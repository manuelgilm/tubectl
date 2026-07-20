package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"tubectl/internal/registry"
	"tubectl/internal/prompt"
	"gopkg.in/yaml.v3"
	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the ~/.tubectl directory structure",
	Long: `Creates the ~/.tubectl directory and all required subdirectories
and files: config.json, registry.json, auth/, transcripts/, prompts/,
plus an emergency prompt file (prompts/yt-bot-answer-comment.yaml).

Run this once before using other commands.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		tubehome, err := TubeCtlHome()
		if err != nil {
			return fmt.Errorf("error defining tubectl home: %w", err)
		}

		err = createFolder(tubehome)
		if err != nil {
			return fmt.Errorf("error creating folder: %w", err)
		}

		// writing the files registry.json and config.json
		err = writeConfigFile(tubehome)
		if err != nil {
			return fmt.Errorf("error creating config file %w", err)
		}

		err = registry.WriteRegistryFile(tubehome)
		if err != nil {
			return fmt.Errorf("error creating registry file %w", err)
		}
		// writing the emergency prompt
		err = writeEmergencyPrompt(tubehome)
		if err != nil {
			return fmt.Errorf("writing emergency prompt: %w", err)
		}

		// Creating additional folders
		err = createFolder(filepath.Join(tubehome, "transcripts"))
		if err != nil {
			return fmt.Errorf("error creating folder: %w", err)
		}

		err = createFolder(filepath.Join(tubehome, "auth"))
		if err != nil {
			return fmt.Errorf("error creating folder: %w", err)
		}

		return nil
	},
}

// the config struct (defines the shape of config.json)
type Config struct {
	OpenAI    OpenAIConfig    `json:"openai"`
	Prompt    PromptConfig    `json:"prompt"`
	BotPrompt BotPromptConfig `json:"bot_prompt"`
}

type OpenAIConfig struct {
	APIKey  string `json:"api_key"`
	BaseURL string `json:"base_url"`
}

type PromptConfig struct {
	ServerURL string `json:"server_url"`
}

type BotPromptConfig struct {
	AnswerCommentModel string `json:"answer_comment_model"`
}



func createFolder(path string) error {
	// Create a folder if it does not exist

	err := os.MkdirAll(path, 0755)
	if err != nil {
		return fmt.Errorf("error creating new directory: %w", err)
	}

	return nil
}
func writeConfigFile(path string) error {
	cfg := Config{
		BotPrompt: BotPromptConfig{
			AnswerCommentModel: "yt-bot-answer-comment",
		},
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("error with Marshal: %w", err)
	}

	return os.WriteFile(filepath.Join(path, "config.json"), data, 0644)
}

func writeEmergencyPrompt(path string) error {
	dir := filepath.Join(path, "prompts")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	p := prompt.PromptFile{
		Template: fmt.Sprintf(`You are Gilsama-Bot, an AI assistant that helps manage YouTube comments for a content creator. Your role is to write friendly and helpful replies to viewer comments.

Guidelines:
- Always start your reply with: [Automated Reply] Gilsama-Bot
- Be warm, appreciative, and conversational
- Reference specific points from the comment or video transcript
- Keep replies concise (2-4 sentences)
- Maintain a friendly and neutral tone regardless of the comment's tone
- If the question cannot be answered from the video context, say: "Oh I don't have the answer for that question and it's not in the video context. Feel free to check other videos or resources!"
- If the user input is off-topic, nonsensical, or hostile, respond politely by steering back to the video content

Comment:
{comment}

Video transcript context:
{transcript}`),
		Vars: []string{"comment", "transcript"},
	}

	data, err := yaml.Marshal(&p)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dir, "yt-bot-answer-comment.yaml"), data, 0644)
}

func init() {
	rootCmd.AddCommand(initCmd)
}
