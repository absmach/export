// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package messages

import "errors"

type message interface {
	encode() map[string]interface{}
}

var (
	_ message = (*Msg)(nil)

	errIncorrectMsgData = errors.New("incorrect message data")
)

type Msg struct {
	Topic   string
	Payload string
}

func (m *Msg) encode() map[string]interface{} {
	return map[string]interface{}{
		"topic":   m.Topic,
		"payload": m.Payload,
	}
}

func (m *Msg) decode(in map[string]interface{}) error {
	if len(in) < 2 {
		return errIncorrectMsgData
	}
	m.Topic = in["topic"].(string)
	m.Payload = in["payload"].(string)
	return nil
}
