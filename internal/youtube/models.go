package youtube

type VideoListResponse struct {
	Items []VideoItem `json:"items"`
}

type VideoItem struct {
	ID      string       `json:"id"`
	Snippet VideoSnippet `json:"snippet"`
}

type VideoSnippet struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	ChannelID   string `json:"channelId"`
}
