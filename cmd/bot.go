package cmd

import (
	"fmt"
	"strings"
	"github.com/spf13/cobra"
	"tubectl/internal/ai"
)

var autoApprove bool
var onlyPrint bool
var promptFile string

// botCmd represents the bot command
var botCmd = &cobra.Command{
	Use:   "bot",
	Short: "YouTube automations powered by AI",
}

var answerCommentCmd = &cobra.Command{
	Use:   "answer-comment",
	Short: "Generates an AI reply to a comment and optionally posts it",
	Long: `Generates an AI reply to a YouTube comment using the video transcript
as context, then optionally posts the reply.

By default the generated reply is shown and the user is prompted for
confirmation before posting. Use --auto-approve to skip the prompt
or --only-print to just display the reply without posting.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := loadClient()
		if err != nil {
			return fmt.Errorf("loading youtube client: %w", err)
		}

		comment, err := client.GetComment(cmd.Context(), commentID)
		if err != nil {
			return fmt.Errorf("getting comment: %w", err)
		}
		commentText := comment.Snippet.TextDisplay

		transcript, err := LoadCachedTranscript(videoID)
		if err != nil {
			return fmt.Errorf("loading cached transcript: %w", err)
		}
		if transcript == nil {
			t, err := client.DownloadTranscript(cmd.Context(), videoID, "")
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Warning: transcript not available: %v\n", err)
			} else {
				transcript = t
				if saveErr := SaveCachedTranscript(transcript); saveErr != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not cache transcript: %v\n", saveErr)
				}
			}
		}

		var transcriptText string
		if transcript != nil {
			var b strings.Builder
			for _, line := range transcript.Lines {
				b.WriteString(line.Text)
				b.WriteString(" ")
			}
			transcriptText = b.String()
		}

		var messages []ai.Message
		if promptFile != "" {
			pf, err := LoadPromptFile(promptFile)
			if err != nil {
				return fmt.Errorf("loading prompt file: %w", err)
			}
			rendered, err := pf.Render(map[string]string{
				"comment":    commentText,
				"transcript": transcriptText,
			})
			if err != nil {
				return fmt.Errorf("rendering prompt: %w", err)
			}
			messages = []ai.Message{
				{Role: "user", Content: rendered},
			}
		} else {
			var err error
			messages, err = BuildMessagesYTBot(commentText, transcriptText)
			if err != nil {
				return fmt.Errorf("building messages: %w", err)
			}
		}

		aiClient, err := loadOpenAIClient("")
		if err != nil {
			return fmt.Errorf("loading AI client: %w", err)
		}
		reply, err := aiClient.Complete(cmd.Context(), messages)
		if err != nil {
			return fmt.Errorf("AI completion failed: %w", err)
		}

		if onlyPrint {
			fmt.Println(reply)
			return nil
		}

		fmt.Fprintf(cmd.ErrOrStderr(), "Generated reply:\n---\n%s\n---\n", reply)

		if !autoApprove {
			fmt.Fprint(cmd.ErrOrStderr(), "Post this reply? [y/N]: ")
			var confirm string
			fmt.Scanln(&confirm)
			if confirm != "y" && confirm != "Y" {
				fmt.Println("Reply cancelled.")
				return nil
			}
		}

		posted, err := client.ReplyToComment(cmd.Context(), commentID, reply)
		if err != nil {
			return fmt.Errorf("posting reply: %w", err)
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "Reply posted (ID: %s)\n", posted.ID)
		fmt.Println(reply)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(botCmd)
	botCmd.AddCommand(answerCommentCmd)

	answerCommentCmd.Flags().StringVar(&videoID, "video-id", "", "YouTube Video ID")
	answerCommentCmd.Flags().StringVar(&commentID, "comment-id", "", "ID of the comment to answer")
	answerCommentCmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "Skip confirmation prompt and post directly")
	answerCommentCmd.Flags().BoolVar(&onlyPrint, "only-print", false, "Generate the reply but do not post it")
	answerCommentCmd.Flags().StringVar(&promptFile, "prompt-file", "", "Path to a YAML prompt file (alternative to the default prompt)")
}
