// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package export

// MessageRepository specifies message writing API.
type MessageRepository interface {
	// Publish method is used to save published message to cloud. A non-nil
	// error is returned to indicate  operation failure.
	Publish([]byte) error
}
