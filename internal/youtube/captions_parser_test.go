package youtube

import (
	"testing"
)

func TestParseSRTTimecode(t *testing.T) {
	tests := []struct {
		input    string
		wantSec  float64
		wantOK   bool
	}{
		{"00:00:00,000", 0, true},
		{"00:01:30,500", 90.5, true},
		{"01:00:00,000", 3600, true},
		{"00:00:01,001", 1.001, true},
		{"invalid", 0, false},
		{"", 0, false},
	}

	for _, tt := range tests {
		got, ok := parseSRTTimecode(tt.input)
		if ok != tt.wantOK {
			t.Errorf("parseSRTTimecode(%q) ok = %v, want %v", tt.input, ok, tt.wantOK)
		}
		if ok && got != tt.wantSec {
			t.Errorf("parseSRTTimecode(%q) = %v, want %v", tt.input, got, tt.wantSec)
		}
	}
}

func TestParseSRTTimestampLine(t *testing.T) {
	tests := []struct {
		input       string
		wantStart   float64
		wantDur     float64
		wantOK      bool
	}{
		{"00:00:00,000 --> 00:00:05,000", 0, 5, true},
		{"00:01:00,000 --> 00:01:30,500", 60, 30.5, true},
		{"invalid", 0, 0, false},
		{"00:00:00,000 --> ", 0, 0, false},
		{" --> 00:00:05,000", 0, 0, false},
		{"", 0, 0, false},
	}

	for _, tt := range tests {
		start, dur, ok := parseSRTTimestampLine(tt.input)
		if ok != tt.wantOK {
			t.Errorf("parseSRTTimestampLine(%q) ok = %v, want %v", tt.input, ok, tt.wantOK)
			continue
		}
		if ok {
			if start != tt.wantStart {
				t.Errorf("parseSRTTimestampLine(%q) start = %v, want %v", tt.input, start, tt.wantStart)
			}
			if dur != tt.wantDur {
				t.Errorf("parseSRTTimestampLine(%q) dur = %v, want %v", tt.input, dur, tt.wantDur)
			}
		}
	}
}

func TestParseSRT(t *testing.T) {
	t.Run("valid srt", func(t *testing.T) {
		data := []byte("1\n00:00:01,000 --> 00:00:04,000\nHello world\n\n2\n00:00:05,000 --> 00:00:07,500\nSecond line\n")
		lines, err := parseSRT(data)
		if err != nil {
			t.Fatalf("parseSRT returned error: %v", err)
		}
		if len(lines) != 2 {
			t.Fatalf("got %d lines, want 2", len(lines))
		}
		if lines[0].Start != 1 || lines[0].Duration != 3 || lines[0].Text != "Hello world" {
			t.Errorf("line 0 = %+v", lines[0])
		}
		if lines[1].Start != 5 || lines[1].Duration != 2.5 || lines[1].Text != "Second line" {
			t.Errorf("line 1 = %+v", lines[1])
		}
	})

	t.Run("strips html tags", func(t *testing.T) {
		data := []byte("1\n00:00:00,000 --> 00:00:02,000\nHello <b>world</b>\n")
		lines, err := parseSRT(data)
		if err != nil {
			t.Fatalf("parseSRT returned error: %v", err)
		}
		if len(lines) != 1 {
			t.Fatalf("got %d lines, want 1", len(lines))
		}
		if lines[0].Text != "Hello world" {
			t.Errorf("text = %q, want %q", lines[0].Text, "Hello world")
		}
	})

	t.Run("empty input", func(t *testing.T) {
		lines, err := parseSRT([]byte{})
		if err != nil {
			t.Fatalf("parseSRT returned error: %v", err)
		}
		if len(lines) != 0 {
			t.Errorf("got %d lines, want 0", len(lines))
		}
	})

	t.Run("crlf line endings", func(t *testing.T) {
		data := []byte("1\r\n00:00:00,000 --> 00:00:01,000\r\nCRLF line\r\n")
		lines, err := parseSRT(data)
		if err != nil {
			t.Fatalf("parseSRT returned error: %v", err)
		}
		if len(lines) != 1 || lines[0].Text != "CRLF line" {
			t.Errorf("line 0 = %+v", lines[0])
		}
	})

	t.Run("malformed timestamp block skipped", func(t *testing.T) {
		data := []byte("1\n00:00:00,000 --> invalid\nText\n\n2\n00:00:01,000 --> 00:00:02,000\nValid\n")
		lines, err := parseSRT(data)
		if err != nil {
			t.Fatalf("parseSRT returned error: %v", err)
		}
		if len(lines) != 1 || lines[0].Text != "Valid" {
			t.Errorf("got %+v, want single valid line", lines)
		}
	})

	t.Run("blank text block skipped", func(t *testing.T) {
		data := []byte("1\n00:00:00,000 --> 00:00:01,000\n  \n\n2\n00:00:01,000 --> 00:00:02,000\nReal text\n")
		lines, err := parseSRT(data)
		if err != nil {
			t.Fatalf("parseSRT returned error: %v", err)
		}
		if len(lines) != 1 || lines[0].Text != "Real text" {
			t.Errorf("got %+v, want single real text", lines)
		}
	})

	t.Run("multiline text collapsed", func(t *testing.T) {
		data := []byte("1\n00:00:00,000 --> 00:00:02,000\nFirst line\nSecond line\n")
		lines, err := parseSRT(data)
		if err != nil {
			t.Fatalf("parseSRT returned error: %v", err)
		}
		if len(lines) != 1 {
			t.Fatalf("got %d lines, want 1", len(lines))
		}
		if lines[0].Text != "First line Second line" {
			t.Errorf("text = %q, want %q", lines[0].Text, "First line Second line")
		}
	})
}

func TestParseTimedText(t *testing.T) {
	t.Run("valid xml", func(t *testing.T) {
		data := []byte(`<transcript><text start="0.5" dur="2.3">Hello world</text><text start="3.0" dur="1.5">Second line</text></transcript>`)
		lines, err := parseTimedText(data)
		if err != nil {
			t.Fatalf("parseTimedText returned error: %v", err)
		}
		if len(lines) != 2 {
			t.Fatalf("got %d lines, want 2", len(lines))
		}
		if lines[0].Start != 0.5 || lines[0].Duration != 2.3 || lines[0].Text != "Hello world" {
			t.Errorf("line 0 = %+v", lines[0])
		}
		if lines[1].Start != 3.0 || lines[1].Duration != 1.5 || lines[1].Text != "Second line" {
			t.Errorf("line 1 = %+v", lines[1])
		}
	})

	t.Run("empty input", func(t *testing.T) {
		data := []byte(`<transcript></transcript>`)
		lines, err := parseTimedText(data)
		if err != nil {
			t.Fatalf("parseTimedText returned error: %v", err)
		}
		if len(lines) != 0 {
			t.Errorf("got %d lines, want 0", len(lines))
		}
	})

	t.Run("malformed xml", func(t *testing.T) {
		_, err := parseTimedText([]byte(`not xml`))
		if err == nil {
			t.Fatal("expected error for malformed XML")
		}
	})

	t.Run("blank text elements skipped", func(t *testing.T) {
		data := []byte(`<transcript><text start="0.5" dur="1.0">   </text><text start="2.0" dur="1.0">Actual</text></transcript>`)
		lines, err := parseTimedText(data)
		if err != nil {
			t.Fatalf("parseTimedText returned error: %v", err)
		}
		if len(lines) != 1 || lines[0].Text != "Actual" {
			t.Errorf("got %+v, want single 'Actual'", lines)
		}
	})
}
