// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package export

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/mainflux/export/internal/pkg/publisher"
	"github.com/mainflux/export/internal/pkg/routes"
	"github.com/mainflux/export/internal/pkg/routes/mfx"
	"github.com/mainflux/export/pkg/config"
	"github.com/mainflux/mainflux/logger"
	nats "github.com/nats-io/nats.go"
)

type Service interface {
	Start(string)
	Connect()
	Disconnect()
	IsConnected() bool
}

var _ Service = (*Exporter)(nil)

type Exporter struct {
	NatsConn  *nats.Conn
	Mqtt      mqtt.Client
	Logger    logger.Logger
	Cfg       config.Config
	Routes    []routes.RouteIF
	Pub       publisher.File
	Connected bool
	mu        sync.Mutex
}

// New create new instance of export service
func New(nc *nats.Conn, c config.Config, logger logger.Logger) (Service, error) {
	routes := make([]routes.RouteIF, 0)

	e := Exporter{
		NatsConn: nc,
		Logger:   logger,
		Cfg:      c,
		Routes:   routes,
	}

	_, err := e.mqttConnect(c, logger)
	if err != nil {
		return nil, err
	}

	pub, err := publisher.NewPublisher("store", e.Mqtt)
	if err != nil {
		return nil, err
	}
	e.Pub = pub
	return &e, nil
}

func (e *Exporter) mqttConnect(conf config.Config, logger logger.Logger) (mqtt.Client, error) {
	conn := func(client mqtt.Client) {
		e.Logger.Info(fmt.Sprintf("Client %s connected", "EXPORT"))
		e.Connect()
	}

	lost := func(client mqtt.Client, err error) {
		e.Logger.Info(fmt.Sprintf("Client %s disconnected", "EXPORT"))
		e.Disconnect()
	}

	opts := mqtt.NewClientOptions().
		AddBroker(conf.MQTT.Host).
		SetClientID("EXPORT").
		SetCleanSession(true).
		SetAutoReconnect(true).
		SetOnConnectHandler(conn).
		SetConnectionLostHandler(lost)

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
		e.Logger.Error(fmt.Sprintf("Client %s had error connecting to the broker: %s\n", "EXPORT", token.Error().Error()))
		return nil, token.Error()
	}
	e.Mqtt = client
	return client, nil
}

// Start method starts consuming messages received from NATS
// and makes routes according to the configuration file.
// Routes export messages to mqtt.
func (e *Exporter) Start(queue string) {
	var route routes.RouteIF

	for _, r := range e.Cfg.Routes {
		switch r.Type {
		case "mfx":
			route = mfx.NewRoute(r.NatsTopic, r.MqttTopic, r.SubTopic, e.Pub, e.Logger)
		default:
			route = routes.NewRoute(r.NatsTopic, r.MqttTopic, r.SubTopic, e.Pub, e.Logger)
		}
		e.Routes = append(e.Routes, route)
		_, err := e.NatsConn.QueueSubscribe(r.NatsTopic, fmt.Sprintf("%s-%s", queue, r.NatsTopic), route.Consume)
		if err != nil {
			e.Logger.Error(fmt.Sprintf("Failed to subscribe for NATS/MQTT %s/%s", r.NatsTopic, r.MqttTopic))
		}
	}
	return
}

func (e *Exporter) IsConnected() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	r := e.Connected
	return r
}

func (e *Exporter) Connect() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.Connected = true

}

func (e *Exporter) Disconnect() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.Connected = false
}
