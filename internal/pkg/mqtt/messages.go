// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mqtt

import (
	"fmt"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/influxdata/config"
	"github.com/mainflux/export/internal/app/export"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/senml"
)

const pointName = "messages"

var _ export.MessageRepository = (*exportRepo)(nil)

const (
	// SenMLJSON represents SenML in JSON format content type.
	SenMLJSON = "application/senml+json"

	// SenMLCBOR represents SenML in CBOR format content type.
	SenMLCBOR = "application/senml+cbor"
)

var formats = map[string]senml.Format{
	SenMLJSON: senml.JSON,
	SenMLCBOR: senml.CBOR,
}

type exportRepo struct {
	client mqtt.Client
	conf   config.Config
	log    logger.Logger
}

type fields map[string]interface{}
type tags map[string]string

// New returns new InfluxDB writer.
func New(client mqtt.Client, conf export.Config, log logger.Logger) export.MessageRepository {
	return &exportRepo{
		client: client,
		conf:   conf,
		log:    log,
	}
}

func (repo *exportRepo) Publish(messages ...interface{}) error {
	topic := fmt.Sprintf("channels/%s/messages", repo.conf.Mqtt.Channel)
	for _, m := range messages {
		msg, ok := m.(mainflux.Message)
		if !ok {
			return fmt.Errorf("Wrong type")
		}

		format, ok := formats[msg.ContentType]
		if !ok {
			format = senml.JSON
		}

		raw, err := senml.Decode(msg.Payload, format)
		if err != nil {
			return fmt.Errorf(fmt.Sprintf("Failed to decode payload message: %s", err))
		}

		pubtopic := fmt.Sprintf("/%s/%s/%s", topic, msg.GetChannel(), msg.GetPublisher(), msg.GetSubtopic())

		payload, err := senml.Encode(raw, senml.JSON)
		if err != nil {
			repo.log.Error(fmt.Sprintf("Failed to publish message on topic %s : %s", pubtopic, err.Error()))
		}
		if token := repo.client.Publish(pubtopic, 0, false, payload); token.Wait() && token.Error() != nil {
			repo.log.Error(fmt.Sprintf("Failed to publish message on topic %s : %s", repo.conf.Mqtt.Channel, token.Error()))
		}
	}
	return nil
}
