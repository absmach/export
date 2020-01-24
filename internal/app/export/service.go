// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package export

import (
	"fmt"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/go-redis/redis"
	"github.com/mainflux/export/internal/pkg/messages"
	"github.com/mainflux/export/internal/pkg/routes"
	"github.com/mainflux/export/internal/pkg/routes/mfx"
	"github.com/mainflux/export/pkg/config"
	log "github.com/mainflux/mainflux/logger"
	nats "github.com/nats-io/nats.go"
)

type Service interface {
	LoadRoutes(queue string, nc *nats.Conn, cacheClient *redis.Client)
}

var _ Service = (*exporter)(nil)

type exporter struct {
	Mqtt   mqtt.Client
	Logger log.Logger
	Cfg    config.Config
	Routes map[string]routes.Route
	Cache  messages.Cache
}

// New create new instance of export service
func New(mqtt mqtt.Client, c config.Config, logger log.Logger) Service {
	routes := make(map[string]routes.Route, 0)
	e := exporter{
		Mqtt:   mqtt,
		Logger: logger,
		Cfg:    c,
		Routes: routes,
	}
	return &e
}

// LoadRoutes method loads route configuration
func (e *exporter) LoadRoutes(queue string) {
	var route routes.Route
	for _, r := range e.Cfg.Routes {
		switch r.Type {
		case "mfx":
			route = mfx.NewRoute(r.NatsTopic, r.MqttTopic, r.SubTopic, e.Mqtt)
		default:
			route = routes.NewRoute(r.NatsTopic, r.MqttTopic, r.SubTopic, e.Mqtt)
		}
		e.Routes[route.NatsTopic()] = route
	}
}

func (e *exporter) Consume(msg *nats.Msg) {
	if _, ok := e.Routes[msg.Subject]; !ok {
		e.Logger.Info(fmt.Sprintf("no configuration for nats topic %s"))
		return
	}

	route := e.Routes[msg.Subject]
	payload, err := route.Consume(msg)
	if err != nil {
		e.Logger.Error(fmt.Sprintf("Failed to consume msg %s", err.Error()))
	}
	Cache.Add(msg.Subject, payload)

}

func (e *exporter) Subscribe(r routes.Route, queue string, nc *nats.Conn) {
	_, err := nc.QueueSubscribe(r.NatsTopic(), fmt.Sprintf("%s-%s", queue, r.NatsTopic()), r.Consume)
	if err != nil {
		e.Logger.Error(fmt.Sprintf("Failed to subscribe for NATS/MQTT %s/%s", r.NatsTopic(), r.MqttTopic()))
	}
}
