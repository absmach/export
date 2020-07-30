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
	cache     messages.Cache
	logger    logger.Logger
	connected chan bool
	status    uint32
	sync.RWMutex
}

const (
	exportGroup = "export"
	count       = 100

	disconnected uint32 = iota
	connected

	NatsSub  = "export"
	NatsAll  = ">"
	Channels = "channels"
	Messages = "messages"
)

var errNoRoutesConfigured = errors.New("No routes configured")

// New create new instance of export service
func New(c config.Config, cache messages.Cache, l logger.Logger) (Service, error) {
	routes := make(map[string]Route)
	id := fmt.Sprintf("export-%s", c.MQTT.Username)

	e := exporter{
		id:        id,
		logger:    l,
		cfg:       c,
		consumers: routes,
		cache:     cache,
		connected: make(chan bool, 1),
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
		if e.cache != nil {
			g, err := e.cache.GroupCreate(r.NatsTopic, exportGroup)
			if err != nil {
				e.logger.Error(fmt.Sprintf("Failed to create stream group: %s", err))
			}
			e.logger.Info(fmt.Sprintf("Stream group %s created %s", r.NatsTopic, g))
		}

	}
	if len(e.consumers) == 0 {
		return errNoRoutesConfigured
	}
	if e.cache != nil {
		go e.startRepublish()
	}
	return nil
}

func (e *exporter) Publish(stream, topic string, payload []byte) error {
	if err := e.publish(topic, payload); err != nil {
		if e.cache == nil {
			return errors.Wrap(errNoCacheConfigured, err)
		}
		// If error occurred and cache is being used
		// we will store data to try to republish later
		_, err = e.cache.Add(stream, topic, payload)
		if err != nil {
			e.logger.Error(fmt.Sprintf("%s `%s`", errFailedToAddToStream.Error(), stream))
			return errors.Wrap(errFailedToAddToStream, err)
		}
	}
	return nil
}

func (e *exporter) Logger() logger.Logger {
	return e.logger
}

func (e *exporter) newRoute(r config.Route) Route {
	return NewRoute(r, e.logger, e)
}

func (e *exporter) startRepublish() {
	// Initial connection established on start up
	<-e.connected
	for _, route := range e.cfg.Routes {
		stream := []string{route.NatsTopic, NatsAll}
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
		e.logger.Info(fmt.Sprintf("Waiting for connection to %s", e.cfg.MQTT.Host))
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
	msgs, read, err := e.cache.ReadGroup(streams, exportGroup, count, e.id)

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
	if e.connectionStatus() != connected {
		e.logger.Error("not connected to mqtt broker")
		return mqtt.ErrNotConnected
	}
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
	e.setConnected(connected)
	e.logger.Debug(fmt.Sprintf("Client %s connected", e.id))
}

func (e *exporter) lost(client mqtt.Client, err error) {
	e.setConnected(disconnected)
	e.logger.Debug(fmt.Sprintf("Client %s disconnected", e.id))
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
	e.status = status
	if e.cache != nil {
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
