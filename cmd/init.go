/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"tubectl/internal/registry"
	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the ~/.tubectl directory structure",
	Long: `Creates the ~/.tubectl directory and all required subdirectories
and files: config.json, registry.json, auth/, transcripts/, prompts/.

Run this once before using other commands.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		tubehome, err := TubeCtlHome()
		if err != nil {
			return fmt.Errorf("Error defining tubectl home: %v ", err)
		}

		err = createFolder(tubehome)
		if err != nil {
			return fmt.Errorf("Error creating folder: %v ", err)
		}

		// writing the files registry.json and config.json
		err = writeConfigFile(tubehome)
		if err != nil {
			return fmt.Errorf("Error creating config file %v ", err)
		}

		err = registry.WriteRegistryFile(tubehome)
		if err != nil {
			return fmt.Errorf("Error creating registry file %v ", err)
		}
		// creating prompt folder
		err = createFolder(filepath.Join(tubehome, "prompts"))
		if err != nil {
			return fmt.Errorf("creating folder: %v ", err)
		}

		// Creating additional folders
		err = createFolder(filepath.Join(tubehome, "transcripts"))
		if err != nil {
			return fmt.Errorf("Error creating folder: %v ", err)
		}

		err = createFolder(filepath.Join(tubehome, "auth"))
		if err != nil {
			return fmt.Errorf("Error creating folder: %v ", err)
		}

		return nil
	},
}

// the config struct (defines the shape of config.json)
type Config struct {
	OpenAI OpenAIConfig `json:"openai"`
	Prompt PromptConfig `json:"prompt"`
}

type OpenAIConfig struct {
	APIKey  string `json:"api_key"`
	BaseURL string `json:"base_url"`
}

type PromptConfig struct {
	ServerURL string `json:"server_url"`
}



func createFolder(path string) error {
	// Create a folder if it does not exist

	err := os.MkdirAll(path, 0755)
	if err != nil {
		return fmt.Errorf("Error creating new directory: %v ", err)
	}

	return nil
}
func writeConfigFile(path string) error {
	//
	var emptyConfig Config

	// marshal the empty config
	data, err := json.MarshalIndent(emptyConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("Error with Marshal: %v ", err)
	}

	err = os.WriteFile(filepath.Join(path, "config.json"), data, 0666)
	if err != nil {
		return fmt.Errorf("Error writing file: %v ", err)
	}
	return nil
}

func init() {
	rootCmd.AddCommand(initCmd)
}
