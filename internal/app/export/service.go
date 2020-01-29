// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package export

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/mainflux/export/internal/pkg/messages"
	"github.com/mainflux/export/internal/pkg/routes"
	"github.com/mainflux/export/internal/pkg/routes/mfx"
	"github.com/mainflux/export/pkg/config"
	logger "github.com/mainflux/mainflux/logger"
	nats "github.com/nats-io/nats.go"
)

type Service interface {
	Start(queue string)
	Subscribe(topic string, nc *nats.Conn)
}

var _ Service = (*exporter)(nil)

type exporter struct {
	ID           string
	Mqtt         mqtt.Client
	Logger       logger.Logger
	Cfg          config.Config
	Consumers    map[string]routes.Route
	Cache        messages.Cache
	publishing   chan bool
	disconnected chan bool
	status       uint32
	sync.RWMutex
}

const (
	exportGroup = "export-group"
	count       = 100

	disconnected uint32 = iota
	connecting
	reconnecting
	connected
)

// New create new instance of export service
func New(c config.Config, cache messages.Cache, l logger.Logger) Service {
	nats := make(map[string]routes.Route, 0)
	id := fmt.Sprintf("export-%s", c.MQTT.Username)

	e := exporter{
		ID:           id,
		Logger:       l,
		Cfg:          c,
		Consumers:    nats,
		Cache:        cache,
		publishing:   make(chan bool),
		disconnected: make(chan bool),
	}
	client, err := e.mqttConnect(c, l)
	if err != nil {
		log.Fatalf(err.Error())
	}
	e.Mqtt = client
	return &e
}

// Start method loads route configuration
func (e *exporter) Start(queue string) {
	var route routes.Route

	msgs, err := e.Cache.Read([]string{"st"}, 1, e.ID)
	if err != nil {
		return
	}
	for _, m := range msgs {
		b, _ := json.Marshal(m)
		fmt.Println(string(b))
	}

	for _, r := range e.Cfg.Routes {
		switch r.Type {
		case "mfx":
			route = mfx.NewRoute(r.NatsTopic, r.MqttTopic, r.SubTopic)
		default:
			route = routes.NewRoute(r.NatsTopic, r.MqttTopic, r.SubTopic)
		}
		e.Consumers[route.NatsTopic()] = route

		res, err := e.Cache.GroupCreate(r.NatsTopic, exportGroup)
		if err != nil {
			e.Logger.Error(fmt.Sprintf("Failed to create stream %s - err: %s", r.NatsTopic, err.Error()))
		}
		e.Logger.Info(fmt.Sprintf("group created %s", res))
	}

	e.startRepublisher()
}

func (e *exporter) Consume(msg *nats.Msg) {
	if _, ok := e.Consumers[msg.Subject]; !ok {
		e.Logger.Info(fmt.Sprintf("no configuration for nats topic %s", msg.Subject))
		return
	}

	route := e.Consumers[msg.Subject]
	payload, err := route.Consume(msg)
	if err != nil {
		e.Logger.Error(fmt.Sprintf("Failed to consume msg %s", err.Error()))
	}
	id, err := e.Cache.Add(msg.Subject, route.MqttTopic(), payload)
	if err != nil {
		e.Logger.Error(fmt.Sprintf("Failed to add to redis stream `%s`", msg.Subject))
	}

	if err = e.publish(route.MqttTopic(), payload); err != nil {
		return
	}

	i, err := e.Cache.Remove(route.NatsTopic(), id)
	if err != nil {
		e.Logger.Error(fmt.Sprintf("Failed to remove %s - %s", id, err.Error()))
	}
	e.Logger.Info(fmt.Sprintf("Entry %s removed %d", id, i))
}

