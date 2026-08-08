package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestLoadOwnerClient_fallsBackToDefault(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TUBECTL_HOME", dir)

	auth := filepath.Join(dir, "auth")
	if err := os.MkdirAll(auth, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	writeTestToken(t, filepath.Join(auth, "youtube.json"))

	client, err := loadOwnerClient(context.Background())
	if err != nil {
		t.Fatalf("loadOwnerClient: %v", err)
	}
	if client == nil {
		t.Fatal("expected a client")
	}
}

func TestLoadOwnerClient_prefersOwnerToken(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TUBECTL_HOME", dir)

	auth := filepath.Join(dir, "auth")
	if err := os.MkdirAll(auth, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	// Owner token present, default token deliberately corrupt: if the owner
	// token were ignored, loading the default would fail.
	writeTestToken(t, filepath.Join(auth, "youtube.owner.json"))
	if err := os.WriteFile(filepath.Join(auth, "youtube.json"), []byte("{not json"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	client, err := loadOwnerClient(context.Background())
	if err != nil {
		t.Fatalf("loadOwnerClient: %v", err)
	}
	if client == nil {
		t.Fatal("expected a client")
	}
}

func TestLoadOwnerClient_errorWhenNoToken(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TUBECTL_HOME", dir)

	if _, err := loadOwnerClient(context.Background()); err == nil {
		t.Fatal("expected error when no token exists")
	}
}

func writeTestToken(t *testing.T, path string) {
	t.Helper()
	tok := youtube.Token{
		AccessToken:  "test-access",
		RefreshToken: "test-refresh",
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(time.Hour),
	}
	data, err := json.Marshal(tok)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}
