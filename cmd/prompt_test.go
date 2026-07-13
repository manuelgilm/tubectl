package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"tubectl/internal/prompt"
)

func TestLoadMlflowClient_withEnvVars(t *testing.T) {
	t.Setenv("MLFLOW_USERNAME", "env-user")
	t.Setenv("MLFLOW_PASSWORD", "env-pass")

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
	t.Setenv("MLFLOW_USERNAME", "")
	t.Setenv("MLFLOW_PASSWORD", "")

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
	t.Setenv("MLFLOW_USERNAME", "")
	t.Setenv("MLFLOW_PASSWORD", "")

	_, err := loadMlflowClient()
	if err == nil {
		t.Fatal("expected error when no credentials available")
	}
}


