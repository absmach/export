// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package export_test

import (
	"os"
	"testing"

	influxdata "github.com/influxdata/influxdb/client/v2"
	log "github.com/mainflux/mainflux/logger"
)

const valueFields = 5

var (
	port        string
	testLog, _  = log.New(os.Stdout, log.Info.String())
	testDB      = "test"
	streamsSize = 250
	selectMsgs  = "SELECT * FROM test..messages"
	dropMsgs    = "DROP SERIES FROM messages"
	clientCfg   = influxdata.HTTPConfig{
		Username: "test",
		Password: "test",
	}
	subtopic = "topic"
)

var (
	v       float64 = 5
	stringV         = "value"
	boolV           = true
	dataV           = "base64"
	sum     float64 = 42
)

func TestSave(t *testing.T) {
	// client := mqtt.Client{}
	// repo := exporter.New(client)

	// cases := []struct {
	// 	desc         string
	// 	msgsNum      int
	// 	expectedSize int
	// }{
	// 	{
	// 		desc:         "save a single message",
	// 		msgsNum:      1,
	// 		expectedSize: 1,
	// 	},
	// 	{
	// 		desc:         "save a batch of messages",
	// 		msgsNum:      streamsSize,
	// 		expectedSize: streamsSize,
	// 	},
	// }

	// for _, tc := range cases {
	// 	// Clean previously saved messages.
	// 	now := time.Now().Unix()
	// 	msg := senml.Message{
	// 		Channel:    "45",
	// 		Publisher:  "2580",
	// 		Protocol:   "http",
	// 		Name:       "test name",
	// 		Unit:       "km",
	// 		UpdateTime: 5456565466,
	// 		Link:       "link",
	// 	}
	// 	var msgs []senml.Message

	// 	for i := 0; i < tc.msgsNum; i++ {
	// 		// Mix possible values as well as value sum.
	// 		count := i % valueFields
	// 		switch count {
	// 		case 0:
	// 			msg.Subtopic = subtopic
	// 			msg.Value = &v
	// 		case 1:
	// 			msg.BoolValue = &boolV
	// 		case 2:
	// 			msg.StringValue = &stringV
	// 		case 3:
	// 			msg.DataValue = &dataV
	// 		case 4:
	// 			msg.Sum = &sum
	// 		}

	// 		msg.Time = float64(now + int64(i))
	// 		msgs = append(msgs, msg)
	// 	}

	// 	err := repo.Save(msgs...)
	// 	assert.Nil(t, err, fmt.Sprintf("Save operation expected to succeed: %s.\n", err))

	// }
}
