package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"encoding/json"
)
var (
	getVideoArgs struct {
		videoID string
	}
	getCommentsArgs struct {
		videoID    string
		maxResults int
		order      string
	}
	getTranscriptArgs struct {
		videoID  string
		noCache  bool
		language string
	}
	postCommentArgs struct {
		videoID string
		text    string
	}
)
var postCommentCmd = &cobra.Command{
	Use: "comment",
	Short: "Comment a video given its id",
	Long: `Posts a top-level comment on a YouTube video.
Requires --video-id and --text.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := loadClient(cmd.Context())
		if err != nil {
			return fmt.Errorf("loading client %w ",err)
		}
		err = client.PostComment(cmd.Context(), postCommentArgs.videoID, postCommentArgs.text)
		if err != nil {
			return fmt.Errorf("posting a comment %w ", err)
		}
		fmt.Println("Comment Posted!")
		return nil
	},
}
var getTranscriptCmd = &cobra.Command{
	Use: "get-transcript",
	Short: "Get video transcript",
	Long: `Downloads a video transcript from YouTube and prints it with
timestamps. Results are cached locally for faster subsequent access.

Use --language to select a specific language (default: en).
Use --no-cache to bypass the cache and fetch fresh data.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !getTranscriptArgs.noCache {
			cached, err := LoadCachedTranscript(getTranscriptArgs.videoID)
			if err != nil {
				return err 
			}
			if cached != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Using Cached Transcript (language: %s, kind: %s)\n\n",
				cached.Language, cached.TrackKind)
				printTranscript(cached) 
				return nil
			}
		}
		client, err := loadClient(cmd.Context())
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.ErrOrStderr(), "Fetching transcript from YouTube API...")
		transcript, err := client.DownloadTranscript(cmd.Context(), getTranscriptArgs.videoID, getTranscriptArgs.language)
		if err != nil {
			return err
		}

		if err := SaveCachedTranscript(transcript); err != nil {
			// Non-fatal: warn but still print the transcript.
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not cache transcript: %v\n", err)
		} else {
			path, _ := TranscriptCachePath(getTranscriptArgs.videoID)
			fmt.Fprintf(cmd.ErrOrStderr(), "Transcript cached to %s\n\n", path)
		}

		printTranscript(transcript)
		return nil

	},
}
var getCommentsCmd = &cobra.Command{
	Use: "comments",
	Short: "List comments for a video",
	RunE: func(cmd *cobra.Command, args []string) error {

		client, err := loadClient(cmd.Context())
		if err != nil {
			return err
		}

		commentThread, err := client.GetComments(cmd.Context(), getCommentsArgs.videoID, getCommentsArgs.maxResults, getCommentsArgs.order)
		if err != nil {
			return fmt.Errorf("Retrieving comments: %w ", err)
		}

		data, err := json.MarshalIndent(commentThread, "", "  ")
		if err != nil {
			return fmt.Errorf("Marshall video %w ", err)
		}

		fmt.Println(string(data))
		return nil
	},

}
var getVideoCmd = &cobra.Command{
	Use:	"get",
	Short:	"Get video details by ID",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := loadClient(cmd.Context())
		if err != nil {
			return err
		}
		video, err := client.GetVideo(cmd.Context(), getVideoArgs.videoID)
		if err != nil {
			return fmt.Errorf("Retrieving video: %w ", err)
		}
		data, err := json.MarshalIndent(video, "", "  ")
		if err != nil {
			return fmt.Errorf("Marshall video %v ", err)
		}
		fmt.Println(string(data))
		return nil
	},
}
var videoCmd = &cobra.Command{
	Use:   "video",
	Short: "YouTube video operations",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("video called")
	},
}

func init() {
	rootCmd.AddCommand(videoCmd)
	videoCmd.AddCommand(getVideoCmd)
	getVideoCmd.Flags().StringVar(&getVideoArgs.videoID, "video-id", "", "YouTube Video ID")
	getVideoCmd.MarkFlagRequired("video-id")

	videoCmd.AddCommand(getCommentsCmd)
	getCommentsCmd.Flags().StringVar(&getCommentsArgs.videoID, "video-id", "", "YouTube Video ID")
	getCommentsCmd.MarkFlagRequired("video-id")
	getCommentsCmd.Flags().IntVar(&getCommentsArgs.maxResults, "max-results", 20, "Max numbers of comments")
	getCommentsCmd.Flags().StringVar(&getCommentsArgs.order, "order", "time", "Comments order")

	videoCmd.AddCommand(getTranscriptCmd)
	getTranscriptCmd.Flags().StringVar(&getTranscriptArgs.videoID, "video-id", "", "YouTube Video ID")
	getTranscriptCmd.MarkFlagRequired("video-id")
	getTranscriptCmd.Flags().StringVar(&getTranscriptArgs.language, "language", "en", "Preferred caption language (e.g. en, es). Defaults to first available.")
	getTranscriptCmd.Flags().BoolVar(&getTranscriptArgs.noCache, "no-cache", false, "Skip the local cache and always fetch from the API")

	videoCmd.AddCommand(postCommentCmd)
	postCommentCmd.Flags().StringVar(&postCommentArgs.videoID, "video-id", "", "YouTube Video ID")
	postCommentCmd.MarkFlagRequired("video-id")
	postCommentCmd.Flags().StringVar(&postCommentArgs.text, "text", "", "Comment content")
	postCommentCmd.MarkFlagRequired("text")
}
