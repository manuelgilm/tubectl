package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/manuelgilm/tubectl/internal/storage"
	"github.com/manuelgilm/tubectl/internal/youtube"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"strings"
	"time"
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
		file     bool
		language string
	}
	postCommentArgs struct {
		videoID     string
		text        string
		autoApprove bool
	}
)
var postCommentCmd = &cobra.Command{
	Use:   "comment",
	Short: "Post a top-level comment on a video",
	Long: `Posts a top-level comment on a YouTube video.
Requires --video-id and --text.

By default a confirmation prompt is shown before posting.
Use --auto-approve to skip the prompt.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := confirmAction(cmd, fmt.Sprintf("Post comment %q on video %s? [y/N]: ", postCommentArgs.text, postCommentArgs.videoID), postCommentArgs.autoApprove); err != nil {
			return err
		}

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
	Use:   "get-transcript",
	Short: "Get video transcript",
	Long: `Downloads a video transcript from YouTube and prints it with
timestamps. Results are cached locally for faster subsequent access.

Use --language to select a specific language (default: en).
Use --no-cache to bypass the cache and fetch fresh data.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if getTranscriptArgs.file {
			path := transcriptFilePath(getTranscriptArgs.videoID)
			if !getTranscriptArgs.noCache {
				if text, err := readTranscriptFile(path); err == nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "Using Cached Transcript (file: %s)\n\n", path)
					fmt.Print(text)
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
			if err := writeTranscriptFile(path, transcript); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not write transcript file: %v\n", err)
			} else {
				fmt.Fprintf(cmd.ErrOrStderr(), "Transcript saved to %s\n\n", path)
			}
			printTranscript(transcript)
			return nil
		}

		db, err := openDB()
		if err != nil {
			return err
		}
		defer db.Close()
		trepo := storage.NewTranscriptRepo(db)

		if !getTranscriptArgs.noCache {
			st, err := trepo.Load(cmd.Context(), getTranscriptArgs.videoID, getTranscriptArgs.language)
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
	Use:   "comments",
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
	Use:   "get",
	Short: "Get video details by ID",
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
	getTranscriptCmd.Flags().StringVar(&getTranscriptArgs.language, "language", "en", "Preferred caption language (e.g. en, es). Defaults to 'en'.")
	getTranscriptCmd.Flags().BoolVar(&getTranscriptArgs.noCache, "no-cache", false, "Skip the local cache and always fetch from the API")
	getTranscriptCmd.Flags().BoolVar(&getTranscriptArgs.file, "file", false, "Use the repo transcript file transcripts/<video-id>.txt instead of the local database (repository as database)")

	videoCmd.AddCommand(postCommentCmd)
	postCommentCmd.Flags().StringVar(&postCommentArgs.videoID, "video-id", "", "YouTube Video ID")
	postCommentCmd.MarkFlagRequired("video-id")
	postCommentCmd.Flags().StringVar(&postCommentArgs.text, "text", "", "Comment content")
	postCommentCmd.MarkFlagRequired("text")
	postCommentCmd.Flags().BoolVar(&postCommentArgs.autoApprove, "auto-approve", false, "Skip confirmation prompt and post directly")
}

func printTranscript(t *youtube.Transcript) {
	for _, line := range formatTranscriptLines(t) {
		fmt.Println(line)
	}
}

// formatTranscriptLines renders a transcript as "[MM:SS] text" lines, used both
// for printing and for the repo transcript files.
func formatTranscriptLines(t *youtube.Transcript) []string {
	lines := make([]string, 0, len(t.Lines))
	for _, line := range t.Lines {
		minutes := int(line.Start) / 60
		seconds := int(line.Start) % 60
		lines = append(lines, fmt.Sprintf("[%02d:%02d] %s", minutes, seconds, line.Text))
	}
	return lines
}

