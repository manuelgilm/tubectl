package youtube

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// ---- parseSRT ----

func TestParseSRT_Basic(t *testing.T) {
	input := "1\n00:00:01,000 --> 00:00:03,500\nHello world\n\n2\n00:00:04,000 --> 00:00:05,000\nGoodbye world\n"
	lines, err := parseSRT([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("want 2 lines, got %d", len(lines))
	}
	if lines[0].Text != "Hello world" {
		t.Errorf("line[0].Text = %q, want %q", lines[0].Text, "Hello world")
	}
	if lines[0].Start != 1.0 {
		t.Errorf("line[0].Start = %f, want 1.0", lines[0].Start)
	}
	if lines[0].Duration != 2.5 {
		t.Errorf("line[0].Duration = %f, want 2.5", lines[0].Duration)
	}
	if lines[1].Text != "Goodbye world" {
		t.Errorf("line[1].Text = %q, want %q", lines[1].Text, "Goodbye world")
	}
}

func TestParseSRT_StripsHTMLTags(t *testing.T) {
	input := "1\n00:00:01,000 --> 00:00:02,000\n<i>Italic</i> and <b>bold</b>\n"
	lines, err := parseSRT([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 1 {
		t.Fatalf("want 1 line, got %d", len(lines))
	}
	if lines[0].Text != "Italic and bold" {
		t.Errorf("HTML not stripped: got %q", lines[0].Text)
	}
}

func TestParseSRT_Empty(t *testing.T) {
	lines, err := parseSRT([]byte(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 0 {
		t.Errorf("want 0 lines, got %d", len(lines))
	}
}

func TestParseSRT_SkipsBlankTextBlock(t *testing.T) {
	// Block with only whitespace text should be skipped.
	input := "1\n00:00:01,000 --> 00:00:02,000\n   \n\n2\n00:00:02,000 --> 00:00:03,000\nHello\n"
	lines, err := parseSRT([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 1 {
		t.Errorf("want 1 line (blank skipped), got %d", len(lines))
	}
}

// ---- parseTimedText ----

func TestParseTimedText_Basic(t *testing.T) {
	input := `<?xml version="1.0" encoding="utf-8"?><transcript><text start="0.5" dur="2.3">Hello</text><text start="3.0" dur="1.5">World</text></transcript>`
	lines, err := parseTimedText([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("want 2 lines, got %d", len(lines))
	}
	if lines[0].Text != "Hello" {
		t.Errorf("line[0].Text = %q, want Hello", lines[0].Text)
	}
	if lines[0].Start != 0.5 {
		t.Errorf("line[0].Start = %f, want 0.5", lines[0].Start)
	}
	if lines[0].Duration != 2.3 {
		t.Errorf("line[0].Duration = %f, want 2.3", lines[0].Duration)
	}
	if lines[1].Text != "World" {
		t.Errorf("line[1].Text = %q, want World", lines[1].Text)
	}
}

func TestParseTimedText_SkipsEmptyLines(t *testing.T) {
	input := `<transcript><text start="0" dur="1">  </text><text start="1" dur="1">Hi</text></transcript>`
	lines, err := parseTimedText([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 1 || lines[0].Text != "Hi" {
		t.Errorf("expected 1 non-empty line, got %d: %v", len(lines), lines)
	}
}

func TestParseTimedText_InvalidXML(t *testing.T) {
	_, err := parseTimedText([]byte("not valid xml <<"))
	if err == nil {
		t.Error("expected error for invalid XML, got nil")
	}
}

// ---- selectTrack ----

func TestSelectTrack_PrefersManualOverASR(t *testing.T) {
	items := []CaptionItem{
		{ID: "asr1", Snippet: CaptionSnippet{Language: "en", TrackKind: "asr"}},
		{ID: "manual1", Snippet: CaptionSnippet{Language: "en", TrackKind: "standard"}},
	}
	got := selectTrack(items, "en")
	if got == nil || got.ID != "manual1" {
		t.Errorf("expected manual1, got %v", got)
	}
}

func TestSelectTrack_FallsBackToASR(t *testing.T) {
	items := []CaptionItem{
		{ID: "asr1", Snippet: CaptionSnippet{Language: "en", TrackKind: "asr"}},
	}
	got := selectTrack(items, "en")
	if got == nil || got.ID != "asr1" {
		t.Errorf("expected asr1, got %v", got)
	}
}

func TestSelectTrack_LanguageFilter(t *testing.T) {
	items := []CaptionItem{
		{ID: "fr1", Snippet: CaptionSnippet{Language: "fr", TrackKind: "standard"}},
		{ID: "en1", Snippet: CaptionSnippet{Language: "en", TrackKind: "standard"}},
	}
	got := selectTrack(items, "en")
	if got == nil || got.ID != "en1" {
		t.Errorf("expected en1, got %v", got)
	}
}

func TestSelectTrack_EmptyLanguageReturnsFirst(t *testing.T) {
	items := []CaptionItem{
		{ID: "fr1", Snippet: CaptionSnippet{Language: "fr", TrackKind: "standard"}},
		{ID: "en1", Snippet: CaptionSnippet{Language: "en", TrackKind: "standard"}},
	}
	got := selectTrack(items, "")
	if got == nil || got.ID != "fr1" {
		t.Errorf("expected fr1 (first item), got %v", got)
	}
}

func TestSelectTrack_NoMatch_ReturnsNil(t *testing.T) {
	items := []CaptionItem{
		{ID: "fr1", Snippet: CaptionSnippet{Language: "fr", TrackKind: "standard"}},
	}
	got := selectTrack(items, "en")
	if got != nil {
		t.Errorf("expected nil for unmatched language, got %v", got)
	}
}

// ---- ListCaptions ----

func TestListCaptions_Success(t *testing.T) {
	fixture := CaptionListResponse{
		Items: []CaptionItem{
			{ID: "cap1", Snippet: CaptionSnippet{Language: "en", TrackKind: "standard"}},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/captions" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		if r.URL.Query().Get("videoId") != "vid123" {
			t.Errorf("videoId param wrong: %q", r.URL.Query().Get("videoId"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(fixture)
	}))
	defer srv.Close()

	tok := &Token{AccessToken: "tok", ExpiresAt: time.Now().Add(time.Hour)}
	c := NewClientWithToken(nil, tok)
	c.baseURL = srv.URL

	result, err := c.ListCaptions(context.Background(), "vid123")
	if err != nil {
		t.Fatalf("ListCaptions: %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].ID != "cap1" {
		t.Errorf("unexpected result: %+v", result)
	}
}

func TestListCaptions_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": 403, "message": "forbidden"},
		})
	}))
	defer srv.Close()

	tok := &Token{AccessToken: "tok", ExpiresAt: time.Now().Add(time.Hour)}
	c := NewClientWithToken(nil, tok)
	c.baseURL = srv.URL

	_, err := c.ListCaptions(context.Background(), "vid")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- downloadCaption ----

func TestDownloadCaption_Success(t *testing.T) {
	srtData := "1\n00:00:01,000 --> 00:00:02,000\nHello\n\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/captions/cap1" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		if r.URL.Query().Get("tfmt") != "srt" {
			t.Errorf("expected tfmt=srt, got %q", r.URL.Query().Get("tfmt"))
		}
		w.Write([]byte(srtData))
	}))
	defer srv.Close()

	tok := &Token{AccessToken: "tok", ExpiresAt: time.Now().Add(time.Hour)}
	c := NewClientWithToken(nil, tok)
	c.baseURL = srv.URL

	body, err := c.downloadCaption(context.Background(), "cap1")
	if err != nil {
		t.Fatalf("downloadCaption: %v", err)
	}
	if string(body) != srtData {
		t.Errorf("body = %q, want %q", string(body), srtData)
	}
}

func TestDownloadCaption_StatusError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("forbidden"))
	}))
	defer srv.Close()

	tok := &Token{AccessToken: "tok", ExpiresAt: time.Now().Add(time.Hour)}
	c := NewClientWithToken(nil, tok)
	c.baseURL = srv.URL

	_, err := c.downloadCaption(context.Background(), "cap1")
	if err == nil {
		t.Fatal("expected error for non-200 status, got nil")
	}
}

