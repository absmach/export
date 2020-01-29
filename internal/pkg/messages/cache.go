// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package messages

// MessageCache contains message  caching interface.
type Cache interface {
	// Add message
	Add(string, string, []byte) (string, error)

	// Removes message from cache
	Remove(string, string) (int64, error)

	// Read group for redis streams
	ReadGroup(streams []string, group string, count int64, consumer string) (map[string]Msg, error)

	// Create redis group for reading streams
	GroupCreate(stream, group string) (string, error)
}
