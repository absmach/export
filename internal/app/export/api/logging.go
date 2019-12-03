// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// +build !test

package api

import (
	"fmt"
	"time"

	"github.com/mainflux/export/internal/app/export"
	log "github.com/mainflux/mainflux/logger"
)

var _ export.MessageRepository = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger log.Logger
	svc    export.MessageRepository
}

// LoggingMiddleware adds logging facilities to the adapter.
func LoggingMiddleware(svc export.MessageRepository, logger log.Logger) export.MessageRepository {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) Publish(msgs ...interface{}) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method save took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Publish(msgs...)
}
