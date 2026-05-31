package youtube

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

// ----- Caption list types -----

// CaptionListResponse is the response from captions.list.
type CaptionListResponse struct {
	Items []CaptionItem `json:"items"`
}

// CaptionItem represents a single caption track.
type CaptionItem struct {
	ID      string          `json:"id"`
	Snippet CaptionSnippet  `json:"snippet"`
}

// CaptionSnippet holds metadata about a caption track.
type CaptionSnippet struct {
	Language     string `json:"language"`
	Name         string `json:"name"`
	TrackKind    string `json:"trackKind"` // "standard", "asr" (auto-generated), "forced"
	IsAutoSynced bool   `json:"isAutoSynced"`
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

// ListCaptions returns the available caption tracks for a video.
func (c *Client) ListCaptions(ctx context.Context, videoID string) (*CaptionListResponse, error) {
	var result CaptionListResponse
	if err := c.get(ctx, "/captions", map[string]string{
		"part":    "snippet",
		"videoId": videoID,
	}, &result); err != nil {
		return nil, fmt.Errorf("listing captions for video %s: %w", videoID, err)
	}
	return &result, nil
}

// DownloadTranscript downloads a caption track and parses it into a Transcript.
// It prefers the first human-made track in the requested language, falling back
// to auto-generated ("asr") if no manual track is found.
// If language is empty, the first available track is used.
func (c *Client) DownloadTranscript(ctx context.Context, videoID, language string) (*Transcript, error) {
	tracks, err := c.ListCaptions(ctx, videoID)
	if err != nil {
		return nil, err
	}
	if len(tracks.Items) == 0 {
		return nil, fmt.Errorf("no caption tracks available for video %s", videoID)
	}

	track := selectTrack(tracks.Items, language)
	if track == nil {
		return nil, fmt.Errorf("no caption track found for language %q on video %s", language, videoID)
	}

	raw, err := c.downloadCaption(ctx, track.ID)
	if err != nil {
		// Fall back to the public timedtext endpoint for videos we don't own.
		return downloadPublicTranscript(ctx, videoID, track.Snippet.Language)
	}

	lines, err := parseSRT(raw)
	if err != nil {
		return nil, fmt.Errorf("parsing caption track: %w", err)
	}

	return &Transcript{
		VideoID:   videoID,
		Language:  track.Snippet.Language,
		TrackKind: track.Snippet.TrackKind,
		CaptionID: track.ID,
		Lines:     lines,
	}, nil
}

// downloadPublicTranscript uses YouTube's undocumented timedtext endpoint to
// fetch captions for public videos without requiring ownership.
func downloadPublicTranscript(ctx context.Context, videoID, language string) (*Transcript, error) {
	url := "https://www.youtube.com/api/timedtext?fmt=srv1&lang=" + language + "&v=" + videoID
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("building timedtext request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("timedtext request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading timedtext response: %w", err)
	}
	if len(body) == 0 {
		return nil, fmt.Errorf("no public captions available for video %s (language: %s)", videoID, language)
	}

	lines, err := parseTimedText(body)
	if err != nil {
		return nil, fmt.Errorf("parsing timedtext: %w", err)
	}

	return &Transcript{
		VideoID:   videoID,
		Language:  language,
		TrackKind: "public",
		Lines:     lines,
	}, nil
}

// selectTrack picks the best caption track: manual > asr, filtered by language.
func selectTrack(items []CaptionItem, language string) *CaptionItem {
	var asr *CaptionItem
	for i := range items {
		item := &items[i]
		if language != "" && item.Snippet.Language != language {
			continue
		}
		if item.Snippet.TrackKind != "asr" {
			return item // prefer manual/standard track
		}
		if asr == nil {
			asr = item
		}
	}
	return asr // fallback to auto-generated
}

// downloadCaption fetches the raw caption body for a given caption ID in SRT format.
func (c *Client) downloadCaption(ctx context.Context, captionID string) ([]byte, error) {
	if err := c.ensureToken(ctx); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.baseURL+"/captions/"+captionID, nil)
	if err != nil {
		return nil, fmt.Errorf("building caption download request: %w", err)
	}

	// Request SRT format — the most reliably returned format by the API.
	q := req.URL.Query()
	q.Set("tfmt", "srt")
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Authorization", "Bearer "+c.token.AccessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("caption download request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading caption response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("caption download returned status %d: %s", resp.StatusCode, string(body))
	}
	if len(body) == 0 {
		return nil, fmt.Errorf("caption download returned empty body — the video may not be owned by your account")
	}

	return body, nil
}
