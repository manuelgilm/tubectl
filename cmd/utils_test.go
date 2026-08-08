package cmd

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/manuelgilm/tubectl/internal/youtube"
	"github.com/spf13/cobra"
)

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

func TestGenerateAnswer_GatewayEndpointNotFound_FallsBackToOpenAI(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("MLFLOW_TRACKING_USERNAME", "user")
	t.Setenv("MLFLOW_TRACKING_PASSWORD", "pass")

	// Gateway is up but has no endpoint for the requested model. This means
	// the model is not deployed there, so the request must fall back to OpenAI.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"detail":{"error_code":"RESOURCE_DOES_NOT_EXIST","message":"GatewayEndpoint not found (name='gpt-4o-mini')"}}`))
	}))
	defer srv.Close()
	t.Setenv("MLFLOW_SERVER_URL", srv.URL)
	t.Setenv("OPENAI_API_KEY", "") // if fallback ran it would error on missing key

	var stderr bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetErr(&stderr)
	cmd.SetContext(context.Background())

	_, err := GenerateAnswer(cmd, "resolved template", "gpt-4o-mini")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "OPENAI_API_KEY") {
		t.Errorf("expected fallback to OpenAI (missing key error), got: %v", err)
	}
	if !strings.Contains(stderr.String(), "falling back to OpenAI") {
		t.Errorf("expected fallback warning on stderr, got: %q", stderr.String())
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
