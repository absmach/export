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
	ID        string
	Mqtt      mqtt.Client
	Logger    log.Logger
	Cfg       config.Config
	Consumers map[string]routes.Route
	Cache     messages.Cache
}

// New create new instance of export service
func New(mqtt mqtt.Client, c config.Config, cache messages.Cache, logger log.Logger) Service {
	nats := make(map[string]routes.Route, 0)
	id := fmt.Sprintf("export-%s", c.MQTT.Username)
	e := exporter{
		ID:        id,
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

		e.Cache.GroupCreate(r.NatsTopic, "export-group")
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
	if token := e.Mqtt.Publish(route.MqttTopic(), 0, false, payload); token.Wait() && token.Error() != nil {
		e.Logger.Error(fmt.Sprintf("Failed to publish to topic %s", route.MqttTopic()))
		return
	}
	if err = e.publish(route.MqttTopic(), payload); err == nil {
		return
	}

	e.Cache.Remove(id)
}

func (e *exporter) Republish() {
	streams := []string{}
	for _, route := range e.Cfg.Routes {
		streams = append(streams, route.NatsTopic)
	}

	xStreams, err := e.Cache.ReadGroup(streams, "export-group", e.ID)
	if err != nil {
		e.Logger.Error(fmt.Sprintf("Failed to republish %s", err.Error()))
	}
	for _, xStream := range xStreams { //Get individual xStream
		//streamName := xStream.Stream
		for _, xMessage := range xStream.Messages { // Get the message from the xStream
			for _, v := range xMessage.Values { // Get the values from the message
				fmt.Println("test:%s", v)
			}

		}
	}
}

func (e *exporter) Subscribe(topic string, queue string, nc *nats.Conn) {
	_, err := nc.QueueSubscribe(topic, queue, e.Consume)
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
