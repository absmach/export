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

type route routes.R

func New(n, m, s string, mqtt mqtt.Client, l logger.Logger) routes.Route {
	return routes.NewRoute(n, m, s, mqtt, l)
}

func (r *route) Consume(m *nats.Msg) {
	r.Publish(m.Data)
}

func (r *route) Publish(bytes []byte) {
	if token := r.Mqtt.Publish(r.MqttTopic, 0, false, bytes); token.Wait() && token.Error() != nil {
		r.Logger.Error(fmt.Sprintf("Failed to publish message on topic %s : %s", r.MqttTopic, token.Error()))
	}
}
