package mlflow

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func newTestMlflowClient(srv *httptest.Server) *Client {
	u, _ := url.Parse(srv.URL)
	return &Client{
		httpClient: srv.Client(),
		url:        u,
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

func TestPromptText(t *testing.T) {
	t.Run("with matching tag", func(t *testing.T) {
		m := &RegisteredModel{
			Name: "test-prompt",
			LatestVersions: []ModelVersion{
				{
					Version: "1",
					Tags: []Tag{
						{Key: "mlflow.prompt.text", Value: "Hello {name}"},
						{Key: "other.tag", Value: "irrelevant"},
					},
				},
			},
		}
		if got := m.PromptText(); got != "Hello {name}" {
			t.Errorf("PromptText() = %q, want %q", got, "Hello {name}")
		}
	})

	t.Run("with no matching tag", func(t *testing.T) {
		m := &RegisteredModel{
			Name: "test-prompt",
			LatestVersions: []ModelVersion{
				{
					Version: "1",
					Tags: []Tag{
						{Key: "other.tag", Value: "irrelevant"},
					},
				},
			},
		}
		if got := m.PromptText(); got != "" {
			t.Errorf("PromptText() = %q, want empty", got)
		}
	})

	t.Run("with no latest versions", func(t *testing.T) {
		m := &RegisteredModel{
			Name:           "test-prompt",
			LatestVersions: nil,
		}
		if got := m.PromptText(); got != "" {
			t.Errorf("PromptText() = %q, want empty", got)
		}
	})
}

func TestGetPromptVersionByAlias(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %s", r.Method)
			}
			if r.URL.Path != "/api/2.0/mlflow/registered-models/alias" {
				t.Errorf("path = %s", r.URL.Path)
			}
			if got := r.URL.Query().Get("name"); got != "my-prompt" {
				t.Errorf("name = %q", got)
			}
			if got := r.URL.Query().Get("alias"); got != "production" {
				t.Errorf("alias = %q", got)
			}
			json.NewEncoder(w).Encode(struct {
				ModelVersion ModelVersion `json:"model_version"`
			}{
				ModelVersion: ModelVersion{
					Name:    "my-prompt",
					Version: "3",
					Tags:    []Tag{{Key: "mlflow.prompt.text", Value: "aliased prompt"}},
				},
			})
		}))
		defer srv.Close()

		c := newTestMlflowClient(srv)
		m, err := c.GetPromptVersionByAlias(context.Background(), "my-prompt", "production")
		if err != nil {
			t.Fatalf("GetPromptVersionByAlias: %v", err)
		}
		if m.Name != "my-prompt" {
			t.Errorf("Name = %q", m.Name)
		}
		if len(m.LatestVersions) != 1 || m.LatestVersions[0].Version != "3" {
			t.Errorf("LatestVersions = %+v", m.LatestVersions)
		}
		if m.PromptText() != "aliased prompt" {
			t.Errorf("PromptText = %q", m.PromptText())
		}
	})

	t.Run("api error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		c := newTestMlflowClient(srv)
		_, err := c.GetPromptVersionByAlias(context.Background(), "my-prompt", "nope")
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestGetPromptRef(t *testing.T) {
	t.Run("name@alias hits alias endpoint", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/2.0/mlflow/registered-models/alias" {
				t.Errorf("path = %s", r.URL.Path)
			}
			json.NewEncoder(w).Encode(struct {
				ModelVersion ModelVersion `json:"model_version"`
			}{
				ModelVersion: ModelVersion{
					Name:    "my-prompt",
					Version: "2",
					Tags:    []Tag{{Key: "mlflow.prompt.text", Value: "via alias"}},
				},
			})
		}))
		defer srv.Close()

		c := newTestMlflowClient(srv)
		m, err := c.GetPromptRef(context.Background(), "my-prompt@production")
		if err != nil {
			t.Fatalf("GetPromptRef: %v", err)
		}
		if m.PromptText() != "via alias" {
			t.Errorf("PromptText = %q", m.PromptText())
		}
	})

	t.Run("bare name hits get endpoint", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/2.0/mlflow/registered-models/get" {
				t.Errorf("path = %s", r.URL.Path)
			}
			json.NewEncoder(w).Encode(RegisteredModelResponse{
				RegisteredModel: RegisteredModel{
					Name: "my-prompt",
					LatestVersions: []ModelVersion{
						{Version: "1", Tags: []Tag{{Key: "mlflow.prompt.text", Value: "latest"}}},
					},
				},
			})
		}))
		defer srv.Close()

		c := newTestMlflowClient(srv)
		m, err := c.GetPromptRef(context.Background(), "my-prompt")
		if err != nil {
			t.Fatalf("GetPromptRef: %v", err)
		}
		if m.PromptText() != "latest" {
			t.Errorf("PromptText = %q", m.PromptText())
		}
	})

	t.Run("name@ treated as no alias", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/2.0/mlflow/registered-models/get" {
				t.Errorf("path = %s", r.URL.Path)
			}
			json.NewEncoder(w).Encode(RegisteredModelResponse{
				RegisteredModel: RegisteredModel{Name: "my-prompt"},
			})
		}))
		defer srv.Close()

		c := newTestMlflowClient(srv)
		m, err := c.GetPromptRef(context.Background(), "my-prompt@")
		if err != nil {
			t.Fatalf("GetPromptRef: %v", err)
		}
		if m.Name != "my-prompt" {
			t.Errorf("Name = %q", m.Name)
		}
	})

	t.Run("empty name errors", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("unexpected request")
		}))
		defer srv.Close()

		c := newTestMlflowClient(srv)
		if _, err := c.GetPromptRef(context.Background(), "@production"); err == nil {
			t.Fatal("expected error for empty prompt name")
		}
	})
}

func TestSplitPromptRef(t *testing.T) {
	cases := map[string]struct{ name, alias string }{
		"my-prompt@production": {name: "my-prompt", alias: "production"},
		"my-prompt":            {name: "my-prompt", alias: ""},
		"my-prompt@":           {name: "my-prompt", alias: ""},
		"@production":          {name: "", alias: "production"},
		"a@b@c":                {name: "a@b", alias: "c"},
	}
	for in, want := range cases {
		name, alias := splitPromptRef(in)
		if name != want.name || alias != want.alias {
			t.Errorf("splitPromptRef(%q) = (%q, %q), want (%q, %q)", in, name, alias, want.name, want.alias)
		}
	}
}
