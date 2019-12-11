// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package export

import (
	"fmt"

	"github.com/mainflux/export/internal/pkg/config"
	"github.com/mainflux/export/internal/pkg/routes/dflt"
	"github.com/mainflux/export/internal/pkg/routes/mflx"
	log "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/transformers"
	nats "github.com/nats-io/nats.go"
)

type Service interface {
	Start(string)
}

var _ Service = (*exporter)(nil)

type exporter struct {
	Nc          *nats.Conn
	Channels    map[string]bool
	Repo        MessageRepository
	Transformer transformers.Transformer
	Logger      log.Logger
	Cfg         config.Config
	Routes      []Route
}

// New create new instance of export service
func New(nc *nats.Conn, repo MessageRepository, c config.Config, transformer transformers.Transformer, channels map[string]bool, fConsume func(*nats.Msg), logger log.Logger) Service {
	routes := make([]Route, 0)
	e := exporter{
		Nc:          nc,
		Channels:    channels,
		Repo:        repo,
		Transformer: transformer,
		Logger:      logger,
		Cfg:         c,
		Routes:      routes,
	}

	return &e
}

// Start method starts consuming messages received from NATS.
// and makes routes according to the configuration file.
// Routes export messages to mqtt.
func (e *exporter) Start(queue string) {
	var route Route
	for _, r := range e.Cfg.Routes {
		switch *r.Type {
		case "mflx":
			route = mflx.New(*r.MqttTopic, *r.NatsTopic, *r.SubTopic)
		default:
			route = dflt.New(*r.MqttTopic, *r.NatsTopic, *r.SubTopic)
		}
		e.Routes = append(e.Routes, route)
		_, err := e.Nc.QueueSubscribe(*r.NatsTopic, queue, route.Consume)
		if err != nil {
			e.Logger.Error(fmt.Sprintf("Failed to subscribe for NATS/MQTT %s/%s", r.NatsTopic, r.MqttTopic))
		}

	}
}
