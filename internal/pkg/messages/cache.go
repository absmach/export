// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package messages

// MessageCache contains message  caching interface.
type Cache interface {
	// Add message
	Add(string, []byte) (string, error)

	// Removes message from cache
	Remove(string) error
}
