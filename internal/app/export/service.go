// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package export

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"strings"
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
	ID        string
	MQTT      mqtt.Client
	Logger    logger.Logger
	Cfg       config.Config
	Consumers map[string]routes.Route
	Cache     messages.Cache
	connected chan bool
	status    uint32
	sync.RWMutex
}

const (
	exportGroup = "export-group"
	count       = 100

	disconnected uint32 = iota
	connected

	NatsSub = "export"
	NatsAll = ">"
)

// New create new instance of export service
func New(c config.Config, cache messages.Cache, l logger.Logger) Service {
	routes := make(map[string]routes.Route)
	id := fmt.Sprintf("export-%s", c.MQTT.Username)

	e := exporter{
		ID:        id,
		Logger:    l,
		Cfg:       c,
		Consumers: routes,
		Cache:     cache,
		connected: make(chan bool, 1),
	}
	client, err := e.mqttConnect(c, l)
	if err != nil {
		log.Fatalf(err.Error())
	}
	e.MQTT = client
	return &e
}

// Start method loads route configuration
func (e *exporter) Start(queue string) {
	var route routes.Route
	for _, r := range e.Cfg.Routes {
		natsTopic := fmt.Sprintf("%s.%s", NatsSub, r.NatsTopic)
		switch r.Type {
		case "mfx":
			route = mfx.NewRoute(natsTopic, r.MqttTopic, r.SubTopic)
		default:
			route = routes.NewRoute(natsTopic, r.MqttTopic, r.SubTopic)
		}
		if !e.validateSubject(route.NatsTopic()) {
			continue
		}
		e.Consumers[route.NatsTopic()] = route
		if e.Cache != nil {
			g, err := e.Cache.GroupCreate(r.NatsTopic, exportGroup)
			if err != nil {
				e.Logger.Error(fmt.Sprintf("Failed to create stream group %s", err.Error()))
			}
			e.Logger.Info(fmt.Sprintf("Stream group %s created %s", r.NatsTopic, g))
		}

	}
	if e.Cache != nil {
		go e.startRepublish()
	}
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

	if err = e.publish(route.MqttTopic(), payload); err != nil {
		if e.Cache == nil {
			return
		}
		// If error occurred we will store data to try to republish
		_, err := e.Cache.Add(msg.Subject, route.MqttTopic(), payload)
		if err != nil {
			e.Logger.Error(fmt.Sprintf("Failed to add to redis stream `%s`", msg.Subject))
		}
	}
}

func (e *exporter) startRepublish() {
	// Initial connection established on start up
	<-e.connected
	e.Logger.Info("Starting republish, waiting for stream data")
	for _, route := range e.Cfg.Routes {
		streams := []string{route.NatsTopic, ">"}
		go func() {
			for {
				msgs, err := e.readMessages(streams)
				if err != nil {
					continue
				}
				e.Logger.Info(fmt.Sprintf("Waiting for connection to %s", e.Cfg.MQTT.Host))
				for {
					// Wait for connection
					if e.IsConnected() || <-e.connected {
						for _, m := range msgs {
							if err := e.publish(m.Topic, []byte(m.Payload)); err != nil {
								e.Logger.Error("Failed to republish message")
							}
						}
						break
					}
				}
			}
		}()
	}
}

func (e *exporter) readMessages(streams []string) (map[string]messages.Msg, error) {
	// Wait for messages in cache, blocking read
	msgs, read, err := e.Cache.ReadGroup(streams, exportGroup, count, e.ID)

	switch err {
	case messages.ErrDecodingData:
		e.Logger.Error(fmt.Sprintf("Failed to decode all data from stream. Read: %d, Failed: %d, Batch: %d.", len(msgs), read, count))
	default:
		e.Logger.Error(fmt.Sprintf("Failed to read from stream %s", err.Error()))
		return nil, err
	}
	e.Logger.Info(fmt.Sprintf("Read %d records from the stream", len(msgs)))
	return msgs, nil
}

func (e *exporter) Subscribe(topic string, nc *nats.Conn) {
	_, err := nc.QueueSubscribe(topic, e.ID, e.Consume)
	if err != nil {
		e.Logger.Error(fmt.Sprintf("Failed to subscribe for NATS %s", topic))
	}
}

func (e *exporter) publish(topic string, payload []byte) error {
	if e.connectionStatus() != connected {
		e.Logger.Error("not connected to mqtt broker")
		return mqtt.ErrNotConnected
	}
	token := e.MQTT.Publish(topic, byte(e.Cfg.MQTT.QoS), e.Cfg.MQTT.Retain, payload)
	if token.Wait() && token.Error() != nil {
		e.Logger.Error(fmt.Sprintf("Failed to publish to topic %s", topic))
		return token.Error()
	}
	return nil
}

func (e *exporter) validateSubject(sub string) bool {
	if strings.ContainsAny(sub, " \t\r\n") {
		return false
	}
	tokens := strings.Split(sub, ".")
	for _, t := range tokens {
		if len(t) == 0 {
			return false
		}
	}
	return true
}

func (e *exporter) conn(client mqtt.Client) {
	e.setConnected(connected)
	e.Logger.Debug(fmt.Sprintf("Client %s connected", e.ID))
}

func (e *exporter) lost(client mqtt.Client, err error) {
	e.setConnected(disconnected)
	e.Logger.Debug(fmt.Sprintf("Client %s disconnected", e.ID))
}

// IsConnected returns a bool signifying whether the client is connected or not.
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
	if e.Cache != nil {
		switch status {
		case connected:
			e.connected <- true
		case disconnected:
			e.connected <- false
		}
	}

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
