package youtube

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestClient(srv *httptest.Server) *Client {
	return &Client{
		httpClient: srv.Client(),
		baseURL:    srv.URL,
		token: &Token{
			AccessToken: "test-token",
			ExpiresAt:   time.Now().Add(time.Hour),
		},
	}
}

func TestNewClient(t *testing.T) {
	tok := &Token{AccessToken: "abc", ExpiresAt: time.Now().Add(time.Hour)}
	c := NewClient(tok)
	if c.baseURL != defaultBaseURL {
		t.Errorf("baseURL = %q", c.baseURL)
	}
	if c.token != tok {
		t.Errorf("token not set")
	}
}

func TestGetVideo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/videos" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("id") != "vid123" {
			t.Errorf("id = %s", r.URL.Query().Get("id"))
		}
		json.NewEncoder(w).Encode(map[string]any{
			"items": []any{
				map[string]any{
					"id": "vid123",
					"snippet": map[string]any{
						"title":       "Test Video",
						"description": "A test",
						"channelId":   "chan1",
					},
				},
			},
		})
	}))
	defer srv.Close()

	c := newTestClient(srv)
	video, err := c.GetVideo(context.Background(), "vid123")
	if err != nil {
		t.Fatalf("GetVideo: %v", err)
	}
	if video.ID != "vid123" || video.Snippet.Title != "Test Video" {
		t.Errorf("video = %+v", video)
	}
}

func TestGetVideo_notFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"items": []any{}})
	}))
	defer srv.Close()

	c := newTestClient(srv)
	_, err := c.GetVideo(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

func TestGetVideo_apiError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"code":    403,
				"message": "forbidden",
			},
		})
	}))
	defer srv.Close()

	c := newTestClient(srv)
	_, err := c.GetVideo(context.Background(), "vid123")
	if err == nil {
		t.Fatal("expected API error")
	}
}

func TestGetComments(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("videoId") != "vid123" {
			t.Errorf("videoId = %s", r.URL.Query().Get("videoId"))
		}
		json.NewEncoder(w).Encode(map[string]any{
			"items": []any{
				map[string]any{
					"id": "c1",
					"snippet": map[string]any{
						"videoId": "vid123",
						"topLevelComment": map[string]any{
							"snippet": map[string]any{
								"authorDisplayName": "User1",
								"textDisplay":       "Great!",
							},
						},
					},
				},
			},
		})
	}))
	defer srv.Close()

	c := newTestClient(srv)
	threads, err := c.GetComments(context.Background(), "vid123", 10, "time")
	if err != nil {
		t.Fatalf("GetComments: %v", err)
	}
	if len(threads) != 1 {
		t.Fatalf("got %d threads, want 1", len(threads))
	}
	if threads[0].ID != "c1" {
		t.Errorf("id = %s", threads[0].ID)
	}
}

func TestGetComments_empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"items": []any{}})
	}))
	defer srv.Close()

	c := newTestClient(srv)
	threads, err := c.GetComments(context.Background(), "vid123", 10, "time")
	if err != nil {
		t.Fatalf("GetComments: %v", err)
	}
	if len(threads) != 0 {
		t.Errorf("got %d threads", len(threads))
	}
}

func TestGetComment(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"items": []any{
				map[string]any{
					"id": "cmt1",
					"snippet": map[string]any{
						"authorDisplayName": "Alice",
						"textDisplay":       "Hello!",
					},
				},
			},
		})
	}))
	defer srv.Close()

	c := newTestClient(srv)
	comment, err := c.GetComment(context.Background(), "cmt1")
	if err != nil {
		t.Fatalf("GetComment: %v", err)
	}
	if comment.ID != "cmt1" || comment.Snippet.TextDisplay != "Hello!" {
		t.Errorf("comment = %+v", comment)
	}
}

func TestGetComment_notFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"items": []any{}})
	}))
	defer srv.Close()

	c := newTestClient(srv)
	_, err := c.GetComment(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDeleteComment(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	if err := c.DeleteComment(context.Background(), "cmt1"); err != nil {
		t.Fatalf("DeleteComment: %v", err)
	}
}

func TestDeleteComment_error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": 404, "message": "not found"},
		})
	}))
	defer srv.Close()

	c := newTestClient(srv)
	if err := c.DeleteComment(context.Background(), "cmt1"); err == nil {
		t.Fatal("expected error")
	}
}

func TestReplyToComment(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"id": "reply1",
			"snippet": map[string]any{
				"authorDisplayName": "Me",
				"textDisplay":       "Thanks!",
			},
		})
	}))
	defer srv.Close()

	c := newTestClient(srv)
	comment, err := c.ReplyToComment(context.Background(), "parent1", "Thanks!")
	if err != nil {
		t.Fatalf("ReplyToComment: %v", err)
	}
	if comment.ID != "reply1" {
		t.Errorf("id = %s", comment.ID)
	}
}

