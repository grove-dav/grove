// SPDX-FileCopyrightText: 2026 Grove contributors
//
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"encoding/json"
	"net/http"
)

// healthzHandler reports liveness. There is no database yet to check for
// readiness; once one exists, this handler can add a DB ping and return 503
// on failure without changing the response shape.
func healthzHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
