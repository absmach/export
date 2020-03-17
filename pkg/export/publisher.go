// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0
package export

import "github.com/mainflux/mainflux/errors"

type Publisher interface {
	Publish(string, string, []byte) errors.Error
}
