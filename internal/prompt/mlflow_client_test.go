package prompt

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestMlflowClient(srv *httptest.Server) *Client {
	return &Client{
		httpClient: srv.Client(),
		baseURL:    srv.URL,
		username:   "testuser",
		password:   "testpass",
	}
}

func TestGetPrompt(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %s", r.Method)
			}
			if r.URL.Query().Get("name") != "my-prompt" {
				t.Errorf("name = %s", r.URL.Query().Get("name"))
			}
			json.NewEncoder(w).Encode(RegisteredModelResponse{
				RegisteredModel: RegisteredModel{
					Name: "my-prompt",
					LatestVersions: []ModelVersion{
						{Version: "1", Tags: []Tag{{Key: "mlflow.prompt.text", Value: "hello"}}},
					},
				},
			})
		}))
		defer srv.Close()

		c := newTestMlflowClient(srv)
		m, err := c.GetPrompt(context.Background(), "my-prompt")
		if err != nil {
			t.Fatalf("GetPrompt: %v", err)
		}
		if m.Name != "my-prompt" {
			t.Errorf("Name = %q", m.Name)
		}
		if m.PromptText() != "hello" {
			t.Errorf("PromptText = %q", m.PromptText())
		}
	})

	t.Run("api error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		c := newTestMlflowClient(srv)
		_, err := c.GetPrompt(context.Background(), "missing")
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("invalid json response", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("not json"))
		}))
		defer srv.Close()

		c := newTestMlflowClient(srv)
		_, err := c.GetPrompt(context.Background(), "bad")
		if err == nil {
			t.Fatal("expected error for invalid JSON")
		}
	})
}

func TestListPrompts(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("filter") != `tag.mlflow.prompt.is_prompt = "true"` {
				t.Errorf("filter = %s", r.URL.Query().Get("filter"))
			}
			json.NewEncoder(w).Encode(ModelVersionSearchResponse{
				ModelVersions: []ModelVersion{
					{Name: "p1", Version: "1"},
					{Name: "p2", Version: "2"},
				},
			})
		}))
		defer srv.Close()

		c := newTestMlflowClient(srv)
		versions, err := c.ListPrompts(context.Background())
		if err != nil {
			t.Fatalf("ListPrompts: %v", err)
		}
		if len(versions) != 2 {
			t.Fatalf("got %d versions, want 2", len(versions))
		}
		if versions[0].Name != "p1" || versions[1].Name != "p2" {
			t.Errorf("versions = %+v", versions)
		}
	})

	t.Run("empty list", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(ModelVersionSearchResponse{})
		}))
		defer srv.Close()

		c := newTestMlflowClient(srv)
		versions, err := c.ListPrompts(context.Background())
		if err != nil {
			t.Fatalf("ListPrompts: %v", err)
		}
		if len(versions) != 0 {
			t.Errorf("got %d versions, want 0", len(versions))
		}
	})

	t.Run("api error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()

		c := newTestMlflowClient(srv)
		_, err := c.ListPrompts(context.Background())
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
