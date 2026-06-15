package youtube

// --------------------------Getting transcript from a video -----------------------------------------
// CaptionSnippet holds metadata about a caption track.
type CaptionSnippet struct {
	Language     string `json:"language"`
	Name         string `json:"name"`
	TrackKind    string `json:"trackKind"` // "standard", "asr" (auto-generated), "forced"
	IsAutoSynced bool   `json:"isAutoSynced"`
}
// CaptionItem represents a single caption track.
type CaptionItem struct {
	ID      string          `json:"id"`
	Snippet CaptionSnippet  `json:"snippet"`
}

// CaptionListResponse is the response from captions.list.
type CaptionListResponse struct {
	Items []CaptionItem `json:"items"`
}



// ----- Transcript types -----

// TranscriptLine is a single timed line from a caption track.
type TranscriptLine struct {
	Start    float64 `json:"start"`
	Duration float64 `json:"duration"`
	Text     string  `json:"text"`
}

// Transcript is the full transcript for a video, ready to be cached.
type Transcript struct {
	VideoID    string           `json:"video_id"`
	Language   string           `json:"language"`
	TrackKind  string           `json:"track_kind"`
	CaptionID  string           `json:"caption_id"`
	Lines      []TranscriptLine `json:"lines"`
}
