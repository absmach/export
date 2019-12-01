// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package export

import (
	"crypto/tls"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/mainflux/mainflux"
	log "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/transformers"
	nats "github.com/nats-io/nats.go"
)

type Exporter interface {
	Start(string) error
	Consume(m *nats.Msg)
	Export(msgs ...interface{})
}

type Consumer struct {
	Nc          *nats.Conn
	Channels    map[string]bool
	Repo        MessageRepository
	Transformer transformers.Transformer
	Logger      log.Logger
	Cons        func(*nats.Msg)
}

type Config struct {
	NatsURL        string
	LogLevel       string
	Port           string
	MqttHost       string
	MqttUsername   string
	MqttPassword   string
	MqttMTLS       bool
	MqttSkipTLSVer bool
	MqttRetain     bool
	MqttChannel    string
	MqttCA         []byte
	MqttCert       tls.Certificate
	MqttQoS        int
	Channels       map[string]bool
}

//
func New(nc *nats.Conn, repo MessageRepository, transformer transformers.Transformer, channels map[string]bool, fConsume func(*nats.Msg), logger log.Logger) Exporter {

	c := Consumer{
		Nc:          nc,
		Channels:    channels,
		Repo:        repo,
		Transformer: transformer,
		Logger:      logger,
		Cons:        fConsume,
	}

	if fConsume == nil {
		c.Cons = c.Consume
	}
	return &c
}

// Start method starts consuming messages received from NATS.
// This method transforms messages to SenML format before
// using MessageRepository to store them.
func (c *Consumer) Start(queue string) error {
	_, err := c.Nc.QueueSubscribe(mainflux.InputChannels, queue, c.Cons)
	return err
}

func (c *Consumer) Consume(m *nats.Msg) {
	msg := mainflux.Message{}

	err := proto.Unmarshal(m.Data, &msg)

	if err == nil {
		msgs := []interface{}{}
		msgs = append(msgs, msg)
		c.Export(msgs...)
	}

	c.Logger.Warn(fmt.Sprintf("Failed to unmarshal received message: %s", err))
	return
}

func (c *Consumer) Export(msgs ...interface{}) {
	if err := c.Repo.Publish(msgs...); err != nil {
		c.Logger.Warn(fmt.Sprintf("Failed to save message: %s", err))
		return
	}
}

func (c *Consumer) channelExists(channel string) bool {
	if _, ok := c.Channels["*"]; ok {
		return true
	}

	_, found := c.Channels[channel]
	return found
}
