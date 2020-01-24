// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package messages

type message interface {
	encode() map[string]interface{}
}

var (
	_ message = (*msg)(nil)
)

type msg struct {
	payload string
}

func (m msg) encode() map[string]interface{} {
	return map[string]interface{}{
		"payload": m.payload,
	}
}
