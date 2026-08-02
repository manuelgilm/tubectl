package cmd

import (
	"encoding/json"

	"github.com/spf13/cobra"
)

var getPromptArgs struct {
	name string
}

var promptCmd = &cobra.Command{
	Use:   "prompt",
	Short: "Commands to communicate with the prompt registry",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Println("prompt called")
	},
}
var listPromptsCmd = &cobra.Command{
	Use:   "list",
	Short: "List available prompts",
	Long:  `It retrieves a list of prompts from the registry`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := loadMlflowClient()
		if err != nil {
			return err
		}
		registeredPrompts, err := client.ListPrompts(cmd.Context())
		if err != nil {
			return err
		}

		bytes, err := json.MarshalIndent(registeredPrompts, "", "  ")
		if err != nil {
			return err
		}
		cmd.Println(string(bytes))
		return nil
	},
}
var getPromptCmd = &cobra.Command{
	Use:   "get",
	Short: "Get Prompt from the Registry",
	Long: `It retrieves a prompt from the prompt registry. 
	The prompt registry is implemented with a remote mlflow
	server
	`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := loadMlflowClient()
		if err != nil {
			return err
		}
		registeredPrompt, err := client.GetPrompt(cmd.Context(), getPromptArgs.name)
		if err != nil {
			return err
		}
		bytes, err := json.MarshalIndent(registeredPrompt, "", "  ")
		if err != nil {
			return err
		}
		cmd.Println(string(bytes))

		return nil
	},
}

func init() {
	rootCmd.AddCommand(promptCmd)
	promptCmd.AddCommand(getPromptCmd)
	promptCmd.AddCommand(listPromptsCmd)
	getPromptCmd.Flags().StringVar(&getPromptArgs.name, "name", "", "Prompt Name")
	getPromptCmd.MarkFlagRequired("name")
}
