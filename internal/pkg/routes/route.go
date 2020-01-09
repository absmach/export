package routes

import "github.com/nats-io/nats.go"

type Route interface {
	Consume(m *nats.Msg)
}
