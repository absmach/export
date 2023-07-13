// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test
// +build !test

package api

import (
	"net/http"

	"github.com/go-zoo/bone"
	"github.com/mainflux/export/pkg/export"
	"github.com/mainflux/mainflux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MakeHandler returns a HTTP API handler with version and metrics.
func MakeHandler(svc export.Service) http.Handler {
	r := bone.New()
	r.Handle("/metrics", promhttp.Handler())
	r.GetFunc("/health", mainflux.Health("export"))
	return r
}
