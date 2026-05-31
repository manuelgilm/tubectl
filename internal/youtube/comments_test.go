package youtube

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestListComments_Success(t *testing.T) {
	fixture := CommentThreadListResponse{
		Items: []CommentThreadItem{
			{Snippet: CommentThreadSnippet{
				TopLevelComment: TopLevelComment{
					Snippet: TopLevelCommentSnippet{TextDisplay: "Great video!", AuthorName: "Alice"},
				},
			}},
			{Snippet: CommentThreadSnippet{
				TopLevelComment: TopLevelComment{
					Snippet: TopLevelCommentSnippet{TextDisplay: "Thanks for this.", AuthorName: "Bob"},
				},
			}},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/commentThreads" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if r.URL.Query().Get("videoId") != "abc123" {
			t.Errorf("videoId param missing or wrong: %q", r.URL.Query().Get("videoId"))
		}
		if r.URL.Query().Get("part") != "snippet" {
			t.Errorf("part param missing: %q", r.URL.Query().Get("part"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(fixture)
	}))
	defer srv.Close()

	tok := &Token{AccessToken: "test-token", ExpiresAt: time.Now().Add(time.Hour)}
	c := NewClientWithToken(nil, tok)
	c.baseURL = srv.URL

	result, err := c.ListComments(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("ListComments failed: %v", err)
	}
	if len(result.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(result.Items))
	}
	got := result.Items[0].Snippet.TopLevelComment.Snippet.TextDisplay
	if got != "Great video!" {
		t.Errorf("first comment = %q, want %q", got, "Great video!")
	}
}

func TestListComments_EmptyResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(CommentThreadListResponse{})
	}))
	defer srv.Close()

	tok := &Token{AccessToken: "tok", ExpiresAt: time.Now().Add(time.Hour)}
	c := NewClientWithToken(nil, tok)
	c.baseURL = srv.URL

	result, err := c.ListComments(context.Background(), "no-comments-video")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(result.Items))
	}
}

func TestListComments_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": 403, "message": "commentsDisabled"},
		})
	}))
	defer srv.Close()

	tok := &Token{AccessToken: "tok", ExpiresAt: time.Now().Add(time.Hour)}
	c := NewClientWithToken(nil, tok)
	c.baseURL = srv.URL

	_, err := c.ListComments(context.Background(), "disabled-video")
	if err == nil {
		t.Fatal("expected error for disabled comments, got nil")
	}
}

func TestListComments_DateFilter(t *testing.T) {
	after := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	fixture := CommentThreadListResponse{
		Items: []CommentThreadItem{
			{Snippet: CommentThreadSnippet{TopLevelComment: TopLevelComment{
				Snippet: TopLevelCommentSnippet{
					TextDisplay: "Old comment",
					PublishedAt: time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
				},
			}}},
			{Snippet: CommentThreadSnippet{TopLevelComment: TopLevelComment{
				Snippet: TopLevelCommentSnippet{
					TextDisplay: "New comment",
					PublishedAt: time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
				},
			}}},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(fixture)
	}))
	defer srv.Close()

	tok := &Token{AccessToken: "tok", ExpiresAt: time.Now().Add(time.Hour)}
	c := NewClientWithToken(nil, tok)
	c.baseURL = srv.URL

	result, err := c.ListComments(context.Background(), "vid123", CommentFilter{PublishedAfter: after})
	if err != nil {
		t.Fatalf("ListComments: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item after date filter, got %d", len(result.Items))
	}
	if result.Items[0].Snippet.TopLevelComment.Snippet.TextDisplay != "New comment" {
		t.Errorf("unexpected item: %q", result.Items[0].Snippet.TopLevelComment.Snippet.TextDisplay)
	}
}

func TestGetComment_Success(t *testing.T) {
	fixture := commentListResponse{
		Items: []Comment{
			{ID: "comment1", Snippet: CommentSnippet{TextDisplay: "Great video!"}},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/comments" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		if r.URL.Query().Get("id") != "comment1" {
			t.Errorf("id param wrong: %q", r.URL.Query().Get("id"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(fixture)
	}))
	defer srv.Close()

	tok := &Token{AccessToken: "tok", ExpiresAt: time.Now().Add(time.Hour)}
	c := NewClientWithToken(nil, tok)
	c.baseURL = srv.URL

	comment, err := c.GetComment(context.Background(), "comment1")
	if err != nil {
		t.Fatalf("GetComment: %v", err)
	}
	if comment.ID != "comment1" {
		t.Errorf("comment ID = %q, want comment1", comment.ID)
	}
	if comment.Snippet.TextDisplay != "Great video!" {
		t.Errorf("TextDisplay = %q, want %q", comment.Snippet.TextDisplay, "Great video!")
	}
}

func TestGetComment_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(commentListResponse{})
	}))
	defer srv.Close()

	tok := &Token{AccessToken: "tok", ExpiresAt: time.Now().Add(time.Hour)}
	c := NewClientWithToken(nil, tok)
	c.baseURL = srv.URL

	_, err := c.GetComment(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error for missing comment, got nil")
	}
}

func TestReplyToComment_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/comments" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		if r.URL.Query().Get("part") != "snippet" {
			t.Errorf("part param missing: %q", r.URL.Query().Get("part"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Comment{ID: "reply1", Snippet: CommentSnippet{TextDisplay: "Thanks!"}})
	}))
	defer srv.Close()

	tok := &Token{AccessToken: "tok", ExpiresAt: time.Now().Add(time.Hour)}
	c := NewClientWithToken(nil, tok)
	c.baseURL = srv.URL

	reply, err := c.ReplyToComment(context.Background(), "parent1", "Thanks!")
	if err != nil {
		t.Fatalf("ReplyToComment: %v", err)
	}
	if reply.ID != "reply1" {
		t.Errorf("reply ID = %q, want reply1", reply.ID)
	}
}
