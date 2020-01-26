// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package export

import (
	"fmt"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/mainflux/export/internal/pkg/messages"
	"github.com/mainflux/export/internal/pkg/routes"
	"github.com/mainflux/export/internal/pkg/routes/mfx"
	"github.com/mainflux/export/pkg/config"
	log "github.com/mainflux/mainflux/logger"
	nats "github.com/nats-io/nats.go"
)

type Service interface {
	LoadRoutes(queue string)
	Subscribe(topic string, queue string, nc *nats.Conn)
}

var _ Service = (*exporter)(nil)

type exporter struct {
	Mqtt      mqtt.Client
	Logger    log.Logger
	Cfg       config.Config
	Consumers map[string]routes.Route
	Cache     messages.Cache
}

// New create new instance of export service
func New(mqtt mqtt.Client, c config.Config, cache messages.Cache, logger log.Logger) Service {
	nats := make(map[string]routes.Route, 0)
	e := exporter{
		Mqtt:      mqtt,
		Logger:    logger,
		Cfg:       c,
		Consumers: nats,
		Cache:     cache,
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
		e.Consumers[route.NatsTopic()] = route
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
	id, err := e.Cache.Add(msg.Subject, payload)
	if err != nil {
		e.Logger.Error(fmt.Sprintf("Failed to add to redis stream `%s`", msg.Subject))
	}

	if token := e.Mqtt.Publish(route.MqttTopic(), 0, false, payload); token.Wait() && token.Error() != nil {
		e.Logger.Error(fmt.Sprintf("Failed to publish to topic %s", route.MqttTopic()))
		return
	}
	e.Cache.Remove(id)
}

func (e *exporter) Subscribe(topic string, queue string, nc *nats.Conn) {
	_, err := nc.QueueSubscribe(topic, queue, e.Consume)
	if err != nil {
		e.Logger.Error(fmt.Sprintf("Failed to subscribe for NATS %s", topic))
	}
}
