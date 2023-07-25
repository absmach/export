// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package export

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"strings"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/mainflux/export/pkg/config"
	"github.com/mainflux/export/pkg/messages"
	logger "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/messaging"
)

var errNoCacheConfigured = errors.New("No cache configured")

type Exporter interface {
	Start(queue string) errors.Error
	Subscribe(ctx context.Context)
	Logger() logger.Logger
}
type Service interface {
	Exporter
	messages.Publisher
}

var _ Service = (*exporter)(nil)

type exporter struct {
	id        string
	mqtt      mqtt.Client
	cfg       config.Config
	consumers map[string]*Route
	logger    logger.Logger
	pubsub    messaging.PubSub
	sync.RWMutex
}

const (
	exportGroup = "export"
	NatsSub     = "export"
	NatsAll     = ">"
	Channels    = "channels"
	Messages    = "messages"
	svcName     = "export"
)

var errNoRoutesConfigured = errors.New("No routes configured")

// New create new instance of export service.
func New(c config.Config, l logger.Logger, pubsub messaging.PubSub) (Service, error) {
	routes := make(map[string]*Route)
	id := fmt.Sprintf("export-%s", c.MQTT.Username)

	e := exporter{
		id:        id,
		logger:    l,
		cfg:       c,
		consumers: routes,
		pubsub:    pubsub,
	}
	client, err := e.mqttConnect(c, l)
	if err != nil {
		return &e, err
	}
	e.mqtt = client
	return &e, nil
}

// Start method loads route configuration.
func (e *exporter) Start(queue string) errors.Error {
	var route *Route
	for _, r := range e.cfg.Routes {
		route = e.newRoute(r)
		if !e.validateSubject(route.NatsTopic) {
			e.logger.Error("Bad NATS subject:" + route.NatsTopic)
			continue
		}
		e.consumers[route.NatsTopic] = route

	}
	if len(e.consumers) == 0 {
		return errNoRoutesConfigured
	}

	return nil
}

func (e *exporter) Publish(stream, topic string, payload []byte) error {
	if err := e.publish(topic, payload); err != nil {
		return errors.Wrap(errNoCacheConfigured, err)
	}
	return nil
}

func (e *exporter) Logger() logger.Logger {
	return e.logger
}

func (e *exporter) newRoute(r config.Route) *Route {
	return NewRoute(r, e.logger, e)
}

type handleFunc func(msg *messaging.Message) error

func (h handleFunc) Handle(msg *messaging.Message) error {
	return h(msg)

}

func (h handleFunc) Cancel() error {
	return nil
}

func handle(mesChan chan *messaging.Message) handleFunc {
	return func(msg *messaging.Message) error {
		mesChan <- msg
		return nil
	}
}

func (e *exporter) Subscribe(ctx context.Context) {
	for _, r := range e.consumers {
		if err := e.pubsub.Subscribe(ctx, svcName, r.NatsTopic, handle(r.Messages)); err != nil {
			e.logger.Error(fmt.Sprintf("Failed to subscribe to NATS %s: %s", r.NatsTopic, err))
		}
		for i := 0; i < r.Workers; i++ {
			go r.Consume()
		}
	}
}

func (e *exporter) publish(topic string, payload []byte) error {
	token := e.mqtt.Publish(topic, byte(e.cfg.MQTT.QoS), e.cfg.MQTT.Retain, payload)
	if token.Wait() && token.Error() != nil {
		e.logger.Error(fmt.Sprintf("Failed to publish to topic %s", topic))
		return token.Error()
	}
	return nil
}

func (e *exporter) validateSubject(sub string) bool {
	if sub == "" {
		return false
	}
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
	e.logger.Debug(fmt.Sprintf("Client %s connected", e.id))
}

func (e *exporter) lost(client mqtt.Client, err error) {
	e.logger.Debug(fmt.Sprintf("Client %s disconnected", e.id))
}

func (e *exporter) mqttConnect(conf config.Config, logger logger.Logger) (mqtt.Client, error) {
	opts := mqtt.NewClientOptions().
		AddBroker(conf.MQTT.Host).
		SetClientID(e.id).
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
		if conf.MQTT.TLSCert.Certificate != nil {
			cfg.Certificates = []tls.Certificate{conf.MQTT.TLSCert}
		}

		opts.SetTLSConfig(cfg)
		opts.SetProtocolVersion(4)
	}
	client := mqtt.NewClient(opts)
	token := client.Connect()
	token.Wait()

	if token.Error() != nil {
		logger.Error(fmt.Sprintf("Client %s had error connecting to the broker: %v", e.id, token.Error()))
		return nil, token.Error()
	}
	return client, nil
}
