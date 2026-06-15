package registry

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAddVideo(t *testing.T) {
	t.Run("add to empty registry", func(t *testing.T) {
		reg := &TubeRegistry{}
		err := AddVideo(reg, "vid1", "Title 1")
		if err != nil {
			t.Fatalf("AddVideo: %v", err)
		}
		if len(reg.Videos) != 1 {
			t.Fatalf("got %d videos, want 1", len(reg.Videos))
		}
		if reg.Videos[0].VideoID != "vid1" || reg.Videos[0].Title != "Title 1" {
			t.Errorf("video = %+v", reg.Videos[0])
		}
	})

	t.Run("duplicate returns error", func(t *testing.T) {
		reg := &TubeRegistry{}
		AddVideo(reg, "vid1", "Title")
		err := AddVideo(reg, "vid1", "Title")
		if err == nil {
			t.Fatal("expected error for duplicate video")
		}
	})

	t.Run("add multiple videos", func(t *testing.T) {
		reg := &TubeRegistry{}
		AddVideo(reg, "a", "A")
		AddVideo(reg, "b", "B")
		AddVideo(reg, "c", "C")
		if len(reg.Videos) != 3 {
			t.Fatalf("got %d videos, want 3", len(reg.Videos))
		}
	})

	t.Run("registered at is set", func(t *testing.T) {
		reg := &TubeRegistry{}
		AddVideo(reg, "v", "t")
		if reg.Videos[0].RegisteredAt.IsZero() {
			t.Error("RegisteredAt should not be zero")
		}
	})
}

func TestRemoveVideo(t *testing.T) {
	t.Run("remove existing", func(t *testing.T) {
		reg := &TubeRegistry{}
		AddVideo(reg, "v1", "")
		AddVideo(reg, "v2", "")
		AddVideo(reg, "v3", "")
		if ok := RemoveVideo(reg, "v2"); !ok {
			t.Fatal("RemoveVideo returned false")
		}
		if len(reg.Videos) != 2 {
			t.Fatalf("got %d videos, want 2", len(reg.Videos))
		}
		if reg.Videos[0].VideoID != "v1" || reg.Videos[1].VideoID != "v3" {
			t.Errorf("remaining = %+v", reg.Videos)
		}
	})

	t.Run("remove non-existent", func(t *testing.T) {
		reg := &TubeRegistry{}
		AddVideo(reg, "v1", "")
		if ok := RemoveVideo(reg, "nonexistent"); ok {
			t.Fatal("RemoveVideo should return false for non-existent")
		}
		if len(reg.Videos) != 1 {
			t.Fatalf("got %d videos, want 1", len(reg.Videos))
		}
	})

	t.Run("remove from single-element list", func(t *testing.T) {
		reg := &TubeRegistry{}
		AddVideo(reg, "v1", "")
		RemoveVideo(reg, "v1")
		if len(reg.Videos) != 0 {
			t.Fatalf("got %d videos, want 0", len(reg.Videos))
		}
	})
}

func TestUpdateVideo(t *testing.T) {
	t.Run("update existing", func(t *testing.T) {
		reg := &TubeRegistry{}
		AddVideo(reg, "v1", "Old Title")
		if ok := UpdateVideo(reg, "v1", "New Title"); !ok {
			t.Fatal("UpdateVideo returned false")
		}
		if reg.Videos[0].Title != "New Title" {
			t.Errorf("title = %q, want %q", reg.Videos[0].Title, "New Title")
		}
	})

	t.Run("update non-existent", func(t *testing.T) {
		reg := &TubeRegistry{}
		if ok := UpdateVideo(reg, "v1", "Title"); ok {
			t.Fatal("UpdateVideo should return false for non-existent")
		}
	})
}

func TestWriteRegistryFile(t *testing.T) {
	dir := t.TempDir()
	err := WriteRegistryFile(dir)
	if err != nil {
		t.Fatalf("WriteRegistryFile: %v", err)
	}

	path := filepath.Join(dir, "registry.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("registry.json not created")
	}

	reg, err := LoadRegistry(path)
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}
	if len(reg.Videos) != 0 {
		t.Errorf("got %d videos, want 0", len(reg.Videos))
	}
}

func TestSaveAndLoadRegistry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.json")

	reg := &TubeRegistry{}
	AddVideo(reg, "v1", "First")
	AddVideo(reg, "v2", "Second")

	err := SaveRegistry(dir, reg)
	if err != nil {
		t.Fatalf("SaveRegistry: %v", err)
	}

	loaded, err := LoadRegistry(path)
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}

	if len(loaded.Videos) != 2 {
		t.Fatalf("got %d videos, want 2", len(loaded.Videos))
	}
	if loaded.Videos[0].VideoID != "v1" || loaded.Videos[0].Title != "First" {
		t.Errorf("video 0 = %+v", loaded.Videos[0])
	}
	if loaded.Videos[1].VideoID != "v2" || loaded.Videos[1].Title != "Second" {
		t.Errorf("video 1 = %+v", loaded.Videos[1])
	}
}

func TestLoadRegistry_errors(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		_, err := LoadRegistry(filepath.Join(t.TempDir(), "nope.json"))
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})

	t.Run("corrupt json", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "registry.json")
		os.WriteFile(path, []byte("{{{"), 0644)
		_, err := LoadRegistry(path)
		if err == nil {
			t.Fatal("expected error for corrupt JSON")
		}
	})
}
