package youtube 

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"fmt"
	"io"
)

const defaultBaseURL = "https://www.googleapis.com/youtube/v3"

type Client struct {
	httpClient		*http.Client
	baseURL			string
	token			*Token
}


func NewClient(token *Token) *Client {
	return &Client{
		httpClient: &http.Client{},
		baseURL:    defaultBaseURL,
		token: 		token,
	}
}

func (c *Client) get(ctx context.Context, path string, params map[string]string, out any) error {

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
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

    return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) delete(ctx context.Context, path string, params map[string]string) error {
    req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+path, nil)
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

    return nil
}

func (c *Client) post(ctx context.Context, path string, params map[string]string, body any, out any) error {

	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshaling request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(data))
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

	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

func (c *Client) PostComment(ctx context.Context, videoID string, text string) error {
	var result CommentThread
	type snippet struct	{
		VideoID	string 	`json:"videoId"`
		TopLevelComment	struct	{
			Snippet	struct {
				TextOriginal	string  `json:"textOriginal"`
			} 	`json:"snippet"`
		} 	`json:"topLevelComment"`
	}

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
	}, body, &result)
	if err != nil {
		return fmt.Errorf("posting a comment %w ", err)
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
		Items	[]Comment `json:"items"`
	}
	if err := c.get(ctx, "/comments", map[string]string{
		"part": "snippet",
		"id": commentID,
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
	type snippet struct	{
		ParentID	string 	`json:"parentId"`
		TextOriginal	string	`json:"textOriginal"`
	}
	type reqBody	struct {
		Snippet	snippet		`json:"snippet"`
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


func (c *Client) GetVideo(ctx context.Context, videoID string) (*Video, error) {
	var result struct {
		Items 	[]Video	`json:"items"`
	}
	err := c.get(ctx, "/videos", map[string]string{
		"part":"snippet",
		"id":videoID,
	}, &result)
	if err != nil {
		return nil, err
	}
	if len(result.Items) == 0 {
		return nil, fmt.Errorf("Video %s not found", videoID)
	}

	return &result.Items[0], nil
}

// downloadCaption fetches the raw caption body for a given caption ID in SRT format.
func (c *Client) downloadCaption(ctx context.Context, captionID string) ([]byte, error) {

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


//  Functions 
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
