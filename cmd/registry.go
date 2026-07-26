package cmd

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"time"
	"github.com/spf13/cobra"
	"github.com/manuelgilm/tubectl/internal/storage"
	"gopkg.in/yaml.v3"
)

type yamlVideoEntry struct {
	ID    string
	Title string
}

func (e *yamlVideoEntry) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err == nil {
		e.ID = s
		return nil
	}
	var obj struct {
		ID    string `yaml:"id"`
		Title string `yaml:"title,omitempty"`
	}
	if err := value.Decode(&obj); err != nil {
		return fmt.Errorf("expected a video ID string or an object with 'id' field")
	}
	if obj.ID == "" {
		return fmt.Errorf("video entry missing 'id' field")
	}
	e.ID = obj.ID
	e.Title = obj.Title
	return nil
}

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
	regSyncArgs struct {
		yamlPath string
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

var registrySyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync registered_videos.yaml into the SQLite registry",
	Long: `Reads a YAML file with video IDs (and optional titles) and ensures all
entries are registered in the local SQLite database.

The YAML supports two formats:
  videos:
    - ITQioNZ_m_U                          # simple ID string
    - id: dQw4w9WgXcQ                      # object without title
      title: "My Video Title"              # object with title (updates existing)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := openDB()
		if err != nil {
			return err
		}
		defer db.Close()

		added, updated, skipped, err := syncYamlEntries(cmd.Context(), db, regSyncArgs.yamlPath)
		if err != nil {
			return err
		}
		fmt.Printf("Synced %s: %d added, %d updated, %d skipped\n", regSyncArgs.yamlPath, added, updated, skipped)
		return nil
	},
}

func syncYamlEntries(ctx context.Context, db *sql.DB, yamlPath string) (added, updated, skipped int, _ error) {
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("reading yaml file: %w", err)
	}

	var list struct {
		Videos []yamlVideoEntry `yaml:"videos"`
	}
	if err := yaml.Unmarshal(data, &list); err != nil {
		return 0, 0, 0, fmt.Errorf("parsing yaml: %w", err)
	}

	repo := storage.NewVideoRepo(db)
	existing, err := repo.List(ctx)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("listing registry: %w", err)
	}

	existingIDs := make(map[string]bool, len(existing))
	for _, v := range existing {
		existingIDs[v.ID] = true
	}

	now := time.Now()
	for _, entry := range list.Videos {
		if _, ok := existingIDs[entry.ID]; ok {
			if entry.Title != "" {
				v, err := repo.Get(ctx, entry.ID)
				if err != nil {
					return 0, 0, 0, fmt.Errorf("get video %s: %w", entry.ID, err)
				}
				if v != nil && v.Title != entry.Title {
					if _, err := repo.Update(ctx, entry.ID, entry.Title); err != nil {
						return 0, 0, 0, fmt.Errorf("update video %s: %w", entry.ID, err)
					}
					updated++
				} else {
					skipped++
				}
			} else {
				skipped++
			}
			continue
		}

		if err := repo.Add(ctx, storage.Video{
			ID:           entry.ID,
			Title:        entry.Title,
			RegisteredAt: now,
			UpdatedAt:    now,
		}); err != nil {
			return 0, 0, 0, fmt.Errorf("add video %s: %w", entry.ID, err)
		}
		added++
	}
	return added, updated, skipped, nil
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

	registryCmd.AddCommand(registrySyncCmd)
	registrySyncCmd.Flags().StringVar(&regSyncArgs.yamlPath, "yaml", "registered_videos.yaml", "Path to the YAML file with video IDs")
}
