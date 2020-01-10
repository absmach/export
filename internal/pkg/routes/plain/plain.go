// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package plain

import (
	"fmt"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/mainflux/export/internal/pkg/routes"
	"github.com/mainflux/mainflux/logger"
	"github.com/nats-io/nats.go"
)

type plainRoute struct {
	route routes.R
}

func NewRoute(n, m, s string, mqtt mqtt.Client, l logger.Logger) routes.Route {
	return &plainRoute{
		route: routes.R{
			NatsTopic: n,
			MqttTopic: m,
			Subtopic:  s,
			Mqtt:      mqtt,
			Logger:    l,
		},
	}
}

func (pr *plainRoute) Consume(m *nats.Msg) {
	fmt.Println("COnsume 1")
	pr.Publish(m.Data)
}

func (pr *plainRoute) Publish(bytes []byte) {
	if token := pr.route.Mqtt.Publish(pr.route.MqttTopic, 0, false, bytes); token.Wait() && token.Error() != nil {
		pr.route.Logger.Error(fmt.Sprintf("Failed to publish message on topic %s : %s", pr.route.MqttTopic, token.Error()))
	}
}
