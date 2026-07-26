package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"github.com/manuelgilm/tubectl/internal/prompt"
	"github.com/manuelgilm/tubectl/internal/youtube"
)

func TestPromptFileRender(t *testing.T) {
	t.Run("all vars present", func(t *testing.T) {
		p := &prompt.PromptFile{
			Template: "Hello {name}, you are {age} years old",
			Vars:     []string{"name", "age"},
		}
		result, err := p.Render(map[string]string{"name": "Alice", "age": "30"})
		if err != nil {
			t.Fatalf("Render: %v", err)
		}
		want := "Hello Alice, you are 30 years old"
		if result != want {
			t.Errorf("got %q, want %q", result, want)
		}
	})

	t.Run("missing var", func(t *testing.T) {
		p := &prompt.PromptFile{
			Template: "Hello {name}",
			Vars:     []string{"name"},
		}
		_, err := p.Render(map[string]string{})
		if err == nil {
			t.Fatal("expected error for missing var")
		}
	})

	t.Run("no vars defined", func(t *testing.T) {
		p := &prompt.PromptFile{
			Template: "Static text",
			Vars:     nil,
		}
		result, err := p.Render(nil)
		if err != nil {
			t.Fatalf("Render: %v", err)
		}
		if result != "Static text" {
			t.Errorf("got %q", result)
		}
	})

	t.Run("multiple occurrences of same var", func(t *testing.T) {
		p := &prompt.PromptFile{
			Template: "{x} + {x} = {y}",
			Vars:     []string{"x", "y"},
		}
		result, err := p.Render(map[string]string{"x": "1", "y": "2"})
		if err != nil {
			t.Fatalf("Render: %v", err)
		}
		if result != "1 + 1 = 2" {
			t.Errorf("got %q", result)
		}
	})
}

func TestLoadPromptFile(t *testing.T) {
	t.Run("valid yaml", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "prompt.yaml")
		content := "template: \"Reply to {comment}\"\nvars:\n  - comment\n"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		p, err := prompt.LoadPromptFile(path)
		if err != nil {
			t.Fatalf("LoadPromptFile: %v", err)
		}
		if p.Template != "Reply to {comment}" {
			t.Errorf("template = %q", p.Template)
		}
		if len(p.Vars) != 1 || p.Vars[0] != "comment" {
			t.Errorf("vars = %v", p.Vars)
		}
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := prompt.LoadPromptFile("/nonexistent/path.yaml")
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("empty template", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "empty.yaml")
		content := "template: \"\"\nvars: []\n"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		_, err := prompt.LoadPromptFile(path)
		if err == nil {
			t.Fatal("expected error for empty template")
		}
	})
}

func TestPrintTranscript(t *testing.T) {
	transcript := &youtube.Transcript{
		Lines: []youtube.TranscriptLine{
			{Start: 0, Text: "Hello"},
			{Start: 65.5, Text: "World"},
		},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printTranscript(transcript)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "[00:00] Hello") {
		t.Errorf("missing first line, got: %s", output)
	}
	if !strings.Contains(output, "[01:05] World") {
		t.Errorf("missing second line, got: %s", output)
	}
}


