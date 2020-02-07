// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package messages

import (
	"github.com/go-redis/redis"
	"github.com/mainflux/mainflux/errors"
)

const (
	streamLen = 1000
)

var _ Cache = (*cache)(nil)

var (
	ErrGrpCreateGroupMissing  = errors.New("Group not created, group not being set")
	ErrGrpCreateStreamMissing = errors.New("Group not created, stream not being set")
	ErrNoStreamData           = errors.New("Empty stream")
	ErrOneStreamOnlyRead      = errors.New("One stream read only")
	ErrDecodingData           = errors.New("Failed to decode read data")
)

type cache struct {
	client *redis.Client
}

// NewMessageCache returns redis message cache implementation.
func NewRedisCache(client *redis.Client) Cache {
	if client != nil {
		return &cache{client: client}
	}
	return nil
}

func (cc *cache) Add(stream, topic string, payload []byte) (string, error) {
	m := Msg{Topic: topic, Payload: string(payload)}
	return cc.add(stream, m.Encode())
}

func (cc *cache) Remove(stream, msgID string) (int64, error) {
	return cc.client.XDel(stream, msgID).Result()
}

func (cc *cache) GroupCreate(stream, group string) (string, error) {
	if stream == "" || group == ""{
		return "", ErrGrpCreateStreamMissing
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

func (cc *cache) ReadGroup(streams []string, group string, count int64, consumer string) (map[string]Msg, int64, error) {
	xReadGroupArgs := &redis.XReadGroupArgs{
		Group:    group,
		Consumer: consumer,
		Streams:  streams,
		Count:    count,
		Block:    0,
	}
	xStreams, err := cc.client.XReadGroup(xReadGroupArgs).Result() //Get Results from redis
	if err != nil {
		return nil, 0, err
	}
	result := make(map[string]Msg)
	if len(xStreams) > 1 {
		return nil, 0, ErrOneStreamOnlyRead
	}
	if len(xStreams) == 0 {
		return nil, 0, ErrNoStreamData
	}
	errors := 0
	for _, xMessage := range xStreams[0].Messages { // Get the message from the xStream
		m := new(Msg)
		err := m.Decode(xMessage.Values)
		if err != nil {
			errors = errors + 1
			continue
		}
		result[xMessage.ID] = *m
	}

	return result, int64(errors), nil
}
