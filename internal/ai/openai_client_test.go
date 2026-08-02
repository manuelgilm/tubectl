package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// mockTracer records CreateSpan calls for testing.
type mockTracer struct {
	spanReqs []SpanRequest
}

func (m *mockTracer) CreateSpan(_ context.Context, req SpanRequest) error {
	m.spanReqs = append(m.spanReqs, req)
	return nil
}

func (m *mockTracer) assertSpanCount(t *testing.T, n int) {
	t.Helper()
	if len(m.spanReqs) != n {
		t.Errorf("CreateSpan called %d times, want %d", len(m.spanReqs), n)
	}
}

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

	t.Run("http status with json error body", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{"message": "invalid_api_key"},
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
		if !strings.Contains(err.Error(), "401") || !strings.Contains(err.Error(), "invalid_api_key") {
			t.Errorf("error = %q, want status and message", err)
		}
	})

	t.Run("http status with plain text body", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte("rate limit exceeded"))
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
		if !strings.Contains(err.Error(), "429") || !strings.Contains(err.Error(), "rate limit exceeded") {
			t.Errorf("error = %q, want status and body", err)
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

func TestCompleteWithTracer(t *testing.T) {
	t.Run("tracer called on success", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"choices": []any{
					map[string]any{
						"message":       map[string]any{"content": "Hello"},
						"finish_reason": "stop",
					},
				},
				"usage": map[string]any{
					"prompt_tokens":     10,
					"completion_tokens": 20,
					"total_tokens":      30,
				},
			})
		}))
		defer srv.Close()

		tr := &mockTracer{}
		c := &Client{
			apiKey:  "sk-test",
			model:   "gpt-4o-mini",
			baseURL: srv.URL,
			http:    srv.Client(),
			tracer:  tr,
		}

		reply, err := c.Complete(context.Background(), []Message{{Role: "user", Content: "Hi"}})
		if err != nil {
			t.Fatalf("Complete: %v", err)
		}
		if reply != "Hello" {
			t.Errorf("reply = %q", reply)
		}

		tr.assertSpanCount(t, 1)

		req := tr.spanReqs[0]
		if req.Model != "gpt-4o-mini" {
			t.Errorf("model = %q", req.Model)
		}
		if req.Response != "Hello" {
			t.Errorf("response = %q", req.Response)
		}
		if req.FinishReason != "stop" {
			t.Errorf("finish_reason = %q", req.FinishReason)
		}
		if req.PromptTokens != 10 || req.CompletionTokens != 20 || req.TotalTokens != 30 {
			t.Errorf("usage: %+v", req)
		}
		if req.Error != "" {
			t.Errorf("error = %q, want empty", req.Error)
		}
		if len(req.TraceID) != 32 {
			t.Errorf("TraceID length = %d, want 32", len(req.TraceID))
		}
	})

	t.Run("tracer called on api error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{"message": "bad key"},
			})
		}))
		defer srv.Close()

		tr := &mockTracer{}
		c := &Client{
			apiKey:  "bad",
			model:   "gpt-4o-mini",
			baseURL: srv.URL,
			http:    srv.Client(),
			tracer:  tr,
		}

		_, err := c.Complete(context.Background(), nil)
		if err == nil {
			t.Fatal("expected error")
		}

		tr.assertSpanCount(t, 1)

		req := tr.spanReqs[0]
		if req.Error == "" {
			t.Error("expected non-empty error in span")
		}
	})

	t.Run("tracer not called when nil", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"choices": []any{
					map[string]any{
						"message": map[string]any{"content": "Hello"},
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

		_, err := c.Complete(context.Background(), []Message{{Role: "user", Content: "Hi"}})
		if err != nil {
			t.Fatalf("Complete: %v", err)
		}
	})
}

func TestWithTracer(t *testing.T) {
	c := NewClient("key", "gpt-4o-mini")
	if c.tracer != nil {
		t.Error("expected nil tracer initially")
	}
	tr := &mockTracer{}
	c.WithTracer(tr)
	if c.tracer != tr {
		t.Error("WithTracer did not set tracer")
	}
}
