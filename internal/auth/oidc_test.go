// SPDX-FileCopyrightText: 2026 Grove contributors
//
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

// fakeIdP is a minimal OIDC provider: discovery document, JWKS, and a helper
// to mint tokens signed with its key.
type fakeIdP struct {
	*httptest.Server
	key  *rsa.PrivateKey
	down atomic.Bool
}

func newFakeIdP(t *testing.T, tls bool) *fakeIdP {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	idp := &fakeIdP{key: key}

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                                idp.URL,
			"jwks_uri":                              idp.URL + "/jwks",
			"authorization_endpoint":                idp.URL + "/authorize",
			"token_endpoint":                        idp.URL + "/token",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{
			Key: &key.PublicKey, KeyID: "k1", Algorithm: "RS256", Use: "sig",
		}}})
	})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if idp.down.Load() {
			http.Error(w, "down", http.StatusServiceUnavailable)
			return
		}
		mux.ServeHTTP(w, r)
	})
	if tls {
		idp.Server = httptest.NewTLSServer(handler)
	} else {
		idp.Server = httptest.NewServer(handler)
	}
	t.Cleanup(idp.Close)
	return idp
}

// token signs claims with the IdP key. Zero-value iss/exp/sub get sane defaults.
func (idp *fakeIdP) token(t *testing.T, claims map[string]any) string {
	t.Helper()
	return signToken(t, idp.key, idp.URL, claims)
}

func signToken(t *testing.T, key *rsa.PrivateKey, issuer string, claims map[string]any) string {
	t.Helper()
	return signTokenAlg(t, jose.RS256, key, issuer, claims)
}

func signTokenAlg(t *testing.T, alg jose.SignatureAlgorithm, key *rsa.PrivateKey, issuer string, claims map[string]any) string {
	t.Helper()
	all := map[string]any{
		"iss": issuer,
		"sub": "u-123",
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	for k, v := range claims {
		if v == nil {
			delete(all, k)
			continue
		}
		all[k] = v
	}
	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: alg, Key: key},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", "k1"),
	)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := jwt.Signed(signer).Claims(all).Serialize()
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func newVerifier(t *testing.T, cfg OIDCConfig) *OIDCVerifier {
	t.Helper()
	v, err := NewOIDCVerifier(cfg)
	if err != nil {
		t.Fatalf("NewOIDCVerifier: %v", err)
	}
	return v
}

func TestOIDCVerifierAcceptsValidToken(t *testing.T) {
	idp := newFakeIdP(t, false)
	v := newVerifier(t, OIDCConfig{Issuer: idp.URL})

	got, err := v.Verify(context.Background(), idp.token(t, map[string]any{
		"sub":                "u-123",
		"preferred_username": "alice",
	}))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	want := Claims{Subject: "u-123", PreferredUsername: "alice"}
	if got != want {
		t.Errorf("claims = %+v, want %+v", got, want)
	}
}

// oCIS's built-in identity provider signs access tokens with PS256.
func TestOIDCVerifierAcceptsSigningAlgorithmsOfOCIS(t *testing.T) {
	idp := newFakeIdP(t, false)
	v := newVerifier(t, OIDCConfig{Issuer: idp.URL})

	for _, alg := range []jose.SignatureAlgorithm{jose.RS256, jose.PS256, jose.PS512} {
		t.Run(string(alg), func(t *testing.T) {
			raw := signTokenAlg(t, alg, idp.key, idp.URL, nil)
			if _, err := v.Verify(context.Background(), raw); err != nil {
				t.Fatalf("Verify: %v", err)
			}
		})
	}
}

func TestOIDCVerifierUsernameFallsBackToOCISLoginClaim(t *testing.T) {
	idp := newFakeIdP(t, false)
	v := newVerifier(t, OIDCConfig{Issuer: idp.URL})

	// oCIS access tokens carry no preferred_username; the login name sits
	// in lg.i.un.
	got, err := v.Verify(context.Background(), idp.token(t, map[string]any{
		"lg": map[string]any{"i": map[string]any{"un": "einstein"}},
	}))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if got.PreferredUsername != "einstein" {
		t.Errorf("PreferredUsername = %q, want %q", got.PreferredUsername, "einstein")
	}

	got, err = v.Verify(context.Background(), idp.token(t, map[string]any{
		"preferred_username": "alice",
		"lg":                 map[string]any{"i": map[string]any{"un": "einstein"}},
	}))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if got.PreferredUsername != "alice" {
		t.Errorf("PreferredUsername = %q, want preferred_username to win", got.PreferredUsername)
	}
}

