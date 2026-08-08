package youtube

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const defaultBaseURL = "https://www.googleapis.com/youtube/v3"

// defaultPublicBaseURL hosts the watch page and timedtext endpoints used by the
// public transcript fallback. Overridable in tests via Client.publicBaseURL.
const defaultPublicBaseURL = "https://www.youtube.com"

// playerUA mimics a browser so YouTube serves the player response and captions.
const playerUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Safari/537.36"

type Client struct {
	httpClient    *http.Client
	baseURL       string
	publicBaseURL string
	token         *Token
}

func (c *Client) publicHost() string {
	if c.publicBaseURL != "" {
		return c.publicBaseURL
	}
	return defaultPublicBaseURL
}

// publicURL builds an absolute URL against the public base (watch page /
// timedtext host), appending the given path and query.
func (c *Client) publicURL(path string, query url.Values) (string, error) {
	base, err := url.Parse(c.publicHost())
	if err != nil {
		return "", fmt.Errorf("parsing public base URL %q: %w", c.publicHost(), err)
	}
	base.Path = base.Path + path
	base.RawQuery = query.Encode()
	return base.String(), nil
}

// NewClient creates a YouTube Data API client using the given OAuth token.
func NewClient(token *Token) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    defaultBaseURL,
		token:      token,
	}
}

