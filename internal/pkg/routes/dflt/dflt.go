package dflt

import (
	"github.com/gogo/protobuf/proto"
	"github.com/mainflux/export/internal/app/export"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/logger"
	"github.com/nats-io/nats.go"
)

type route struct {
	natsTopic string
	mqttTopic string
	subtopic  string
	logger    logger.Logger 
	mqtt      mqtt.Client
}

func New(n, m, s string, mqtt mqtt.Client, logger logger.Logger) export.Route {
	return &route{
		natsTopic: n,
		mqttTopic: m,
		subtopic:  s,
		mqtt: mqtt,
		logger: logger,
	}
}

func (r *route) Consume(m *nats.Msg) {
	msg := mainflux.Message{}
	err := proto.Unmarshal(m.Data, &msg)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to unmarshall message %s", err.Error())

	}
	r.Publish(msg.Payload)
	return
}

func (r *route) Publish(bytes []byte) {
	token := r.mqtt.Publish(r.mqttTopic, 0, false, bytes); token.Wait() && token.Error() != nil {
	    r.logger.Error(fmt.Sprintf("Failed to publish message on topic %s : %s", c.mqttTopic, token.Error()))
	}
		
}
