// SPDX-FileCopyrightText: 2026 Grove contributors
//
// SPDX-License-Identifier: Apache-2.0

// Package server builds and runs Grove's HTTP server.
package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// Server wraps the HTTP server serving Grove's endpoints.
type Server struct {
	httpServer *http.Server
	logger     *slog.Logger
	addr       string
}

// New builds a Server listening on addr, logging through logger.
func New(addr string, logger *slog.Logger) *Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthzHandler)
	mux.HandleFunc("GET /metrics", metricsHandler())

	return &Server{
		httpServer: &http.Server{
			Addr:    addr,
			Handler: withRequestLogging(logger, mux),
		},
		logger: logger,
		addr:   addr,
	}
}

// Run starts the server and blocks until ctx is canceled, then shuts down
// gracefully.
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		s.logger.Info("http server listening", "addr", s.addr)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("server: listen and serve: %w", err)
			return
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		s.logger.Info("shutting down http server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server: shutdown: %w", err)
		}
		s.logger.Info("http server stopped cleanly")
		return nil
	}
}

func withRequestLogging(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)

		level := slog.LevelInfo
		if r.URL.Path == "/healthz" || r.URL.Path == "/metrics" {
			level = slog.LevelDebug
		}
		logger.Log(r.Context(), level, "http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
