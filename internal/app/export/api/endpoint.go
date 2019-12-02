// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/export/internal/app/export"
)

func addEndpoint(svc export.Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		return addRes{}, nil
	}
}

// func viewEndpoint(svc export.Service) endpoint.Endpoint {
// }

// func listEndpoint(svc export.Service) endpoint.Endpoint {
// }

// func removeEndpoint(svc export.Service) endpoint.Endpoint {
// 	return func(_ context.Context, request interface{}) (interface{}, error) {
// 		req := request.(entityReq)

// 		if err := req.validate(); err != nil {
// 			return removeRes{}, err
// 		}

// 		if err := svc.Remove(req.key, req.id); err != nil {
// 			return nil, err
// 		}

// 		return removeRes{}, nil
// 	}
// }
