// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package export

// MessageRepository specifies message writing API.
type MessageRepository interface {
	// Publish method is used to publish message to cloud.
	Publish([]byte) error
}
