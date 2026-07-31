// SPDX-FileCopyrightText: 2026 Grove contributors
//
// SPDX-License-Identifier: Apache-2.0

// Package config loads Grove's configuration from defaults, an optional YAML
// file, and environment variables, in that precedence order (env wins).
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/grove-dav/grove/internal/logging"
)

// Config holds Grove's runtime configuration.
type Config struct {
	HTTPAddr   string `yaml:"http_addr"`
	LogLevel   string `yaml:"log_level"`
	OIDCIssuer string `yaml:"oidc_issuer"`
	DBDSN      string `yaml:"db_dsn"`
}

// Default returns the built-in configuration defaults.
func Default() Config {
	return Config{
		HTTPAddr: ":8080",
		LogLevel: "info",
	}
}

// Load builds a Config starting from Default(), overlaying the YAML file at
// path (if path is non-empty), then overlaying any set GROVE_* environment
// variables. Environment variables always win over the file, which always
// wins over defaults.
func Load(path string) (Config, error) {
	cfg := Default()

	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return Config{}, fmt.Errorf("config: reading %s: %w", path, err)
		}
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return Config{}, fmt.Errorf("config: parsing %s: %w", path, err)
		}
	}

	if v, ok := os.LookupEnv("GROVE_HTTP_ADDR"); ok {
		cfg.HTTPAddr = v
	}
	if v, ok := os.LookupEnv("GROVE_LOG_LEVEL"); ok {
		cfg.LogLevel = v
	}
	if v, ok := os.LookupEnv("GROVE_OIDC_ISSUER"); ok {
		cfg.OIDCIssuer = v
	}
	if v, ok := os.LookupEnv("GROVE_DB_DSN"); ok {
		cfg.DBDSN = v
	}

	return cfg, nil
}

// Validate checks that cfg is well-formed.
func (cfg Config) Validate() error {
	if cfg.HTTPAddr == "" {
		return fmt.Errorf("config: http_addr must not be empty")
	}
	if _, err := logging.ParseLevel(cfg.LogLevel); err != nil {
		return fmt.Errorf("config: log_level: %w", err)
	}
	return nil
}
