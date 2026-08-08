package ai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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

	t.Run("basic auth header", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, pass, ok := r.BasicAuth()
			if !ok || user != "mlflow-user" || pass != "secret" {
				t.Errorf("basic auth = %q/%q (ok=%v), want mlflow-user/secret", user, pass, ok)
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"choices": []any{
					map[string]any{
						"message": map[string]any{"content": "via gateway"},
					},
				},
			})
		}))
		defer srv.Close()

		c := NewClient("irrelevant", "gpt-4o-mini").
			WithBaseURL(srv.URL).
			WithHTTPClient(srv.Client()).
			WithBasicAuth("mlflow-user", "secret")

		reply, err := c.Complete(context.Background(), []Message{{Role: "user", Content: "Hi"}})
		if err != nil {
			t.Fatalf("Complete: %v", err)
		}
		if reply != "via gateway" {
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

func TestWithBasicAuth(t *testing.T) {
	c := NewClient("key", "gpt-4o-mini")
	if c.username != "" || c.password != "" {
		t.Error("expected empty credentials initially")
	}
	c.WithBasicAuth("user", "pass")
	if c.username != "user" || c.password != "pass" {
		t.Errorf("credentials = %q/%q", c.username, c.password)
	}

	c2 := NewClient("key", "gpt-4o-mini").WithBaseURL("http://gateway")
	if c2.BaseURL() != "http://gateway" {
		t.Errorf("BaseURL = %q", c2.BaseURL())
	}
}

func TestWithBaseURLAndHTTPClient(t *testing.T) {
	c := NewClient("key", "gpt-4o-mini")
	if c.BaseURL() != defaultBaseURL {
		t.Errorf("BaseURL = %q, want %q", c.BaseURL(), defaultBaseURL)
	}
	c.WithBaseURL("http://custom")
	if c.BaseURL() != "http://custom" {
		t.Errorf("BaseURL = %q", c.BaseURL())
	}
}

func TestConnectivityError(t *testing.T) {
	t.Run("network failure wrapped as ConnectivityError", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		url := srv.URL
		srv.Close()

		c := &Client{
			apiKey:  "sk-test",
			model:   "gpt-4o-mini",
			baseURL: url,
			http:    srv.Client(),
		}

		_, err := c.Complete(context.Background(), []Message{{Role: "user", Content: "Hi"}})
		if err == nil {
			t.Fatal("expected error for closed server")
		}
		var connErr *ConnectivityError
		if !errors.As(err, &connErr) {
			t.Errorf("expected *ConnectivityError, got %T", err)
		}
	})

	t.Run("http 4xx is NOT a ConnectivityError", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": {"message": "bad key"}}`))
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
		var connErr *ConnectivityError
		if errors.As(err, &connErr) {
			t.Errorf("did not expect *ConnectivityError for HTTP response")
		}
	})

	t.Run("200 empty choices is NOT a ConnectivityError", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"choices": []any{}})
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
			t.Fatal("expected error")
		}
		var connErr *ConnectivityError
		if errors.As(err, &connErr) {
			t.Errorf("did not expect *ConnectivityError for empty choices")
		}
	})
}

func TestComplete_StatusCode(t *testing.T) {
	t.Run("non-json error body included in message", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte("upstream exploded"))
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
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "502") || !strings.Contains(err.Error(), "upstream exploded") {
			t.Errorf("error = %v", err)
		}
		var statusErr *HTTPStatusError
		if !errors.As(err, &statusErr) {
			t.Fatalf("expected *HTTPStatusError, got %T", err)
		}
		if statusErr.StatusCode != http.StatusBadGateway {
			t.Errorf("StatusCode = %d", statusErr.StatusCode)
		}
		if statusErr.Body != "upstream exploded" {
			t.Errorf("Body = %q", statusErr.Body)
		}
	})

	t.Run("empty body on non-2xx still errors", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusGatewayTimeout)
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
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "504") {
			t.Errorf("error = %v", err)
		}
		var statusErr *HTTPStatusError
		if !errors.As(err, &statusErr) {
			t.Fatalf("expected *HTTPStatusError, got %T", err)
		}
		if statusErr.StatusCode != http.StatusGatewayTimeout {
			t.Errorf("StatusCode = %d", statusErr.StatusCode)
		}
		if statusErr.Body != "" {
			t.Errorf("Body = %q, want empty", statusErr.Body)
		}
	})
}
