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
	"time"
	"github.com/manuelgilm/tubectl/internal/ai"
	"github.com/manuelgilm/tubectl/internal/prompt"
	"github.com/manuelgilm/tubectl/internal/storage"
	"github.com/manuelgilm/tubectl/internal/trace"
	"github.com/manuelgilm/tubectl/internal/youtube"
	"github.com/spf13/cobra"
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
	username := os.Getenv("MLFLOW_TRACKING_USERNAME")
	password := os.Getenv("MLFLOW_TRACKING_PASSWORD")
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
	tracer := trace.NewMLflowTracer(serverURL, username, password)
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
	username := os.Getenv("MLFLOW_TRACKING_USERNAME")
	password := os.Getenv("MLFLOW_TRACKING_PASSWORD")

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
		return nil, fmt.Errorf("MLflow credentials not found. Set MLFLOW_TRACKING_USERNAME/MLFLOW_TRACKING_PASSWORD env vars or run 'tubectl auth mlflow --username <user> --password <pass>'")
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


func GetTranscriptText(cmd *cobra.Command, videoID string) (string, error) {
	db, err := openDB()
	if err != nil {
		return "", err
	}
	defer db.Close()

	repo := storage.NewTranscriptRepo(db)
	st, err := repo.Load(cmd.Context(), videoID)
	if err != nil {
		return "", fmt.Errorf("loading cached transcript: %w", err)
	}

	if st == nil {
		client, err := loadClient(cmd.Context())
		if err != nil {
			return "", err
		}
		t, err := client.DownloadTranscript(cmd.Context(), videoID, "")
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: transcript not available: %v\n", err)
			return "", nil
		}
		st, err = transcriptToStored(t)
		if err != nil {
			return "", err
		}
		if saveErr := repo.Save(cmd.Context(), st); saveErr != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not cache transcript: %v\n", saveErr)
		}
	}

	return st.Content, nil
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
		return "" , fmt.Errorf("loading MLflow client: %w", err)
	}
	registered, err := mlflowClient.GetPrompt(cmd.Context(), promptName)
	if err != nil {
		return "", fmt.Errorf("fetching prompt %q from MLflow: %w", promptName, err)
	}
	template := registered.PromptText()
	if template == "" {
		return "", fmt.Errorf("prompt %q has no prompt text in MLflow",promptName)
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
	if err == nil {
		registered, err := mlflowClient.GetPrompt(cmd.Context(), modelName)
		if err == nil {
			template := registered.PromptText()
			if template != "" {
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

func GenerateAnswer(cmd *cobra.Command, resolvedTemplate string) (string, error) {
	messages := []ai.Message{
		{Role: "system", Content: resolvedTemplate},
	}
	
	aiClient, err := loadOpenAIClient(cmd.Context(), "")
	if err != nil {
		return "", fmt.Errorf("loading AI client: %w", err)
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


func transcriptToStored(t *youtube.Transcript) (*storage.StoredTranscript, error) {
	linesJSON, err := json.Marshal(t.Lines)
	if err != nil {
		return nil, fmt.Errorf("marshal transcript lines: %w", err)
	}
	var content strings.Builder
	for _, l := range t.Lines {
		content.WriteString(l.Text)
		content.WriteString(" ")
	}
	return &storage.StoredTranscript{
		VideoID:   t.VideoID,
		Language:  t.Language,
		TrackKind: t.TrackKind,
		CaptionID: t.CaptionID,
		Content:   strings.TrimSpace(content.String()),
		Lines:     string(linesJSON),
		CachedAt:  time.Now(),
	}, nil
}

func storedToTranscript(st *storage.StoredTranscript) (*youtube.Transcript, error) {
	var lines []youtube.TranscriptLine
	if err := json.Unmarshal([]byte(st.Lines), &lines); err != nil {
		return nil, fmt.Errorf("unmarshal transcript lines: %w", err)
	}
	return &youtube.Transcript{
		VideoID:   st.VideoID,
		Language:  st.Language,
		TrackKind: st.TrackKind,
		CaptionID: st.CaptionID,
		Lines:     lines,
	}, nil
}

func renderTemplate(template, commentText, transcriptText string) string {
	result := strings.ReplaceAll(template, "{{comment}}", commentText)
	result = strings.ReplaceAll(result, "{{transcript}}", transcriptText)
	return result
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
	if err := maybeImportJSON(db); err != nil {
		db.Close()
		return nil, err
	}
	if err := maybeImportTranscripts(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

type legacyRegistry struct {
	Videos []legacyVideo `json:"videos"`
}

type legacyVideo struct {
	Title        string    `json:"title"`
	VideoID      string    `json:"video_id"`
	PublishedAt  time.Time `json:"published_at"`
	RegisteredAt time.Time `json:"registered_at"`
}

func maybeImportJSON(db *sql.DB) error {
	home, err := TubeCtlHome()
	if err != nil {
		return err
	}
	jsonPath := filepath.Join(home, "registry.json")
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return fmt.Errorf("reading old registry.json: %w", err)
	}

	var reg legacyRegistry
	if err := json.Unmarshal(data, &reg); err != nil {
		return fmt.Errorf("parsing old registry.json: %w", err)
	}

	repo := storage.NewVideoRepo(db)
	now := time.Now()
	for _, v := range reg.Videos {
		vv := storage.Video{
			ID:           v.VideoID,
			Title:        v.Title,
			PublishedAt:  v.PublishedAt,
			RegisteredAt: v.RegisteredAt,
			UpdatedAt:    v.RegisteredAt,
		}
		if vv.RegisteredAt.IsZero() {
			vv.RegisteredAt = now
		}
		if vv.UpdatedAt.IsZero() {
			vv.UpdatedAt = now
		}
		if err := repo.Add(context.Background(), vv); err != nil {
			return fmt.Errorf("importing video %s: %w", v.VideoID, err)
		}
	}

	if err := os.Rename(jsonPath, jsonPath+".migrated"); err != nil {
		return fmt.Errorf("renaming registry.json: %w", err)
	}
	fmt.Printf("Imported %d videos from registry.json into tubectl.db\n", len(reg.Videos))
	return nil
}

func maybeImportTranscripts(db *sql.DB) error {
	home, err := TubeCtlHome()
	if err != nil {
		return err
	}
	transcriptsDir := filepath.Join(home, "transcripts")
	entries, err := os.ReadDir(transcriptsDir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("reading transcripts dir: %w", err)
	}

	trepo := storage.NewTranscriptRepo(db)
	var imported int
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(transcriptsDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: skipping corrupt transcript %s: %v\n", entry.Name(), err)
			continue
		}
		var t youtube.Transcript
		if err := json.Unmarshal(data, &t); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: skipping corrupt transcript %s: %v\n", entry.Name(), err)
			continue
		}
		st, err := transcriptToStored(&t)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: skipping transcript %s: %v\n", entry.Name(), err)
			continue
		}
		if err := trepo.Save(context.Background(), st); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not save transcript %s: %v\n", entry.Name(), err)
			continue
		}
		os.Rename(path, path+".migrated")
		imported++
	}
	if imported > 0 {
		fmt.Printf("Imported %d transcripts into tubectl.db\n", imported)
	}
	return nil
}
