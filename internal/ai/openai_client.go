package ai

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"
)

const defaultBaseURL = "https://api.openai.com/v1"

// Tracer is an optional observer of LLM calls.
type Tracer interface {
	CreateSpan(ctx context.Context, req SpanRequest) error
}

// SpanRequest captures everything needed to build an OTel span.
type SpanRequest struct {
	TraceID          string
	Name             string
	StartTime        time.Time
	EndTime          time.Time
	Model            string
	Messages         []Message
	Response         string
	FinishReason     string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	LatencyMs        int64
	Tags             map[string]string // surfaced as MLflow trace tags
	Error            string            // non-empty on failure
}

type Client struct {
	apiKey 		string
	model		string
	baseURL		string
	http		*http.Client
	tracer		Tracer
	tags		map[string]string
}

// WithTracer sets a tracer on the client.
func (c *Client) WithTracer(t Tracer) *Client {
	c.tracer = t
	return c
}

// WithTags sets trace tags attached to spans produced by this client.
func (c *Client) WithTags(tags map[string]string) *Client {
	c.tags = tags
	return c
}

// NewClient creates an OpenAI client with the given API key and model.
// Model Defaults to "gpt-40-mini"
func NewClient(apiKey string, model string) *Client {
	if model =="" {
		model = "gpt-4o-mini"
	}

	return &Client{
		apiKey: apiKey,
		model: model,
		baseURL: defaultBaseURL,
		http: &http.Client{},

	}
}

// Message is a single chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Complete sends a chat completion request and returns the reply text.
func (c *Client) Complete(ctx context.Context, messages []Message) (_ string, callErr error) {
	start := time.Now()

	// Trace data populated after the API response is decoded.
	var traceID string
	var finishReason, content string
	var promptTokens, completionTokens, totalTokens int

	if c.tracer != nil {
		tid := make([]byte, 16)
		if _, err := rand.Read(tid); err == nil {
			traceID = hex.EncodeToString(tid)
		}
	}
	defer func() {
		if c.tracer == nil || traceID == "" {
			return
		}
		errStr := ""
		if callErr != nil {
			errStr = callErr.Error()
		}
		spanReq := SpanRequest{
			TraceID:          traceID,
			Name:             "openai_chat",
			StartTime:        start,
			EndTime:          time.Now(),
			Model:            c.model,
			Messages:         messages,
			Response:         content,
			FinishReason:     finishReason,
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      totalTokens,
			LatencyMs:        time.Since(start).Milliseconds(),
			Tags:             c.tags,
			Error:            errStr,
		}
		if serr := c.tracer.CreateSpan(ctx, spanReq); serr != nil {
			fmt.Fprintf(os.Stderr, "Warning: trace span: %v\n", serr)
		}
	}()

	body, err := json.Marshal(map[string]any{
		"model":    c.model,
		"messages": messages,
	})
	if err != nil {
		return "", fmt.Errorf("marshaling request: %w", err)
	}

	u, err := url.JoinPath(c.baseURL, "chat", "completions")
	if err != nil {
		return "", fmt.Errorf("building URL: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("openai request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
		Usage *struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decoding response: %w", err)
	}
	if result.Error != nil {
		return "", fmt.Errorf("openai error: %s", result.Error.Message)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("openai returned no choices")
	}

	content = result.Choices[0].Message.Content
	finishReason = result.Choices[0].FinishReason
	if result.Usage != nil {
		promptTokens = result.Usage.PromptTokens
		completionTokens = result.Usage.CompletionTokens
		totalTokens = result.Usage.TotalTokens
	}

	return content, nil
}
