// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package writer

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/mainflux/export/writers"
	"github.com/mainflux/mainflux"
	log "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/transformers"
	nats "github.com/nats-io/nats.go"
)

var _ writers.Writer = (*exporter)(nil)

type exporter struct {
	writer writers.Writer
	logger log.Logger
}

func New(nc *nats.Conn, repo writers.MessageRepository, transformer transformers.Transformer, channels map[string]bool, fConsume func(*nats.Msg), logger log.Logger) writers.Writer {
	e := exporter{logger: logger}
	w := writers.New(nc, repo, transformer, channels, e.Consume, logger)
	e.writer = w
	return &e
}

// Start method starts consuming messages received from NATS.
// This method transforms messages to SenML format before
// using MessageRepository to store them.
func (e *exporter) Start(queue string) error {
	return e.writer.Start(queue)
}

func (e *exporter) Consume(m *nats.Msg) {
	msg := mainflux.Message{}

	if err := proto.Unmarshal(m.Data, &msg); err != nil {
		e.logger.Warn(fmt.Sprintf("Failed to unmarshal received message: %s", err))
		return
	}

	msgs := []interface{}{}
	msgs = append(msgs, msg)

	e.Write(msgs...)
}

func (e *exporter) Write(msgs ...interface{}) {
	e.writer.Write(msgs...)
}
