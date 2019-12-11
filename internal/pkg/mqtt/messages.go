// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mqtt

import (
	"fmt"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/mainflux/export/internal/app/export"
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
	topic  string
	log    logger.Logger
}

// New returns new InfluxDB writer.
func New(client mqtt.Client, topic string, log logger.Logger) export.MessageRepository {
	return &exportRepo{
		client: client,
		topic:  topic,
		log:    log,
	}
}

func (repo *exportRepo) Publish(message []byte) error {
	if token := repo.client.Publish(repo.topic, 0, false, message); token.Wait() && token.Error() != nil {
		repo.log.Error(fmt.Sprintf("Failed to publish message on topic %s : %s", repo.topic, token.Error()))
	}
	return nil
}
