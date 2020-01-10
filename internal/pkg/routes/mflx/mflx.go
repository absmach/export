package mflx

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

type route struct {
	natsTopic string
	mqttTopic string
	subtopic  string
	logger    logger.Logger
	mqtt      mqtt.Client
}

func New(n, m, s string, mqtt mqtt.Client, logger logger.Logger) routes.Route {
	return &route{
		natsTopic: n,
		mqttTopic: m,
		subtopic:  s,
		mqtt:      mqtt,
		logger:    logger,
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
	r.Publish(payload)
	return
}

func (r *route) Publish(bytes []byte) {
	if token := r.mqtt.Publish(r.mqttTopic, 0, false, bytes); token.Wait() && token.Error() != nil {
		r.logger.Error(fmt.Sprintf("Failed to publish message on topic %s : %s", r.mqttTopic, token.Error()))
	}
}