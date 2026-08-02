package cmd

import (
	"encoding/json"
	"github.com/manuelgilm/tubectl/internal/storage"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateFolder(t *testing.T) {
	dir := t.TempDir()
	testPath := filepath.Join(dir, "subdir")

	err := createFolder(testPath)
	if err != nil {
		t.Fatalf("createFolder failed: %v ", err)
	}

	if _, err := os.Stat(testPath); os.IsNotExist(err) {
		t.Fatal("Expected directory to exist")
	}
}

func TestWriteConfigFile(t *testing.T) {
	dir := t.TempDir()

	err := writeConfigFile(io.Discard, dir)
	if err != nil {
		t.Fatalf("Write File failed: %v ", err)
	}

	var emptyConfig Config

	configPath := filepath.Join(dir, "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed while reading the file %v ", err)
	}
	err = json.Unmarshal(data, &emptyConfig)
	if err != nil {
		t.Fatalf("Error while unmarshaling %v ", err)
	}

	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("stat config.json: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("config.json mode = %o, want 600", perm)
	}
}

func TestWriteConfigFile_repairsExistingPerms(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(configPath, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := writeConfigFile(io.Discard, dir); err != nil {
		t.Fatalf("writeConfigFile: %v", err)
	}

	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("stat config.json: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("config.json mode = %o, want 600", perm)
	}
}

func TestCreateDatabase(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "tubectl.db")

	db, err := storage.New(dbPath)
	if err != nil {
		t.Fatalf("storage.New failed: %v", err)
	}
	defer db.Close()

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Fatal("expected tubectl.db to exist")
	}

	var version int
	err = db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM _migrations").Scan(&version)
	if err != nil {
		t.Fatalf("reading migrations: %v", err)
	}
	if version == 0 {
		t.Fatal("expected migrations to have run")
	}
}

func TestCreateFolder_alreadyExists(t *testing.T) {
	dir := t.TempDir()

	err := createFolder(dir)
	if err != nil {
		t.Fatalf("createFolder on existing dir: %v", err)
	}
}

func TestCreateFolder_nested(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "a", "b", "c")

	err := createFolder(nested)
	if err != nil {
		t.Fatalf("createFolder nested: %v", err)
	}

	if _, err := os.Stat(nested); os.IsNotExist(err) {
		t.Fatal("expected nested directory to exist")
	}
}
