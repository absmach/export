// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package export

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/mainflux/export/internal/pkg/messages"
	"github.com/mainflux/export/internal/pkg/routes"
	"github.com/mainflux/export/internal/pkg/routes/mfx"
	"github.com/mainflux/export/pkg/config"
	logger "github.com/mainflux/mainflux/logger"
	nats "github.com/nats-io/nats.go"
)

type Service interface {
	LoadRoutes(queue string)
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
}

const (
	exportGroup = "export-group"
	count       = 100
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

// LoadRoutes method loads route configuration
func (e *exporter) LoadRoutes(queue string) {
	var route routes.Route
	for _, r := range e.Cfg.Routes {
		switch r.Type {
		case "mfx":
			route = mfx.NewRoute(r.NatsTopic, r.MqttTopic, r.SubTopic)
		default:
			route = routes.NewRoute(r.NatsTopic, r.MqttTopic, r.SubTopic)
		}

		if e.validateTopic(route.NatsTopic()) {
			e.Consumers[route.NatsTopic()] = route
			e.Cache.GroupCreate(r.NatsTopic, exportGroup)
		}

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

func (e *exporter) Republish() {
	for _, route := range e.Cfg.Routes {
		streams := []string{route.NatsTopic, "$"}
		go func() {
			for {
				msgs, err := e.Cache.ReadGroup(streams, exportGroup, count, e.ID)
				if err != nil {
					e.Logger.Error(fmt.Sprintf("Failed to read from stream %s", err.Error()))
				}
				e.Logger.Info(fmt.Sprintf("Read %d records from the stream", len(msgs)))
				for _, m := range msgs {
					e.publish(m.Topic, []byte(m.Payload))
				}
			}
		}()
	}
}

func (e *exporter) Subscribe(topic string, nc *nats.Conn) {
	_, err := nc.QueueSubscribe(topic, e.ID, e.Consume)
	if err != nil {
		e.Logger.Error(fmt.Sprintf("Failed to subscribe for NATS %s", topic))
	}
}

func (e *exporter) publish(topic string, payload []byte) error {
	if token := e.Mqtt.Publish(topic, byte(e.Cfg.MQTT.QoS), e.Cfg.MQTT.Retain, payload); token.Wait() && token.Error() != nil {
		e.Logger.Error(fmt.Sprintf("Failed to publish to topic %s", topic))
		return token.Error()
	}
	return nil
}

func (e *exporter) validateTopic(s string) bool {
	return true
}

func (e *exporter) conn(client mqtt.Client) {
	e.Logger.Debug(fmt.Sprintf("Client %s connected", e.ID))
}

func (e *exporter) lost(client mqtt.Client, err error) {
	e.Logger.Debug(fmt.Sprintf("Client %s disconnected", e.ID))
	fmt.Println("lost")
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
