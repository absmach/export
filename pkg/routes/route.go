// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package routes

import (
	"fmt"
	"strings"

	"github.com/mainflux/export/pkg/messages"
	"github.com/mainflux/mainflux/logger"
	nats "github.com/nats-io/nats.go"
)

const (
	// Number of the workers depends on the connection capacity
	// as well as on payload size that needs to be sent.
	// Number of workers also determines the size of the buffer
	// that recieves messages from NATS.
	// For regular telemetry SenML messages 10 workers is enough.
	workers = 10
)

// Route - message route, tells which nats topic messages goes to which mqtt topic.
// Later we can add direction and other combination like ( nats-nats).
// Route is used in mfx and plain. route.go has base implementation and mfx.go
// has extended implementation.

type Route struct {
	NatsTopic string
	MqttTopic string
	Subtopic  string
	Messages  chan *nats.Msg
	Workers   int
	logger    logger.Logger
	pub       messages.Publisher
}

func NewRoute(n, m, s string, w int, log logger.Logger, pub messages.Publisher) Route {
	if w == 0 {
		w = workers
	}
	r := Route{
		NatsTopic: n,
		MqttTopic: m,
		Subtopic:  s,
		Messages:  make(chan *nats.Msg, w),
		Workers:   w,
		logger:    log,
		pub:       pub,
	}
	return r
}

func (r *Route) Process(data []byte) ([]byte, error) {
	return data, nil
}

func (r *Route) Consume() {
	for msg := range r.Messages {
		payload, err := r.Process(msg.Data)
		if err != nil {
			r.logger.Error(fmt.Sprintf("Failed to consume message %s", err))
		}
		topic := fmt.Sprintf("%s/%s", r.MqttTopic, strings.ReplaceAll(msg.Subject, ".", "/"))
		if err := r.pub.Publish(msg.Subject, topic, payload); err != nil {
			r.logger.Error(fmt.Sprintf("Failed to publish on route %s: %s", r.MqttTopic, err))
		}
		r.logger.Debug(fmt.Sprintf("Published to:%s , payload:%s", msg.Subject, string(payload[:50])))
	}
}
