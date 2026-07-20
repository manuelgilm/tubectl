package youtube

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func SaveToken(path string, token *Token) error {
	data, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("marshal token: %w", err)
	}

	err = os.WriteFile(path, data, 0600)
	if err != nil {
		return fmt.Errorf("saving token: %w", err)
	}
	return nil
}

func TranscriptCachePath(videoID string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	transcriptsDir := filepath.Join(home, ".tubectl", "transcripts")
	if err := os.MkdirAll(transcriptsDir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(transcriptsDir, videoID+".json"), nil
}

func SaveCachedTranscript(t *Transcript) error {
	path, err := TranscriptCachePath(t.VideoID)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(t)
}

func LoadCachedTranscript(videoID string) (*Transcript, error) {
	path, err := TranscriptCachePath(videoID)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var t Transcript
	if err := json.NewDecoder(f).Decode(&t); err != nil {
		return nil, fmt.Errorf("corrupt transcript cache for %s: %w", videoID, err)
	}
	return &t, nil
}
