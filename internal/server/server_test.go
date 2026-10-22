// SPDX-FileCopyrightText: 2026 Grove contributors
//
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/grove-dav/grove/internal/auth"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestHealthz(t *testing.T) {
	s := New(":0", testLogger(), nil)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Body.String(); !strings.Contains(got, `"status":"ok"`) {
		t.Errorf("body = %q, want it to contain %q", got, `"status":"ok"`)
	}
}

func TestMetrics(t *testing.T) {
	s := New(":0", testLogger(), nil)
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Body.String(); !strings.Contains(got, "go_goroutines") {
		t.Errorf("body does not contain expected metric name %q", "go_goroutines")
	}
}

func TestUnknownRoute(t *testing.T) {
	s := New(":0", testLogger(), nil)
	req := httptest.NewRequest(http.MethodGet, "/nope", nil)
	rec := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestRunGracefulShutdown(t *testing.T) {
	s := New("127.0.0.1:0", testLogger(), nil)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- s.Run(ctx)
	}()

	// Give the server a moment to start listening, then request shutdown.
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned error after shutdown: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after context cancellation")
	}
}

type stubVerifier struct{ err error }

func (s stubVerifier) Verify(_ context.Context, _ string) (auth.Claims, error) {
	return auth.Claims{Subject: "u-1"}, s.err
}

func TestAuthMiddlewareProtectsEverythingButPublicPaths(t *testing.T) {
	s := New(":0", testLogger(), auth.Middleware(stubVerifier{}, testLogger()))

	tests := []struct {
		name   string
		path   string
		header string
		want   int
	}{
		{"healthz is public", "/healthz", "", http.StatusOK},
		{"metrics is public", "/metrics", "", http.StatusOK},
		{"unregistered path without token fails closed", "/dav/calendars/alice/", "", http.StatusUnauthorized},
		{"unregistered path with token reaches the router", "/dav/calendars/alice/", "Bearer tok", http.StatusNotFound},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			rec := httptest.NewRecorder()
			s.httpServer.Handler.ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("status = %d, want %d", rec.Code, tc.want)
			}
		})
	}
}
