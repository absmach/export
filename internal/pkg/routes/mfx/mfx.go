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
	// ContentTypeJSON represents SenML in JSON format content type.
	ContentTypeJSON = "application/senml+json"

	// ContentTypeCBOR represents SenML in CBOR format content type.
	ContentTypeCBOR = "application/senml+cbor"
)

var formats = map[string]senml.Format{
	ContentTypeJSON: senml.JSON,
	ContentTypeCBOR: senml.CBOR,
}

type mfxRoute struct {
	route routes.Route
}

func NewRoute(n, m, s string, mqtt mqtt.Client, l logger.Logger) routes.Route {
	return mfxRoute{
		route: routes.NewRoute(n, m, s, mqtt, l),
	}
}

func (mr mfxRoute) NatsTopic() string {
	return mr.route.NatsTopic()
}

func (mr mfxRoute) MqttTopic() string {
	return mr.route.MqttTopic()
}

func (mr mfxRoute) Subtopic() string {
	return mr.route.Subtopic()
}

func (mr mfxRoute) Logger() logger.Logger {
	return mr.route.Logger()
}

func (mr mfxRoute) Mqtt() mqtt.Client {
	return mr.route.Mqtt()
}

func (mr mfxRoute) Consume(m *nats.Msg) {
	msg := mainflux.Message{}
	err := proto.Unmarshal(m.Data, &msg)
	if err != nil {
		mr.route.Logger().Error(fmt.Sprintf("Failed to unmarshal %s", err.Error()))
	}
	format, ok := formats[msg.ContentType]
	if !ok {
		format = senml.JSON
	}

	raw, err := senml.Decode(msg.Payload, format)
	if err != nil {
		mr.route.Logger().Error(fmt.Sprintf("Failed to decode payload message: %s", err))
	}

	payload, err := senml.Encode(raw, senml.JSON)
	if err != nil {
		mr.route.Logger().Error(fmt.Sprintf("Failed to encode payload message: %s", err))
		return
	}
	mr.Forward(payload)
}

func (mr mfxRoute) Forward(payload []byte) {
	mr.route.Forward(payload)
}