func TestDownloadCaption_EmptyBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// no body written
	}))
	defer srv.Close()

	tok := &Token{AccessToken: "tok", ExpiresAt: time.Now().Add(time.Hour)}
	c := NewClientWithToken(nil, tok)
	c.baseURL = srv.URL

	_, err := c.downloadCaption(context.Background(), "cap1")
	if err == nil {
		t.Fatal("expected error for empty body, got nil")
	}
}

// ---- DownloadTranscript ----

func TestDownloadTranscript_Success(t *testing.T) {
	captionsList := CaptionListResponse{
		Items: []CaptionItem{
			{ID: "cap1", Snippet: CaptionSnippet{Language: "en", TrackKind: "standard"}},
		},
	}
	srtData := "1\n00:00:01,000 --> 00:00:02,000\nHello world\n\n"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/captions":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(captionsList)
		case "/captions/cap1":
			w.Write([]byte(srtData))
		default:
			t.Errorf("unexpected path: %q", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	tok := &Token{AccessToken: "tok", ExpiresAt: time.Now().Add(time.Hour)}
	c := NewClientWithToken(nil, tok)
	c.baseURL = srv.URL

	transcript, err := c.DownloadTranscript(context.Background(), "vid123", "en")
	if err != nil {
		t.Fatalf("DownloadTranscript: %v", err)
	}
	if transcript.VideoID != "vid123" {
		t.Errorf("VideoID = %q, want vid123", transcript.VideoID)
	}
	if transcript.Language != "en" {
		t.Errorf("Language = %q, want en", transcript.Language)
	}
	if len(transcript.Lines) != 1 || transcript.Lines[0].Text != "Hello world" {
		t.Errorf("unexpected lines: %v", transcript.Lines)
	}
}

func TestDownloadTranscript_NoTracksAvailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(CaptionListResponse{})
	}))
	defer srv.Close()

	tok := &Token{AccessToken: "tok", ExpiresAt: time.Now().Add(time.Hour)}
	c := NewClientWithToken(nil, tok)
	c.baseURL = srv.URL

	_, err := c.DownloadTranscript(context.Background(), "vid123", "en")
	if err == nil {
		t.Fatal("expected error for video with no caption tracks")
	}
}

func TestDownloadTranscript_NoMatchingLanguage(t *testing.T) {
	captionsList := CaptionListResponse{
		Items: []CaptionItem{
			{ID: "fr1", Snippet: CaptionSnippet{Language: "fr", TrackKind: "standard"}},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(captionsList)
	}))
	defer srv.Close()

	tok := &Token{AccessToken: "tok", ExpiresAt: time.Now().Add(time.Hour)}
	c := NewClientWithToken(nil, tok)
	c.baseURL = srv.URL

	_, err := c.DownloadTranscript(context.Background(), "vid123", "en")
	if err == nil {
		t.Fatal("expected error when no track matches requested language")
	}
}
