package youtube

import (
	"context"
	"fmt"
	"time"
)

// ----- Comment thread types -----

// TopLevelCommentSnippet holds the content fields of a top-level comment.
type TopLevelCommentSnippet struct {
	TextDisplay string `json:"textDisplay"`
	AuthorName  string `json:"authorDisplayName"`
	PublishedAt string `json:"publishedAt"`
}

// TopLevelComment wraps the snippet of a top-level comment.
type TopLevelComment struct {
	ID      string                 `json:"id"`
	Snippet TopLevelCommentSnippet `json:"snippet"`
}

// CommentThreadSnippet holds the top-level comment of a thread.
type CommentThreadSnippet struct {
	TopLevelComment TopLevelComment `json:"topLevelComment"`
}

// CommentThreadItem is a single comment thread returned by the API.
type CommentThreadItem struct {
	ID      string               `json:"id"`
	Snippet CommentThreadSnippet `json:"snippet"`
}

// CommentThreadListResponse is the response from commentThreads.list.
type CommentThreadListResponse struct {
	Items []CommentThreadItem `json:"items"`
}

// CommentFilter holds optional filters applied client-side to ListComments results.
type CommentFilter struct {
	// PublishedAfter keeps only comments published strictly after this time.
	// Zero value disables this filter.
	PublishedAfter time.Time
}

// ListComments returns comment threads for a video.
// Optionally pass a CommentFilter to apply client-side filtering.
func (c *Client) ListComments(ctx context.Context, videoID string, opts ...CommentFilter) (*CommentThreadListResponse, error) {
	var result CommentThreadListResponse
	if err := c.get(ctx, "/commentThreads", map[string]string{
		"part":    "snippet",
		"videoId": videoID,
	}, &result); err != nil {
		return nil, fmt.Errorf("listing comments for video %s: %w", videoID, err)
	}

	if len(opts) > 0 && !opts[0].PublishedAfter.IsZero() {
		cutoff := opts[0].PublishedAfter
		filtered := result.Items[:0]
		for _, item := range result.Items {
			pub, err := time.Parse(time.RFC3339, item.Snippet.TopLevelComment.Snippet.PublishedAt)
			if err != nil || pub.After(cutoff) {
				filtered = append(filtered, item)
			}
		}
		result.Items = filtered
	}

	return &result, nil
}

// ----- Single comment types -----

// CommentSnippet holds the content of a single comment or reply.
type CommentSnippet struct {
	TextDisplay  string `json:"textDisplay"`
	TextOriginal string `json:"textOriginal"`
	AuthorName   string `json:"authorDisplayName"`
	PublishedAt  string `json:"publishedAt"`
	UpdatedAt    string `json:"updatedAt"`
	LikeCount    int64  `json:"likeCount"`
	ParentID     string `json:"parentId,omitempty"`
}

// Comment is a single YouTube comment (used for direct lookups and replies).
type Comment struct {
	ID      string         `json:"id"`
	Snippet CommentSnippet `json:"snippet"`
}

type commentListResponse struct {
	Items []Comment `json:"items"`
}

// GetComment fetches a single comment by ID.
func (c *Client) GetComment(ctx context.Context, commentID string) (*Comment, error) {
	var result commentListResponse
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

// ReplyToComment posts a reply to an existing comment thread.
// Requires ScopeYoutubeForceSsl.
func (c *Client) ReplyToComment(ctx context.Context, parentCommentID, text string) (*Comment, error) {
	type snippet struct {
		ParentID     string `json:"parentId"`
		TextOriginal string `json:"textOriginal"`
	}
	type reqBody struct {
		Snippet snippet `json:"snippet"`
	}
	body := reqBody{Snippet: snippet{ParentID: parentCommentID, TextOriginal: text}}

	var result Comment
	if err := c.post(ctx, "/comments", map[string]string{"part": "snippet"}, body, &result); err != nil {
		return nil, fmt.Errorf("replying to comment %s: %w", parentCommentID, err)
	}
	return &result, nil
}
