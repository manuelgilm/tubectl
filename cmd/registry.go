/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"path/filepath"
	"github.com/spf13/cobra"
	"tubectl/internal/registry"
	"encoding/json"
)

var videoID, title string
var registryUpdateCmd = &cobra.Command{
	Use: "update",
	Short: "Update a video's title in the registry",
	RunE: func(cmd *cobra.Command, args []string) error {
		home, err := TubeCtlHome()
		if err != nil {
			return err
		}
		reg, err := registry.LoadRegistry(filepath.Join(home, "registry.json"))
		if err != nil {
			return err
		}
		if !registry.UpdateVideo(reg, videoID, title){
			return fmt.Errorf("Video %s not found", videoID)
		}
		return registry.SaveRegistry(home, reg)
	},
}
var registryDeleteCmd = &cobra.Command{
	Use: "delete",
	Short: "Remove a video from the registry",
	RunE: func(cmd *cobra.Command, args []string) error {
		home, err := TubeCtlHome()
		if err != nil {
			return err
		}
		reg, err := registry.LoadRegistry(filepath.Join(home, "registry.json"))
		if err != nil {
			return err
		}
		if !registry.RemoveVideo(reg, videoID) {
			return fmt.Errorf("video %s not found in registry", videoID)
		}
		return registry.SaveRegistry(home,reg)
	},
}
var registryListCmd = &cobra.Command{
	Use: "list",
	Short: "List all registered videos",
	RunE: func(cmd *cobra.Command, args []string) error {
		home, err := TubeCtlHome()
		if err != nil {
			return err 
		}
		reg, err := registry.LoadRegistry(filepath.Join(home, "registry.json"))
		if err != nil {
			return err
		}
		data, err := json.MarshalIndent(reg, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	},
}
var registryAddCmd = &cobra.Command{
	Use: "add",
	Short: "Add a video to the registry",
	RunE: func(cmd *cobra.Command, args []string) error {
		home, err := TubeCtlHome()
		if err != nil {
			return err
		}
		reg, err := registry.LoadRegistry(filepath.Join(home, "registry.json"))
		if err != nil {
			return err
		}
		err = registry.AddVideo(reg, videoID, title)
		if err != nil {
			return err
		}
		return registry.SaveRegistry(home, reg)
	},
}
var registryCmd = &cobra.Command{
	Use:   "registry",
	Short: "Manage the local video registry",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("registry called")
	},
}

func init() {
	rootCmd.AddCommand(registryCmd)
	registryCmd.AddCommand(registryAddCmd)
	registryAddCmd.Flags().StringVar(&videoID, "video-id", "", "YouTube Video ID")
	registryAddCmd.Flags().StringVar(&title, "title", "", "YouTube Video Title")
	
	registryCmd.AddCommand(registryListCmd)
	registryCmd.AddCommand(registryDeleteCmd)
	registryDeleteCmd.Flags().StringVar(&videoID, "video-id", "", "YouTube Video ID")

	registryCmd.AddCommand(registryUpdateCmd)
	registryUpdateCmd.Flags().StringVar(&videoID, "video-id", "", "YouTube Video ID")
	registryUpdateCmd.Flags().StringVar(&title, "title", "", "YouTube Video Title")
}
