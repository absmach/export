// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package messages

import (
	"github.com/go-redis/redis"
)

const (
	msgPrefix = "prefix"
	streamLen = 1000
)

var _ Cache = (*cache)(nil)

type cache struct {
	client *redis.Client
}

// NewMessageCache returns redis message cache implementation.
func NewRedisCache(client *redis.Client) Cache {
	return &cache{client: client}
}

func (cc *cache) Add(stream string, payload []byte) error {
	m := msg{payload: string(payload)}.encode()
	return cc.add(stream, m)
}

func (cc *cache) Remove(msgID string) error {
	return cc.client.Del(msgID).Err()
}

func (cc *cache) add(stream string, m map[string]interface{}) (string, error) {

	record := &redis.XAddArgs{
		Stream:       stream,
		MaxLenApprox: streamLen,
		Values:       m,
	}

	return cc.client.XAdd(record).Result()
}
