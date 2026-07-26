package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type VideoRepo struct{ db *sql.DB }

func NewVideoRepo(db *sql.DB) *VideoRepo {
	return &VideoRepo{db: db}
}

func (r *VideoRepo) Add(ctx context.Context, v Video) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO videos (id, title, description, channel_id, published_at, registered_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		v.ID, v.Title, v.Description, v.ChannelID,
		formatTime(v.PublishedAt), formatTime(v.RegisteredAt), formatTime(v.UpdatedAt),
	)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("video %s already registered", v.ID)
		}
		return fmt.Errorf("add video %s: %w", v.ID, err)
	}
	return nil
}

func (r *VideoRepo) Get(ctx context.Context, id string) (*Video, error) {
	v := &Video{}
	var pubAt, regAt, updAt string
	err := r.db.QueryRowContext(ctx, `
		SELECT id, title, description, channel_id, published_at, registered_at, updated_at
		FROM videos WHERE id = ?`, id).Scan(
		&v.ID, &v.Title, &v.Description, &v.ChannelID, &pubAt, &regAt, &updAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get video %s: %w", id, err)
	}
	v.PublishedAt = parseTime(pubAt)
	v.RegisteredAt = parseTime(regAt)
	v.UpdatedAt = parseTime(updAt)
	return v, nil
}

func (r *VideoRepo) List(ctx context.Context) ([]Video, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, title, description, channel_id, published_at, registered_at, updated_at
		FROM videos ORDER BY published_at DESC, registered_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list videos: %w", err)
	}
	defer rows.Close()

	vs := make([]Video, 0)
	for rows.Next() {
		var v Video
		var pubAt, regAt, updAt string
		if err := rows.Scan(&v.ID, &v.Title, &v.Description, &v.ChannelID, &pubAt, &regAt, &updAt); err != nil {
			return nil, fmt.Errorf("scan video: %w", err)
		}
		v.PublishedAt = parseTime(pubAt)
		v.RegisteredAt = parseTime(regAt)
		v.UpdatedAt = parseTime(updAt)
		vs = append(vs, v)
	}
	return vs, rows.Err()
}

func (r *VideoRepo) Update(ctx context.Context, id, title string) (*Video, error) {
	now := formatTime(time.Now())
	res, err := r.db.ExecContext(ctx, `UPDATE videos SET title = ?, updated_at = ? WHERE id = ?`, title, now, id)
	if err != nil {
		return nil, fmt.Errorf("update video %s: %w", id, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, nil
	}
	return r.Get(ctx, id)
}

func (r *VideoRepo) Delete(ctx context.Context, id string) (bool, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM videos WHERE id = ?`, id)
	if err != nil {
		return false, fmt.Errorf("delete video %s: %w", id, err)
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "UNIQUE constraint")
}
