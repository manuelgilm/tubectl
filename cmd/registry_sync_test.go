package cmd

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/manuelgilm/tubectl/internal/storage"
	"gopkg.in/yaml.v3"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := storage.New(filepath.Join(t.TempDir(), "tubectl.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func yamlUnmarshalSingle(yamlFragment string, v interface{}) error {
	return yaml.Unmarshal([]byte(yamlFragment), v)
}

func TestYamlVideoEntry_UnmarshalYAML(t *testing.T) {
	t.Run("string format", func(t *testing.T) {
		var entry yamlVideoEntry
		if err := yamlUnmarshalSingle(`"ITQioNZ_m_U"`, &entry); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if entry.ID != "ITQioNZ_m_U" {
			t.Errorf("ID = %q", entry.ID)
		}
		if entry.Title != "" {
			t.Errorf("Title = %q, want empty", entry.Title)
		}
	})

	t.Run("object without title", func(t *testing.T) {
		var entry yamlVideoEntry
		if err := yamlUnmarshalSingle(`{id: dQw4w9WgXcQ}`, &entry); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if entry.ID != "dQw4w9WgXcQ" {
			t.Errorf("ID = %q", entry.ID)
		}
		if entry.Title != "" {
			t.Errorf("Title = %q, want empty", entry.Title)
		}
	})

	t.Run("object with title", func(t *testing.T) {
		var entry yamlVideoEntry
		if err := yamlUnmarshalSingle(`{id: De2hGBxJ2j4, title: "My Video"}`, &entry); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if entry.ID != "De2hGBxJ2j4" {
			t.Errorf("ID = %q", entry.ID)
		}
		if entry.Title != "My Video" {
			t.Errorf("Title = %q", entry.Title)
		}
	})

	t.Run("object missing id", func(t *testing.T) {
		var entry yamlVideoEntry
		err := yamlUnmarshalSingle(`{title: "No ID"}`, &entry)
		if err == nil {
			t.Fatal("expected error for missing id")
		}
	})

	t.Run("number treated as string id", func(t *testing.T) {
		var entry yamlVideoEntry
		if err := yamlUnmarshalSingle(`42`, &entry); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if entry.ID != "42" {
			t.Errorf("ID = %q, want \"42\"", entry.ID)
		}
	})
}

func TestSyncYamlEntries(t *testing.T) {
	t.Run("add new videos from yaml", func(t *testing.T) {
		db := newTestDB(t)
		dir := t.TempDir()
		yamlPath := filepath.Join(dir, "videos.yaml")

		content := []byte("videos:\n  - vid1\n  - vid2\n  - id: vid3\n    title: \"Third Video\"\n")
		if err := os.WriteFile(yamlPath, content, 0644); err != nil {
			t.Fatal(err)
		}

		added, updated, skipped, err := syncYamlEntries(context.Background(), db, yamlPath)
		if err != nil {
			t.Fatalf("syncYamlEntries: %v", err)
		}
		if added != 3 {
			t.Errorf("added = %d, want 3", added)
		}
		if updated != 0 {
			t.Errorf("updated = %d, want 0", updated)
		}
		if skipped != 0 {
			t.Errorf("skipped = %d, want 0", skipped)
		}
	})

	t.Run("skip existing videos", func(t *testing.T) {
		db := newTestDB(t)
		dir := t.TempDir()
		yamlPath := filepath.Join(dir, "videos.yaml")

		repo := storage.NewVideoRepo(db)
		now := time.Now()
		repo.Add(context.Background(), storage.Video{ID: "vid1", Title: "", RegisteredAt: now, UpdatedAt: now})

		content := []byte("videos:\n  - vid1\n  - vid2\n")
		if err := os.WriteFile(yamlPath, content, 0644); err != nil {
			t.Fatal(err)
		}

		added, _, skipped, err := syncYamlEntries(context.Background(), db, yamlPath)
		if err != nil {
			t.Fatalf("syncYamlEntries: %v", err)
		}
		if added != 1 {
			t.Errorf("added = %d, want 1", added)
		}
		if skipped != 1 {
			t.Errorf("skipped = %d, want 1", skipped)
		}
	})

	t.Run("update title when changed", func(t *testing.T) {
		db := newTestDB(t)
		dir := t.TempDir()
		yamlPath := filepath.Join(dir, "videos.yaml")

		repo := storage.NewVideoRepo(db)
		now := time.Now()
		repo.Add(context.Background(), storage.Video{ID: "vid1", Title: "Old Title", RegisteredAt: now, UpdatedAt: now})

		content := []byte("videos:\n  - id: vid1\n    title: \"New Title\"\n")
		if err := os.WriteFile(yamlPath, content, 0644); err != nil {
			t.Fatal(err)
		}

		added, updated, skipped, err := syncYamlEntries(context.Background(), db, yamlPath)
		if err != nil {
			t.Fatalf("syncYamlEntries: %v", err)
		}
		if added != 0 {
			t.Errorf("added = %d, want 0", added)
		}
		if updated != 1 {
			t.Errorf("updated = %d, want 1", updated)
		}
		if skipped != 0 {
			t.Errorf("skipped = %d, want 0", skipped)
		}

		v, _ := repo.Get(context.Background(), "vid1")
		if v.Title != "New Title" {
			t.Errorf("title = %q, want %q", v.Title, "New Title")
		}
	})

	t.Run("missing yaml file", func(t *testing.T) {
		db := newTestDB(t)
		_, _, _, err := syncYamlEntries(context.Background(), db, "/nonexistent/path.yaml")
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})

	t.Run("malformed yaml", func(t *testing.T) {
		db := newTestDB(t)
		dir := t.TempDir()
		yamlPath := filepath.Join(dir, "bad.yaml")

		os.WriteFile(yamlPath, []byte("{{{broken"), 0644)

		_, _, _, err := syncYamlEntries(context.Background(), db, yamlPath)
		if err == nil {
			t.Fatal("expected error for malformed yaml")
		}
	})
}

func TestSyncYamlEmptyFile(t *testing.T) {
	db := newTestDB(t)
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "empty.yaml")

	os.WriteFile(yamlPath, []byte("videos: []\n"), 0644)

	added, updated, skipped, err := syncYamlEntries(context.Background(), db, yamlPath)
	if err != nil {
		t.Fatalf("syncYamlEntries: %v", err)
	}
	if added != 0 || updated != 0 || skipped != 0 {
		t.Errorf("expected all zero, got %d/%d/%d", added, updated, skipped)
	}
}
