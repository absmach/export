// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// +build !test

package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/go-zoo/bone"
	"github.com/mainflux/export/internal/app/export"
	"github.com/mainflux/mainflux"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	kithttp "github.com/go-kit/kit/transport/http"
)

const (
	contentType  = "application/json"
	maxLimit     = 100
	defaultLimit = 10
)

var (
	errUnsupportedContentType = errors.New("unsupported content type")
	errInvalidQueryParams     = errors.New("invalid query params")
	fullMatch                 = []string{"state", "external_id", "mainflux_id", "mainflux_key"}
	partialMatch              = []string{"name"}
)

// MakeHandler returns a HTTP API handler with version and metrics.
func MakeHandler(svc export.Service) http.Handler {
	r := bone.New()
	r.GetFunc("/version", export.Version())
	r.Handle("/metrics", promhttp.Handler())
	return r
}