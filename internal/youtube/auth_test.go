package youtube

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestTokenValid(t *testing.T) {
	t.Run("valid token", func(t *testing.T) {
		tok := &Token{
			AccessToken: "abc",
			ExpiresAt:   time.Now().Add(time.Hour),
		}
		if !tok.Valid() {
			t.Fatal("expected valid token")
		}
	})

	t.Run("expired token", func(t *testing.T) {
		tok := &Token{
			AccessToken: "abc",
			ExpiresAt:   time.Now().Add(-time.Hour),
		}
		if tok.Valid() {
			t.Fatal("expected expired token")
		}
	})

	t.Run("empty access token", func(t *testing.T) {
		tok := &Token{
			AccessToken: "",
			ExpiresAt:   time.Now().Add(time.Hour),
		}
		if tok.Valid() {
			t.Fatal("expected invalid token with empty access token")
		}
	})

	t.Run("nil receiver", func(t *testing.T) {
		var tok *Token
		if tok.Valid() {
			t.Fatal("expected invalid for nil receiver")
		}
	})
}

func TestNewYoutubeProvider(t *testing.T) {
	p := NewYoutubeProvider("/some/path")
	if p.tokenPath != "/some/path" {
		t.Errorf("tokenPath = %q, want %q", p.tokenPath, "/some/path")
	}
	if p.config != nil {
		t.Error("config should be nil")
	}
}

func TestExists(t *testing.T) {
	t.Run("existing file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.txt")
		if err := os.WriteFile(path, []byte("data"), 0644); err != nil {
			t.Fatal(err)
		}
		if !exists(path) {
			t.Fatal("expected file to exist")
		}
	})

	t.Run("non-existing file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "nonexistent")
		if exists(path) {
			t.Fatal("expected file to not exist")
		}
	})
}

func TestSaveAndLoadToken(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")

	original := &Token{
		AccessToken:  "access123",
		RefreshToken: "refresh456",
		TokenType:    "Bearer",
		ExpiresAt:    time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	if err := SaveToken(path, original); err != nil {
		t.Fatalf("SaveToken: %v", err)
	}

	loaded, err := LoadToken(path)
	if err != nil {
		t.Fatalf("LoadToken: %v", err)
	}

	if loaded.AccessToken != original.AccessToken {
		t.Errorf("AccessToken = %q, want %q", loaded.AccessToken, original.AccessToken)
	}
	if loaded.RefreshToken != original.RefreshToken {
		t.Errorf("RefreshToken = %q, want %q", loaded.RefreshToken, original.RefreshToken)
	}
	if loaded.TokenType != original.TokenType {
		t.Errorf("TokenType = %q, want %q", loaded.TokenType, original.TokenType)
	}
	if !loaded.ExpiresAt.Equal(original.ExpiresAt) {
		t.Errorf("ExpiresAt = %v, want %v", loaded.ExpiresAt, original.ExpiresAt)
	}
}

func TestLoadToken_errors(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		_, err := LoadToken(filepath.Join(t.TempDir(), "no_token.json"))
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "bad.json")
		if err := os.WriteFile(path, []byte("not json"), 0644); err != nil {
			t.Fatal(err)
		}
		_, err := LoadToken(path)
		if err == nil {
			t.Fatal("expected error for invalid JSON")
		}
	})
}

func TestTokenFromOAuth2(t *testing.T) {
	tok := tokenFromOAuth2(&oauth2.Token{
		AccessToken:  "at",
		RefreshToken: "rt",
		TokenType:    "Bearer",
		Expiry:       time.Date(2030, 6, 15, 12, 0, 0, 0, time.UTC),
	})
	if tok.AccessToken != "at" {
		t.Errorf("AccessToken = %q", tok.AccessToken)
	}
	if tok.RefreshToken != "rt" {
		t.Errorf("RefreshToken = %q", tok.RefreshToken)
	}
	if tok.TokenType != "Bearer" {
		t.Errorf("TokenType = %q", tok.TokenType)
	}
	if !tok.ExpiresAt.Equal(time.Date(2030, 6, 15, 12, 0, 0, 0, time.UTC)) {
		t.Errorf("ExpiresAt = %v", tok.ExpiresAt)
	}
}
