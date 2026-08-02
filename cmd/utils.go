package cmd

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"github.com/manuelgilm/tubectl/internal/ai"
	"github.com/manuelgilm/tubectl/internal/prompt"
	"github.com/manuelgilm/tubectl/internal/storage"
	"github.com/manuelgilm/tubectl/internal/trace"
	"github.com/manuelgilm/tubectl/internal/youtube"
	"github.com/spf13/cobra"
)

// Function to get the tubectl home directoy
func TubeCtlHome() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}
	if home == "" {
		return "", fmt.Errorf("home directory is empty")
	}
	return filepath.Join(home, ".tubectl"), nil

}

func loadOpenAIClient(ctx context.Context, model string) (*ai.Client, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		if cfg, err := loadConfig(); err == nil {
			apiKey = cfg.OpenAI.APIKey
		}
	}

	if model == "" {
		model = "gpt-4o-mini"
	}
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY not found in environment variables or config.json")
	}
	c := ai.NewClient(apiKey, model)
	if tracer := newMLflowTracer(ctx); tracer != nil {
		c.WithTracer(tracer)
	}
	return c, nil
}

type mlflowCreds struct {
	username  string
	password  string
	serverURL string
}

// resolveMLflowCreds resolves MLflow credentials from environment variables,
// falling back to the credentials file written by `tubectl auth mlflow`.
func resolveMLflowCreds() (mlflowCreds, error) {
	username := os.Getenv("MLFLOW_TRACKING_USERNAME")
	password := os.Getenv("MLFLOW_TRACKING_PASSWORD")
	serverURL := os.Getenv("MLFLOW_SERVER_URL")

	if username == "" || password == "" {
		// Fall back entirely to the credentials file. Never mix half the env
		// pair (e.g. username from env) with a password from the file.
		home, err := TubeCtlHome()
		if err != nil {
			return mlflowCreds{}, err
		}
		creds, err := prompt.LoadCredentials(filepath.Join(home, "auth", "mlflow.json"))
		if err != nil {
			return mlflowCreds{}, fmt.Errorf("MLflow credentials not found. Set MLFLOW_TRACKING_USERNAME/MLFLOW_TRACKING_PASSWORD env vars or run 'tubectl auth mlflow --username <user> --password <pass>'")
		}
		username = creds.Username
		password = creds.Password
		if serverURL == "" {
			serverURL = creds.ServerURL
		}
	}
	if serverURL == "" {
		serverURL = prompt.DefaultMlflowServer
	}
	if username == "" || password == "" {
		return mlflowCreds{}, fmt.Errorf("MLflow credentials not found. Set MLFLOW_TRACKING_USERNAME/MLFLOW_TRACKING_PASSWORD env vars or run 'tubectl auth mlflow --username <user> --password <pass>'")
	}
	return mlflowCreds{username: username, password: password, serverURL: serverURL}, nil
}

func newMLflowTracer(_ context.Context) *trace.MLflowTracer {
	if os.Getenv("MLFLOW_TRACING_ENABLED") != "true" {
		return nil
	}
	creds, err := resolveMLflowCreds()
	if err != nil {
		return nil
	}
	tracer := trace.NewMLflowTracer(creds.serverURL, creds.username, creds.password)
	if expID := os.Getenv("MLFLOW_EXPERIMENT_ID"); expID != "" {
		tracer.WithExperimentID(expID)
	}
	return tracer
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
	creds, err := resolveMLflowCreds()
	if err != nil {
		return nil, err
	}
	return prompt.NewClient(creds.username, creds.password, creds.serverURL), nil
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


func ResolvePromptTemplate(cmd *cobra.Command, promptName, promptFile, commentText, transcriptText string) (string, error) {

	var resolvedTemplate string
	var err error
	switch {
		case promptName != "":
			resolvedTemplate, err = ResolvePromptFromMLflowRegistry(cmd, promptName, commentText, transcriptText)
			if err != nil {
				return "", err
			}
		case promptFile != "":
			resolvedTemplate, err = ResolvePromptFromFile(promptFile, commentText, transcriptText)
			if err != nil {
				return "", err
			}
		default:
			resolvedTemplate, err = resolveBotPrompt(cmd, cmd.ErrOrStderr(), commentText, transcriptText)
			if err != nil {
				return "", fmt.Errorf("resolving prompt: %w", err)
			}
		}
	return resolvedTemplate, nil
}

func ResolvePromptFromFile(promptFile, commentText, transcriptText string) (string, error) {
	pf, err := prompt.LoadPromptFile(promptFile)
	if err != nil {
		return "",fmt.Errorf("loading prompt file: %w", err)
	}
	rendered, err := pf.Render(map[string]string{
		"comment":    commentText,
		"transcript": transcriptText,
	})
	if err != nil {
		return "", fmt.Errorf("rendering prompt: %w", err)
	}
	return rendered, nil
}

func ResolvePromptFromMLflowRegistry(cmd *cobra.Command, promptName, commentText, transcriptText string) (string, error) {
	mlflowClient, err := loadMlflowClient()
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Warning: MLflow client unavailable (%v), falling back to default prompt\n", err)
		return prompt.DefaultBotPromptText(commentText, transcriptText), nil
	}
	registered, err := mlflowClient.GetPrompt(cmd.Context(), promptName)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Warning: MLflow prompt %q fetch failed (%v), falling back to default prompt\n", promptName, err)
		return prompt.DefaultBotPromptText(commentText, transcriptText), nil
	}
	template := registered.PromptText()
	if template == "" {
		fmt.Fprintf(cmd.ErrOrStderr(), "Warning: MLflow prompt %q has no prompt text, falling back to default prompt\n", promptName)
		return prompt.DefaultBotPromptText(commentText, transcriptText), nil
	}
	rendered := renderTemplate(template, commentText, transcriptText)	
	return rendered, nil
}

