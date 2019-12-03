// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// +build !test
package api

import (
	"net/http"

	"github.com/go-zoo/bone"
	"github.com/mainflux/export/internal/app/export"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MakeHandler returns a HTTP API handler with version and metrics.
func MakeHandler(svc export.Service) http.Handler {
	r := bone.New()
	r.GetFunc("/version", export.Version())
	r.Handle("/metrics", promhttp.Handler())
	return r
}
