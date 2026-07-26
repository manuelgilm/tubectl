package storage

import (
	"context"
	"database/sql"
	"fmt"
)

var migrations = []string{
	`CREATE TABLE IF NOT EXISTS _migrations (
		version INTEGER PRIMARY KEY
	)`,
	`CREATE TABLE IF NOT EXISTS videos (
		id            TEXT PRIMARY KEY,
		title         TEXT NOT NULL,
		description   TEXT NOT NULL DEFAULT '',
		channel_id    TEXT NOT NULL DEFAULT '',
		published_at  TEXT NOT NULL DEFAULT '',
		registered_at TEXT NOT NULL,
		updated_at    TEXT NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS transcripts (
		video_id   TEXT NOT NULL REFERENCES videos(id),
		language   TEXT NOT NULL DEFAULT 'en',
		track_kind TEXT NOT NULL DEFAULT '',
		caption_id TEXT NOT NULL DEFAULT '',
		content    TEXT NOT NULL,
		lines      TEXT NOT NULL,
		cached_at  TEXT NOT NULL,
		UNIQUE(video_id, language)
	)`,
	`CREATE TRIGGER IF NOT EXISTS after_delete_video AFTER DELETE ON videos
	BEGIN
		DELETE FROM transcripts WHERE video_id = OLD.id;
	END`,
}

func Migrate(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, migrations[0]); err != nil {
		return fmt.Errorf("migration v0: %w", err)
	}

	var current int
	err := db.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) FROM _migrations`).Scan(&current)
	if err != nil {
		return fmt.Errorf("read migration version: %w", err)
	}

	for i := current + 1; i < len(migrations); i++ {
		if _, err := db.ExecContext(ctx, migrations[i]); err != nil {
			return fmt.Errorf("migration v%d: %w", i, err)
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO _migrations (version) VALUES (?)`, i); err != nil {
			return fmt.Errorf("record migration v%d: %w", i, err)
		}
	}
	return nil
}
