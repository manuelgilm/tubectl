package cmd

import (
	"fmt"
	"github.com/manuelgilm/tubectl/internal/prompting"
	"github.com/spf13/cobra"
)

var completeArgs struct {
	model string
	query string
}

var completeCmd = &cobra.Command{
	Use:   "complete",
	Short: "Use OpenAI completion AI",
	Long: `Sends a prompt to the OpenAI-compatible API and prints the response.
Requires --query. When MLflow credentials are configured, the request is
sent to the MLflow gateway and --model names the gateway endpoint;
otherwise it falls back to OpenAI (--model is an OpenAI model, default
gpt-4o-mini, requiring OPENAI_API_KEY).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if completeArgs.query == "" {
			return fmt.Errorf("--query is required")
		}

		openaiClient, err := loadOpenAIClient(cmd.Context(), completeArgs.model)
		if err != nil {
			return fmt.Errorf("loading openai client: %w", err)
		}

		messages, err := prompting.BuildMessagesYTBot(completeArgs.query, "")
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
	completeCmd.Flags().StringVar(&completeArgs.model, "model", "", "Gateway endpoint name (with MLflow creds) or OpenAI model. Defaults to gpt-4o-mini")
	completeCmd.Flags().StringVar(&completeArgs.query, "query", "", "User prompt text")
	completeCmd.MarkFlagRequired("query")
}
