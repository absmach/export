package dflt

import (
	"fmt"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/mainflux/export/internal/app/export"
	"github.com/mainflux/export/internal/pkg/routes"
	"github.com/mainflux/export/internal/pkg/storage"
	"github.com/mainflux/mainflux/logger"
	"github.com/nats-io/nats.go"
)

type route struct {
	natsTopic string
	mqttTopic string
	subtopic  string
	logger    logger.Logger
	mqtt      mqtt.Client
	expSvc    export.Service
}

func New(e export.Service, n, m, s string, mqtt mqtt.Client, logger logger.Logger) routes.Route {
	return &route{
		natsTopic: n,
		mqttTopic: m,
		subtopic:  s,
		mqtt:      mqtt,
		logger:    logger,
		expSvc:    e,
	}
}

func (r *route) Consume(m *nats.Msg) {
	err = r.expSvc.Publish(m.Data, r.mqttTopic)
	if err!= nil {
		r.logger.Error(fmt.Sprintf("Failed to publish message on topic %s : %s", r.mqttTopic, token.Error()))
	}
	return
}


