// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package dflt

import (
	"fmt"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/mainflux/export/internal/pkg/routes"
	"github.com/mainflux/mainflux/logger"
	"github.com/nats-io/nats.go"
)

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
	r.Publish(m.Data)
}

func (r *route) Publish(bytes []byte) {
	if token := r.mqtt.Publish(r.mqttTopic, 0, false, bytes); token.Wait() && token.Error() != nil {
		r.logger.Error(fmt.Sprintf("Failed to publish message on topic %s : %s", r.mqttTopic, token.Error()))
	}
}
