/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"encoding/json"
)

var commentID string
var text	string

var replyToCommentCmd = &cobra.Command{
	Use: "reply",
	Short: "Reply to a given comment using the YouTube Data API",
	Long: `Replies to an existing comment thread. Requires --comment-id
(the parent comment ID) and --text (the reply content).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := loadClient()
		if err != nil {
			return fmt.Errorf("loading client %v ", err)
		}
		comment, err := client.ReplyToComment(cmd.Context(), commentID, text)
		if err != nil {
			return fmt.Errorf("replying to comment %s : %w", commentID, err)
		}
		fmt.Printf("comment: %s posted \n", comment.Snippet.TextDisplay)

		return nil
	},
}

var deleteCommentCmd = &cobra.Command{
	Use: "delete",
	Short: "Delete a comment",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := loadClient()
		if err != nil {
			return fmt.Errorf("loading client %v ", err)
		}
		err = client.DeleteComment(cmd.Context(), commentID)
		if err != nil {
			return fmt.Errorf("deleting comment %s : %w ", commentID, err)
		}
		fmt.Printf("comment %s deleted! \n ", commentID)
		return nil
	},
}

var getCommentCmd = &cobra.Command{
	Use: "get",
	Short: "Get a comment by ID",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := loadClient()
		if err != nil {
			return fmt.Errorf("loading client: %v ", err)
		}
		comment, err := client.GetComment(cmd.Context(), commentID)
		if err != nil {
			return fmt.Errorf("getting comment %s : %w ", commentID, err)
		}
		
		data, err := json.MarshalIndent(comment.Snippet.TextDisplay, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling comment %v ", err)
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
	getCommentCmd.Flags().StringVar(&commentID, "comment-id", "", "Id of the comment")

	commentCmd.AddCommand(replyToCommentCmd)
	replyToCommentCmd.Flags().StringVar(&commentID, "comment-id", "", "Id of the Parent comment")
	replyToCommentCmd.Flags().StringVar(&text, "text", "", "text to use as reply")

	commentCmd.AddCommand(deleteCommentCmd)
	deleteCommentCmd.Flags().StringVar(&commentID, "comment-id", "", "Id of the Parent comment")

}
