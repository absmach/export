// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mfx

import (
	"github.com/gogo/protobuf/proto"
	"github.com/mainflux/export/internal/app/export/publish"
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

func NewRoute(n, m, s string, log logger.Logger, pub publish.Publisher) routes.Route {
	return mfxRoute{
		route: routes.NewRoute(n, m, s, log, pub),
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

func (mr mfxRoute) Consume(m *nats.Msg) {
	mr.route.Consume(m)
}

func (mr mfxRoute) Process(data []byte) ([]byte, error) {
	msg := mainflux.Message{}
	err := proto.Unmarshal(data, &msg)
	if err != nil {
		return []byte{}, err
	}
	format, ok := formats[msg.ContentType]
	if !ok {
		format = senml.JSON
	}

	raw, err := senml.Decode(msg.Payload, format)
	if err != nil {
		return []byte{}, err
	}

	payload, err := senml.Encode(raw, senml.JSON)
	if err != nil {
		return []byte{}, err
	}
	return payload, nil
}
