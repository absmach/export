// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0
package publish

type Publisher interface {
	Publish(string, string, []byte)
}
