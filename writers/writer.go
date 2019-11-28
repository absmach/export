// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package writers

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/mainflux/mainflux"
	log "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/transformers"
	"github.com/mainflux/mainflux/transformers/senml"
	nats "github.com/nats-io/nats.go"
)

type Writer interface {
	Start(string) error
	Consume(m *nats.Msg)
	Write(msgs ...interface{})
}

type Consumer struct {
	Nc          *nats.Conn
	Channels    map[string]bool
	Repo        MessageRepository
	Transformer transformers.Transformer
	Logger      log.Logger
	Cons        func(*nats.Msg)
}

//
func New(nc *nats.Conn, repo MessageRepository, transformer transformers.Transformer, channels map[string]bool, fConsume func(*nats.Msg), logger log.Logger) Writer {

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
	var msg mainflux.Message
	if err := proto.Unmarshal(m.Data, &msg); err != nil {
		c.Logger.Warn(fmt.Sprintf("Failed to unmarshal received message: %s", err))
		return
	}

	t, err := c.Transformer.Transform(msg)
	if err != nil {
		c.Logger.Warn(fmt.Sprintf("Failed to tranform received message: %s", err))
		return
	}
	norm, ok := t.([]senml.Message)
	if !ok {
		c.Logger.Warn("Invalid message format from the Transformer output.")
		return
	}
	var msgs []interface{}
	for _, v := range norm {
		if c.channelExists(v.Channel) {
			msgs = append(msgs, v)
		}
	}

	c.Write(msgs...)
}

func (c *Consumer) Write(msgs ...interface{}) {
	if err := c.Repo.Save(msgs...); err != nil {
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
