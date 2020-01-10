// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package routes

import "github.com/nats-io/nats.go"

type Route interface {
	Consume(m *nats.Msg)
	Publish([]byte)
}
