package mlflow

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/manuelgilm/tubectl/internal"
)

func TestSaveAndLoadCredentials(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "auth", "mlflow.json")

	original := &Credentials{
		ServerURL: "https://mlflow.example.com",
		Username:  "admin",
		Password:  "secret123",
	}

	if err := SaveCredentials(path, original); err != nil {
		t.Fatalf("SaveCredentials: %v", err)
	}

	loaded, err := LoadCredentials(path)
	if err != nil {
		t.Fatalf("LoadCredentials: %v", err)
	}

	if loaded.ServerURL != original.ServerURL {
		t.Errorf("ServerURL = %q, want %q", loaded.ServerURL, original.ServerURL)
	}
	if loaded.Username != original.Username {
		t.Errorf("Username = %q, want %q", loaded.Username, original.Username)
	}
	if loaded.Password != original.Password {
		t.Errorf("Password = %q, want %q", loaded.Password, original.Password)
	}
}

func TestLoadCredentials_errors(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		_, err := LoadCredentials(filepath.Join(t.TempDir(), "nope.json"))
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})

	t.Run("corrupt json", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "bad.json")
		if err := os.WriteFile(path, []byte("{{{"), 0644); err != nil {
			t.Fatal(err)
		}
		_, err := LoadCredentials(path)
		if err == nil {
			t.Fatal("expected error for corrupt JSON")
		}
	})
}

func TestMLflowProviderLogin(t *testing.T) {
	t.Run("saves credentials", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "creds.json")

		p := NewMLflowProvider(path)
		err := p.Login(context.Background(), internal.Options{
			Username: "user1",
			Password: "pass1",
		})
		if err != nil {
			t.Fatalf("Login: %v", err)
		}

		creds, err := LoadCredentials(path)
		if err != nil {
			t.Fatalf("LoadCredentials: %v", err)
		}
		if creds.Username != "user1" || creds.Password != "pass1" {
			t.Errorf("got %+v", creds)
		}
	})

	t.Run("rejects empty username", func(t *testing.T) {
		p := NewMLflowProvider(filepath.Join(t.TempDir(), "creds.json"))
		err := p.Login(context.Background(), internal.Options{
			Username: "",
			Password: "pass",
		})
		if err == nil {
			t.Fatal("expected error for empty username")
		}
	})

	t.Run("rejects empty password", func(t *testing.T) {
		p := NewMLflowProvider(filepath.Join(t.TempDir(), "creds.json"))
		err := p.Login(context.Background(), internal.Options{
			Username: "user",
			Password: "",
		})
		if err == nil {
			t.Fatal("expected error for empty password")
		}
	})
}

func TestMLflowProviderLogout(t *testing.T) {
	t.Run("removes credentials file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "creds.json")
		SaveCredentials(path, &Credentials{Username: "u", Password: "p"})

		p := NewMLflowProvider(path)
		if err := p.Logout(); err != nil {
			t.Fatalf("Logout: %v", err)
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Error("credentials file still exists")
		}
	})
}

func TestMLflowProviderStatus(t *testing.T) {
	t.Run("authenticated when file exists", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "creds.json")
		SaveCredentials(path, &Credentials{Username: "u", Password: "p"})

		p := NewMLflowProvider(path)
		status, err := p.Status()
		if err != nil {
			t.Fatalf("Status: %v", err)
		}
		if !status.Authenticated {
			t.Error("expected authenticated")
		}
	})

	t.Run("not authenticated when file missing", func(t *testing.T) {
		p := NewMLflowProvider(filepath.Join(t.TempDir(), "nonexistent.json"))
		_, err := p.Status()
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
