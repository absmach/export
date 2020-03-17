// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0
package messages

import "github.com/mainflux/mainflux/errors"

type Publisher interface {
	Publish(string, string, []byte) errors.Error
}
