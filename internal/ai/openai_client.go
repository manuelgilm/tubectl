package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"fmt"
)

const defaultBaseURL = "https://api.openai.com/v1"

type Client struct {
	apiKey 		string
	model		string
	baseURL		string
	http		*http.Client
}

//New clients creates an OpenAI client witht he given API key and model.
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
func (c *Client) Complete(ctx context.Context, messages []Message) (string, error) {
	body, err := json.Marshal(map[string]any{
		"model":    c.model,
		"messages": messages,
	})
	if err != nil {
		return "", fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/chat/completions", bytes.NewReader(body))
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
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
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
	return result.Choices[0].Message.Content, nil
}
