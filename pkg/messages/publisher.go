// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0
package messages

import "github.com/mainflux/mainflux/errors"

type Publisher interface {
	Publish(stream string, topic string, msg []byte) errors.Error
}
