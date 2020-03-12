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

const (
	workers = 200
)

type route struct {
	natsTopic string
	mqttTopic string
	subtopic  string
	logger    logger.Logger
	pub       publish.Publisher
	messages  chan *nats.Msg
	sub       *nats.Subscription
}

type Route interface {
	Consume(*nats.Msg)
	Process(data []byte) ([]byte, error)
	NatsTopic() string
	MqttTopic() string
	Subtopic() string
	Subscribe(g string, nc *nats.Conn) error
}

func NewRoute(n, m, s string, log logger.Logger, pub publish.Publisher) Route {
	r := route{
		natsTopic: n,
		mqttTopic: m,
		subtopic:  s,
		logger:    log,
		pub:       pub,
		messages:  make(chan *nats.Msg, workers),
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

func (r *route) Consume(msg *nats.Msg) {
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

func (r *route) Subscribe(group string, nc *nats.Conn) error {
	sub, err := nc.ChanQueueSubscribe(r.NatsTopic(), group, r.messages)
	if err != nil {
		r.logger.Error(fmt.Sprintf("Failed to subscribe to NATS %s: %s", r.NatsTopic(), err))
		return err
	}
	r.logger.Info(fmt.Sprintf("Starting %d workers", workers))
	r.sub = sub
	for i := 0; i < workers; i++ {
		go r.runWorker()
	}

	return nil
}

func (r *route) runWorker() {
	for msg := range r.messages {
		r.Consume(msg)
	}

}
