package cmd

import (
	"os"
	"tubectl/internal/youtube"
	"tubectl/internal/ai"
	"github.com/spf13/cobra"
	"fmt"
	"path/filepath"
)

var rootCmd = &cobra.Command{
	Use:   "tubectl",
	Short: "CLI for YouTube management and AI-powered automations",
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
}
func loadOpenAIClient(model string) (*ai.Client, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")

	if model == "" {
		model = "gpt-4o-mini"
	}
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY Not found in environment variables")
	}
	return ai.NewClient(apiKey, model), nil
}

func loadClient() (*youtube.Client, error) {
	home, err := TubeCtlHome()
	if err != nil {
		return nil, err 
	}
	
	token, err := youtube.LoadToken(filepath.Join(home, "auth", "youtube.json"))
	if err == nil {
		if !token.Valid() {
			return nil, fmt.Errorf("Token Expired")
		}
	} else {
		return nil, fmt.Errorf("Loading token: %v ", err)
	}
	
	
	client := youtube.NewClient(token)
	return client, nil
}