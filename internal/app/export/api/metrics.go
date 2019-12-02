// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"time"

	"github.com/go-kit/kit/metrics"
	"github.com/mainflux/export/internal/app/export"
)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	repo    export.MessageRepository
}

// MetricsMiddleware returns new message repository
// with Save method wrapped to expose metrics.
func MetricsMiddleware(repo export.MessageRepository, counter metrics.Counter, latency metrics.Histogram) export.MessageRepository {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		repo:    repo,
	}
}

func (mm *metricsMiddleware) Publish(msgs ...interface{}) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "handle_message").Add(1)
		mm.latency.With("method", "handle_message").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return mm.repo.Publish(msgs...)
}
