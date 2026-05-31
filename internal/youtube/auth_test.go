package youtube

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newTestOAuthConfig(serverURL string) *OAuthConfig {
	return &OAuthConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURI:  "http://localhost/callback",
		Scopes:       []string{ScopeYoutubeReadonly},
	}
}

func TestAuthCodeURL(t *testing.T) {
	cfg := newTestOAuthConfig("")
	authURL := cfg.AuthCodeURL("mystate")

	parsed, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("invalid URL: %v", err)
	}

	q := parsed.Query()
	tests := []struct{ key, want string }{
		{"client_id", cfg.ClientID},
		{"redirect_uri", cfg.RedirectURI},
		{"response_type", "code"},
		{"access_type", "offline"},
		{"state", "mystate"},
	}
	for _, tt := range tests {
		if got := q.Get(tt.key); got != tt.want {
			t.Errorf("query param %q = %q, want %q", tt.key, got, tt.want)
		}
	}
	if !strings.Contains(q.Get("scope"), ScopeYoutubeReadonly) {
		t.Errorf("scope %q missing %q", q.Get("scope"), ScopeYoutubeReadonly)
	}
}

func TestExchange_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parsing form: %v", err)
		}
		if r.Form.Get("grant_type") != "authorization_code" {
			t.Errorf("unexpected grant_type: %s", r.Form.Get("grant_type"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResponse{
			AccessToken:  "access-abc",
			RefreshToken: "refresh-xyz",
			TokenType:    "Bearer",
			ExpiresIn:    3600,
		})
	}))
	defer srv.Close()

	// Patch the token URL to point at the test server.
	orig := tokenURL
	t.Cleanup(func() {
		// tokenURL is a package-level const, so we use a variable in tests.
	})
	_ = orig // used below via the patched variable

	cfg := newTestOAuthConfig(srv.URL)
	tok, err := exchangeWithURL(context.Background(), cfg, srv.URL, url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {"auth-code-123"},
		"redirect_uri": {cfg.RedirectURI},
	})
	if err != nil {
		t.Fatalf("Exchange failed: %v", err)
	}
	if tok.AccessToken != "access-abc" {
		t.Errorf("AccessToken = %q, want %q", tok.AccessToken, "access-abc")
	}
	if tok.RefreshToken != "refresh-xyz" {
		t.Errorf("RefreshToken = %q, want %q", tok.RefreshToken, "refresh-xyz")
	}
	if tok.ExpiresAt.Before(time.Now()) {
		t.Error("token should not be expired immediately after exchange")
	}
}

func TestExchange_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResponse{
			Error:     "invalid_grant",
			ErrorDesc: "Token has been expired or revoked.",
		})
	}))
	defer srv.Close()

	cfg := newTestOAuthConfig(srv.URL)
	_, err := exchangeWithURL(context.Background(), cfg, srv.URL, url.Values{
		"grant_type": {"authorization_code"},
		"code":       {"bad-code"},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid_grant") {
		t.Errorf("error %q should mention invalid_grant", err.Error())
	}
}

func TestRefresh_PreservesRefreshToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Simulate Google not returning a new refresh token on refresh.
		json.NewEncoder(w).Encode(tokenResponse{
			AccessToken: "new-access-token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		})
	}))
	defer srv.Close()

	cfg := newTestOAuthConfig(srv.URL)
	tok, err := exchangeWithURL(context.Background(), cfg, srv.URL, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {"old-refresh-token"},
	})
	if err != nil {
		t.Fatalf("Refresh failed: %v", err)
	}
	if tok.AccessToken != "new-access-token" {
		t.Errorf("AccessToken = %q, want new-access-token", tok.AccessToken)
	}
	if tok.RefreshToken != "old-refresh-token" {
		t.Errorf("RefreshToken should be preserved as %q, got %q", "old-refresh-token", tok.RefreshToken)
	}
}

func TestTokenValid(t *testing.T) {
	valid := &Token{AccessToken: "tok", ExpiresAt: time.Now().Add(time.Hour)}
	if !valid.Valid() {
		t.Error("token with future expiry should be valid")
	}

	expired := &Token{AccessToken: "tok", ExpiresAt: time.Now().Add(-time.Second)}
	if expired.Valid() {
		t.Error("token with past expiry should not be valid")
	}

	empty := &Token{}
	if empty.Valid() {
		t.Error("token with empty AccessToken should not be valid")
	}
}

func TestSaveAndLoadToken(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")

	original := &Token{
		AccessToken:  "save-access",
		RefreshToken: "save-refresh",
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(time.Hour).Truncate(time.Second),
	}

	if err := SaveToken(path, original); err != nil {
		t.Fatalf("SaveToken: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat token file: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("token file permissions = %o, want 0600", info.Mode().Perm())
	}

	loaded, err := LoadToken(path)
	if err != nil {
		t.Fatalf("LoadToken: %v", err)
	}
	if loaded.AccessToken != original.AccessToken {
		t.Errorf("AccessToken = %q, want %q", loaded.AccessToken, original.AccessToken)
	}
	if loaded.RefreshToken != original.RefreshToken {
		t.Errorf("RefreshToken = %q, want %q", loaded.RefreshToken, original.RefreshToken)
	}
	if !loaded.ExpiresAt.Equal(original.ExpiresAt) {
		t.Errorf("ExpiresAt = %v, want %v", loaded.ExpiresAt, original.ExpiresAt)
	}
}

func TestLoadToken_FileNotFound(t *testing.T) {
	_, err := LoadToken("/nonexistent/path/token.json")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestExchange_ViaPublicMethod(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parsing form: %v", err)
		}
		if r.Form.Get("grant_type") != "authorization_code" {
			t.Errorf("unexpected grant_type: %s", r.Form.Get("grant_type"))
		}
		if r.Form.Get("code") != "auth-code-456" {
			t.Errorf("unexpected code: %s", r.Form.Get("code"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResponse{
			AccessToken:  "new-access",
			RefreshToken: "new-refresh",
			TokenType:    "Bearer",
			ExpiresIn:    3600,
		})
	}))
	defer srv.Close()

	cfg := &OAuthConfig{
		ClientID:      "cid",
		ClientSecret:  "csec",
		RedirectURI:   "urn:ietf:wg:oauth:2.0:oob",
		tokenEndpoint: srv.URL,
	}
	tok, err := cfg.Exchange(context.Background(), "auth-code-456")
	if err != nil {
		t.Fatalf("Exchange failed: %v", err)
	}
	if tok.AccessToken != "new-access" {
		t.Errorf("AccessToken = %q, want new-access", tok.AccessToken)
	}
	if tok.RefreshToken != "new-refresh" {
		t.Errorf("RefreshToken = %q, want new-refresh", tok.RefreshToken)
	}
}
