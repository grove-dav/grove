// SPDX-FileCopyrightText: 2026 Grove contributors
//
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
)

// discoveryTimeout bounds each request to the identity provider (discovery
// document and key set).
const discoveryTimeout = 10 * time.Second

// signingAlgs lists the RSA signature algorithms accepted on tokens. go-oidc
// defaults to RS256 only, but the oCIS identity provider signs with PS256.
var signingAlgs = []string{
	oidc.RS256, oidc.RS384, oidc.RS512,
	oidc.PS256, oidc.PS384, oidc.PS512,
}

// OIDCConfig configures an OIDCVerifier.
type OIDCConfig struct {
	// Issuer is the OIDC issuer URL, e.g. the oCIS public URL.
	Issuer string
	// Audience, when set, must appear in the token's "aud" claim. oCIS sets
	// "aud" to the OAuth client ID of the app that requested the token (for
	// example "web"), so a value here admits only that client. When empty the
	// audience isn't checked.
	Audience string
	// CAFile is an optional PEM bundle of extra CAs to trust when talking to
	// the issuer, on top of the system roots.
	CAFile string
}

// OIDCVerifier validates access tokens against an OIDC issuer's published
// keys. The issuer is contacted on first use and again after any failure, so
// Grove can start before the identity provider is ready.
type OIDCVerifier struct {
	cfg    OIDCConfig
	client *http.Client

	mu       sync.Mutex
	verifier *oidc.IDTokenVerifier
}

// NewOIDCVerifier validates cfg and loads the CA bundle. It doesn't contact the
// issuer.
func NewOIDCVerifier(cfg OIDCConfig) (*OIDCVerifier, error) {
	if cfg.Issuer == "" {
		return nil, errors.New("auth: oidc issuer must not be empty")
	}

	client := &http.Client{Timeout: discoveryTimeout}
	if cfg.CAFile != "" {
		pool, err := loadCAPool(cfg.CAFile)
		if err != nil {
			return nil, err
		}
		client.Transport = &http.Transport{
			Proxy:           http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12},
		}
	}
	return &OIDCVerifier{cfg: cfg, client: client}, nil
}

func loadCAPool(path string) (*x509.CertPool, error) {
	pemBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("auth: reading oidc CA file: %w", err)
	}
	pool, err := x509.SystemCertPool()
	if err != nil {
		pool = x509.NewCertPool()
	}
	if !pool.AppendCertsFromPEM(pemBytes) {
		return nil, fmt.Errorf("auth: oidc CA file %s contains no PEM certificates", path)
	}
	return pool, nil
}

// Verify implements Verifier.
func (v *OIDCVerifier) Verify(ctx context.Context, rawToken string) (Claims, error) {
	verifier, err := v.idTokenVerifier(ctx)
	if err != nil {
		return Claims{}, fmt.Errorf("%w: %w", ErrUnavailable, err)
	}

	// The verifier checks signature, issuer, expiry and (optionally) audience.
	token, err := verifier.Verify(oidc.ClientContext(ctx, v.client), rawToken)
	if err != nil {
		return Claims{}, fmt.Errorf("%w: %w", ErrInvalidToken, err)
	}
	if token.Subject == "" {
		return Claims{}, fmt.Errorf("%w: missing sub claim", ErrInvalidToken)
	}

	var extra struct {
		PreferredUsername string `json:"preferred_username"`
		// oCIS access tokens carry the login name here instead of in
		// preferred_username.
		Login struct {
			Identity struct {
				Username string `json:"un"`
			} `json:"i"`
		} `json:"lg"`
	}
	if err := token.Claims(&extra); err != nil {
		return Claims{}, fmt.Errorf("%w: decoding claims: %w", ErrInvalidToken, err)
	}
	username := extra.PreferredUsername
	if username == "" {
		username = extra.Login.Identity.Username
	}
	return Claims{Subject: token.Subject, PreferredUsername: username}, nil
}

// idTokenVerifier returns the cached go-oidc verifier, running discovery
// first if it hasn't succeeded yet.
func (v *OIDCVerifier) idTokenVerifier(ctx context.Context) (*oidc.IDTokenVerifier, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.verifier != nil {
		return v.verifier, nil
	}

	provider, err := oidc.NewProvider(oidc.ClientContext(ctx, v.client), v.cfg.Issuer)
	if err != nil {
		return nil, err
	}
	v.verifier = provider.Verifier(&oidc.Config{
		ClientID:             v.cfg.Audience,
		SkipClientIDCheck:    v.cfg.Audience == "",
		SupportedSigningAlgs: signingAlgs,
	})
	return v.verifier, nil
}