func TestPostComment(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"id": "thread1",
			"snippet": map[string]any{
				"videoId": "vid123",
				"topLevelComment": map[string]any{
					"snippet": map[string]any{
						"authorDisplayName": "Me",
						"textDisplay":       "My comment",
					},
				},
			},
		})
	}))
	defer srv.Close()

	c := newTestClient(srv)
	if err := c.PostComment(context.Background(), "vid123", "My comment"); err != nil {
		t.Fatalf("PostComment: %v", err)
	}
}

func TestPostComment_error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": 400, "message": "bad request"},
		})
	}))
	defer srv.Close()

	c := newTestClient(srv)
	if err := c.PostComment(context.Background(), "vid123", "text"); err == nil {
		t.Fatal("expected error")
	}
}

func TestListCaptions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"items": []any{
				map[string]any{
					"id": "cap1",
					"snippet": map[string]any{
						"language":  "en",
						"name":      "English",
						"trackKind": "standard",
					},
				},
			},
		})
	}))
	defer srv.Close()

	c := newTestClient(srv)
	resp, err := c.ListCaptions(context.Background(), "vid123")
	if err != nil {
		t.Fatalf("ListCaptions: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("got %d items", len(resp.Items))
	}
	if resp.Items[0].ID != "cap1" || resp.Items[0].Snippet.Language != "en" {
		t.Errorf("item = %+v", resp.Items[0])
	}
}

func TestListCaptions_empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"items": []any{}})
	}))
	defer srv.Close()

	c := newTestClient(srv)
	resp, err := c.ListCaptions(context.Background(), "vid123")
	if err != nil {
		t.Fatalf("ListCaptions: %v", err)
	}
	if len(resp.Items) != 0 {
		t.Errorf("got %d items", len(resp.Items))
	}
}

func TestSelectTrack(t *testing.T) {
	t.Run("prefers manual over asr", func(t *testing.T) {
		items := []CaptionItem{
			{ID: "asr1", Snippet: CaptionSnippet{Language: "en", TrackKind: "asr"}},
			{ID: "manual1", Snippet: CaptionSnippet{Language: "en", TrackKind: "standard"}},
		}
		track := selectTrack(items, "")
		if track == nil || track.ID != "manual1" {
			t.Errorf("got %+v, want manual1", track)
		}
	})

	t.Run("falls back to asr", func(t *testing.T) {
		items := []CaptionItem{
			{ID: "asr1", Snippet: CaptionSnippet{Language: "en", TrackKind: "asr"}},
		}
		track := selectTrack(items, "")
		if track == nil || track.ID != "asr1" {
			t.Errorf("got %+v, want asr1", track)
		}
	})

	t.Run("filters by language", func(t *testing.T) {
		items := []CaptionItem{
			{ID: "en1", Snippet: CaptionSnippet{Language: "en", TrackKind: "standard"}},
			{ID: "es1", Snippet: CaptionSnippet{Language: "es", TrackKind: "standard"}},
		}
		track := selectTrack(items, "es")
		if track == nil || track.ID != "es1" {
			t.Errorf("got %+v, want es1", track)
		}
	})

	t.Run("returns nil when no match", func(t *testing.T) {
		items := []CaptionItem{
			{ID: "en1", Snippet: CaptionSnippet{Language: "en", TrackKind: "standard"}},
		}
		track := selectTrack(items, "fr")
		if track != nil {
			t.Errorf("got %+v, want nil", track)
		}
	})

	t.Run("empty items", func(t *testing.T) {
		track := selectTrack(nil, "")
		if track != nil {
			t.Errorf("got %+v, want nil", track)
		}
	})
}

