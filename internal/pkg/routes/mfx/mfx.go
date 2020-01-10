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

type mfxRoute struct {
	route routes.R
}

func NewRoute(n, m, s string, mqtt mqtt.Client, l logger.Logger) routes.Route {
	return &mfxRoute{
		route: routes.R{
			NatsTopic: n,
			MqttTopic: m,
			Subtopic:  s,
			Mqtt:      mqtt,
			Logger:    l,
		},
	}
}

func (mr mfxRoute) Consume(m *nats.Msg) {
	fmt.Println("COnsume 2")
	msg := mainflux.Message{}
	err := proto.Unmarshal(m.Data, &msg)
	if err != nil {
		mr.route.Logger.Error(fmt.Sprintf("Failed to unmarshal %s", err.Error()))
	}
	format, ok := formats[msg.ContentType]
	if !ok {
		format = senml.JSON
	}

	raw, err := senml.Decode(msg.Payload, format)
	if err != nil {
		mr.route.Logger.Error(fmt.Sprintf("Failed to decode payload message: %s", err))
	}

	payload, err := senml.Encode(raw, senml.JSON)
	mr.route.Publish(payload)
	return
}

func (mr mfxRoute) Publish(bytes []byte) {
	if token := mr.route.Mqtt.Publish(mr.route.MqttTopic, 0, false, bytes); token.Wait() && token.Error() != nil {
		mr.route.Logger.Error(fmt.Sprintf("Failed to publish message on topic %s : %s", mr.route.MqttTopic, token.Error()))
	}
}
