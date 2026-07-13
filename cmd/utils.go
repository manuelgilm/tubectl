package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"path/filepath"
	"encoding/json"
	"gopkg.in/yaml.v3"
	"tubectl/internal/youtube"
	"tubectl/internal/prompt"
	"tubectl/internal/ai"
)

func printTranscript(t *youtube.Transcript) {
	for _, line := range t.Lines {
		minutes := int(line.Start) / 60
		seconds := int(line.Start) % 60
		fmt.Printf("[%02d:%02d] %s\n", minutes, seconds, line.Text)
	}
}

// Function to get the tubectl home directoy
func TubeCtlHome() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".tubectl"), nil

}

// transcriptCachePath returns the path for a cached transcript JSON file.
func TranscriptCachePath(videoID string) (string, error) {
	dir, err := TubeCtlHome()
	if err != nil {
		return "", err
	}
	transcriptsDir := filepath.Join(dir, "transcripts")
	if err := os.MkdirAll(transcriptsDir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(transcriptsDir, videoID+".json"), nil
}

// saveCachedTranscript writes a transcript to the local cache.
func SaveCachedTranscript(t *youtube.Transcript) error {
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


// loadCachedTranscript reads a transcript from the local cache.
// Returns nil, nil if the cache file does not exist yet.
func LoadCachedTranscript(videoID string) (*youtube.Transcript, error) {
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

	var t youtube.Transcript
	if err := json.NewDecoder(f).Decode(&t); err != nil {
		return nil, fmt.Errorf("corrupt transcript cache for %s: %w", videoID, err)
	}
	return &t, nil
}


type PromptFile struct {
	Template	string `yaml:"template"`
	Vars		[]string 	`yaml:"vars"`
}

func (p *PromptFile) Render(vars map[string]string) (string, error) {
	result := p.Template
	for _, v := range p.Vars {
		val, ok := vars[v]
		if !ok {
			return "", fmt.Errorf("missing variable: %s ", v)
		}
		result = strings.ReplaceAll(result, "{"+v+"}", val)
	}
	return result, nil
}

func LoadPromptFile(path string) (*PromptFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading prompt file: %w ", err)
	}
	var p PromptFile
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parsing prompt file: %s ", path)
	}
	if p.Template == "" {
		return nil, fmt.Errorf("prompt file %s has empty template", path)
	}
	return &p, nil
}


func BuildMessagesYTBot(text string, transcript string) ([]ai.Message, error) {
	renderedTemplate := fmt.Sprintf(`
You are Gilsama-Bot, an AI assistant that helps manage YouTube comments for a content creator. Your role is to write friendly and helpful replies to viewer comments.

Guidelines:
- Always start your reply with: [Automated Reply] Gilsama-Bot:
- Be warm, appreciative, and conversational
- Reference specific points from the comment or video transcript
- Keep replies concise (2-4 sentences)
- Maintain a friendly and neutral tone regardless of the comment's tone
- If the question cannot be answered from the video context, say: "Oh I don't have the answer for that question and it's not in the video context. Feel free to check other videos or resources!"
- If the user input is off-topic, nonsensical, or hostile, respond politely by steering back to the video content

Comment:
%s

Video transcript context:
%s
`, text, transcript)
	return []ai.Message{
		{Role: "system", Content: renderedTemplate},
	}, nil
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

func loadConfig() (Config, error) {
	home, err := TubeCtlHome()
	if err != nil {
		return Config{}, err
	}
	data, err := os.ReadFile(filepath.Join(home, "config.json"))
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func loadMlflowClient() (*prompt.Client, error) {
	username := os.Getenv("MLFLOW_USERNAME")
	password := os.Getenv("MLFLOW_PASSWORD")

	if username != "" && password != "" {
		return prompt.NewClient(username, password, ""), nil
	}

	home, err := TubeCtlHome()
	if err != nil {
		return nil, err
	}
	creds, err := prompt.LoadCredentials(filepath.Join(home, "auth", "mlflow.json"))
	if err != nil {
		return nil, fmt.Errorf("MLflow credentials not found. Set MLFLOW_USERNAME/MLFLOW_PASSWORD env vars or run 'tubectl auth mlflow --username <user> --password <pass>'")
	}
	return prompt.NewClient(creds.Username, creds.Password, creds.ServerURL), nil
}

func loadClient(ctx context.Context) (*youtube.Client, error) {
	home, err := TubeCtlHome()
	if err != nil {
		return nil, err 
	}
	
	tokenPath := filepath.Join(home, "auth", "youtube.json")
	token, err := youtube.LoadToken(tokenPath)
	if err != nil {
		return nil, fmt.Errorf("loading token: %w", err)
	}

	if !token.Valid() {
		newToken, err := youtube.RefreshToken(ctx, tokenPath)
		if err != nil {
			return nil, fmt.Errorf("token expired and refresh failed: %w", err)
		}
		token = newToken
	}

	return youtube.NewClient(token), nil
}