func TestOIDCVerifierRejectsBadTokens(t *testing.T) {
	idp := newFakeIdP(t, false)
	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	tests := map[string]string{
		"expired":        idp.token(t, map[string]any{"exp": time.Now().Add(-time.Hour).Unix()}),
		"wrong issuer":   idp.token(t, map[string]any{"iss": "https://evil.example"}),
		"unknown signer": signToken(t, otherKey, idp.URL, nil),
		"missing sub":    idp.token(t, map[string]any{"sub": nil}),
		"empty sub":      idp.token(t, map[string]any{"sub": ""}),
		"not a jwt":      "definitely-not-a-jwt",
	}
	for name, raw := range tests {
		t.Run(name, func(t *testing.T) {
			v := newVerifier(t, OIDCConfig{Issuer: idp.URL})
			_, err := v.Verify(context.Background(), raw)
			if !errors.Is(err, ErrInvalidToken) {
				t.Fatalf("err = %v, want ErrInvalidToken", err)
			}
		})
	}
}

func TestOIDCVerifierAudience(t *testing.T) {
	idp := newFakeIdP(t, false)

	t.Run("matching audience accepted", func(t *testing.T) {
		v := newVerifier(t, OIDCConfig{Issuer: idp.URL, Audience: "grove"})
		if _, err := v.Verify(context.Background(), idp.token(t, map[string]any{"aud": "grove"})); err != nil {
			t.Fatalf("Verify: %v", err)
		}
	})
	t.Run("other audience rejected", func(t *testing.T) {
		v := newVerifier(t, OIDCConfig{Issuer: idp.URL, Audience: "grove"})
		_, err := v.Verify(context.Background(), idp.token(t, map[string]any{"aud": "web"}))
		if !errors.Is(err, ErrInvalidToken) {
			t.Fatalf("err = %v, want ErrInvalidToken", err)
		}
	})
	t.Run("audience ignored when not configured", func(t *testing.T) {
		v := newVerifier(t, OIDCConfig{Issuer: idp.URL})
		if _, err := v.Verify(context.Background(), idp.token(t, map[string]any{"aud": "anything"})); err != nil {
			t.Fatalf("Verify: %v", err)
		}
	})
}

func TestOIDCVerifierRecoversWhenIdentityProviderComesUp(t *testing.T) {
	idp := newFakeIdP(t, false)
	idp.down.Store(true)
	v := newVerifier(t, OIDCConfig{Issuer: idp.URL})
	raw := idp.token(t, nil)

	if _, err := v.Verify(context.Background(), raw); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("while down: err = %v, want ErrUnavailable", err)
	}

	idp.down.Store(false)
	if _, err := v.Verify(context.Background(), raw); err != nil {
		t.Fatalf("after recovery: %v", err)
	}
}

func TestOIDCVerifierUnreachableIssuer(t *testing.T) {
	idp := newFakeIdP(t, false)
	issuer := idp.URL
	idp.Close()

	v := newVerifier(t, OIDCConfig{Issuer: issuer})
	_, err := v.Verify(context.Background(), "a.b.c")
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("err = %v, want ErrUnavailable", err)
	}
}

func TestOIDCVerifierCABundle(t *testing.T) {
	idp := newFakeIdP(t, true)
	raw := idp.token(t, nil)

	t.Run("untrusted certificate", func(t *testing.T) {
		v := newVerifier(t, OIDCConfig{Issuer: idp.URL})
		if _, err := v.Verify(context.Background(), raw); !errors.Is(err, ErrUnavailable) {
			t.Fatalf("err = %v, want ErrUnavailable", err)
		}
	})
	t.Run("certificate trusted via CA file", func(t *testing.T) {
		caFile := filepath.Join(t.TempDir(), "ca.pem")
		pemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: idp.Certificate().Raw})
		if err := os.WriteFile(caFile, pemBytes, 0o600); err != nil {
			t.Fatal(err)
		}
		v := newVerifier(t, OIDCConfig{Issuer: idp.URL, CAFile: caFile})
		if _, err := v.Verify(context.Background(), raw); err != nil {
			t.Fatalf("Verify: %v", err)
		}
	})
}

func TestNewOIDCVerifierValidatesConfig(t *testing.T) {
	dir := t.TempDir()
	emptyPEM := filepath.Join(dir, "empty.pem")
	if err := os.WriteFile(emptyPEM, []byte("not a certificate"), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := map[string]OIDCConfig{
		"empty issuer":          {},
		"missing CA file":       {Issuer: "https://ocis.example", CAFile: filepath.Join(dir, "nope.pem")},
		"CA file without certs": {Issuer: "https://ocis.example", CAFile: emptyPEM},
	}
	for name, cfg := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := NewOIDCVerifier(cfg); err == nil {
				t.Fatal("NewOIDCVerifier succeeded, want an error")
			}
		})
	}
}