func (c *Client) get(ctx context.Context, path string, params map[string]string, out any) error {

	endpoint, err := url.JoinPath(c.baseURL, path)
	if err != nil {
		return fmt.Errorf("building URL: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}

	q := req.URL.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Authorization", "Bearer "+c.token.AccessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return youtubeAPIError(resp)
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) delete(ctx context.Context, path string, params map[string]string) error {
	endpoint, err := url.JoinPath(c.baseURL, path)
	if err != nil {
		return fmt.Errorf("building URL: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}

	q := req.URL.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Authorization", "Bearer "+c.token.AccessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return youtubeAPIError(resp)
	}

	return nil
}

func (c *Client) post(ctx context.Context, path string, params map[string]string, body any, out any) error {

	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshaling request body: %w", err)
	}

	endpoint, err := url.JoinPath(c.baseURL, path)
	if err != nil {
		return fmt.Errorf("building URL: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}

	q := req.URL.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Authorization", "Bearer "+c.token.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return youtubeAPIError(resp)
	}

	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

func (c *Client) PostComment(ctx context.Context, videoID string, text string) error {
	body := map[string]any{
		"snippet": map[string]any{
			"videoId": videoID,
			"topLevelComment": map[string]any{
				"snippet": map[string]any{
					"textOriginal": text,
				},
			},
		},
	}
	err := c.post(ctx, "/commentThreads", map[string]string{
		"part": "snippet",
	}, body, nil)
	if err != nil {
		return fmt.Errorf("posting a comment: %w", err)
	}
	return nil
}
func (c *Client) GetComments(ctx context.Context, videoID string, maxResults int, order string) ([]CommentThread, error) {
	var result struct {
		Items []CommentThread `json:"items"`
	}
	err := c.get(ctx, "/commentThreads", map[string]string{
		"part":       "snippet",
		"videoId":    videoID,
		"maxResults": fmt.Sprintf("%d", maxResults),
		"order":      order,
	}, &result)
	if err != nil {
		return nil, err
	}
	return result.Items, nil
}

func (c *Client) GetComment(ctx context.Context, commentID string) (*Comment, error) {
	var result struct {
		Items []Comment `json:"items"`
	}
	if err := c.get(ctx, "/comments", map[string]string{
		"part": "snippet",
		"id":   commentID,
	}, &result); err != nil {
		return nil, fmt.Errorf("getting comment %s: %w", commentID, err)
	}
	if len(result.Items) == 0 {
		return nil, fmt.Errorf("comment %s not found", commentID)
	}

	return &result.Items[0], nil
}

func (c *Client) DeleteComment(ctx context.Context, commentID string) error {
	return c.delete(ctx, "/comments", map[string]string{"id": commentID})
}

func (c *Client) ReplyToComment(ctx context.Context, parentID, text string) (*Comment, error) {
	type snippet struct {
		ParentID     string `json:"parentId"`
		TextOriginal string `json:"textOriginal"`
	}
	type reqBody struct {
		Snippet snippet `json:"snippet"`
	}

	var result Comment

	body := reqBody{Snippet: snippet{ParentID: parentID, TextOriginal: text}}
	if err := c.post(ctx, "/comments", map[string]string{
		"part": "snippet",
	}, body, &result); err != nil {
		return nil, fmt.Errorf("replying to comment %s: %w", parentID, err)
	}
	return &result, nil
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
		// Keep the original API error in the failure path so the real cause
		// (ownership, quota, scope, ...) is visible instead of swallowed.
		pub, pubErr := c.downloadPublicTranscript(ctx, videoID, track.Snippet.Language)
		if pubErr != nil {
			return nil, fmt.Errorf("caption download failed for video %s (track kind %q, language %q): %v; public fallback failed: %w",
				videoID, track.Snippet.TrackKind, track.Snippet.Language, err, pubErr)
		}
		return pub, nil
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

func (c *Client) GetVideo(ctx context.Context, videoID string) (*Video, error) {
	var result struct {
		Items []Video `json:"items"`
	}
	err := c.get(ctx, "/videos", map[string]string{
		"part": "snippet",
		"id":   videoID,
	}, &result)
	if err != nil {
		return nil, err
	}
	if len(result.Items) == 0 {
		return nil, fmt.Errorf("video %s not found", videoID)
	}

	return &result.Items[0], nil
}

// downloadCaption fetches the raw caption body for a given caption ID in SRT format.
func (c *Client) downloadCaption(ctx context.Context, captionID string) ([]byte, error) {

	endpoint, err := url.JoinPath(c.baseURL, "captions", captionID)
	if err != nil {
		return nil, fmt.Errorf("building caption download URL: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
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

// downloadPublicTranscript fetches captions for public videos without
// requiring ownership. It first tries the signed timedtext URL that YouTube
// embeds in the watch page's player response (required for auto-generated
// tracks), then falls back to the bare timedtext endpoint.
func (c *Client) downloadPublicTranscript(ctx context.Context, videoID, language string) (*Transcript, error) {
	if tracks, err := c.playerCaptionTracks(ctx, videoID); err == nil {
		if track := selectPlayerTrack(tracks, language); track != nil {
			if t, err := c.fetchSignedTranscript(ctx, videoID, track); err == nil {
				return t, nil
			}
		}
	}
	return c.fetchNaiveTimedText(ctx, videoID, language)
}

type playerCaptionTrack struct {
	BaseURL      string `json:"baseUrl"`
	LanguageCode string `json:"languageCode"`
	Kind         string `json:"kind"` // "asr" for auto-generated tracks
	TrackName    string `json:"trackName"`
}

// playerCaptionTracks fetches the watch page and extracts the caption tracks
// embedded in its ytInitialPlayerResponse JSON.
func (c *Client) playerCaptionTracks(ctx context.Context, videoID string) ([]playerCaptionTrack, error) {
	watchURL, err := c.publicURL("/watch", url.Values{"v": []string{videoID}})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, watchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building watch page request: %w", err)
	}
	req.Header.Set("User-Agent", playerUA)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching watch page: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading watch page: %w", err)
	}
	return extractCaptionTracks(body)
}

// fetchSignedTranscript downloads a caption track through the signed baseUrl
// taken from the player response.
func (c *Client) fetchSignedTranscript(ctx context.Context, videoID string, track *playerCaptionTrack) (*Transcript, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, track.BaseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building timedtext request: %w", err)
	}
	req.Header.Set("User-Agent", playerUA)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("timedtext request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading timedtext response: %w", err)
	}
	if len(body) == 0 {
		return nil, fmt.Errorf("no public captions available for video %s (language: %s)", videoID, track.LanguageCode)
	}

	lines, err := parseTimedText(body)
	if err != nil {
		return nil, fmt.Errorf("parsing timedtext: %w", err)
	}

	return &Transcript{
		VideoID:   videoID,
		Language:  track.LanguageCode,
		TrackKind: "public",
		Lines:     lines,
	}, nil
}

// fetchNaiveTimedText uses YouTube's undocumented timedtext endpoint without a
// signature. It works for some manual tracks but returns an empty body for
// auto-generated ones.
func (c *Client) fetchNaiveTimedText(ctx context.Context, videoID, language string) (*Transcript, error) {
	endpoint, err := c.publicURL("/api/timedtext", url.Values{
		"fmt":  []string{"srv1"},
		"lang": []string{language},
		"v":    []string{videoID},
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("building timedtext request: %w", err)
	}
	req.Header.Set("User-Agent", playerUA)

	resp, err := c.httpClient.Do(req)
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

// extractCaptionTracks locates the "captionTracks" array inside the watch
// page's ytInitialPlayerResponse and unmarshals it. JSON escapes such as
// \u0026 in baseUrl are decoded by the JSON decoder.
func extractCaptionTracks(page []byte) ([]playerCaptionTrack, error) {
	const marker = `"captionTracks":[`
	idx := bytes.Index(page, []byte(marker))
	if idx < 0 {
		return nil, fmt.Errorf("no captionTracks found in player response")
	}
	start := idx + len(marker) - 1

	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(page); i++ {
		ch := page[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}
		switch ch {
		case '"':
			inString = true
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				var tracks []playerCaptionTrack
				if err := json.Unmarshal(page[start:i+1], &tracks); err != nil {
					return nil, fmt.Errorf("parsing captionTracks: %w", err)
				}
				return tracks, nil
			}
		}
	}
	return nil, fmt.Errorf("unterminated captionTracks array")
}

// selectPlayerTrack picks the best caption track: manual > asr, filtered by
// language, mirroring selectTrack.
func selectPlayerTrack(tracks []playerCaptionTrack, language string) *playerCaptionTrack {
	var asr *playerCaptionTrack
	for i := range tracks {
		track := &tracks[i]
		if language != "" && track.LanguageCode != language {
			continue
		}
		if track.Kind != "asr" {
			return track
		}
		if asr == nil {
			asr = track
		}
	}
	return asr
}

// youtubeAPIError parses a YouTube API error response into a Go error.
func youtubeAPIError(resp *http.Response) error {
	var apiErr struct {
		Error struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
		} `json:"error"`
	}
	if decodeErr := json.NewDecoder(resp.Body).Decode(&apiErr); decodeErr != nil {
		return fmt.Errorf("youtube api error (status %d, could not parse body: %w)", resp.StatusCode, decodeErr)
	}
	return fmt.Errorf("youtube api error %d: %s", apiErr.Error.Code, apiErr.Error.Message)
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
