// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package transformers

// Transformer specifies API form Message transformer.
type Transformer interface {
	// Transform Mainflux message to any other format.
	Transform([]byte) (interface{}, error)
}
