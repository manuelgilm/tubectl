package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type StoredTranscript struct {
	VideoID   string
	Language  string
	TrackKind string
	CaptionID string
	Content   string
	Lines     string
	CachedAt  time.Time
}

type TranscriptRepo struct{ db *sql.DB }

func NewTranscriptRepo(db *sql.DB) *TranscriptRepo {
	return &TranscriptRepo{db: db}
}

func (r *TranscriptRepo) Save(ctx context.Context, t *StoredTranscript) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO transcripts (video_id, language, track_kind, caption_id, content, lines, cached_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(video_id, language) DO UPDATE SET
			track_kind = excluded.track_kind,
			caption_id = excluded.caption_id,
			content = excluded.content,
			lines = excluded.lines,
			cached_at = excluded.cached_at`,
		t.VideoID, t.Language, t.TrackKind, t.CaptionID, t.Content, t.Lines, formatTime(t.CachedAt),
	)
	if err != nil {
		return fmt.Errorf("save transcript %s: %w", t.VideoID, err)
	}
	return nil
}

func (r *TranscriptRepo) Load(ctx context.Context, videoID string) (*StoredTranscript, error) {
	t := &StoredTranscript{}
	var cachedAt string
	err := r.db.QueryRowContext(ctx, `
		SELECT video_id, language, track_kind, caption_id, content, lines, cached_at
		FROM transcripts WHERE video_id = ?`, videoID).Scan(
		&t.VideoID, &t.Language, &t.TrackKind, &t.CaptionID, &t.Content, &t.Lines, &cachedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load transcript %s: %w", videoID, err)
	}
	t.CachedAt = parseTime(cachedAt)
	return t, nil
}

func (r *TranscriptRepo) Delete(ctx context.Context, videoID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM transcripts WHERE video_id = ?`, videoID)
	if err != nil {
		return fmt.Errorf("delete transcript %s: %w", videoID, err)
	}
	return nil
}