func (e *exporter) startRepublisher() {
	streams := []string{}
	for _, route := range e.Cfg.Routes {
		streams = append(streams, route.NatsTopic)
	}
	<-e.publishing
	go func() {
		for {
			e.Logger.Info("Waiting for disconnect")
			<-e.disconnected
			e.Logger.Info("Waiting to get online to republish")
			<-e.publishing
			e.Logger.Info("Start republishing")
			for {
				msgs, err := e.Cache.Read(streams, count, e.ID)
				if err != nil {
					e.Logger.Error(fmt.Sprintf("Failed to read from stream %s", err.Error()))
				}
				e.Logger.Info(fmt.Sprintf("Read %d records from the stream", len(msgs)))

				for _, m := range msgs {
					b, _ := json.Marshal(m)
					fmt.Println(string(b))
				}

				if len(msgs) < count {
					break
				}
			}
		}
	}()
}

func (e *exporter) Subscribe(topic string, nc *nats.Conn) {
	_, err := nc.QueueSubscribe(topic, e.ID, e.Consume)
	if err != nil {
		e.Logger.Error(fmt.Sprintf("Failed to subscribe for NATS %s", topic))
	}
}

func (e *exporter) publish(topic string, payload []byte) error {
	if e.connectionStatus() == connected {
		token := e.Mqtt.Publish(topic, byte(e.Cfg.MQTT.QoS), e.Cfg.MQTT.Retain, payload)
		if token.Wait() && token.Error() != nil {
			e.Logger.Error(fmt.Sprintf("Failed to publish to topic %s", topic))
			return token.Error()
		}
		return nil
	}
	return mqtt.ErrNotConnected
}

func (e *exporter) conn(client mqtt.Client) {
	e.setConnected(connected)
	e.Logger.Debug(fmt.Sprintf("Client %s connected", e.ID))
	e.publishing <- true
}

func (e *exporter) lost(client mqtt.Client, err error) {
	e.setConnected(disconnected)
	e.Logger.Debug(fmt.Sprintf("Client %s disconnected", e.ID))
	fmt.Println("lost")
	e.disconnected <- true
}

// IsConnected returns a bool signifying whether
// the client is connected or not.
func (e *exporter) IsConnected() bool {
	e.RLock()
	defer e.RUnlock()
	status := atomic.LoadUint32(&e.status)
	switch {
	case status == connected:
		return true
	default:
		return false
	}
}

func (e *exporter) connectionStatus() uint32 {
	e.RLock()
	defer e.RUnlock()
	status := atomic.LoadUint32(&e.status)
	return status
}

func (e *exporter) setConnected(status uint32) {
	e.Lock()
	defer e.Unlock()
	atomic.StoreUint32(&e.status, uint32(status))
}

func (e *exporter) mqttConnect(conf config.Config, logger logger.Logger) (mqtt.Client, error) {

	opts := mqtt.NewClientOptions().
		AddBroker(conf.MQTT.Host).
		SetClientID(e.ID).
		SetCleanSession(true).
		SetAutoReconnect(true).
		SetOnConnectHandler(e.conn).
		SetConnectionLostHandler(e.lost)

	if conf.MQTT.Username != "" && conf.MQTT.Password != "" {
		opts.SetUsername(conf.MQTT.Username)
		opts.SetPassword(conf.MQTT.Password)
	}

	if conf.MQTT.MTLS {
		cfg := &tls.Config{
			InsecureSkipVerify: conf.MQTT.SkipTLSVer,
		}

		if conf.MQTT.CA != nil {
			cfg.RootCAs = x509.NewCertPool()
			cfg.RootCAs.AppendCertsFromPEM(conf.MQTT.CA)
		}
		if conf.MQTT.Cert.Certificate != nil {
			cfg.Certificates = []tls.Certificate{conf.MQTT.Cert}
		}

		cfg.BuildNameToCertificate()
		opts.SetTLSConfig(cfg)
		opts.SetProtocolVersion(4)
	}
	client := mqtt.NewClient(opts)
	token := client.Connect()
	token.Wait()

	if token.Error() != nil {
		logger.Error(fmt.Sprintf("Client %s had error connecting to the broker: %s\n", e.ID, token.Error().Error()))
		return nil, token.Error()
	}
	return client, nil
}
