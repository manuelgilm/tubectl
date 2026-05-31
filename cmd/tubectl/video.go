package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"tubectl/internal/youtube"
)

var videoID string

var videoCmd = &cobra.Command{
	Use:   "video",
	Short: "Interact with YouTube videos",
}

var getCommentsCmd = &cobra.Command{
	Use:   "get-comments",
	Short: "List comments for a video",
	Example: `  tubectl video --video-id dQw4w9WgXcQ get-comments
  tubectl video --video-id dQw4w9WgXcQ get-comments --published-after 2024-01-01`,
	RunE: runGetComments,
}

var publishedAfterFlag string

var languageFlag string

var getTranscriptCmd = &cobra.Command{
	Use:   "get-transcript",
	Short: "Fetch and display the transcript (captions) of a video",
	Long: `Downloads the caption track for a video and prints the transcript.
The transcript is cached in ~/.config/tubectl/transcripts/<videoID>.json.
On subsequent calls the cached version is used — no API quota is consumed.`,
	Example: `  tubectl video --video-id dQw4w9WgXcQ get-transcript
  tubectl video --video-id dQw4w9WgXcQ get-transcript --language en
  tubectl video --video-id dQw4w9WgXcQ get-transcript --no-cache`,
	RunE: runGetTranscript,
}

var noCache bool

func init() {
	videoCmd.PersistentFlags().StringVar(&videoID, "video-id", "", "YouTube video ID (required)")
	videoCmd.MarkPersistentFlagRequired("video-id")

	getCommentsCmd.Flags().StringVar(&publishedAfterFlag, "published-after", "", "Only show comments published after this date (YYYY-MM-DD)")

	getTranscriptCmd.Flags().StringVar(&languageFlag, "language", "", "Preferred caption language (e.g. en, es). Defaults to first available.")
	getTranscriptCmd.Flags().BoolVar(&noCache, "no-cache", false, "Skip the local cache and always fetch from the API")

	videoCmd.AddCommand(getCommentsCmd)
	videoCmd.AddCommand(getTranscriptCmd)
}

func runGetTranscript(cmd *cobra.Command, args []string) error {
	if !noCache {
		cached, err := loadCachedTranscript(videoID)
		if err != nil {
			return err
		}
		if cached != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "Using cached transcript (language: %s, kind: %s)\n\n",
				cached.Language, cached.TrackKind)
			printTranscript(cached)
			return nil
		}
	}

	client, err := loadClient()
	if err != nil {
		return err
	}

	fmt.Fprintln(cmd.ErrOrStderr(), "Fetching transcript from YouTube API...")
	transcript, err := client.DownloadTranscript(cmd.Context(), videoID, languageFlag)
	if err != nil {
		return err
	}

	if err := saveCachedTranscript(transcript); err != nil {
		// Non-fatal: warn but still print the transcript.
		fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not cache transcript: %v\n", err)
	} else {
		path, _ := transcriptCachePath(videoID)
		fmt.Fprintf(cmd.ErrOrStderr(), "Transcript cached to %s\n\n", path)
	}

	printTranscript(transcript)
	return nil
}

func printTranscript(t *youtube.Transcript) {
	for _, line := range t.Lines {
		minutes := int(line.Start) / 60
		seconds := int(line.Start) % 60
		fmt.Printf("[%02d:%02d] %s\n", minutes, seconds, line.Text)
	}
}

func runGetComments(cmd *cobra.Command, args []string) error {
	client, err := loadClient()
	if err != nil {
		return err
	}

	var opts []youtube.CommentFilter
	if publishedAfterFlag != "" {
		cutoff, err := time.Parse("2006-01-02", publishedAfterFlag)
		if err != nil {
			return fmt.Errorf("invalid --published-after date %q: use YYYY-MM-DD format", publishedAfterFlag)
		}
		opts = append(opts, youtube.CommentFilter{PublishedAfter: cutoff})
	}

	result, err := client.ListComments(cmd.Context(), videoID, opts...)
	if err != nil {
		return err
	}

	if len(result.Items) == 0 {
		fmt.Println("No comments found.")
		return nil
	}

	fmt.Printf("%d comment(s) for video %s\n\n", len(result.Items), videoID)
	for _, item := range result.Items {
		s := item.Snippet.TopLevelComment.Snippet
		date := s.PublishedAt
		if t, err := time.Parse(time.RFC3339, s.PublishedAt); err == nil {
			date = t.Format("2006-01-02")
		}
		fmt.Printf("  comment-id: %s\n  [%s] %s\n  %s\n\n",
			item.Snippet.TopLevelComment.ID, s.AuthorName, date, s.TextDisplay)
	}
	return nil
}
