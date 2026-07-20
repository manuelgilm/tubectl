package cmd

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"github.com/spf13/cobra"
	"tubectl/internal/ai"
	"tubectl/internal/prompt"
	"tubectl/internal/youtube"
)

var answerCommentArgs struct {
	videoID    string
	commentID  string
	autoApprove bool
	onlyPrint   bool
	promptFile  string
}

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
		client, err := loadClient(cmd.Context())
		if err != nil {
			return fmt.Errorf("loading youtube client: %w", err)
		}

		comment, err := client.GetComment(cmd.Context(), answerCommentArgs.commentID)
		if err != nil {
			return fmt.Errorf("getting comment: %w", err)
		}
		commentText := comment.Snippet.TextDisplay

		transcript, err := youtube.LoadCachedTranscript(answerCommentArgs.videoID)
		if err != nil {
			return fmt.Errorf("loading cached transcript: %w", err)
		}
		if transcript == nil {
			t, err := client.DownloadTranscript(cmd.Context(), answerCommentArgs.videoID, "")
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Warning: transcript not available: %v\n", err)
			} else {
				transcript = t
				if saveErr := youtube.SaveCachedTranscript(transcript); saveErr != nil {
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

		var resolvedTemplate string
		if answerCommentArgs.promptFile != "" {
			pf, err := prompt.LoadPromptFile(answerCommentArgs.promptFile)
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
			resolvedTemplate = rendered
		} else {
			resolvedTemplate, err = resolveBotPrompt(cmd.Context(), cmd.ErrOrStderr(), commentText, transcriptText)
			if err != nil {
				return fmt.Errorf("resolving prompt: %w", err)
			}
		}

		messages := []ai.Message{
			{Role: "system", Content: resolvedTemplate},
		}

		aiClient, err := loadOpenAIClient(cmd.Context(), "")
		if err != nil {
			return fmt.Errorf("loading AI client: %w", err)
		}
		reply, err := aiClient.Complete(cmd.Context(), messages)
		if err != nil {
			return fmt.Errorf("AI completion failed: %w", err)
		}

		if answerCommentArgs.onlyPrint {
			fmt.Println(reply)
			return nil
		}

		fmt.Fprintf(cmd.ErrOrStderr(), "Generated reply:\n---\n%s\n---\n", reply)

		if !answerCommentArgs.autoApprove {
			fmt.Fprint(cmd.ErrOrStderr(), "Post this reply? [y/N]: ")
			var confirm string
			if _, err := fmt.Scanln(&confirm); err != nil {
				return fmt.Errorf("reading confirmation (use --auto-approve in non-interactive mode): %w", err)
			}
			if confirm != "y" && confirm != "Y" {
				fmt.Println("Reply cancelled.")
				return nil
			}
		}

		posted, err := client.ReplyToComment(cmd.Context(), answerCommentArgs.commentID, reply)
		if err != nil {
			return fmt.Errorf("posting reply: %w", err)
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "Reply posted (ID: %s)\n", posted.ID)
		fmt.Println(reply)
		return nil
	},
}

func resolveBotPrompt(ctx context.Context, w io.Writer, commentText, transcriptText string) (string, error) {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(w, "Warning: could not load config: %v\n", err)
		return tryLocalPrompt(Config{}, commentText, transcriptText)
	}

	modelName := cfg.BotPrompt.AnswerCommentModel
	if modelName == "" {
		return tryLocalPrompt(cfg, commentText, transcriptText)
	}

	mlflowClient, err := loadMlflowClient()
	if err == nil {
		registered, err := mlflowClient.GetPrompt(ctx, modelName)
		if err == nil {
			template := registered.PromptText()
			if template != "" {
				rendered := renderTemplate(template, commentText, transcriptText)
				return rendered, nil
			}
		}
	}

	return tryLocalPrompt(cfg, commentText, transcriptText)
}

func tryLocalPrompt(cfg Config, commentText, transcriptText string) (string, error) {
	home, err := TubeCtlHome()
	if err != nil {
		return prompt.DefaultBotPromptText(commentText, transcriptText), nil
	}

	modelName := cfg.BotPrompt.AnswerCommentModel
	if modelName == "" {
		modelName = "yt-bot-answer-comment"
	}

	pf, err := prompt.LoadPromptFile(filepath.Join(home, "prompts", modelName+".yaml"))
	if err != nil {
		return prompt.DefaultBotPromptText(commentText, transcriptText), nil
	}

	rendered, err := pf.Render(map[string]string{
		"comment":    commentText,
		"transcript": transcriptText,
	})
	if err != nil {
		return prompt.DefaultBotPromptText(commentText, transcriptText), nil
	}

	return rendered, nil
}

func renderTemplate(template, commentText, transcriptText string) string {
	result := strings.ReplaceAll(template, "{comment}", commentText)
	result = strings.ReplaceAll(result, "{transcript}", transcriptText)
	return result
}


func init() {
	rootCmd.AddCommand(botCmd)
	botCmd.AddCommand(answerCommentCmd)

	answerCommentCmd.Flags().StringVar(&answerCommentArgs.videoID, "video-id", "", "YouTube Video ID")
	answerCommentCmd.MarkFlagRequired("video-id")
	answerCommentCmd.Flags().StringVar(&answerCommentArgs.commentID, "comment-id", "", "ID of the comment to answer")
	answerCommentCmd.MarkFlagRequired("comment-id")
	answerCommentCmd.Flags().BoolVar(&answerCommentArgs.autoApprove, "auto-approve", false, "Skip confirmation prompt and post directly")
	answerCommentCmd.Flags().BoolVar(&answerCommentArgs.onlyPrint, "only-print", false, "Generate the reply but do not post it")
	answerCommentCmd.Flags().StringVar(&answerCommentArgs.promptFile, "prompt-file", "", "Path to a YAML prompt file (alternative to the default prompt)")
}
