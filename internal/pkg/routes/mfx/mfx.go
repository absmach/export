// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mfx

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/mainflux/export/internal/pkg/publisher"
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
	route routes.Route
}

func NewRoute(n, m, s string, pub publisher.File, l logger.Logger) routes.RouteIF {
	return &mfxRoute{
		route: routes.Route{
			NatsTopic: n,
			MqttTopic: m,
			Subtopic:  s,
			Pub:       pub,
			Logger:    l,
		},
	}
}

func (mr mfxRoute) Consume(m *nats.Msg) {
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
	mr.route.Pub.Publish(bytes, mr.route.MqttTopic)
}
