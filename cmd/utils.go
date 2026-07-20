package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"tubectl/internal/ai"
	"tubectl/internal/prompt"
	"tubectl/internal/trace"
	"tubectl/internal/youtube"
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

func loadOpenAIClient(ctx context.Context, model string) (*ai.Client, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")

	if model == "" {
		model = "gpt-4o-mini"
	}
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY not found in environment variables")
	}
	c := ai.NewClient(apiKey, model)
	if tracer := newMLflowTracer(ctx); tracer != nil {
		c.WithTracer(tracer)
	}
	return c, nil
}

func newMLflowTracer(_ context.Context) *trace.MLflowTracer {
	if os.Getenv("MLFLOW_TRACING_ENABLED") != "true" {
		return nil
	}
	username := os.Getenv("MLFLOW_USERNAME")
	password := os.Getenv("MLFLOW_PASSWORD")
	serverURL := os.Getenv("MLFLOW_SERVER_URL")

	if username == "" || password == "" {
		home, err := TubeCtlHome()
		if err != nil {
			return nil
		}
		creds, err := prompt.LoadCredentials(filepath.Join(home, "auth", "mlflow.json"))
		if err != nil {
			return nil
		}
		username = creds.Username
		password = creds.Password
		if creds.ServerURL != "" {
			serverURL = creds.ServerURL
		}
	}
	if serverURL == "" {
		serverURL = prompt.DefaultMlflowServer
	}
	return trace.NewMLflowTracer(serverURL, username, password)
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
		serverURL := os.Getenv("MLFLOW_SERVER_URL")
		return prompt.NewClient(username, password, serverURL), nil
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