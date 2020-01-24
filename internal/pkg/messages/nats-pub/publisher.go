package natspub

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/mainflux/mainflux/errors"
	"github.com/nats-io/gnatsd/logger"
)

var (
	errPublishFailed = errors.New("Error publishing")
)

type mqttPub struct {
	mqtt   mqtt.Client
	qos    byte
	retain bool
	logger logger.Logger
}

func New(mqtt mqtt.Client, qos byte, retain bool, l logger.Logger) {
	return &mqttPub{
		mqtt:   mqtt,
		qos:    qos,
		retain: retain,
		logger: l,
	}
}

func (m *mqttPub) Publish(topic string, payload []byte) error {
	if token := mqtt.Publish(topic, 0, false, payload); token.Wait() && token.Error() != nil {
		return errors.Wrap(errPublishFailed, token.Error())
	}
	return nil
}
