// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mfx

import (
	"fmt"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gogo/protobuf/proto"
	"github.com/mainflux/export/internal/pkg/routes"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/senml"
	"github.com/nats-io/nats.go"
)

const (
	// JSON represents SenML in JSON format content type.
	JSON = "application/senml+json"

	// CBOR represents SenML in CBOR format content type.
	CBOR = "application/senml+cbor"
)

var formats = map[string]senml.Format{
	JSON: senml.JSON,
	CBOR: senml.CBOR,
}

type route routes.R

func New(n, m, s string, mqtt mqtt.Client, l logger.Logger) routes.Route {
	return routes.NewRoute(n, m, s, mqtt, l)
}

func (r route) Consume(m *nats.Msg) {
	msg := mainflux.Message{}
	err := proto.Unmarshal(m.Data, &msg)
	if err != nil {
		r.Logger.Error(fmt.Sprintf("Failed to unmarshal %s", err.Error()))
	}
	format, ok := formats[msg.ContentType]
	if !ok {
		format = senml.JSON
	}

	raw, err := senml.Decode(msg.Payload, format)
	if err != nil {
		r.Logger.Error(fmt.Sprintf("Failed to decode payload message: %s", err))
	}

	payload, err := senml.Encode(raw, senml.JSON)
	r.Publish(payload)
	return
}

func (r route) Publish(bytes []byte) {
	if token := r.Mqtt.Publish(r.MqttTopic, 0, false, bytes); token.Wait() && token.Error() != nil {
		r.Logger.Error(fmt.Sprintf("Failed to publish message on topic %s : %s", r.MqttTopic, token.Error()))
	}
}
