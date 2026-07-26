package storage

import "time"

type Video struct {
	ID           string
	Title        string
	Description  string
	ChannelID    string
	PublishedAt  time.Time
	RegisteredAt time.Time
	UpdatedAt    time.Time
}
