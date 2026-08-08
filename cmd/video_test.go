package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/manuelgilm/tubectl/internal/youtube"
)

func TestFormatTranscriptLines(t *testing.T) {
	tr := &youtube.Transcript{
		Lines: []youtube.TranscriptLine{
			{Start: 0, Text: "Hello"},
			{Start: 65.5, Text: "World"},
		},
	}
	lines := formatTranscriptLines(tr)
	if len(lines) != 2 || lines[0] != "[00:00] Hello" || lines[1] != "[01:05] World" {
		t.Errorf("lines = %q", lines)
	}
}

func TestStripTranscriptTimestamps(t *testing.T) {
	t.Run("timedtext format", func(t *testing.T) {
		input := "[00:00] Hello world\n[01:05] Second line\n\n[02:10] Third\n[09:00] Plain"
		got := stripTranscriptTimestamps(input)
		want := "Hello world Second line Third Plain"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("srt format", func(t *testing.T) {
		input := "1\n00:00:00,400 --> 00:00:05,600\nHello world\n\n2\n00:00:05,600 --> 00:00:09,000\nSecond line\n"
		got := stripTranscriptTimestamps(input)
		want := "Hello world Second line"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("empty input", func(t *testing.T) {
		if got := stripTranscriptTimestamps(""); got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})
}

func TestIsTimingLine(t *testing.T) {
	cases := []struct {
		line string
		want bool
	}{
		{"00:00:00,400 --> 00:00:05,600", true},
		{"0:00:00.400,0:00:05.600", true},
		{"00:00:05,600", true},
		{"1", true},
		{"", false},
		{"Hello world", false},
		{"[00:00] Hello world", false},
		{"0:00:00.400,0:00:05.600A", false},
	}
	for _, c := range cases {
		if got := isTimingLine(c.line); got != c.want {
			t.Errorf("isTimingLine(%q) = %v, want %v", c.line, got, c.want)
		}
	}
}

func TestTranscriptFileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TUBECTL_TRANSCRIPT_DIR", dir)

	path := transcriptFilePath("abc123")
	wantPath := filepath.Join(dir, "abc123.txt")
	if path != wantPath {
		t.Fatalf("path = %q, want %q", path, wantPath)
	}

	tr := &youtube.Transcript{
		VideoID:  "abc123",
		Language: "en",
		Lines: []youtube.TranscriptLine{
			{Start: 0, Text: "First"},
			{Start: 3.5, Text: "Second"},
		},
	}
	if err := writeTranscriptFile(path, tr); err != nil {
		t.Fatalf("writeTranscriptFile: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(data), "[00:00] First") || !strings.Contains(string(data), "[00:03] Second") {
		t.Errorf("file content = %q", string(data))
	}

	text, err := readTranscriptFile(path)
	if err != nil {
		t.Fatalf("readTranscriptFile: %v", err)
	}
	if got := stripTranscriptTimestamps(text); got != "First Second" {
		t.Errorf("stripped = %q", got)
	}
}
