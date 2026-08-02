package youtube

import (
	"time"
)

// --------------------------Getting video metadata -----------------------------------------
type Video struct {
	ID      string `json:"id"`
	Snippet struct {
		Title       string    `json:"title"`
		Description string    `json:"description"`
		PublishedAt time.Time `json:"publishedAt"`
		ChannelID   string    `json:"channelId"`
	} `json:"snippet"`
}
