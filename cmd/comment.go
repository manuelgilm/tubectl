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
	Short: "Reply a given comment using the Youtube Data API",
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
// commentCmd represents the comment command
var commentCmd = &cobra.Command{
	Use:   "comment",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
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
