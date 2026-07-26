package storage

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	db.Exec("PRAGMA foreign_keys=ON")
	if err := Migrate(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestVideoRepo_Add(t *testing.T) {
	t.Run("add to empty db", func(t *testing.T) {
		db := newTestDB(t)
		repo := NewVideoRepo(db)

		err := repo.Add(context.Background(), Video{
			ID: "vid1", Title: "Title 1", RegisteredAt: time.Now(), UpdatedAt: time.Now(),
		})
		if err != nil {
			t.Fatalf("Add: %v", err)
		}

		vs, err := repo.List(context.Background())
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(vs) != 1 {
			t.Fatalf("got %d videos, want 1", len(vs))
		}
		if vs[0].ID != "vid1" || vs[0].Title != "Title 1" {
			t.Errorf("video = %+v", vs[0])
		}
	})

	t.Run("duplicate returns error", func(t *testing.T) {
		db := newTestDB(t)
		repo := NewVideoRepo(db)
		now := time.Now()

		repo.Add(context.Background(), Video{ID: "vid1", Title: "T", RegisteredAt: now, UpdatedAt: now})
		err := repo.Add(context.Background(), Video{ID: "vid1", Title: "T", RegisteredAt: now, UpdatedAt: now})
		if err == nil {
			t.Fatal("expected error for duplicate video")
		}
	})

	t.Run("add multiple videos", func(t *testing.T) {
		db := newTestDB(t)
		repo := NewVideoRepo(db)
		now := time.Now()

		repo.Add(context.Background(), Video{ID: "a", Title: "A", RegisteredAt: now, UpdatedAt: now})
		repo.Add(context.Background(), Video{ID: "b", Title: "B", RegisteredAt: now, UpdatedAt: now})
		repo.Add(context.Background(), Video{ID: "c", Title: "C", RegisteredAt: now, UpdatedAt: now})

		vs, _ := repo.List(context.Background())
		if len(vs) != 3 {
			t.Fatalf("got %d videos, want 3", len(vs))
		}
	})

	t.Run("registered at is set", func(t *testing.T) {
		db := newTestDB(t)
		repo := NewVideoRepo(db)
		now := time.Now()

		repo.Add(context.Background(), Video{ID: "v", Title: "t", RegisteredAt: now, UpdatedAt: now})
		v, _ := repo.Get(context.Background(), "v")
		if v.RegisteredAt.IsZero() {
			t.Error("RegisteredAt should not be zero")
		}
	})
}

func TestVideoRepo_Get(t *testing.T) {
	t.Run("get existing", func(t *testing.T) {
		db := newTestDB(t)
		repo := NewVideoRepo(db)
		now := time.Now()

		repo.Add(context.Background(), Video{
			ID: "v1", Title: "Title", Description: "Desc", ChannelID: "ch1",
			PublishedAt: now, RegisteredAt: now, UpdatedAt: now,
		})

		v, err := repo.Get(context.Background(), "v1")
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if v == nil {
			t.Fatal("Get returned nil")
		}
		if v.ID != "v1" || v.Title != "Title" || v.Description != "Desc" || v.ChannelID != "ch1" {
			t.Errorf("video = %+v", v)
		}
	})

	t.Run("get non-existing", func(t *testing.T) {
		db := newTestDB(t)
		repo := NewVideoRepo(db)

		v, err := repo.Get(context.Background(), "nonexistent")
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if v != nil {
			t.Fatal("expected nil for non-existing video")
		}
	})
}

func TestVideoRepo_Delete(t *testing.T) {
	t.Run("delete existing", func(t *testing.T) {
		db := newTestDB(t)
		repo := NewVideoRepo(db)
		now := time.Now()

		repo.Add(context.Background(), Video{ID: "v1", Title: "", RegisteredAt: now, UpdatedAt: now})
		repo.Add(context.Background(), Video{ID: "v2", Title: "", RegisteredAt: now, UpdatedAt: now})
		repo.Add(context.Background(), Video{ID: "v3", Title: "", RegisteredAt: now, UpdatedAt: now})

		ok, err := repo.Delete(context.Background(), "v2")
		if err != nil {
			t.Fatalf("Delete: %v", err)
		}
		if !ok {
			t.Fatal("Delete returned false")
		}

		vs, _ := repo.List(context.Background())
		if len(vs) != 2 {
			t.Fatalf("got %d videos, want 2", len(vs))
		}
		ids := map[string]bool{}
		for _, v := range vs {
			ids[v.ID] = true
		}
		if !ids["v1"] || !ids["v3"] {
			t.Errorf("remaining = %+v", vs)
		}
	})

	t.Run("delete non-existent", func(t *testing.T) {
		db := newTestDB(t)
		repo := NewVideoRepo(db)

		ok, err := repo.Delete(context.Background(), "nonexistent")
		if err != nil {
			t.Fatalf("Delete: %v", err)
		}
		if ok {
			t.Fatal("Delete should return false for non-existent")
		}
	})

	t.Run("delete from single-element list", func(t *testing.T) {
		db := newTestDB(t)
		repo := NewVideoRepo(db)
		now := time.Now()

		repo.Add(context.Background(), Video{ID: "v1", Title: "", RegisteredAt: now, UpdatedAt: now})
		ok, _ := repo.Delete(context.Background(), "v1")
		if !ok {
			t.Fatal("Delete returned false")
		}
		vs, _ := repo.List(context.Background())
		if len(vs) != 0 {
			t.Fatalf("got %d videos, want 0", len(vs))
		}
	})
}

func TestVideoRepo_Update(t *testing.T) {
	t.Run("update existing", func(t *testing.T) {
		db := newTestDB(t)
		repo := NewVideoRepo(db)
		now := time.Now()

		repo.Add(context.Background(), Video{ID: "v1", Title: "Old Title", RegisteredAt: now, UpdatedAt: now})
		v, err := repo.Update(context.Background(), "v1", "New Title")
		if err != nil {
			t.Fatalf("Update: %v", err)
		}
		if v == nil {
			t.Fatal("Update returned nil")
		}
		if v.Title != "New Title" {
			t.Errorf("title = %q, want %q", v.Title, "New Title")
		}
	})

	t.Run("update non-existent", func(t *testing.T) {
		db := newTestDB(t)
		repo := NewVideoRepo(db)

		v, err := repo.Update(context.Background(), "v1", "Title")
		if err != nil {
			t.Fatalf("Update: %v", err)
		}
		if v != nil {
			t.Fatal("Update should return nil for non-existent")
		}
	})
}

func TestVideoRepo_List(t *testing.T) {
	t.Run("empty list", func(t *testing.T) {
		db := newTestDB(t)
		repo := NewVideoRepo(db)

		vs, err := repo.List(context.Background())
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(vs) != 0 {
			t.Fatalf("got %d videos, want 0", len(vs))
		}
	})

	t.Run("order by registered_at desc", func(t *testing.T) {
		db := newTestDB(t)
		repo := NewVideoRepo(db)

		repo.Add(context.Background(), Video{ID: "a", Title: "A", RegisteredAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), UpdatedAt: time.Now()})
		repo.Add(context.Background(), Video{ID: "b", Title: "B", RegisteredAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), UpdatedAt: time.Now()})

		vs, _ := repo.List(context.Background())
		if vs[0].ID != "b" || vs[1].ID != "a" {
			t.Errorf("expected b first, got %+v", vs)
		}
	})
}

func TestFormatParseTime(t *testing.T) {
	t.Run("roundtrip", func(t *testing.T) {
		now := time.Now().Truncate(time.Second)
		s := formatTime(now)
		parsed := parseTime(s)
		if !parsed.Equal(now) {
			t.Errorf("got %v, want %v", parsed, now)
		}
	})

	t.Run("zero time", func(t *testing.T) {
		if s := formatTime(time.Time{}); s != "" {
			t.Errorf("expected empty string, got %q", s)
		}
		if pt := parseTime(""); !pt.IsZero() {
			t.Error("expected zero time")
		}
	})

	t.Run("invalid string returns zero", func(t *testing.T) {
		if pt := parseTime("not-a-time"); !pt.IsZero() {
			t.Error("expected zero time for invalid input")
		}
	})
}
