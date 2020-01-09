package mflx

import (
	"fmt"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gogo/protobuf/proto"
	"github.com/mainflux/export/internal/app/export"
	"github.com/mainflux/export/internal/pkg/routes"
	"github.com/mainflux/export/internal/pkg/storage"
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

type route struct {
	natsTopic string
	mqttTopic string
	subtopic  string
	logger    logger.Logger
	mqtt      mqtt.Client
	expSvc    export.Service
}

func New(e export.Service, n, m, s string, mqtt mqtt.Client, logger logger.Logger) routes.Route {
	return &route{
		natsTopic: n,
		mqttTopic: m,
		subtopic:  s,
		mqtt:      mqtt,
		logger:    logger,
		expSvc:    export.Service,
	}
}

func (r *route) Consume(m *nats.Msg) {
	msg := mainflux.Message{}
	err := proto.Unmarshal(m.Data, &msg)
	if err != nil {
		r.logger.Error(fmt.Sprintf("Failed to unmarshal %s", err.Error()))
	}
	format, ok := formats[msg.ContentType]
	if !ok {
		format = senml.JSON
	}

	raw, err := senml.Decode(msg.Payload, format)
	if err != nil {
		r.logger.Error(fmt.Sprintf("Failed to decode payload message: %s", err))
	}

	payload, err := senml.Encode(raw, senml.JSON)
	if err != nil {
		r.logger.Error(fmt.Sprintf("Failed to encode %s", err.Error()))
		return 
	}
	err = r.expSvc.Publish(payload, r.mqttTopic)
	if err!= nil {
		r.logger.Error(fmt.Sprintf("Failed to publish message on topic %s : %s", r.mqttTopic, token.Error()))
	}
	return
}

