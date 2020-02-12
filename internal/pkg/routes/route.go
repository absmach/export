// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package routes

import (
	"fmt"
	"strings"

	"github.com/mainflux/export/internal/app/export/publish"
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
	publisher publish.Publisher
	logger    logger.Logger
	pub       publish.Publisher
}

type Route interface {
	Consume(*nats.Msg)
	Process(data []byte) ([]byte, error)
	NatsTopic() string
	MqttTopic() string
	Subtopic() string
}

func NewRoute(n, m, s string, log logger.Logger, pub publish.Publisher) Route {
	r := route{
		natsTopic: n,
		mqttTopic: m,
		subtopic:  s,
		logger:    log,
		pub:       pub,
	}
	return r
}

func (r route) NatsTopic() string {
	return r.natsTopic
}

func (r route) MqttTopic() string {
	return r.mqttTopic
}

func (r route) Subtopic() string {
	return r.subtopic
}

func (r route) Process(data []byte) ([]byte, error) {
	return data, nil
}

func (r route) Consume(msg *nats.Msg) {
	payload, err := r.Process(msg.Data)
	if err != nil {
		r.logger.Error(fmt.Sprintf("Failed to consume msg %s", err.Error()))
	}
	topic := fmt.Sprintf("%s/%s", r.MqttTopic(), strings.ReplaceAll(msg.Subject, ".", "/"))
	r.pub.Publish(msg.Subject, topic, payload)
}
