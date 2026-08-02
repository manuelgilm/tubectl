package cmd

import (
	"encoding/json"
	"github.com/manuelgilm/tubectl/internal/storage"
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

	err := writeConfigFile(dir)
	if err != nil {
		t.Fatalf("Write File failed: %v ", err)
	}

	var emptyConfig Config

	data, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatalf("Failed while reading the file %v ", err)
	}
	err = json.Unmarshal(data, &emptyConfig)
	if err != nil {
		t.Fatalf("Error while unmarshaling %v ", err)
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
