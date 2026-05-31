package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestClient(t *testing.T, handler http.Handler) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	c := NewClient("test-api-key", "gpt-4o-mini")
	c.baseURL = srv.URL
	return c, srv
}

func TestComplete_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-api-key" {
			t.Errorf("wrong Authorization header: %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("wrong Content-Type: %q", r.Header.Get("Content-Type"))
		}

		// Verify the request body contains the expected messages.
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decoding request body: %v", err)
		}
		if body["model"] != "gpt-4o-mini" {
			t.Errorf("model = %q, want gpt-4o-mini", body["model"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": "Hello, I'm happy to help!"}},
			},
		})
	}))
	defer srv.Close()

	c := NewClient("test-api-key", "gpt-4o-mini")
	c.baseURL = srv.URL

	reply, err := c.Complete(context.Background(), []Message{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: "Say hello."},
	})
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	if reply != "Hello, I'm happy to help!" {
		t.Errorf("reply = %q, want %q", reply, "Hello, I'm happy to help!")
	}
}

func TestComplete_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": "invalid_api_key",
			},
		})
	}))
	defer srv.Close()

	c := NewClient("bad-key", "")
	c.baseURL = srv.URL

	_, err := c.Complete(context.Background(), []Message{{Role: "user", Content: "hi"}})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "openai error: invalid_api_key" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestComplete_NoChoices(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{},
		})
	}))
	defer srv.Close()

	c := NewClient("test-api-key", "")
	c.baseURL = srv.URL

	_, err := c.Complete(context.Background(), []Message{{Role: "user", Content: "hi"}})
	if err == nil {
		t.Fatal("expected error for empty choices, got nil")
	}
}

func TestNewClient_DefaultModel(t *testing.T) {
	c := NewClient("key", "")
	if c.model != "gpt-4o-mini" {
		t.Errorf("default model = %q, want gpt-4o-mini", c.model)
	}
}
