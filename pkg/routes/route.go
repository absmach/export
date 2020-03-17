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

// Route - message route, tells which nats topic messages goes to which mqtt topic.
// Later we can add direction and other combination like ( nats-nats).
// Route is used in mfx and plain. Route is like base implementation and mfx and plain
// are extended implementation.

const (
	workers = 200
)

type route struct {
	natsTopic string
	mqttTopic string
	subtopic  string
	logger    logger.Logger
	pub       messages.Publisher
	messages  chan *nats.Msg
	sub       *nats.Subscription
	workers   int
}

type Route interface {
	Consume()
	Process(data []byte) ([]byte, error)
	MessagesBuffer() chan *nats.Msg
	NatsTopic() string
	MqttTopic() string
	Subtopic() string
}

func NewRoute(n, m, s string, w int, log logger.Logger, pub messages.Publisher) Route {
	if w == 0 {
		w = workers
	}
	r := route{
		natsTopic: n,
		mqttTopic: m,
		subtopic:  s,
		logger:    log,
		pub:       pub,
		messages:  make(chan *nats.Msg, w),
		workers:   w,
	}
	return &r
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

func (r *route) Process(data []byte) ([]byte, error) {
	return data, nil
}

func (r *route) MessagesBuffer() chan *nats.Msg {
	return r.messages
}

func (r *route) Consume() {
	for msg := range r.messages {

		payload, err := r.Process(msg.Data)
		if err != nil {
			r.logger.Error(fmt.Sprintf("Failed to consume message %s", err))
		}
		topic := fmt.Sprintf("%s/%s", r.MqttTopic(), strings.ReplaceAll(msg.Subject, ".", "/"))
		if err := r.pub.Publish(msg.Subject, topic, payload); err != nil {
			r.logger.Error(fmt.Sprintf("Failed to publish on route %s: %s", r.MqttTopic(), err))
		}
		r.logger.Debug(fmt.Sprintf("Published to:%s , payload:%s", msg.Subject, string(payload[:50])))
	}
}

// func (r *route) runWorker() {
// 	for msg := range r.messages {
// 		r.Consume(msg)
// 	}

// }
