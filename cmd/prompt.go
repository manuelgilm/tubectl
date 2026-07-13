/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"encoding/json"
	"tubectl/internal/prompt"
	"github.com/spf13/cobra"
	"os"
)

var promptName string

// promptCmd represents the prompt command
var promptCmd = &cobra.Command{
	Use:   "prompt",
	Short: "Commands to communicate with the prompt registry",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("prompt called")
	},
}
var listPromptsCmd = &cobra.Command{
	Use: "list",
	Short:"List available prompts",
	Long: `It retrieves a list of prompts from the registry`,
	RunE: func(cmd *cobra.Command, args []string) error {
		password := os.Getenv("MLFLOW_PASSWORD")
		username := os.Getenv("MLFLOW_USERNAME")
		client := prompt.NewClient(username, password)
		registeredPrompts, err := client.ListPrompts(cmd.Context())
		if err != nil {
			return err
		}

		bytes, err := json.MarshalIndent(registeredPrompts, "", "  ")
		if err != nil {
			return err 
		}
		fmt.Println(string(bytes))
		return nil
	},
} 
var getPromptCmd = &cobra.Command{
	Use: "get",
	Short: "Get Prompt from the Registry",
	Long: `It retrieves a prompt from the prompt registry. 
	The prompt registry is implemented with a remote mlflow
	server
	`,
	RunE: func(cmd *cobra.Command, args []string) error {
		password := os.Getenv("MLFLOW_PASSWORD")
		username := os.Getenv("MLFLOW_USERNAME")
		client := prompt.NewClient(username, password)

		registeredPrompt, err := client.GetPrompt(cmd.Context(), promptName)
		if err != nil {
			return err
		}
		bytes, err := json.MarshalIndent(registeredPrompt,"", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(bytes))
		
		return nil
	},
}

func init() {
	rootCmd.AddCommand(promptCmd)
	promptCmd.AddCommand(getPromptCmd)
	promptCmd.AddCommand(listPromptsCmd)
	getPromptCmd.Flags().StringVar(&promptName, "name", "", "Prompt Name")


	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// promptCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// promptCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
