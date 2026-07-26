package cmd

import (
	"fmt"
	"path/filepath"
	"github.com/spf13/cobra"
	"github.com/manuelgilm/tubectl/internal/registry"
	"encoding/json"
)

var (
	regAddArgs struct {
		videoID string
		title   string
	}
	regDeleteArgs struct {
		videoID string
	}
	regUpdateArgs struct {
		videoID string
		title   string
	}
)
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
		if !registry.UpdateVideo(reg, regUpdateArgs.videoID, regUpdateArgs.title){
			return fmt.Errorf("video %s not found", regUpdateArgs.videoID)
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
		if !registry.RemoveVideo(reg, regDeleteArgs.videoID) {
			return fmt.Errorf("video %s not found in registry", regDeleteArgs.videoID)
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
		err = registry.AddVideo(reg, regAddArgs.videoID, regAddArgs.title)
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
	registryAddCmd.Flags().StringVar(&regAddArgs.videoID, "video-id", "", "YouTube Video ID")
	registryAddCmd.MarkFlagRequired("video-id")
	registryAddCmd.Flags().StringVar(&regAddArgs.title, "title", "", "YouTube Video Title")
	
	registryCmd.AddCommand(registryListCmd)
	registryCmd.AddCommand(registryDeleteCmd)
	registryDeleteCmd.Flags().StringVar(&regDeleteArgs.videoID, "video-id", "", "YouTube Video ID")
	registryDeleteCmd.MarkFlagRequired("video-id")

	registryCmd.AddCommand(registryUpdateCmd)
	registryUpdateCmd.Flags().StringVar(&regUpdateArgs.videoID, "video-id", "", "YouTube Video ID")
	registryUpdateCmd.MarkFlagRequired("video-id")
	registryUpdateCmd.Flags().StringVar(&regUpdateArgs.title, "title", "", "YouTube Video Title")
	registryUpdateCmd.MarkFlagRequired("title")
}
