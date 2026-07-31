// SPDX-FileCopyrightText: 2026 Grove contributors
//
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// metricsHandler exposes Prometheus metrics. client_golang auto-registers
// go_* and process_* collectors on import, which is sufficient
// instrumentation until Grove has application-specific metrics to add.
func metricsHandler() http.HandlerFunc {
	h := promhttp.Handler()
	return func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r)
	}
}
