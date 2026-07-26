package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/manuelgilm/tubectl/internal/storage"
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
			return fmt.Errorf("loading client: %w", err)
		}
		err = client.PostComment(cmd.Context(), postCommentArgs.videoID, postCommentArgs.text)
		if err != nil {
			return fmt.Errorf("posting a comment: %w", err)
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
		db, err := openDB()
		if err != nil {
			return err
		}
		defer db.Close()
		trepo := storage.NewTranscriptRepo(db)

		if !getTranscriptArgs.noCache {
			st, err := trepo.Load(cmd.Context(), getTranscriptArgs.videoID)
			if err != nil {
				return err
			}
			if st != nil {
				t, err := storedToTranscript(st)
				if err != nil {
					return err
				}
				fmt.Fprintf(cmd.ErrOrStderr(), "Using Cached Transcript (language: %s, kind: %s)\n\n",
					t.Language, t.TrackKind)
				printTranscript(t)
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

		st, err := transcriptToStored(transcript)
		if err != nil {
			return err
		}
		if err := trepo.Save(cmd.Context(), st); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not cache transcript: %v\n", err)
		} else {
			fmt.Fprintf(cmd.ErrOrStderr(), "Transcript cached to local database\n\n")
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
			return fmt.Errorf("retrieving comments: %w", err)
		}

		data, err := json.MarshalIndent(commentThread, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling comments: %w", err)
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
			return fmt.Errorf("retrieving video: %w", err)
		}
		data, err := json.MarshalIndent(video, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling video: %w", err)
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