func TestExtractCaptionTracks(t *testing.T) {
	t.Run("extracts and decodes tracks", func(t *testing.T) {
		page := []byte(`var ytInitialPlayerResponse = {"captions":{"playerCaptionsTracklistRenderer":{"captionTracks":[{"baseUrl":"https://www.youtube.com/api/timedtext?v=abc\u0026kind=asr\u0026lang=en","languageCode":"en","kind":"asr","name":{"simpleText":"English (auto)"}},{"baseUrl":"https://www.youtube.com/api/timedtext?v=abc\u0026lang=es","languageCode":"es","kind":"asr"}]}}}};`)
		tracks, err := extractCaptionTracks(page)
		if err != nil {
			t.Fatalf("extractCaptionTracks: %v", err)
		}
		if len(tracks) != 2 {
			t.Fatalf("got %d tracks, want 2", len(tracks))
		}
		if tracks[0].LanguageCode != "en" || tracks[0].Kind != "asr" {
			t.Errorf("track[0] = %+v", tracks[0])
		}
		if want := "https://www.youtube.com/api/timedtext?v=abc&kind=asr&lang=en"; tracks[0].BaseURL != want {
			t.Errorf("BaseURL = %q, want %q", tracks[0].BaseURL, want)
		}
	})

	t.Run("brackets inside strings are skipped", func(t *testing.T) {
		page := []byte(`{"captionTracks":[{"baseUrl":"u[0]","name":{"simpleText":"a [weird] name"},"languageCode":"en"}]}`)
		tracks, err := extractCaptionTracks(page)
		if err != nil {
			t.Fatalf("extractCaptionTracks: %v", err)
		}
		if len(tracks) != 1 || tracks[0].BaseURL != "u[0]" {
			t.Errorf("got %+v", tracks)
		}
	})

	t.Run("no captionTracks", func(t *testing.T) {
		if _, err := extractCaptionTracks([]byte(`{"playabilityStatus":{"status":"OK"}}`)); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("unterminated array", func(t *testing.T) {
		if _, err := extractCaptionTracks([]byte(`{"captionTracks":[{"baseUrl":"x"`)); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestSelectPlayerTrack(t *testing.T) {
	tracks := []playerCaptionTrack{
		{LanguageCode: "en", Kind: "asr"},
		{LanguageCode: "en", Kind: ""},
		{LanguageCode: "es", Kind: ""},
	}

	t.Run("prefers manual over asr for matching language", func(t *testing.T) {
		track := selectPlayerTrack(tracks, "en")
		if track == nil || track.Kind != "" {
			t.Errorf("got %+v, want manual en", track)
		}
	})

	t.Run("language filter", func(t *testing.T) {
		track := selectPlayerTrack(tracks, "es")
		if track == nil || track.LanguageCode != "es" {
			t.Errorf("got %+v, want es", track)
		}
	})

	t.Run("no language picks first manual", func(t *testing.T) {
		track := selectPlayerTrack(tracks, "")
		if track == nil || track.Kind != "" {
			t.Errorf("got %+v, want manual", track)
		}
	})

	t.Run("no match returns nil", func(t *testing.T) {
		if track := selectPlayerTrack(tracks, "fr"); track != nil {
			t.Errorf("got %+v, want nil", track)
		}
	})
}

func TestDownloadPublicTranscript(t *testing.T) {
	const xml = `<transcript><text start="0.5" dur="2.3">Hello world</text><text start="3.0" dur="1.5">Second line</text></transcript>`

	t.Run("uses signed baseUrl from player response", func(t *testing.T) {
		var srv *httptest.Server
		srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/watch":
				json.NewEncoder(w).Encode(struct {
					CaptionTracks []playerCaptionTrack `json:"captionTracks"`
				}{
					CaptionTracks: []playerCaptionTrack{
						{BaseURL: srv.URL + "/timedtext?kind=asr&lang=en", LanguageCode: "en", Kind: "asr"},
					},
				})
			case "/timedtext":
				if r.URL.Query().Get("kind") != "asr" {
					t.Errorf("expected kind=asr on signed URL, got %s", r.URL.RawQuery)
				}
				w.Write([]byte(xml))
			default:
				t.Errorf("unexpected path %s", r.URL.Path)
			}
		}))
		defer srv.Close()

		c := &Client{httpClient: srv.Client(), publicBaseURL: srv.URL}
		tr, err := c.downloadPublicTranscript(context.Background(), "vid123", "en")
		if err != nil {
			t.Fatalf("downloadPublicTranscript: %v", err)
		}
		if tr.Language != "en" || len(tr.Lines) != 2 {
			t.Errorf("got %+v", tr)
		}
	})

	t.Run("falls back to naive timedtext when no captionTracks", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/watch":
				w.Write([]byte(`{"playabilityStatus":{"status":"OK"}}`))
			case "/api/timedtext":
				if r.URL.Query().Get("lang") != "en" {
					t.Errorf("lang = %s", r.URL.Query().Get("lang"))
				}
				w.Write([]byte(xml))
			default:
				t.Errorf("unexpected path %s", r.URL.Path)
			}
		}))
		defer srv.Close()

		c := &Client{httpClient: srv.Client(), publicBaseURL: srv.URL}
		tr, err := c.downloadPublicTranscript(context.Background(), "vid123", "en")
		if err != nil {
			t.Fatalf("downloadPublicTranscript: %v", err)
		}
		if len(tr.Lines) != 2 {
			t.Errorf("got %+v", tr)
		}
	})

	t.Run("errors when nothing is available", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"playabilityStatus":{"status":"OK"}}`))
		}))
		defer srv.Close()

		c := &Client{httpClient: srv.Client(), publicBaseURL: srv.URL}
		if _, err := c.downloadPublicTranscript(context.Background(), "vid123", "en"); err == nil {
			t.Fatal("expected error")
		}
	})
}
