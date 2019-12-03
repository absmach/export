// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package plain

import (
	"github.com/mainflux/export/internal/pkg/transformers"
)

type transformer struct{}

// New returns transformer service implementation for SenML messages.
func New() transformers.Transformer {
	return transformer{}
}

func (n transformer) Transform(m []byte) (interface{}, error) {
	return m, nil
}
