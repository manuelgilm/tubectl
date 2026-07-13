/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"encoding/json"
)
var maxResults int
var order string
var noCache bool
var languageFlag string

var postCommentCmd = &cobra.Command{
	Use: "comment",
	Short: "Comment a video given its id",
	Long: `Posts a top-level comment on a YouTube video.
Requires --video-id and --text.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := loadClient()
		if err != nil {
			return fmt.Errorf("loading client %v ",err)
		}
		err = client.PostComment(cmd.Context(), videoID, text)
		if err != nil {
			return fmt.Errorf("posting a comment %v ", err)
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
		if !noCache {
			cached, err := LoadCachedTranscript(videoID)
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
		client, err := loadClient()
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.ErrOrStderr(), "Fetching transcript from YouTube API...")
		transcript, err := client.DownloadTranscript(cmd.Context(), videoID, languageFlag)
		if err != nil {
			return err
		}

		if err := SaveCachedTranscript(transcript); err != nil {
			// Non-fatal: warn but still print the transcript.
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not cache transcript: %v\n", err)
		} else {
			path, _ := TranscriptCachePath(videoID)
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

		client, err := loadClient()
		if err != nil {
			return err
		}

		commentThread, err := client.GetComments(cmd.Context(), videoID, maxResults, order)
		if err != nil {
			return fmt.Errorf("Retrieving comments: %v ", err)
		}

		data, err := json.MarshalIndent(commentThread, "", "  ")
		if err != nil {
			return fmt.Errorf("Marshall video %v ", err)
		}

		fmt.Println(string(data))
		return nil
	},

}
var getVideoCmd = &cobra.Command{
	Use:	"get",
	Short:	"Get video details by ID",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := loadClient()
		if err != nil {
			return err
		}
		video, err := client.GetVideo(cmd.Context(), videoID)
		if err != nil {
			return fmt.Errorf("Retrieving video: %v ", err)
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
	getVideoCmd.Flags().StringVar(&videoID, "video-id", "", "YouTube Video ID")

	videoCmd.AddCommand(getCommentsCmd)
	getCommentsCmd.Flags().StringVar(&videoID, "video-id", "", "YouTube Video ID")
	getCommentsCmd.Flags().IntVar(&maxResults, "max-results", 20, "Max numbers of comments")
	getCommentsCmd.Flags().StringVar(&order, "order", "time", "Comments order")

	videoCmd.AddCommand(getTranscriptCmd)
	getTranscriptCmd.Flags().StringVar(&videoID, "video-id", "", "YouTube Video ID")
	getTranscriptCmd.Flags().StringVar(&languageFlag, "language", "en", "Preferred caption language (e.g. en, es). Defaults to first available.")
	getTranscriptCmd.Flags().BoolVar(&noCache, "no-cache", false, "Skip the local cache and always fetch from the API")

	videoCmd.AddCommand(postCommentCmd)
	postCommentCmd.Flags().StringVar(&videoID, "video-id", "", "YouTube Video ID")
	postCommentCmd.Flags().StringVar(&text, "text", "", "Comment content")
}
