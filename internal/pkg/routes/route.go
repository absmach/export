// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package routes

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/mainflux/mainflux/logger"
	"github.com/nats-io/nats.go"
)

type R struct {
	NatsTopic string
	MqttTopic string
	Subtopic  string
	Logger    logger.Logger
	Mqtt      mqtt.Client
}

type Route interface {
	Consume(m *nats.Msg)
	Publish([]byte)
}

func (r R) Consume(m *nats.Msg) {}
func (r R) Publish([]byte)      {}
