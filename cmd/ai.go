package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var model string
var query string

var completeCmd = &cobra.Command{
	Use:   "complete",
	Short: "Use OpenAI completion AI",
	Long: `Sends a prompt to the OpenAI API and prints the response.
Requires --query. Optionally set --model to override the default
(gpt-4o-mini). Requires OPENAI_API_KEY environment variable.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if query == "" {
			return fmt.Errorf("--query is required")
		}

		openaiClient, err := loadOpenAIClient(model)
		if err != nil {
			return fmt.Errorf("loading openai client: %w", err)
		}

		messages, err := BuildMessagesYTBot(query, "Video without context")
		if err != nil {
			return fmt.Errorf("building messages: %w", err)
		}

		completion, err := openaiClient.Complete(cmd.Context(), messages)
		if err != nil {
			return fmt.Errorf("AI completion failed: %w", err)
		}

		fmt.Println(completion)
		return nil
	},
}

var aiCmd = &cobra.Command{
	Use:   "ai",
	Short: "Interact with LLMs (OpenAI)",
}

func init() {
	rootCmd.AddCommand(aiCmd)
	aiCmd.AddCommand(completeCmd)
	completeCmd.Flags().StringVar(&model, "model", "", "Name of the API model. Defaults to gpt-4o-mini")
	completeCmd.Flags().StringVar(&query, "query", "", "User prompt text")
}
