// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package export

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"strings"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/mainflux/export/pkg/config"
	"github.com/mainflux/export/pkg/messages"
	"github.com/mainflux/mainflux/errors"
	logger "github.com/mainflux/mainflux/logger"
	nats "github.com/nats-io/nats.go"
)

var (
	errNoCacheConfigured   = errors.New("No cache configured")
	errFailedToAddToStream = errors.New("Failed to add to redis stream")
)

type Exporter interface {
	Start(queue string) errors.Error
	Subscribe(nc *nats.Conn)
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
	consumers map[string]Route
	logger    logger.Logger
	connected chan bool
	status    uint32
	sync.RWMutex
}

const (
	disconnected uint32 = iota
	connected
	exportGroup = "export"
	count       = 100

	NatsSub  = "export"
	NatsAll  = ">"
	Channels = "channels"
	Messages = "messages"
)

var errNoRoutesConfigured = errors.New("No routes configured")

// New create new instance of export service
func New(c config.Config, l logger.Logger) (Service, error) {
	routes := make(map[string]Route)
	id := fmt.Sprintf("export-%s", c.MQTT.Username)

	e := exporter{
		id:        id,
		logger:    l,
		cfg:       c,
		consumers: routes,
	}
	client, err := e.mqttConnect(c, l)
	if err != nil {
		return &e, err
	}
	e.mqtt = client
	return &e, nil
}

// Start method loads route configuration
func (e *exporter) Start(queue string) errors.Error {
	var route Route
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

func (e *exporter) newRoute(r config.Route) Route {
	return NewRoute(r, e.logger, e)
}

func (e *exporter) Subscribe(nc *nats.Conn) {
	for _, r := range e.consumers {
		_, err := nc.ChanQueueSubscribe(r.NatsTopic, exportGroup, r.Messages)
		if err != nil {
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

		cfg.BuildNameToCertificate()
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
