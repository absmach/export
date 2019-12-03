// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package export

import (
	"fmt"

	"github.com/mainflux/export/internal/pkg/config"
	"github.com/mainflux/export/internal/pkg/transformers"
	log "github.com/mainflux/mainflux/logger"
	nats "github.com/nats-io/nats.go"
)

type Service interface {
	Start(string, transformers.Transformer, string) error
	Consume(m *nats.Msg)
	Export(s string, msgs ...interface{})
}

var _ Service = (*exportService)(nil)

type exporter struct {
	Transformer transformers.Transformer
}
type exportService struct {
	Nc        *nats.Conn
	Channels  map[string]bool
	Repo      MessageRepository
	Logger    log.Logger
	Cons      func(*nats.Msg)
	Routes    map[string]config.Route
	Exporters map[string]exporter
}

// New create new instance of export service
func New(nc *nats.Conn, repo MessageRepository, conf config.Config, fConsume func(*nats.Msg), logger log.Logger) Service {
	routes := map[string]config.Route{}
	exporters := map[string]exporter{}
	for _, r := range conf.Routes {
		routes[*r.NatsTopic] = r
		exporters[*r.NatsTopic] = exporter{}
	}
	c := exportService{
		Nc:        nc,
		Repo:      repo,
		Logger:    logger,
		Cons:      fConsume,
		Routes:    routes,
		Exporters: exporters,
	}

	if fConsume == nil {
		c.Cons = c.Consume
	}
	return &c
}

// Start method starts consuming messages received from NATS.
// This method transforms messages to SenML format before
// using MessageRepository to store them.
func (c *exportService) Start(topic string, t transformers.Transformer, queue string) error {
	exp, ok := c.Exporters[topic]
	if !ok {
		c.Logger.Error(fmt.Sprintf("There is no exporter for nats subject %s", topic))
		return fmt.Errorf(fmt.Sprintf("There is no exporter for nats subject %s", topic))
	}
	exp.Transformer = t
	_, err := c.Nc.QueueSubscribe(topic, queue, c.Cons)
	return err
}

func (c *exportService) Consume(m *nats.Msg) {
	r, ok := c.Routes[m.Subject]
	if !ok {
		c.Logger.Error(fmt.Sprintf("There is no mapping for nats subject %s", m.Subject))
		return
	}
	exp, ok := c.Exporters[m.Subject]
	if !ok {
		c.Logger.Error(fmt.Sprintf("There is no exporter for nats subject %s", m.Subject))
		return
	}

	t, err := exp.Transformer.Transform(m.Data)
	if err != nil {
		c.Logger.Warn(fmt.Sprintf("Failed to tranform received message: %s", err))
		return
	}
	if err == nil {
		msgs := []interface{}{}
		msgs = append(msgs, t)
		c.Export(*r.MqttTopic, msgs...)
	}
	c.Logger.Warn(fmt.Sprintf("Failed to unmarshal received message: %s", err))
	return
}

func (c *exportService) Export(topic string, msgs ...interface{}) {
	if err := c.Repo.Publish(topic, msgs...); err != nil {
		c.Logger.Warn(fmt.Sprintf("Failed to save message: %s", err))
		return
	}
}

func (c *exportService) channelExists(channel string) bool {
	if _, ok := c.Channels["*"]; ok {
		return true
	}
	_, found := c.Channels[channel]
	return found
}
