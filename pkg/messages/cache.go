// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package messages

// Cache contains message  caching interface.
type Cache interface {
	// Add message
	Add(stream string, topic string, msg []byte) (string, error)

	// Removes message from cache
	Remove(stream string, msgID string) (int64, error)

	// Read group for redis streams
	ReadGroup(streams []string, group string, count int64, consumer string) (map[string]Msg, int64, error)

	// Create redis group for reading streams
	GroupCreate(stream, group string) (string, error)
}
