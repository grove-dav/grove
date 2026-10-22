// SPDX-FileCopyrightText: 2026 Grove contributors
//
// SPDX-License-Identifier: Apache-2.0

// Package auth authenticates requests to Grove using OIDC bearer tokens issued
// by the oCIS identity provider.
package auth

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"path"
	"strings"
)

var (
	// ErrInvalidToken is returned by a Verifier when the token is malformed,
	// expired, signed by an unknown key, or issued for another audience.
	ErrInvalidToken = errors.New("invalid token")

	// ErrUnavailable is returned by a Verifier when the identity provider
	// can't be reached, so the token could not be checked either way.
	ErrUnavailable = errors.New("identity provider unavailable")
)

// Claims is the identity extracted from a verified token.
type Claims struct {
	// Subject is the stable user identifier (the "sub" claim). All ownership
	// keys on it.
	Subject string
	// PreferredUsername is for display only.
	PreferredUsername string
}

// Verifier checks a raw bearer token and returns the identity it carries.
type Verifier interface {
	Verify(ctx context.Context, rawToken string) (Claims, error)
}

type claimsKey struct{}

// ClaimsFromContext returns the claims that Middleware attached to ctx.
func ClaimsFromContext(ctx context.Context) (Claims, bool) {
	c, ok := ctx.Value(claimsKey{}).(Claims)
	return c, ok
}

// Middleware requires a valid "Authorization: Bearer" token on every request
// except the public paths, and stores the verified claims in the request
// context.
func Middleware(v Verifier, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isPublic(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			raw, ok := bearerToken(r)
			if !ok {
				unauthorized(w, "")
				return
			}

			claims, err := v.Verify(r.Context(), raw)
			switch {
			case err == nil:
				ctx := context.WithValue(r.Context(), claimsKey{}, claims)
				next.ServeHTTP(w, r.WithContext(ctx))
			case errors.Is(err, ErrUnavailable):
				logger.Error("token verification unavailable", "error", err)
				writeError(w, http.StatusServiceUnavailable, "identity provider unavailable")
			default:
				logger.Debug("token rejected", "error", err)
				unauthorized(w, "invalid_token")
			}
		})
	}
}

// isPublic reports whether path is served without authentication. The path is
// cleaned first so that "/web/../dav/..." can't borrow a public prefix.
func isPublic(p string) bool {
	p = path.Clean(p)
	switch p {
	case "/healthz", "/metrics":
		return true
	}
	return strings.HasPrefix(p, "/.well-known/") || strings.HasPrefix(p, "/web/")
}

func bearerToken(r *http.Request) (string, bool) {
	scheme, token, found := strings.Cut(r.Header.Get("Authorization"), " ")
	if !found || !strings.EqualFold(scheme, "Bearer") {
		return "", false
	}
	token = strings.TrimSpace(token)
	return token, token != ""
}

func unauthorized(w http.ResponseWriter, errCode string) {
	challenge := `Bearer realm="grove"`
	if errCode != "" {
		challenge += `, error="` + errCode + `"`
	}
	w.Header().Set("WWW-Authenticate", challenge)
	writeError(w, http.StatusUnauthorized, "unauthorized")
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
