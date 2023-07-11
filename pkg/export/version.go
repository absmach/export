// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package export

import (
	"encoding/json"
	"net/http"
)

const version string = "0.0.1"

// VersionInfo contains version endpoint response.
type VersionInfo struct {
	// Service contains service name.
	Service string `json:"service"`

	// Version contains service current version value.
	Version string `json:"version"`
}

// Version exposes an HTTP handler for retrieving service version.
func Version() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		res := VersionInfo{"export", version}

		data, err := json.Marshal(res)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}

		if _, err = w.Write(data); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	})
}
