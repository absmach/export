// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package messages

import (
	"fmt"

	"github.com/go-redis/redis"
	"github.com/mainflux/mainflux/errors"
)

const (
	msgPrefix = "prefix"
	streamLen = 1000
)

var _ Cache = (*cache)(nil)

var (
	errGrpCreateGroupMissing  = errors.New("Group not created, group not being set")
	errGrpCreateStreamMissing = errors.New("Group not created, stream not being set")
)

type cache struct {
	client *redis.Client
}

// NewMessageCache returns redis message cache implementation.
func NewRedisCache(client *redis.Client) Cache {
	return &cache{client: client}
}

func (cc *cache) Add(stream, topic string, payload []byte) (string, error) {
	m := msg{topic: topic, payload: string(payload)}.encode()
	return cc.add(stream, m)
}

func (cc *cache) Remove(stream, msgID string) (int64, error) {
	return cc.client.XDel(stream, msgID).Result()
}

func (cc *cache) GroupCreate(stream, group string) (string, error) {
	if stream == "" {
		return "", errGrpCreateStreamMissing
	}
	if group == "" {
		return "", errGrpCreateGroupMissing
	}
	return cc.client.XGroupCreateMkStream(stream, group, "$").Result()
}

func (cc *cache) add(stream string, m map[string]interface{}) (string, error) {

	record := &redis.XAddArgs{
		Stream:       stream,
		MaxLenApprox: streamLen,
		Values:       m,
	}

	return cc.client.XAdd(record).Result()
}

func (cc *cache) Read(streams []string, count int64, consumer string) ([]interface{}, error) {
	xReadArgs := &redis.XReadArgs{
		Streams: []string{"test.test", "0-0"},
		Count:   count,
		Block:   0,
	}
	xStreams, err := cc.client.XRead(xReadArgs).Result() //Get Results from XRead command
	if err != nil {
		return nil, err
	}
	for _, xStream := range xStreams { //Get individual xStream
		//streamName := xStream.Stream
		for _, xMessage := range xStream.Messages { // Get the message from the xStream

		}
	}
	return nil, err
}

func (cc *cache) ReadGroup(streams []string, group string, count int64, consumer string) ([]interface{}, error) {

	xReadGroupArgs := &redis.XReadGroupArgs{
		Group:    group,
		Consumer: consumer,
		Streams:  streams,
		Count:    count,
		Block:    0,
	}

	xStreams, err := cc.client.XReadGroup(xReadGroupArgs).Result() //Get Results from XRead command

	// cc.client.XReadGroup(streams, "export-group", consumer)
	if err != nil {
		return nil, err
	}

	for _, xStream := range xStreams { //Get individual xStream
		//streamName := xStream.Stream
		for _, xMessage := range xStream.Messages { // Get the message from the xStream
			for _, v := range xMessage.Values { // Get the values from the message
				// TO DO
				fmt.Println("test:%s", v)
			}

		}
	}
	return nil, err
}
