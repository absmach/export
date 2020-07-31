// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0
package messages

type Publisher interface {
	// Publish message to topic, save into stream if publish fails
	Publish(stream string, topic string, msg []byte) error
}