func resolveBotPrompt(cmd *cobra.Command, w io.Writer, commentText, transcriptText string) (string, error) {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(w, "Warning: could not load config: %v\n", err)
		return tryLocalPrompt(Config{}, commentText, transcriptText)
	}

	modelName := cfg.BotPrompt.AnswerCommentModel
	if modelName == "" {
		return tryLocalPrompt(cfg, commentText, transcriptText)
	}

	mlflowClient, err := loadMlflowClient()
	if err != nil {
		fmt.Fprintf(w, "Warning: MLflow client unavailable (%v), falling back to local prompt\n", err)
	} else {
		registered, err := mlflowClient.GetPrompt(cmd.Context(), modelName)
		if err != nil {
			fmt.Fprintf(w, "Warning: MLflow prompt %q fetch failed (%v), falling back to local prompt\n", modelName, err)
		} else {
			template := registered.PromptText()
			if template == "" {
				fmt.Fprintf(w, "Warning: MLflow prompt %q has empty text, falling back to local prompt\n", modelName)
			} else {
				rendered := renderTemplate(template, commentText, transcriptText)
				return rendered, nil
			}
		}
	}

	return tryLocalPrompt(cfg, commentText, transcriptText)
}

func tryLocalPrompt(cfg Config, commentText, transcriptText string) (string, error) {
	home, err := TubeCtlHome()
	if err != nil {
		return prompt.DefaultBotPromptText(commentText, transcriptText), nil
	}

	modelName := cfg.BotPrompt.AnswerCommentModel
	if modelName == "" {
		modelName = "yt-bot-answer-comment"
	}
	if strings.ContainsAny(modelName, "/\\") {
		modelName = "yt-bot-answer-comment"
	}

	pf, err := prompt.LoadPromptFile(filepath.Join(home, "prompts", modelName+".yaml"))
	if err != nil {
		return prompt.DefaultBotPromptText(commentText, transcriptText), nil
	}

	rendered, err := pf.Render(map[string]string{
		"comment":    commentText,
		"transcript": transcriptText,
	})
	if err != nil {
		return prompt.DefaultBotPromptText(commentText, transcriptText), nil
	}

	return rendered, nil
}
func ResolveComment(cmd *cobra.Command, commentID string) (string, error){
	client, err := loadClient(cmd.Context())
	if err != nil {
		return "", err
	}
	comment, err := client.GetComment(cmd.Context(), commentID)
	if err != nil {
		return "", fmt.Errorf("getting comment: %w", err)
	}
	commentText := comment.Snippet.TextDisplay	
	return commentText, nil
}

func GenerateAnswer(cmd *cobra.Command, resolvedTemplate, model string, tags map[string]string) (string, error) {
	messages := []ai.Message{
		{Role: "system", Content: resolvedTemplate},
	}
	
	aiClient, err := loadOpenAIClient(cmd.Context(), model)
	if err != nil {
		return "", fmt.Errorf("loading AI client: %w", err)
	}
	if len(tags) > 0 {
		aiClient.WithTags(tags)
	}
	reply, err := aiClient.Complete(cmd.Context(), messages)
	if err != nil {
		return "", fmt.Errorf("AI completion failed: %w", err)
	}
	return reply, nil
}

func replyComment(cmd *cobra.Command, commentID, reply string, autoApprove bool) error {

	fmt.Fprintf(cmd.ErrOrStderr(), "Generated reply:\n---\n%s\n---\n", reply)
	if !autoApprove {
		fmt.Fprint(cmd.ErrOrStderr(), "Post this reply? [y/N]: ")
		var confirm string
		if _, err := fmt.Scanln(&confirm); err != nil {
			return fmt.Errorf("reading confirmation (use --auto-approve in non-interactive mode): %w", err)
		}
		if confirm != "y" && confirm != "Y" {
			fmt.Println("Reply cancelled.")
			return nil
		}
	}
	client, err := loadClient(cmd.Context())
	if err != nil {
		return err
	}
	posted, err := client.ReplyToComment(cmd.Context(), commentID, reply)
	if err != nil {
		return fmt.Errorf("posting reply: %w", err)
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "Reply posted (ID: %s)\n", posted.ID)
	fmt.Println(reply)
	return nil	
}


func renderTemplate(template, commentText, transcriptText string) string {
	r := strings.NewReplacer("{comment}", commentText, "{transcript}", transcriptText)
	return r.Replace(template)
}

func openDB() (*sql.DB, error) {
	home, err := TubeCtlHome()
	if err != nil {
		return nil, err
	}
	dbPath := filepath.Join(home, "tubectl.db")
	db, err := storage.New(dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}
	return db, nil
}
