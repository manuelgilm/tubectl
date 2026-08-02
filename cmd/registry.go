package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/manuelgilm/tubectl/internal/storage"
	"github.com/spf13/cobra"
	"time"
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
	Use:   "update",
	Short: "Update a video's title in the registry",
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := openDB()
		if err != nil {
			return err
		}
		defer db.Close()

		repo := storage.NewVideoRepo(db)
		v, err := repo.Update(cmd.Context(), regUpdateArgs.videoID, regUpdateArgs.title)
		if err != nil {
			return err
		}
		if v == nil {
			return fmt.Errorf("video %s not found", regUpdateArgs.videoID)
		}
		return nil
	},
}

var registryDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Remove a video from the registry",
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := openDB()
		if err != nil {
			return err
		}
		defer db.Close()

		repo := storage.NewVideoRepo(db)
		ok, err := repo.Delete(cmd.Context(), regDeleteArgs.videoID)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("video %s not found in registry", regDeleteArgs.videoID)
		}
		return nil
	},
}

var registryListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all registered videos",
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := openDB()
		if err != nil {
			return err
		}
		defer db.Close()

		repo := storage.NewVideoRepo(db)
		vs, err := repo.List(cmd.Context())
		if err != nil {
			return err
		}
		data, err := json.MarshalIndent(vs, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	},
}

var registryAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a video to the registry",
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := openDB()
		if err != nil {
			return err
		}
		defer db.Close()

		repo := storage.NewVideoRepo(db)
		v := storage.Video{
			ID:           regAddArgs.videoID,
			Title:        regAddArgs.title,
			RegisteredAt: time.Now(),
			UpdatedAt:    time.Now(),
		}
		return repo.Add(cmd.Context(), v)
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
