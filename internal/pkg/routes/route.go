// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package routes

import (
	"fmt"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/mainflux/mainflux/logger"
	"github.com/nats-io/nats.go"
)

// Route - message route, tells which nats topic messages goes to which mqtt topic.
// Later we can add direction and other combination like ( nats-nats).
// Route is used in mfx and plain. Route is like base implementation and mfx and plain
// are extended implementation.
type route struct {
	natsTopic string
	mqttTopic string
	subtopic  string
	logger    logger.Logger
	mqtt      mqtt.Client
}

type Route interface {
	Consume(m *nats.Msg)
	Forward([]byte)
	NatsTopic() string
	MqttTopic() string
	Subtopic() string
	Logger() logger.Logger
	Mqtt() mqtt.Client
}

func NewRoute(n, m, s string, mqtt mqtt.Client, l logger.Logger) Route {
	return &route{
		natsTopic: n,
		mqttTopic: m,
		subtopic:  s,
		mqtt:      mqtt,
		logger:    l,
	}
}

func (r *route) NatsTopic() string {
	return r.natsTopic
}

func (r *route) MqttTopic() string {
	return r.mqttTopic
}

func (r *route) Subtopic() string {
	return r.subtopic
}

func (r *route) Logger() logger.Logger {
	return r.logger
}

func (r *route) Mqtt() mqtt.Client {
	return r.mqtt
}

func (r *route) Consume(m *nats.Msg) {
	r.Forward(m.Data)
}

func (r *route) Forward(bytes []byte) {
	if token := r.Mqtt().Publish(r.MqttTopic(), 0, false, bytes); token.Wait() && token.Error() != nil {
		r.Logger().Error(fmt.Sprintf("Failed to publish message on topic %s : %s", r.MqttTopic(), token.Error()))
	}
}
