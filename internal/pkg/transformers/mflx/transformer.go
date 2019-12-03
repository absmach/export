// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mflx

import (
	"github.com/gogo/protobuf/proto"
	"github.com/mainflux/export/internal/pkg/transformers"
	"github.com/mainflux/mainflux"
)

type transformer struct{}

// New returns transformer service implementation for SenML messages.
func New() transformers.Transformer {
	return transformer{}
}

func (n transformer) Transform(m []byte) (interface{}, error) {
	msg := mainflux.Message{}
	err := proto.Unmarshal(m, &msg)
	if err != nil {
		return msg, err
	}

	return msg, nil
}
