package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"encoding/json"
)

var (
	getCommentArgs struct {
		commentID string
	}
	replyToCommentArgs struct {
		commentID string
		text      string
	}
	deleteCommentArgs struct {
		commentID string
	}
)
var replyToCommentCmd = &cobra.Command{
	Use: "reply",
	Short: "Reply to a given comment using the YouTube Data API",
	Long: `Replies to an existing comment thread. Requires --comment-id
(the parent comment ID) and --text (the reply content).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := loadClient(cmd.Context())
		if err != nil {
			return fmt.Errorf("loading client %w ", err)
		}
		comment, err := client.ReplyToComment(cmd.Context(), replyToCommentArgs.commentID, replyToCommentArgs.text)
		if err != nil {
			return fmt.Errorf("replying to comment %s : %w", replyToCommentArgs.commentID, err)
		}
		fmt.Printf("comment: %s posted \n", comment.Snippet.TextDisplay)

		return nil
	},
}

var deleteCommentCmd = &cobra.Command{
	Use: "delete",
	Short: "Delete a comment",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := loadClient(cmd.Context())
		if err != nil {
			return fmt.Errorf("loading client %w ", err)
		}
		err = client.DeleteComment(cmd.Context(), deleteCommentArgs.commentID)
		if err != nil {
			return fmt.Errorf("deleting comment %s : %w ", deleteCommentArgs.commentID, err)
		}
		fmt.Printf("comment %s deleted! \n ", deleteCommentArgs.commentID)
		return nil
	},
}

var getCommentCmd = &cobra.Command{
	Use: "get",
	Short: "Get a comment by ID",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := loadClient(cmd.Context())
		if err != nil {
			return fmt.Errorf("loading client: %w ", err)
		}
		comment, err := client.GetComment(cmd.Context(), getCommentArgs.commentID)
		if err != nil {
			return fmt.Errorf("getting comment %s : %w ", getCommentArgs.commentID, err)
		}
		
		data, err := json.MarshalIndent(comment.Snippet.TextDisplay, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling comment %w ", err)
		}
		fmt.Println(string(data))
		return nil
	},
}
var commentCmd = &cobra.Command{
	Use:   "comment",
	Short: "YouTube comment operations",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("comment called")
	},
}

func init() {
	rootCmd.AddCommand(commentCmd)
	commentCmd.AddCommand(getCommentCmd)
	getCommentCmd.Flags().StringVar(&getCommentArgs.commentID, "comment-id", "", "Id of the comment")
	getCommentCmd.MarkFlagRequired("comment-id")

	commentCmd.AddCommand(replyToCommentCmd)
	replyToCommentCmd.Flags().StringVar(&replyToCommentArgs.commentID, "comment-id", "", "Id of the Parent comment")
	replyToCommentCmd.MarkFlagRequired("comment-id")
	replyToCommentCmd.Flags().StringVar(&replyToCommentArgs.text, "text", "", "text to use as reply")
	replyToCommentCmd.MarkFlagRequired("text")

	commentCmd.AddCommand(deleteCommentCmd)
	deleteCommentCmd.Flags().StringVar(&deleteCommentArgs.commentID, "comment-id", "", "Id of the Parent comment")
	deleteCommentCmd.MarkFlagRequired("comment-id")

}
