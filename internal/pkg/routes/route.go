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
	logger    logger.Logger
	pub       publish.Publisher
	workers   int
	messages  chan *nats.Msg
	sub       *nats.Subscription
}

var (
	workers = 10
)

type Route interface {
	Consume(*nats.Msg)
	Process(data []byte) ([]byte, error)
	NatsTopic() string
	MqttTopic() string
	Subtopic() string
	Subscribe(nc *nats.Conn) error
	Clean()
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
		r.logger.Error(fmt.Sprintf("Failed to publish on route %s: %s", r.MqttTopic(), err.Error()))
	}
	r.logger.Debug(fmt.Sprintf("Published to:%s , payload:%s", msg.Subject, string(payload[:50])))
}

func (r *route) Subscribe(nc *nats.Conn) error {
	sub, err := nc.ChanSubscribe(r.NatsTopic(), r.messages)
	if err != nil {
		r.logger.Error(fmt.Sprintf("Failed to subscribe to check results channel: %s", err))
		return err
	}
	r.sub = sub
	go func() {
		for msg := range r.messages {
			go r.Consume(msg)
		}
	}()
	return nil
}

func (r *route) Clean() {
	r.sub.Unsubscribe()
	close(r.messages)
}
