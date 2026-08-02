package youtube

// --------------------------Getting Comments from a video -----------------------------------------
type CommentThread struct {
	ID      string `json:"id"`
	Snippet struct {
		VideoID         string `json:"videoId"`
		TopLevelComment struct {
			Snippet struct {
				AuthorDisplayName string `json:"authorDisplayName"`
				TextDisplay       string `json:"textDisplay"`
				PublishedAt       string `json:"publishedAt"`
			} `json:"snippet"`
		} `json:"topLevelComment"`
	} `json:"snippet"`
}

type Comment struct {
	ID      string `json:"id"`
	Snippet struct {
		AuthorDisplayName string `json:"authorDisplayName"`
		TextDisplay       string `json:"textDisplay"`
		PublishedAt       string `json:"publishedAt"`
	} `json:"snippet"`
}
