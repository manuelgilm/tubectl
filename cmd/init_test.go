package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"github.com/manuelgilm/tubectl/internal/registry"
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

func TestWriteRegistryFile(t *testing.T) {
	dir := t.TempDir()

	err := registry.WriteRegistryFile(dir)
	if err != nil {
		t.Fatalf("Failed while writing registry file %v ", err)
	}

	var emptyRegistryFile registry.TubeRegistry

	data, err := os.ReadFile(filepath.Join(dir, "registry.json"))
	if err != nil {
		t.Fatalf("Failed Reading the registry file %v ", err)
	}

	err = json.Unmarshal(data, &emptyRegistryFile)
	if err != nil {
		t.Fatalf("Error while unmarshalling %v ", err)
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
