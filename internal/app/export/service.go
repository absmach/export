// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package export

import (
	"fmt"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/mainflux/export/internal/pkg/routes"
	"github.com/mainflux/export/internal/pkg/routes/dflt"
	"github.com/mainflux/export/internal/pkg/routes/mflx"
	"github.com/mainflux/export/pkg/config"
	log "github.com/mainflux/mainflux/logger"
	nats "github.com/nats-io/nats.go"
)

type Service interface {
	Start(string)
	Publish(b []byte, topic string) error
	Connect()
	Disconnect()
	IsConnected() bool
}

var _ Service = (*Exporter)(nil)

type Exporter struct {
	Name      string
	Nc        *nats.Conn
	Mqtt      mqtt.Client
	Logger    log.Logger
	Cfg       config.Config
	Routes    []routes.Route
	Connected bool
	mu        sync.Mutex
}

// New create new instance of export service
func New(nc *nats.Conn, mqtt mqtt.Client, c config.Config, logger log.Logger) Service {
	routes := make([]routes.Route, 0)
	e := Exporter{
		Nc:     nc,
		Mqtt:   mqtt,
		Logger: logger,
		Cfg:    c,
		Routes: routes,
		mu:     sync.Mutex{},
	}

	return &e
}

// Start method starts consuming messages received from NATS.
// and makes routes according to the configuration file.
// Routes export messages to mqtt.
func (e *Exporter) Start(queue string) {
	var route routes.Route
	for _, r := range e.Cfg.Routes {
		switch r.Type {
		case "mflx":
			route = mflx.New(e, r.NatsTopic, r.MqttTopic, r.SubTopic, e.Mqtt, e.Logger)
		default:
			route = dflt.New(e, r.NatsTopic, r.MqttTopic, r.SubTopic, e.Mqtt, e.Logger)
		}
		e.Routes = append(e.Routes, route)
		_, err := e.Nc.QueueSubscribe(r.NatsTopic, fmt.Sprintf("%s-%s", queue, r.NatsTopic), route.Consume)
		if err != nil {
			e.Logger.Error(fmt.Sprintf("Failed to subscribe for NATS/MQTT %s/%s", r.NatsTopic, r.MqttTopic))
		}

	}
}
func (e *Exporter) Publish(b []byte, topic string) error {
	return e.Publish(b, topic)
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
