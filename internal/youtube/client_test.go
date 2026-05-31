package youtube

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// newTestClient creates a Client wired to a test HTTP server.
func newTestClient(t *testing.T, handler http.Handler) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	tok := &Token{AccessToken: "test-token", ExpiresAt: time.Now().Add(time.Hour)}
	c := NewClientWithToken(nil, tok)
	// Override the base URL to point at the test server.
	c.baseURL = srv.URL
	return c, srv
}

func TestGet_Success(t *testing.T) {
	type response struct {
		Items []string `json:"items"`
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("missing or wrong Authorization header: %q", r.Header.Get("Authorization"))
		}
		if r.URL.Path != "/videos" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		if r.URL.Query().Get("part") != "snippet" {
			t.Errorf("missing query param part")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response{Items: []string{"a", "b"}})
	})

	c, srv := newTestClient(t, handler)
	defer srv.Close()

	var out response
	err := c.get(context.Background(), "/videos", map[string]string{"part": "snippet"}, &out)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if len(out.Items) != 2 {
		t.Errorf("Items len = %d, want 2", len(out.Items))
	}
}

func TestGet_APIError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"code":    403,
				"message": "forbidden",
			},
		})
	})

	c, srv := newTestClient(t, handler)
	defer srv.Close()

	var out any
	err := c.get(context.Background(), "/videos", nil, &out)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGet_ExpiredTokenNoRefresh(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not reach the server when token is expired and no refresh is available")
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	expiredTok := &Token{AccessToken: "old", ExpiresAt: time.Now().Add(-time.Minute)}
	c := NewClientWithToken(nil, expiredTok)
	c.baseURL = srv.URL

	var out any
	err := c.get(context.Background(), "/videos", nil, &out)
	if err == nil {
		t.Fatal("expected error for expired token with no refresh config")
	}
}

func TestGet_AutoRefresh(t *testing.T) {
	const newAccessToken = "refreshed-token"

	// First call goes to the token endpoint; second to the API.
	refreshCalled := false
	apiCalled := false

	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		refreshCalled = true
		json.NewEncoder(w).Encode(tokenResponse{
			AccessToken: newAccessToken,
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		})
	}))
	defer tokenSrv.Close()

	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCalled = true
		if r.Header.Get("Authorization") != "Bearer "+newAccessToken {
			t.Errorf("expected refreshed token in header, got %q", r.Header.Get("Authorization"))
		}
		json.NewEncoder(w).Encode(map[string]any{"items": []any{}})
	}))
	defer apiSrv.Close()

	cfg := &OAuthConfig{
		ClientID:     "cid",
		ClientSecret: "csec",
		tokenEndpoint: tokenSrv.URL,
	}
	expiredTok := &Token{
		AccessToken:  "expired",
		RefreshToken: "refresh-tok",
		ExpiresAt:    time.Now().Add(-time.Minute),
	}
	c := NewClientWithToken(cfg, expiredTok)
	c.baseURL = apiSrv.URL

	var out any
	if err := c.get(context.Background(), "/videos", nil, &out); err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if !refreshCalled {
		t.Error("expected token refresh to be called")
	}
	if !apiCalled {
		t.Error("expected API to be called after refresh")
	}
}
