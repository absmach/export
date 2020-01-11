// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package export

import (
	"fmt"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/mainflux/export/internal/pkg/routes"
	"github.com/mainflux/export/internal/pkg/routes/mfx"
	"github.com/mainflux/export/pkg/config"
	log "github.com/mainflux/mainflux/logger"
	nats "github.com/nats-io/nats.go"
)

type Service interface {
	Start(string)
}

var _ Service = (*exporter)(nil)

type exporter struct {
	NatsConn *nats.Conn
	Mqtt     mqtt.Client
	Logger   log.Logger
	Cfg      config.Config
	Routes   []routes.RouteIF
}

// New create new instance of export service
func New(nc *nats.Conn, mqtt mqtt.Client, c config.Config, logger log.Logger) Service {
	routes := make([]routes.RouteIF, 0)
	e := exporter{
		NatsConn: nc,
		Mqtt:     mqtt,
		Logger:   logger,
		Cfg:      c,
		Routes:   routes,
	}

	return &e
}

// Start method starts consuming messages received from NATS.
// and makes routes according to the configuration file.
// Routes export messages to mqtt.
func (e *exporter) Start(queue string) {
	var route routes.RouteIF
	for _, r := range e.Cfg.Routes {
		switch r.Type {
		case "mfx":
			route = mfx.NewRoute(r.NatsTopic, r.MqttTopic, r.SubTopic, e.Mqtt, e.Logger)
		default:
			route = routes.NewRoute(r.NatsTopic, r.MqttTopic, r.SubTopic, e.Mqtt, e.Logger)
		}
		e.Routes = append(e.Routes, route)
		_, err := e.NatsConn.QueueSubscribe(r.NatsTopic, fmt.Sprintf("%s-%s", queue, r.NatsTopic), route.Consume)
		if err != nil {
			e.Logger.Error(fmt.Sprintf("Failed to subscribe for NATS/MQTT %s/%s", r.NatsTopic, r.MqttTopic))
		}

	}
}
