package cmd

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/manuelgilm/tubectl/internal/prompting"
	"github.com/manuelgilm/tubectl/internal/youtube"
	"github.com/spf13/cobra"
)

func TestPromptFileRender(t *testing.T) {
	t.Run("all vars present", func(t *testing.T) {
		p := &prompting.PromptFile{
			Template: "Hello {name}, you are {age} years old",
			Vars:     []string{"name", "age"},
		}
		result, err := p.Render(map[string]string{"name": "Alice", "age": "30"})
		if err != nil {
			t.Fatalf("Render: %v", err)
		}
		want := "Hello Alice, you are 30 years old"
		if result != want {
			t.Errorf("got %q, want %q", result, want)
		}
	})

	t.Run("missing var", func(t *testing.T) {
		p := &prompting.PromptFile{
			Template: "Hello {name}",
			Vars:     []string{"name"},
		}
		_, err := p.Render(map[string]string{})
		if err == nil {
			t.Fatal("expected error for missing var")
		}
	})

	t.Run("no vars defined", func(t *testing.T) {
		p := &prompting.PromptFile{
			Template: "Static text",
			Vars:     nil,
		}
		result, err := p.Render(nil)
		if err != nil {
			t.Fatalf("Render: %v", err)
		}
		if result != "Static text" {
			t.Errorf("got %q", result)
		}
	})

	t.Run("multiple occurrences of same var", func(t *testing.T) {
		p := &prompting.PromptFile{
			Template: "{x} + {x} = {y}",
			Vars:     []string{"x", "y"},
		}
		result, err := p.Render(map[string]string{"x": "1", "y": "2"})
		if err != nil {
			t.Fatalf("Render: %v", err)
		}
		if result != "1 + 1 = 2" {
			t.Errorf("got %q", result)
		}
	})
}

func TestResolvePromptFromMLflowRegistry_FallsBackToDefault(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("MLFLOW_TRACKING_USERNAME", "")
	t.Setenv("MLFLOW_TRACKING_PASSWORD", "")

	cmd := &cobra.Command{}
	cmd.SetErr(&bytes.Buffer{})

	template, err := ResolvePromptFromMLflowRegistry(cmd, "sama_bot_system_prompt", "hello comment", "transcript text")
	if err != nil {
		t.Fatalf("expected fallback without error, got: %v", err)
	}
	if !strings.Contains(template, "Gilsama-Bot") {
		t.Errorf("expected fallback to default prompt, got: %s", template)
	}
	if !strings.Contains(template, "hello comment") {
		t.Errorf("expected comment text in template, got: %s", template)
	}
}

func TestLoadPromptFile(t *testing.T) {
	t.Run("valid yaml", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "prompt.yaml")
		content := "template: \"Reply to {comment}\"\nvars:\n  - comment\n"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		p, err := prompting.LoadPromptFile(path)
		if err != nil {
			t.Fatalf("LoadPromptFile: %v", err)
		}
		if p.Template != "Reply to {comment}" {
			t.Errorf("template = %q", p.Template)
		}
		if len(p.Vars) != 1 || p.Vars[0] != "comment" {
			t.Errorf("vars = %v", p.Vars)
		}
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := prompting.LoadPromptFile("/nonexistent/path.yaml")
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("empty template", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "empty.yaml")
		content := "template: \"\"\nvars: []\n"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		_, err := prompting.LoadPromptFile(path)
		if err == nil {
			t.Fatal("expected error for empty template")
		}
	})
}

func TestLoadOpenAIClient_GatewayPreference(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("MLFLOW_TRACKING_USERNAME", "user")
	t.Setenv("MLFLOW_TRACKING_PASSWORD", "pass")
	t.Setenv("MLFLOW_SERVER_URL", "https://ml.example.com")
	t.Setenv("OPENAI_API_KEY", "")

	c, err := loadOpenAIClient(context.Background(), "gpt-4o-mini")
	if err != nil {
		t.Fatalf("loadOpenAIClient: %v", err)
	}
	if !strings.Contains(c.BaseURL(), "/gateway/mlflow/v1") {
		t.Errorf("expected gateway base URL, got %q", c.BaseURL())
	}
	if !strings.Contains(c.BaseURL(), "https://ml.example.com") {
		t.Errorf("expected MLflow server in base URL, got %q", c.BaseURL())
	}
}

func TestLoadOpenAIClient_FallsBackToOpenAIWithoutCreds(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("MLFLOW_TRACKING_USERNAME", "")
	t.Setenv("MLFLOW_TRACKING_PASSWORD", "")
	t.Setenv("MLFLOW_SERVER_URL", "")
	t.Setenv("OPENAI_API_KEY", "sk-test")

	c, err := loadOpenAIClient(context.Background(), "gpt-4o-mini")
	if err != nil {
		t.Fatalf("loadOpenAIClient: %v", err)
	}
	if strings.Contains(c.BaseURL(), "/gateway/mlflow/v1") {
		t.Errorf("expected direct OpenAI, got gateway base URL %q", c.BaseURL())
	}
}

func TestPrintTranscript(t *testing.T) {
	transcript := &youtube.Transcript{
		Lines: []youtube.TranscriptLine{
			{Start: 0, Text: "Hello"},
			{Start: 65.5, Text: "World"},
		},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printTranscript(transcript)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "[00:00] Hello") {
		t.Errorf("missing first line, got: %s", output)
	}
	if !strings.Contains(output, "[01:05] World") {
		t.Errorf("missing second line, got: %s", output)
	}
}

func TestGenerateAnswer_ServerResponded_NoFallback(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("MLFLOW_TRACKING_USERNAME", "user")
	t.Setenv("MLFLOW_TRACKING_PASSWORD", "pass")

	// Gateway responds but rejects the request (5xx). This is a config/server
	// problem and must surface loudly, NOT fall back to OpenAI.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("gateway upstream error"))
	}))
	defer srv.Close()
	t.Setenv("MLFLOW_SERVER_URL", srv.URL)
	t.Setenv("OPENAI_API_KEY", "") // if fallback ran it would error on missing key

	cmd := &cobra.Command{}
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetContext(context.Background())

	_, err := GenerateAnswer(cmd, "resolved template", "gpt-4o-mini")
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "OPENAI_API_KEY") {
		t.Fatalf("server-responded error fell back to OpenAI: %v", err)
	}
	if !strings.Contains(err.Error(), "502") {
		t.Errorf("expected gateway status error, got: %v", err)
	}
}

func TestGenerateAnswer_Connectivity_FallsBackToOpenAI(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("MLFLOW_TRACKING_USERNAME", "user")
	t.Setenv("MLFLOW_TRACKING_PASSWORD", "pass")

	// Gateway is unreachable (closed server).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()
	t.Setenv("MLFLOW_SERVER_URL", url)
	t.Setenv("OPENAI_API_KEY", "") // fallback will attempt OpenAI and fail on missing key

	cmd := &cobra.Command{}
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetContext(context.Background())

	_, err := GenerateAnswer(cmd, "resolved template", "gpt-4o-mini")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "OPENAI_API_KEY") {
		t.Errorf("expected fallback to OpenAI (missing key error), got: %v", err)
	}
}
