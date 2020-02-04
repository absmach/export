// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package routes

import (
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
}

type Route interface {
	Consume(m *nats.Msg) ([]byte, error)
	NatsTopic() string
	MqttTopic() string
	Subtopic() string
}

func NewRoute(n, m, s string) Route {
	r := &route{
		natsTopic: n,
		mqttTopic: m,
		subtopic:  s,
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

func (r *route) Consume(m *nats.Msg) ([]byte, error) {
	return m.Data, nil
}
