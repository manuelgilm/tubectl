package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var answerCommentArgs struct {
	videoID            string
	commentID          string
	autoApprove        bool
	onlyPrint          bool
	promptFile         string
	promptName         string
	transcriptLanguage string
	model              string
}

// botCmd represents the bot command
var botCmd = &cobra.Command{
	Use:   "bot",
	Short: "YouTube automations powered by AI",
}

var answerCommentCmd = &cobra.Command{
	Use: "answer-comment",
	Short: "Generates an AI reply to a comment and optionally posts it",
	Long:	`Generates an AI reply to a YouTube comment using the video transcript
as context, then optionally posts the reply.

By default the generated reply is shown and the user is prompted for
confirmation before posting. Use --auto-approve to skip the prompt
or --only-print to just display the reply without posting.`,
	RunE: func(cmd *cobra.Command, args []string) error {
	var err error
	transcript, err := GetTranscriptText(cmd, answerCommentArgs.videoID, answerCommentArgs.transcriptLanguage)
	if err != nil {
		return err
	}
	if transcript == "" {
		fmt.Fprintln(cmd.ErrOrStderr(), "Warning: transcript not available, AI reply will have no video context")
	}
		commentText, err := ResolveComment(cmd, answerCommentArgs.commentID)
		if err != nil {
			return err
		}
		resolvedTemplate, err := ResolvePromptTemplate(cmd, answerCommentArgs.promptName, answerCommentArgs.promptFile, commentText, transcript)
		if err != nil {
			return err
		}
		reply, err := GenerateAnswer(cmd, resolvedTemplate, answerCommentArgs.model, map[string]string{
			"source":     "youtube-comment",
			"comment_id": answerCommentArgs.commentID,
			"video_id":   answerCommentArgs.videoID,
		})
		if err != nil {
			return err
		}

		if answerCommentArgs.onlyPrint {
			fmt.Println(reply)
			return nil
		}

		return replyComment(cmd, answerCommentArgs.commentID, reply, answerCommentArgs.autoApprove)		
	},
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
	answerCommentCmd.Flags().StringVar(&answerCommentArgs.promptName, "prompt-name", "", "Prompt name in MLflow (takes precedence over --prompt-file)")
	answerCommentCmd.Flags().StringVar(&answerCommentArgs.transcriptLanguage, "transcript-language", "en", "Language of the transcript (e.g. en, es)")
	answerCommentCmd.Flags().StringVar(&answerCommentArgs.model, "model", "", "OpenAI model name (default: gpt-4o-mini)")
}
