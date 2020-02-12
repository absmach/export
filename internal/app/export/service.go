// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package export

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/mainflux/export/internal/pkg/messages"
	"github.com/mainflux/export/internal/pkg/routes"
	"github.com/mainflux/export/internal/pkg/routes/mfx"
	"github.com/mainflux/export/pkg/config"
	logger "github.com/mainflux/mainflux/logger"
	nats "github.com/nats-io/nats.go"
)

type Service interface {
	Start(queue string) error
	Subscribe(topic string, nc *nats.Conn)
	Logger() logger.Logger
}

var _ Service = (*exporter)(nil)

type exporter struct {
	ID        string
	MQTT      mqtt.Client
	Cfg       config.Config
	Consumers map[string]routes.Route
	Cache     messages.Cache
	logger    logger.Logger
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

var errNoRoutesConfigured = errors.New("No routes configured")

// New create new instance of export service
func New(c config.Config, cache messages.Cache, l logger.Logger) Service {
	routes := make(map[string]routes.Route)
	id := fmt.Sprintf("export-%s", c.MQTT.Username)

	e := exporter{
		ID:        id,
		logger:    l,
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
func (e *exporter) Start(queue string) error {
	var route routes.Route
	for _, r := range e.Cfg.Routes {
		natsTopic := fmt.Sprintf("%s.%s", NatsSub, r.NatsTopic)
		switch r.Type {
		case "mfx":
			route = mfx.NewRoute(natsTopic, r.MqttTopic, r.SubTopic, e.logger, e)
		default:
			route = routes.NewRoute(natsTopic, r.MqttTopic, r.SubTopic, e.logger, e)
		}
		if !e.validateSubject(route.NatsTopic()) {
			continue
		}
		e.Consumers[route.NatsTopic()] = route
		if e.Cache != nil {
			g, err := e.Cache.GroupCreate(r.NatsTopic, exportGroup)
			if err != nil {
				e.logger.Error(fmt.Sprintf("Failed to create stream group %s", err.Error()))
			}
			e.logger.Info(fmt.Sprintf("Stream group %s created %s", r.NatsTopic, g))
		}

	}
	if len(e.Consumers) == 0 {
		return errNoRoutesConfigured
	}
	if e.Cache != nil {
		go e.startRepublish()
	}
	return nil
}

func (e *exporter) Publish(subject, topic string, payload []byte) {
	if err := e.publish(topic, payload); err == nil {
		return
	}

	if e.Cache == nil {
		return
	}
	// If error occurred we will store data to try to republish
	_, err := e.Cache.Add(subject, topic, payload)
	if err != nil {
		e.logger.Error(fmt.Sprintf("Failed to add to redis stream `%s`", subject))
	}
}

func (e *exporter) Logger() logger.Logger {
	return e.logger
}

func (e *exporter) startRepublish() {
	// Initial connection established on start up
	<-e.connected
	for _, route := range e.Cfg.Routes {
		stream := []string{route.NatsTopic, ">"}
		go e.republish(stream)
	}
}

func (e *exporter) republish(stream []string) {
	for {
		e.logger.Info("Republish, waiting for stream data")
		msgs, err := e.readMessages(stream)
		if err != nil {
			continue
		}
		e.logger.Info(fmt.Sprintf("Waiting for connection to %s", e.Cfg.MQTT.Host))
		for {
			// Wait for connection
			if e.IsConnected() || <-e.connected {
				for _, m := range msgs {
					if err := e.publish(m.Topic, []byte(m.Payload)); err != nil {
						e.logger.Error("Failed to republish message")
					}
				}
				break
			}
		}
	}
}

func (e *exporter) readMessages(streams []string) (map[string]messages.Msg, error) {
	// Wait for messages in cache, blocking read
	msgs, read, err := e.Cache.ReadGroup(streams, exportGroup, count, e.ID)

	switch err {
	case messages.ErrDecodingData:
		e.logger.Error(fmt.Sprintf("Failed to decode all data from stream. Read: %d, Failed: %d, Batch: %d.", len(msgs), read, count))
	default:
		e.logger.Error(fmt.Sprintf("Failed to read from stream %s", err.Error()))
		return nil, err
	}
	e.logger.Info(fmt.Sprintf("Read %d records from the stream", len(msgs)))
	return msgs, nil
}

func (e *exporter) Subscribe(topic string, nc *nats.Conn) {
	for _, r := range e.Consumers {
		_, err := nc.QueueSubscribe(topic, e.ID, r.Consume)
		if err != nil {
			e.logger.Error(fmt.Sprintf("Failed to subscribe for NATS %s", topic))
		}
	}
}

func (e *exporter) publish(topic string, payload []byte) error {
	if e.connectionStatus() != connected {
		e.logger.Error("not connected to mqtt broker")
		return mqtt.ErrNotConnected
	}
	token := e.MQTT.Publish(topic, byte(e.Cfg.MQTT.QoS), e.Cfg.MQTT.Retain, payload)
	if token.Wait() && token.Error() != nil {
		e.logger.Error(fmt.Sprintf("Failed to publish to topic %s", topic))
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
	e.logger.Debug(fmt.Sprintf("Client %s connected", e.ID))
}

func (e *exporter) lost(client mqtt.Client, err error) {
	e.setConnected(disconnected)
	e.logger.Debug(fmt.Sprintf("Client %s disconnected", e.ID))
}

// IsConnected returns a bool signifying whether the client is connected or not.
func (e *exporter) IsConnected() bool {
	e.RLock()
	defer e.RUnlock()
	switch {
	case e.status == connected:
		return true
	default:
		return false
	}
}

func (e *exporter) connectionStatus() uint32 {
	e.RLock()
	defer e.RUnlock()
	return e.status
}

func (e *exporter) setConnected(status uint32) {
	e.Lock()
	defer e.Unlock()
	if e.Cache != nil {
		switch e.status {
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
