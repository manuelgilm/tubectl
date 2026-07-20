package cmd

import (
	// "context"
	"fmt"
	// "io"
	// "path/filepath"
	// "strings"
	"github.com/spf13/cobra"
	// "tubectl/internal/ai"
	// "tubectl/internal/prompt"
	// "tubectl/internal/youtube"
)

var answerCommentArgs struct {
	videoID     string
	commentID   string
	autoApprove bool
	onlyPrint   bool
	promptFile  string
	promptName  string
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
		transcript, err := GetTranscriptText(cmd, answerCommentArgs.videoID)
		if err != nil {
			return err
		}
		commentText, err := ResolveComment(cmd, answerCommentArgs.commentID)
		if err != nil {
			return err
		}
		resolvedTemplate, err := ResolvePromptTemplate(cmd, answerCommentArgs.promptName, answerCommentArgs.promptFile, commentText, transcript)
		if err != nil {
			return err
		}
		reply, err := GenerateAnswer(cmd, resolvedTemplate)
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
}
