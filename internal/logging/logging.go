// SPDX-FileCopyrightText: 2026 Grove contributors
//
// SPDX-License-Identifier: Apache-2.0

// Package logging configures Grove's structured logger.
package logging

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// ParseLevel parses a case-insensitive level name (debug, info, warn, error)
// into a slog.Level.
func ParseLevel(s string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("logging: unknown level %q", s)
	}
}

// New builds a JSON-handler slog.Logger writing to stdout at the given level.
func New(level string) (*slog.Logger, error) {
	lvl, err := ParseLevel(level)
	if err != nil {
		return nil, err
	}
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	return slog.New(h), nil
}
