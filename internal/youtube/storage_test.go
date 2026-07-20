package youtube

import "testing"

func TestTranscriptCacheRoundtrip(t *testing.T) {
	// Override home by setting HOME so TranscriptCachePath uses our temp dir
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir()) // Windows compat

	transcript := &Transcript{
		VideoID:   "test123",
		Language:  "en",
		TrackKind: "asr",
		Lines: []TranscriptLine{
			{Start: 1.0, Duration: 2.0, Text: "hello"},
		},
	}

	if err := SaveCachedTranscript(transcript); err != nil {
		t.Fatalf("SaveCachedTranscript: %v", err)
	}

	loaded, err := LoadCachedTranscript("test123")
	if err != nil {
		t.Fatalf("LoadCachedTranscript: %v", err)
	}
	if loaded.VideoID != "test123" || loaded.Language != "en" {
		t.Errorf("loaded = %+v", loaded)
	}
	if len(loaded.Lines) != 1 || loaded.Lines[0].Text != "hello" {
		t.Errorf("lines = %+v", loaded.Lines)
	}
}
