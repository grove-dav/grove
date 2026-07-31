// SPDX-FileCopyrightText: 2026 Grove contributors
//
// SPDX-License-Identifier: Apache-2.0

// Command grove runs the Grove calendar/contacts service.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/grove-dav/grove/internal/config"
	"github.com/grove-dav/grove/internal/logging"
	"github.com/grove-dav/grove/internal/server"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "grove:", err)
		os.Exit(1)
	}
}

func run() error {
	configPath := flag.String("config", "", "path to a grove.yaml config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	logger, err := logging.New(cfg.LogLevel)
	if err != nil {
		return err
	}
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	srv := server.New(cfg.HTTPAddr, logger)
	return srv.Run(ctx)
}
