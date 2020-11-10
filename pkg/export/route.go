// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package export

import (
	"fmt"
	"math"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/mainflux/export/pkg/config"
	"github.com/mainflux/export/pkg/messages"
	"github.com/mainflux/mainflux/errors"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/messaging"
	nats "github.com/nats-io/nats.go"
)

const (
	// Number of the workers depends on the connection capacity
	// as well as on payload size that needs to be sent.
	// Number of workers also determines the size of the buffer
	// that recieves messages from NATS.
	// For regular telemetry SenML messages 10 workers is enough.
	workers      = 10
	sliceLen     = 50
	defaultType  = "default"
	mainfluxType = "mfx"
	JSON         = "application/senml+json"
)

var errUnsupportedType = errors.New("route type is not supported")

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
	Type      string
	logger    logger.Logger
	pub       messages.Publisher
}

func NewRoute(rc config.Route, log logger.Logger, pub messages.Publisher) *Route {
	w := rc.Workers
	if w == 0 {
		w = workers
	}
	r := &Route{
		NatsTopic: rc.NatsTopic + "." + NatsAll,
		MqttTopic: rc.MqttTopic,
		Subtopic:  rc.SubTopic,
		Type:      rc.Type,
		Workers:   w,
		Messages:  make(chan *nats.Msg, w),
		logger:    log,
		pub:       pub,
	}
	return r
}

func (r *Route) Process(data []byte) ([]byte, error) {
	switch r.Type {
	case defaultType:
		return data, nil
	case mainfluxType:
		var msg messaging.Message
		err := proto.Unmarshal(data, &msg)
		if err != nil {
			return nil, err
		}
		return msg.Payload, nil
	default:
		return nil, errUnsupportedType
	}

}

func (r *Route) Consume() {
	for msg := range r.Messages {
		payload, err := r.Process(msg.Data)
		if err != nil {
			r.logger.Error(fmt.Sprintf("Failed to consume message %s", err))
		}
		topic := r.MqttTopic
		if r.Subtopic != "" {
			topic = fmt.Sprintf("%s/%s", r.MqttTopic, r.Subtopic)
		}
		topic = fmt.Sprintf("%s/%s", topic, strings.ReplaceAll(msg.Subject, ".", "/"))
		if err := r.pub.Publish(msg.Subject, topic, payload); err != nil {
			r.logger.Error(fmt.Sprintf("Failed to publish on route %s: %s", r.MqttTopic, err))
		}
		r.msgDebug(msg.Subject, payload)
	}
}

func (r *Route) msgDebug(sub string, payload []byte) {
	p := ""
	if l := math.Min(float64(sliceLen), float64(len(payload))); len(payload) > 0 {
		p = string(payload[:int(l)])
	}
	r.logger.Debug(fmt.Sprintf("Published to: %s, payload: %s", sub, p))
}
