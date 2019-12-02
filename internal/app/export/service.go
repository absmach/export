// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package export

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/mainflux/export/internal/pkg/config"
	"github.com/mainflux/mainflux"
	log "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/transformers"
	nats "github.com/nats-io/nats.go"
)

type Service interface {
	Start(string) error
	Consume(m *nats.Msg)
	Export(s string, msgs ...interface{})
}

var _ Service = (*exporter)(nil)

type exporter struct {
	Nc          *nats.Conn
	Channels    map[string]bool
	Repo        MessageRepository
	Transformer transformers.Transformer
	Logger      log.Logger
	Cons        func(*nats.Msg)
	Routes      map[string]config.Route
}

// New create new instance of export service
func New(nc *nats.Conn, repo MessageRepository, transformer transformers.Transformer, conf config.Config, fConsume func(*nats.Msg), logger log.Logger) Service {
	routes := map[string]config.Route{}
	for _, r := range conf.Routes {
		routes[*r.NatsTopic] = r
	}
	c := exporter{
		Nc:          nc,
		Repo:        repo,
		Transformer: transformer,
		Logger:      logger,
		Cons:        fConsume,
		Routes:      routes,
	}

	if fConsume == nil {
		c.Cons = c.Consume
	}
	return &c
}

// Start method starts consuming messages received from NATS.
// This method transforms messages to SenML format before
// using MessageRepository to store them.
func (c *exporter) Start(queue string) error {
	_, err := c.Nc.QueueSubscribe(mainflux.InputChannels, queue, c.Cons)
	return err
}

func (c *exporter) Consume(m *nats.Msg) {
	r, ok := c.Routes[m.Subject]
	if !ok {
		c.Logger.Error(fmt.Sprintf("There is no mapping for nats subject %s", m.Subject))
		return
	}
	msg := mainflux.Message{}

	err := proto.Unmarshal(m.Data, &msg)

	if err == nil {
		msgs := []interface{}{}
		msgs = append(msgs, msg)
		c.Export(*r.MqttTopic, msgs...)
	}

	c.Logger.Warn(fmt.Sprintf("Failed to unmarshal received message: %s", err))
	return
}

func (c *exporter) Export(topic string, msgs ...interface{}) {
	if err := c.Repo.Publish(topic, msgs...); err != nil {
		c.Logger.Warn(fmt.Sprintf("Failed to save message: %s", err))
		return
	}
}

func (c *exporter) channelExists(channel string) bool {
	if _, ok := c.Channels["*"]; ok {
		return true
	}

	_, found := c.Channels[channel]
	return found
}