// transcriptFilePath returns the repo transcript file path for a video, e.g.
// transcripts/<video-id>.txt. The directory can be overridden via the
// TUBECTL_TRANSCRIPT_DIR environment variable.
func transcriptFilePath(videoID string) string {
	dir := os.Getenv("TUBECTL_TRANSCRIPT_DIR")
	if dir == "" {
		dir = "transcripts"
	}
	return filepath.Join(dir, videoID+".txt")
}

// writeTranscriptFile writes a transcript as timestamped lines to the repo
// transcript file (used by the CI workflow to keep transcripts in git).
func writeTranscriptFile(path string, t *youtube.Transcript) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating transcript directory: %w", err)
	}
	lines := formatTranscriptLines(t)
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		return fmt.Errorf("writing transcript file: %w", err)
	}
	return nil
}

func readTranscriptFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// stripTranscriptTimestamps removes the "[MM:SS] " prefix from each line of a
// timestamped transcript file, producing the plain spoken text used in prompts.
func stripTranscriptTimestamps(text string) string {
	var b strings.Builder
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if trimmed[0] == '[' {
			if i := strings.IndexByte(trimmed, ']'); i > 0 && i <= 8 {
				trimmed = strings.TrimSpace(trimmed[i+1:])
			}
		}
		b.WriteString(trimmed)
		b.WriteString(" ")
	}
	return strings.TrimSpace(b.String())
}

func transcriptToStored(t *youtube.Transcript) (*storage.StoredTranscript, error) {
	linesJSON, err := json.Marshal(t.Lines)
	if err != nil {
		return nil, fmt.Errorf("marshal transcript lines: %w", err)
	}
	var content strings.Builder
	for _, l := range t.Lines {
		content.WriteString(l.Text)
		content.WriteString(" ")
	}
	return &storage.StoredTranscript{
		VideoID:   t.VideoID,
		Language:  t.Language,
		TrackKind: t.TrackKind,
		CaptionID: t.CaptionID,
		Content:   strings.TrimSpace(content.String()),
		Lines:     string(linesJSON),
		CachedAt:  time.Now(),
	}, nil
}

func storedToTranscript(st *storage.StoredTranscript) (*youtube.Transcript, error) {
	var lines []youtube.TranscriptLine
	if err := json.Unmarshal([]byte(st.Lines), &lines); err != nil {
		return nil, fmt.Errorf("unmarshal transcript lines: %w", err)
	}
	return &youtube.Transcript{
		VideoID:   st.VideoID,
		Language:  st.Language,
		TrackKind: st.TrackKind,
		CaptionID: st.CaptionID,
		Lines:     lines,
	}, nil
}

func GetTranscriptText(cmd *cobra.Command, videoID, language string) (string, error) {
	db, err := openDB()
	if err != nil {
		return "", err
	}
	defer db.Close()

	// 1) Local SQLite cache is the source of truth when working locally.
	repo := storage.NewTranscriptRepo(db)
	st, err := repo.Load(cmd.Context(), videoID, language)
	if err != nil {
		return "", fmt.Errorf("loading cached transcript: %w", err)
	}
	if st != nil {
		return st.Content, nil
	}

	// 2) Repo transcript file (e.g. transcripts/<video-id>.txt). Used by the CI
	// workflow, where the local database is empty on every run and transcripts
	// are committed to git instead.
	if text, err := readTranscriptFile(transcriptFilePath(videoID)); err == nil {
		return stripTranscriptTimestamps(text), nil
	}

	// 3) Live fetch, cached to the local database for later use.
	client, err := loadClient(cmd.Context())
	if err != nil {
		return "", err
	}
	t, err := client.DownloadTranscript(cmd.Context(), videoID, language)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Warning: transcript not available: %v\n", err)
		return "", nil
	}
	st, err = transcriptToStored(t)
	if err != nil {
		return "", err
	}
	if saveErr := repo.Save(cmd.Context(), st); saveErr != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not cache transcript: %v\n", saveErr)
	}
	return st.Content, nil
}
