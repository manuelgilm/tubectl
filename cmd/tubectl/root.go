package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"tubectl/internal/youtube"
)

var rootCmd = &cobra.Command{
	Use:   "tubectl",
	Short: "A CLI for the YouTube Data API v3",
	Long: `tubectl lets you authenticate with YouTube and interact with
videos, comments, and replies from the command line.`,
}

// Execute is the entry point called from main.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(videoCmd)
	rootCmd.AddCommand(commentCmd)
}

// tokenFilePath returns the path where the OAuth token is stored.
func tokenFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".tubectl-token.json"
	}
	dir := filepath.Join(home, ".config", "tubectl")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return ".tubectl-token.json"
	}
	return filepath.Join(dir, "token.json")
}

// oauthCfg builds an OAuthConfig from environment variables.
func oauthCfg() (*youtube.OAuthConfig, error) {
	clientID := os.Getenv("YOUTUBE_CLIENT_ID")
	clientSecret := os.Getenv("YOUTUBE_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("YOUTUBE_CLIENT_ID and YOUTUBE_CLIENT_SECRET environment variables must be set")
	}
	return &youtube.OAuthConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		// OOB redirect — Google displays the code on screen; ideal for CLIs.
		RedirectURI: "urn:ietf:wg:oauth:2.0:oob",
		Scopes:      []string{youtube.ScopeYoutubeForceSsl},
	}, nil
}

// loadClient loads the stored token and returns an authenticated client.
func loadClient() (*youtube.Client, error) {
	cfg, err := oauthCfg()
	if err != nil {
		return nil, err
	}
	tok, err := youtube.LoadToken(tokenFilePath())
	if err != nil {
		return nil, fmt.Errorf("not authenticated — run: tubectl auth login")
	}
	return youtube.NewClientWithToken(cfg, tok), nil
}

// configDir returns ~/.config/tubectl, creating it if needed.
func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".config", "tubectl")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return dir, nil
}

// transcriptCachePath returns the path for a cached transcript JSON file.
func transcriptCachePath(videoID string) (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	transcriptsDir := filepath.Join(dir, "transcripts")
	if err := os.MkdirAll(transcriptsDir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(transcriptsDir, videoID+".json"), nil
}

// loadCachedTranscript reads a transcript from the local cache.
// Returns nil, nil if the cache file does not exist yet.
func loadCachedTranscript(videoID string) (*youtube.Transcript, error) {
	path, err := transcriptCachePath(videoID)
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

	var t youtube.Transcript
	if err := json.NewDecoder(f).Decode(&t); err != nil {
		return nil, fmt.Errorf("corrupt transcript cache for %s: %w", videoID, err)
	}
	return &t, nil
}

// saveCachedTranscript writes a transcript to the local cache.
func saveCachedTranscript(t *youtube.Transcript) error {
	path, err := transcriptCachePath(t.VideoID)
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
