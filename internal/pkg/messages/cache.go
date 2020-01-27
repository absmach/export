// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package messages

import "github.com/go-redis/redis"

// MessageCache contains message  caching interface.
type Cache interface {
	// Add message
	Add(string, string, []byte) (string, error)

	// Removes message from cache
	Remove(string) error

	// Read group for redis streams
	ReadGroup(streams []string, group, consumer string) ([]redis.XStream, error)

	// Create redis group for reading streams
	GroupCreate(stream, group string) (string, error)
}
