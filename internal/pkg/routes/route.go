// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package routes

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/mainflux/mainflux/errors"
	"github.com/nats-io/nats.go"
)

const (
	filePrefix = "msgid"
)

var (
	errBuffering     = errors.New("Message buffering will not work")
	errPublishFailed = errors.New("Failed publishing")
)

// Route - message route, tells which nats topic messages goes to which mqtt topic.
// Later we can add direction and other combination like ( nats-nats).
// Route is used in mfx and plain. Route is like base implementation and mfx and plain
// are extended implementation.
type route struct {
	natsTopic string
	mqttTopic string
	subtopic  string
	mqtt      mqtt.Client
}

type Route interface {
	Consume(m *nats.Msg) ([]byte, error)
	Forward([]byte) errors.Error
	NatsTopic() string
	MqttTopic() string
	Subtopic() string
	Mqtt() mqtt.Client
}

func NewRoute(n, m, s string, mqtt mqtt.Client) Route {
	r := &route{
		natsTopic: n,
		mqttTopic: m,
		subtopic:  s,
		mqtt:      mqtt,
	}
	return r
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

func (r *route) Mqtt() mqtt.Client {
	return r.mqtt
}

func (r *route) Consume(m *nats.Msg) ([]byte, error) {
	return m.Data, nil
}

func (r *route) Forward(bytes []byte) errors.Error {
	if token := r.Mqtt().Publish(r.MqttTopic(), 0, false, bytes); token.Wait() && token.Error() != nil {
		return errPublishFailed
	}
	return nil
}
