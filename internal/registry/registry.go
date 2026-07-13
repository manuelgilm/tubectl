package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
	"path/filepath"
)
// Defines the registry
type TubeRegistry struct {
	Videos []RegisteredVideo `json:"videos"`
}

type RegisteredVideo struct {
	Title        string    `json:"title"`
	VideoID      string    `json:"video_id"`
	PublishedAt  time.Time `json:"published_at"`
	RegisteredAt time.Time `json:"registered_at"`
}
//CRUD FUNCTIONS
func LoadRegistry(path string) (*TubeRegistry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading registry file: %w ", err)
	}

	var registry TubeRegistry
	err = json.Unmarshal(data, &registry)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling registry: %w ", err)
	}
	return &registry, nil
}

func AddVideo(reg *TubeRegistry, videoID string, title string) error {
	
	for _, v:= range reg.Videos {
		if v.VideoID == videoID {
			return fmt.Errorf("video %s already registered", videoID)
		}
	}
	reg.Videos = append(reg.Videos, RegisteredVideo{
		Title: title,
		VideoID: videoID,
		PublishedAt: time.Time{},
		RegisteredAt: time.Now(),
	})
	return nil
}

func RemoveVideo(reg *TubeRegistry, videoID string) bool {
	for i , v := range reg.Videos {
		if v.VideoID == videoID {
			reg.Videos = append(reg.Videos[:i], reg.Videos[i+1:]...)
			return true
		}
	}
	return false
}

func UpdateVideo(reg *TubeRegistry, videoID string, title string) bool {
	for i , v := range reg.Videos {
		if v.VideoID == videoID {
			reg.Videos[i].Title = title
			return true
		}
	}
	return false
}
// Functions
func WriteRegistryFile(path string) error {
	var emptyRegistry TubeRegistry

	//marshal the empty registry file
	data, err := json.MarshalIndent(emptyRegistry, "", "  ")

	if err != nil {
		return fmt.Errorf("Error with Marshal: %w ", err)
	}

	err = os.WriteFile(filepath.Join(path, "registry.json"), data, 0644)

	return err

}

func SaveRegistry(path string, reg *TubeRegistry) error {
	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling registry: %w", err)
	}
	return os.WriteFile(filepath.Join(path, "registry.json"), data, 0644)	
	
}