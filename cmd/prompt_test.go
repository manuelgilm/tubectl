package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/manuelgilm/tubectl/internal/prompt"
)

func TestLoadMlflowClient_withEnvVars(t *testing.T) {
	t.Setenv("MLFLOW_TRACKING_USERNAME", "env-user")
	t.Setenv("MLFLOW_TRACKING_PASSWORD", "env-pass")

	client, err := loadMlflowClient()
	if err != nil {
		t.Fatalf("loadMlflowClient: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestLoadMlflowClient_fallbackToCredentials(t *testing.T) {
	home := t.TempDir()
	credsDir := filepath.Join(home, ".tubectl", "auth")
	credsPath := filepath.Join(credsDir, "mlflow.json")
	if err := os.MkdirAll(credsDir, 0700); err != nil {
		t.Fatal(err)
	}

	prompt.SaveCredentials(credsPath, &prompt.Credentials{
		Username: "file-user",
		Password: "file-pass",
	})

	t.Setenv("HOME", home)
	// Unset env vars to test fallback
	t.Setenv("MLFLOW_TRACKING_USERNAME", "")
	t.Setenv("MLFLOW_TRACKING_PASSWORD", "")

	client, err := loadMlflowClient()
	if err != nil {
		t.Fatalf("loadMlflowClient: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestLoadMlflowClient_errorWhenUnavailable(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("MLFLOW_TRACKING_USERNAME", "")
	t.Setenv("MLFLOW_TRACKING_PASSWORD", "")

	_, err := loadMlflowClient()
	if err == nil {
		t.Fatal("expected error when no credentials available")
	}
}

func writeTestMlflowCreds(t *testing.T, home string) {
	t.Helper()
	credsDir := filepath.Join(home, ".tubectl", "auth")
	if err := os.MkdirAll(credsDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := prompt.SaveCredentials(filepath.Join(credsDir, "mlflow.json"), &prompt.Credentials{
		Username:  "file-user",
		Password:  "file-pass",
		ServerURL: "https://file.example.com",
	}); err != nil {
		t.Fatal(err)
	}
}

func TestResolveMLflowCreds_partialEnvUsesFile(t *testing.T) {
	home := t.TempDir()
	writeTestMlflowCreds(t, home)
	t.Setenv("HOME", home)
	t.Setenv("MLFLOW_TRACKING_USERNAME", "env-user")
	t.Setenv("MLFLOW_TRACKING_PASSWORD", "")

	creds, err := resolveMLflowCreds()
	if err != nil {
		t.Fatalf("resolveMLflowCreds: %v", err)
	}
	// A partial env pair must not be mixed with file creds: the file wins entirely.
	if creds.username != "file-user" || creds.password != "file-pass" {
		t.Errorf("got %s/%s, want file-user/file-pass", creds.username, creds.password)
	}
	if creds.serverURL != "https://file.example.com" {
		t.Errorf("serverURL = %q", creds.serverURL)
	}
}

func TestResolveMLflowCreds_envPairWins(t *testing.T) {
	home := t.TempDir()
	writeTestMlflowCreds(t, home)
	t.Setenv("HOME", home)
	t.Setenv("MLFLOW_TRACKING_USERNAME", "env-user")
	t.Setenv("MLFLOW_TRACKING_PASSWORD", "env-pass")
	t.Setenv("MLFLOW_SERVER_URL", "https://env.example.com")

	creds, err := resolveMLflowCreds()
	if err != nil {
		t.Fatalf("resolveMLflowCreds: %v", err)
	}
	if creds.username != "env-user" || creds.password != "env-pass" {
		t.Errorf("got %s/%s, want env-user/env-pass", creds.username, creds.password)
	}
	if creds.serverURL != "https://env.example.com" {
		t.Errorf("serverURL = %q", creds.serverURL)
	}
}
