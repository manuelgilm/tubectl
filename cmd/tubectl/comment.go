package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"tubectl/internal/openai"
)

var commentID string

var commentCmd = &cobra.Command{
	Use:   "comment",
	Short: "Interact with YouTube comments",
}

var getContentCmd = &cobra.Command{
	Use:   "get-content",
	Short: "Display the content of a comment",
	Example: `  tubectl comment --comment-id Ugxyz123 get-content`,
	RunE:  runGetContent,
}

var replyCmd = &cobra.Command{
	Use:   "reply <text>",
	Short: "Post a reply to a comment",
	Long: `Post a reply to a comment thread. The reply text is taken from
the positional arguments so quoting is optional for single-word replies.`,
	Example: `  tubectl comment --comment-id Ugxyz123 reply "Great point!"
  tubectl comment --comment-id Ugxyz123 reply Great point!`,
	Args: cobra.MinimumNArgs(1),
	RunE: runReply,
}

var suggestReplyCmd = &cobra.Command{
	Use:   "suggest-reply",
	Short: "Use AI to suggest a reply based on the video transcript",
	Long: `Fetches the comment, loads the cached transcript for the given video,
and asks OpenAI to draft a reply. You can then post it or discard it.`,
	Example: `  tubectl comment --comment-id Ugxyz123 suggest-reply --video-id dQw4w9WgXcQ`,
	RunE: runSuggestReply,
}

var (
	suggestVideoID  string
	suggestPrintOnly   bool
	suggestAutoApprove bool
)

func init() {
	commentCmd.PersistentFlags().StringVar(&commentID, "comment-id", "", "YouTube comment ID (required)")
	commentCmd.MarkPersistentFlagRequired("comment-id")

	suggestReplyCmd.Flags().StringVar(&suggestVideoID, "video-id", "", "Video ID to load the transcript from (required)")
	suggestReplyCmd.MarkFlagRequired("video-id")
	suggestReplyCmd.Flags().BoolVar(&suggestPrintOnly, "print-only", false, "Print the suggestion without prompting to post")
	suggestReplyCmd.Flags().BoolVar(&suggestAutoApprove, "auto-approve", false, "Post the suggestion automatically without prompting")

	commentCmd.AddCommand(getContentCmd)
	commentCmd.AddCommand(replyCmd)
	commentCmd.AddCommand(suggestReplyCmd)
}

func runSuggestReply(cmd *cobra.Command, args []string) error {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("OPENAI_API_KEY environment variable must be set")
	}

	// Load transcript from cache.
	transcript, err := loadCachedTranscript(suggestVideoID)
	if err != nil {
		return err
	}
	if transcript == nil {
		return fmt.Errorf("no cached transcript for video %s — run: tubectl video --video-id %s get-transcript",
			suggestVideoID, suggestVideoID)
	}

	// Fetch the comment.
	ytClient, err := loadClient()
	if err != nil {
		return err
	}
	comment, err := ytClient.GetComment(cmd.Context(), commentID)
	if err != nil {
		return err
	}
	commentText := comment.Snippet.TextDisplay

	// Build transcript context (cap at ~3000 words to stay within token limits).
	var sb strings.Builder
	words := 0
	for _, line := range transcript.Lines {
		sb.WriteString(line.Text)
		sb.WriteByte('\n')
		words += len(strings.Fields(line.Text))
		if words >= 3000 {
			break
		}
	}

	// Call OpenAI.
	ai := openai.NewClient(apiKey, "")
	suggestion, err := ai.Complete(cmd.Context(), []openai.Message{
		{
			Role: "system",
			Content: "You are a helpful YouTube content creator. " +
				"Use the provided video transcript as context to write a friendly, " +
				"concise reply to a viewer comment. Reply only with the reply text itself, no preamble.",
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("Video transcript:\n%s\n\nViewer comment:\n%s", sb.String(), commentText),
		},
	})
	if err != nil {
		return err
	}

	fmt.Printf("Comment:\n  %s\n\n", commentText)
	fmt.Printf("Suggested reply:\n  %s\n\n", suggestion)

	if suggestPrintOnly {
		return nil
	}

	if !suggestAutoApprove {
		fmt.Print("Post this reply? [y/N] ")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		if strings.ToLower(strings.TrimSpace(scanner.Text())) != "y" {
			fmt.Println("Discarded.")
			return nil
		}
	}

	posted, err := ytClient.ReplyToComment(cmd.Context(), commentID, suggestion)
	if err != nil {
		return err
	}
	fmt.Printf("Reply posted (id: %s)\n", posted.ID)
	return nil
}

func runGetContent(cmd *cobra.Command, args []string) error {
	client, err := loadClient()
	if err != nil {
		return err
	}

	comment, err := client.GetComment(cmd.Context(), commentID)
	if err != nil {
		return err
	}

	s := comment.Snippet
	published := s.PublishedAt
	if t, err := time.Parse(time.RFC3339, s.PublishedAt); err == nil {
		published = t.Format("2006-01-02 15:04:05 UTC")
	}

	fmt.Printf("Author:    %s\n", s.AuthorName)
	fmt.Printf("Published: %s\n", published)
	fmt.Printf("Likes:     %d\n", s.LikeCount)
	fmt.Printf("ID:        %s\n", comment.ID)
	fmt.Println()
	fmt.Println(s.TextDisplay)

	return nil
}

func runReply(cmd *cobra.Command, args []string) error {
	text := strings.Join(args, " ")

	client, err := loadClient()
	if err != nil {
		return err
	}

	reply, err := client.ReplyToComment(cmd.Context(), commentID, text)
	if err != nil {
		return err
	}

	fmt.Printf("Reply posted successfully (id: %s)\n", reply.ID)
	return nil
}
