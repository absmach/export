// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package export

import (
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/mainflux/export/internal/pkg/routes"
	"github.com/mainflux/export/internal/pkg/routes/mfx"
	"github.com/mainflux/export/pkg/config"
	log "github.com/mainflux/mainflux/logger"
	nats "github.com/nats-io/nats.go"
)

type Service interface {
	Start(queue string, nc *nats.Conn)
}

const heartbeatSubject = "heartbeat"

var _ Service = (*exporter)(nil)

type exporter struct {
	Mqtt   mqtt.Client
	Logger log.Logger
	Cfg    config.Config
	Routes []routes.Route
}

// New create new instance of export service
func New(mqtt mqtt.Client, c config.Config, logger log.Logger) Service {
	routes := make([]routes.Route, 0)
	e := exporter{
		Mqtt:   mqtt,
		Logger: logger,
		Cfg:    c,
		Routes: routes,
	}

	return &e
}

// Start method starts consuming messages received from NATS
// and makes routes according to the configuration file.
// Routes export messages to mqtt.
func (e *exporter) Start(queue string, nc *nats.Conn) {
	var route routes.Route
	for _, r := range e.Cfg.Routes {
		switch r.Type {
		case "mfx":
			route = mfx.NewRoute(r.NatsTopic, r.MqttTopic, r.SubTopic, e.Mqtt, e.Logger)
		default:
			route = routes.NewRoute(r.NatsTopic, r.MqttTopic, r.SubTopic, e.Mqtt, e.Logger)
		}
		e.Routes = append(e.Routes, route)
		e.Subscribe(route, queue, nc)
	}
	go e.heartbeat(nc)
}

func (e *exporter) heartbeat(nc *nats.Conn) {
	// Publish heartbeat
	ticker := time.NewTicker(10000 * time.Millisecond)
	go func() {
		for {
			select {
			case <-ticker.C:
				subject := fmt.Sprintf("%s.%s", heartbeatSubject, "export")
				nc.Publish(subject, []byte{})
			}
		}
	}()
}

func (e *exporter) Subscribe(r routes.Route, queue string, nc *nats.Conn) {
	_, err := nc.QueueSubscribe(r.NatsTopic(), fmt.Sprintf("%s-%s", queue, r.NatsTopic()), r.Consume)
	if err != nil {
		e.Logger.Error(fmt.Sprintf("Failed to subscribe for NATS/MQTT %s/%s", r.NatsTopic(), r.MqttTopic()))
	}
}
