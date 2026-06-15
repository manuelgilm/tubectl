package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tubectl/internal/youtube"
)

func TestPromptFileRender(t *testing.T) {
	t.Run("all vars present", func(t *testing.T) {
		p := &PromptFile{
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
		p := &PromptFile{
			Template: "Hello {name}",
			Vars:     []string{"name"},
		}
		_, err := p.Render(map[string]string{})
		if err == nil {
			t.Fatal("expected error for missing var")
		}
	})

	t.Run("no vars defined", func(t *testing.T) {
		p := &PromptFile{
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
		p := &PromptFile{
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
		p, err := LoadPromptFile(path)
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
		_, err := LoadPromptFile("/nonexistent/path.yaml")
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
		_, err := LoadPromptFile(path)
		if err == nil {
			t.Fatal("expected error for empty template")
		}
	})
}

func TestBuildMessagesYTBot(t *testing.T) {
	messages, err := BuildMessagesYTBot("Great video!", "This is the transcript content.")
	if err != nil {
		t.Fatalf("BuildMessagesYTBot: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("got %d messages, want 1", len(messages))
	}
	if messages[0].Role != "system" {
		t.Errorf("role = %q, want system", messages[0].Role)
	}
	if !strings.Contains(messages[0].Content, "Great video!") {
		t.Errorf("content missing comment text")
	}
	if !strings.Contains(messages[0].Content, "This is the transcript content.") {
		t.Errorf("content missing transcript text")
	}
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

func TestTranscriptCacheRoundtrip(t *testing.T) {
	// Test that transcript JSON serialization/deserialization works correctly.
	transcript := &youtube.Transcript{
		VideoID:   "test123",
		Language:  "en",
		TrackKind: "asr",
		Lines: []youtube.TranscriptLine{
			{Start: 1.0, Duration: 2.0, Text: "hello"},
		},
	}

	dir := t.TempDir()
	p := filepath.Join(dir, "transcript.json")
	f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		t.Fatal(err)
	}
	enc := json.NewEncoder(f)
	if err := enc.Encode(transcript); err != nil {
		f.Close()
		t.Fatal(err)
	}
	f.Close()

	f2, err := os.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	defer f2.Close()
	var loaded youtube.Transcript
	if err := json.NewDecoder(f2).Decode(&loaded); err != nil {
		t.Fatal(err)
	}
	if loaded.VideoID != "test123" || loaded.Language != "en" {
		t.Errorf("loaded = %+v", loaded)
	}
	if len(loaded.Lines) != 1 || loaded.Lines[0].Text != "hello" {
		t.Errorf("lines = %+v", loaded.Lines)
	}
}
