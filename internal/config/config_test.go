// SPDX-FileCopyrightText: 2026 Grove contributors
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load(\"\"): unexpected error: %v", err)
	}
	want := Default()
	if cfg != want {
		t.Errorf("Load(\"\") = %+v, want %+v", cfg, want)
	}
}

func TestLoadFileOverridesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "grove.yaml")
	yamlContent := "http_addr: \":9090\"\n"
	if err := os.WriteFile(path, []byte(yamlContent), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load(%q): unexpected error: %v", path, err)
	}
	if cfg.HTTPAddr != ":9090" {
		t.Errorf("HTTPAddr = %q, want %q", cfg.HTTPAddr, ":9090")
	}
	// LogLevel was omitted from the file, so the default survives.
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q (default should survive)", cfg.LogLevel, "info")
	}
}

func TestLoadEnvOverridesFileAndDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "grove.yaml")
	yamlContent := "http_addr: \":9090\"\nlog_level: \"warn\"\n"
	if err := os.WriteFile(path, []byte(yamlContent), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	t.Setenv("GROVE_HTTP_ADDR", ":7070")
	t.Setenv("GROVE_LOG_LEVEL", "debug")
	t.Setenv("GROVE_OIDC_ISSUER", "https://idp.example.com")
	t.Setenv("GROVE_OIDC_AUDIENCE", "grove")
	t.Setenv("GROVE_OIDC_CA_FILE", "/etc/grove/ca.pem")
	t.Setenv("GROVE_DB_DSN", "postgres://example")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load(%q): unexpected error: %v", path, err)
	}
	if cfg.HTTPAddr != ":7070" {
		t.Errorf("HTTPAddr = %q, want env value %q", cfg.HTTPAddr, ":7070")
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want env value %q", cfg.LogLevel, "debug")
	}
	if cfg.OIDCIssuer != "https://idp.example.com" {
		t.Errorf("OIDCIssuer = %q, want env value", cfg.OIDCIssuer)
	}
	if cfg.OIDCAudience != "grove" {
		t.Errorf("OIDCAudience = %q, want env value", cfg.OIDCAudience)
	}
	if cfg.OIDCCAFile != "/etc/grove/ca.pem" {
		t.Errorf("OIDCCAFile = %q, want env value", cfg.OIDCCAFile)
	}
	if cfg.DBDSN != "postgres://example" {
		t.Errorf("DBDSN = %q, want env value", cfg.DBDSN)
	}
}

func TestLoadFileReadsOIDCSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "grove.yaml")
	yamlContent := "oidc_issuer: https://ocis.example\noidc_audience: grove\noidc_ca_file: /etc/grove/ca.pem\n"
	if err := os.WriteFile(path, []byte(yamlContent), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load(%q): unexpected error: %v", path, err)
	}
	if cfg.OIDCIssuer != "https://ocis.example" || cfg.OIDCAudience != "grove" || cfg.OIDCCAFile != "/etc/grove/ca.pem" {
		t.Errorf("OIDC settings = %q, %q, %q", cfg.OIDCIssuer, cfg.OIDCAudience, cfg.OIDCCAFile)
	}
}

func TestLoadMissingFile(t *testing.T) {
	if _, err := Load("/nonexistent/grove.yaml"); err == nil {
		t.Fatal("Load with nonexistent path: expected error, got nil")
	}
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{"valid", Config{HTTPAddr: ":8080", LogLevel: "info", OIDCIssuer: "https://ocis.example"}, false},
		{"empty addr", Config{HTTPAddr: "", LogLevel: "info", OIDCIssuer: "https://ocis.example"}, true},
		{"bad level", Config{HTTPAddr: ":8080", LogLevel: "bogus", OIDCIssuer: "https://ocis.example"}, true},
		{"missing issuer", Config{HTTPAddr: ":8080", LogLevel: "info"}, true},
	}

	for _, tc := range cases {
		err := tc.cfg.Validate()
		if tc.wantErr && err == nil {
			t.Errorf("%s: expected error, got nil", tc.name)
		}
		if !tc.wantErr && err != nil {
			t.Errorf("%s: unexpected error: %v", tc.name, err)
		}
	}
}
