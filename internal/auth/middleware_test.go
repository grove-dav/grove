// SPDX-FileCopyrightText: 2026 Grove contributors
//
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeVerifier struct {
	claims Claims
	err    error
	gotRaw string
	calls  int
}

func (f *fakeVerifier) Verify(_ context.Context, raw string) (Claims, error) {
	f.calls++
	f.gotRaw = raw
	return f.claims, f.err
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// serve runs a request through the middleware wrapped around a handler that
// records whether it was reached.
func serve(t *testing.T, v Verifier, req *http.Request) (rec *httptest.ResponseRecorder, reached bool, got Claims) {
	t.Helper()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		got, _ = ClaimsFromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})
	rec = httptest.NewRecorder()
	Middleware(v, testLogger())(next).ServeHTTP(rec, req)
	return rec, reached, got
}

func TestMiddlewarePublicPathsSkipAuth(t *testing.T) {
	for _, path := range []string{
		"/healthz",
		"/metrics",
		"/.well-known/caldav",
		"/.well-known/carddav",
		"/web/grove.js",
		"/web/assets/chunk-abc.js",
	} {
		t.Run(path, func(t *testing.T) {
			v := &fakeVerifier{}
			rec, reached, _ := serve(t, v, httptest.NewRequest(http.MethodGet, path, nil))
			if !reached || rec.Code != http.StatusNoContent {
				t.Fatalf("status = %d, reached = %v; want the request to pass through", rec.Code, reached)
			}
			if v.calls != 0 {
				t.Errorf("verifier called %d times for a public path", v.calls)
			}
		})
	}
}

func TestMiddlewareProtectedPathsRequireToken(t *testing.T) {
	for _, path := range []string{
		"/dav/calendars/alice/personal/",
		"/graph/v1.0/me/events",
		"/healthzx",
		"/metrics/extra",
		"/web",
		"/webx/grove.js",
		"/.well-known",
		"/web/../dav/calendars/alice/",
		"/",
	} {
		t.Run(path, func(t *testing.T) {
			v := &fakeVerifier{}
			rec, reached, _ := serve(t, v, httptest.NewRequest(http.MethodGet, path, nil))
			if reached {
				t.Fatal("request reached the handler without a token")
			}
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
			}
			if got := rec.Header().Get("WWW-Authenticate"); got != `Bearer realm="grove"` {
				t.Errorf("WWW-Authenticate = %q, want %q", got, `Bearer realm="grove"`)
			}
		})
	}
}

func TestMiddlewareRejectsNonBearerAuthorization(t *testing.T) {
	for _, header := range []string{
		"Basic YWxpY2U6c2VjcmV0",
		"Bearer",
		"Bearer ",
		"Token abc",
		"abc",
	} {
		t.Run(header, func(t *testing.T) {
			v := &fakeVerifier{}
			req := httptest.NewRequest(http.MethodGet, "/dav/calendars/alice/", nil)
			req.Header.Set("Authorization", header)
			rec, reached, _ := serve(t, v, req)
			if reached || rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, reached = %v; want 401 and no pass-through", rec.Code, reached)
			}
			if v.calls != 0 {
				t.Errorf("verifier called %d times for a malformed header", v.calls)
			}
		})
	}
}

func TestMiddlewarePassesVerifiedClaimsToHandler(t *testing.T) {
	want := Claims{Subject: "u-123", PreferredUsername: "alice"}
	v := &fakeVerifier{claims: want}
	req := httptest.NewRequest(http.MethodGet, "/graph/v1.0/me/events", nil)
	req.Header.Set("Authorization", "bearer tok-abc")

	rec, reached, got := serve(t, v, req)

	if !reached || rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, reached = %v; want the request to pass through", rec.Code, reached)
	}
	if v.gotRaw != "tok-abc" {
		t.Errorf("verifier got token %q, want %q", v.gotRaw, "tok-abc")
	}
	if got != want {
		t.Errorf("claims in context = %+v, want %+v", got, want)
	}
}

func TestMiddlewareInvalidToken(t *testing.T) {
	v := &fakeVerifier{err: fmt.Errorf("%w: expired", ErrInvalidToken)}
	req := httptest.NewRequest(http.MethodGet, "/graph/v1.0/me/events", nil)
	req.Header.Set("Authorization", "Bearer bad")

	rec, reached, _ := serve(t, v, req)

	if reached || rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, reached = %v; want 401 and no pass-through", rec.Code, reached)
	}
	want := `Bearer realm="grove", error="invalid_token"`
	if got := rec.Header().Get("WWW-Authenticate"); got != want {
		t.Errorf("WWW-Authenticate = %q, want %q", got, want)
	}
}

func TestMiddlewareIdentityProviderUnavailable(t *testing.T) {
	v := &fakeVerifier{err: fmt.Errorf("%w: connection refused", ErrUnavailable)}
	req := httptest.NewRequest(http.MethodGet, "/graph/v1.0/me/events", nil)
	req.Header.Set("Authorization", "Bearer tok")

	rec, reached, _ := serve(t, v, req)

	if reached || rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, reached = %v; want 503 and no pass-through", rec.Code, reached)
	}
}

func TestMiddlewareUnexpectedVerifierErrorFailsClosed(t *testing.T) {
	v := &fakeVerifier{err: fmt.Errorf("something else")}
	req := httptest.NewRequest(http.MethodGet, "/graph/v1.0/me/events", nil)
	req.Header.Set("Authorization", "Bearer tok")

	rec, reached, _ := serve(t, v, req)

	if reached || rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, reached = %v; want 401 and no pass-through", rec.Code, reached)
	}
}

func TestClaimsFromContextEmpty(t *testing.T) {
	if _, ok := ClaimsFromContext(context.Background()); ok {
		t.Error("ClaimsFromContext on an empty context reported ok")
	}
}
