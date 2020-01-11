// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package routes

import (
	"github.com/mainflux/export/internal/pkg/publisher"
	"github.com/mainflux/mainflux/logger"
	"github.com/nats-io/nats.go"
)

// Route - message route, tells which nats topic messages goes to which mqtt topic.
// Later we can add direction and other combination like ( nats-nats).
// Route is used in mfx and plain. Route is like base implementation and mfx and plain
// are extended implemntation.
type Route struct {
	NatsTopic string
	MqttTopic string
	Subtopic  string
	Logger    logger.Logger
	Pub       publisher.File
}

type RouteIF interface {
	Consume(m *nats.Msg)
}

func NewRoute(n, m, s string, pub publisher.File, l logger.Logger) RouteIF {
	return &Route{
		NatsTopic: n,
		MqttTopic: m,
		Subtopic:  s,
		Pub:       pub,
		Logger:    l,
	}
}

func (r Route) Consume(m *nats.Msg) {
	r.Publish(m.Data)
}

func (r Route) Publish(bytes []byte) {
	r.Pub.Publish(bytes, r.MqttTopic)
}
