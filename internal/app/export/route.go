package export

import "github.com/nats-io/nats.go"

type Route interface {
	Consume(m *nats.Msg)
	Publish([]byte)
}
