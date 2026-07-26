package storage

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

func addTestVideo(t *testing.T, db *sql.DB, id string) {
	t.Helper()
	repo := NewVideoRepo(db)
	now := time.Now()
	if err := repo.Add(context.Background(), Video{ID: id, Title: id, RegisteredAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("addTestVideo: %v", err)
	}
}

func TestTranscriptRepo_SaveLoad(t *testing.T) {
	db := newTestDB(t)
	repo := NewTranscriptRepo(db)
	addTestVideo(t, db, "vid1")
	now := time.Now()

	t.Run("save and load", func(t *testing.T) {
		st := &StoredTranscript{
			VideoID:   "vid1",
			Language:  "en",
			TrackKind: "asr",
			CaptionID: "capt1",
			Content:   "hello world",
			Lines:     `[{"start":1,"duration":2,"text":"hello"},{"start":3,"duration":1,"text":"world"}]`,
			CachedAt:  now,
		}

		if err := repo.Save(context.Background(), st); err != nil {
			t.Fatalf("Save: %v", err)
		}

		loaded, err := repo.Load(context.Background(), "vid1")
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if loaded == nil {
			t.Fatal("Load returned nil")
		}
		if loaded.VideoID != "vid1" || loaded.Language != "en" || loaded.TrackKind != "asr" {
			t.Errorf("loaded = %+v", loaded)
		}
		if loaded.Content != "hello world" {
			t.Errorf("content = %q", loaded.Content)
		}
		if loaded.Lines != st.Lines {
			t.Errorf("lines = %q", loaded.Lines)
		}
	})

	t.Run("overwrite existing", func(t *testing.T) {
		st := &StoredTranscript{
			VideoID:  "vid1",
			Language: "en",
			Content:  "updated content",
			Lines:    `[]`,
			CachedAt: time.Now(),
		}
		if err := repo.Save(context.Background(), st); err != nil {
			t.Fatalf("Save overwrite: %v", err)
		}

		loaded, _ := repo.Load(context.Background(), "vid1")
		if loaded.Content != "updated content" {
			t.Errorf("content = %q, want %q", loaded.Content, "updated content")
		}
	})

	t.Run("load not found", func(t *testing.T) {
		loaded, err := repo.Load(context.Background(), "nonexistent")
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if loaded != nil {
			t.Fatal("expected nil for missing transcript")
		}
	})
}

func TestTranscriptRepo_Delete(t *testing.T) {
	db := newTestDB(t)
	repo := NewTranscriptRepo(db)
	addTestVideo(t, db, "vid1")

	repo.Save(context.Background(), &StoredTranscript{
		VideoID:  "vid1",
		Language: "en",
		Content:  "delete me",
		Lines:    `[]`,
		CachedAt: time.Now(),
	})

	if err := repo.Delete(context.Background(), "vid1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	loaded, _ := repo.Load(context.Background(), "vid1")
	if loaded != nil {
		t.Fatal("expected nil after delete")
	}
}
