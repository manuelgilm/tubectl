package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient(t *testing.T) {
	t.Run("default model", func(t *testing.T) {
		c := NewClient("key", "")
		if c.model != "gpt-4o-mini" {
			t.Errorf("model = %q, want gpt-4o-mini", c.model)
		}
	})

	t.Run("custom model", func(t *testing.T) {
		c := NewClient("key", "gpt-4")
		if c.model != "gpt-4" {
			t.Errorf("model = %q", c.model)
		}
	})

	t.Run("api key set", func(t *testing.T) {
		c := NewClient("sk-test", "gpt-4o-mini")
		if c.apiKey != "sk-test" {
			t.Errorf("apiKey = %q", c.apiKey)
		}
	})

	t.Run("default base url", func(t *testing.T) {
		c := NewClient("key", "gpt-4o-mini")
		if c.baseURL != defaultBaseURL {
			t.Errorf("baseURL = %q", c.baseURL)
		}
	})
}

func TestComplete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("method = %s", r.Method)
			}
			if r.Header.Get("Authorization") != "Bearer sk-test" {
				t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"choices": []any{
					map[string]any{
						"message": map[string]any{
							"content": "Hello from AI",
						},
					},
				},
			})
		}))
		defer srv.Close()

		c := &Client{
			apiKey:  "sk-test",
			model:   "gpt-4o-mini",
			baseURL: srv.URL,
			http:    srv.Client(),
		}

		reply, err := c.Complete(context.Background(), []Message{
			{Role: "user", Content: "Hi"},
		})
		if err != nil {
			t.Fatalf("Complete: %v", err)
		}
		if reply != "Hello from AI" {
			t.Errorf("reply = %q", reply)
		}
	})

	t.Run("api error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{
					"message": "invalid_api_key",
				},
			})
		}))
		defer srv.Close()

		c := &Client{
			apiKey:  "bad",
			model:   "gpt-4o-mini",
			baseURL: srv.URL,
			http:    srv.Client(),
		}

		_, err := c.Complete(context.Background(), nil)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("empty choices", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"choices": []any{},
			})
		}))
		defer srv.Close()

		c := &Client{
			apiKey:  "sk-test",
			model:   "gpt-4o-mini",
			baseURL: srv.URL,
			http:    srv.Client(),
		}

		_, err := c.Complete(context.Background(), nil)
		if err == nil {
			t.Fatal("expected error for empty choices")
		}
	})

	t.Run("non-json response", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("not json"))
		}))
		defer srv.Close()

		c := &Client{
			apiKey:  "sk-test",
			model:   "gpt-4o-mini",
			baseURL: srv.URL,
			http:    srv.Client(),
		}

		_, err := c.Complete(context.Background(), nil)
		if err == nil {
			t.Fatal("expected error for non-JSON response")
		}
	})

	t.Run("network error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		srv.Close()

		c := &Client{
			apiKey:  "sk-test",
			model:   "gpt-4o-mini",
			baseURL: srv.URL,
			http:    srv.Client(),
		}

		_, err := c.Complete(context.Background(), nil)
		if err == nil {
			t.Fatal("expected error for closed server")
		}
	})
}
