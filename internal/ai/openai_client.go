package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.openai.com/v1"

// ConnectivityError indicates the request never reached or completed against
// the server (a transport-level failure), as opposed to an HTTP response the
// server did return (non-2xx, error body, empty choices...). Callers use it to
// distinguish "server unreachable" from "server answered and disliked/misanswered
// the request".
type ConnectivityError struct {
	Err error
}

func (e *ConnectivityError) Error() string {
	return "openai request failed: " + e.Err.Error()
}

func (e *ConnectivityError) Unwrap() error {
	return e.Err
}

// HTTPStatusError represents a non-2xx response from the server, carrying the
// HTTP status code and the server-provided error body so callers can decide
// whether a fallback is appropriate (e.g. a gateway 404 meaning the model is
// not deployed there).
type HTTPStatusError struct {
	StatusCode int
	Body       string
}

func (e *HTTPStatusError) Error() string {
	if e.Body == "" {
		return fmt.Sprintf("openai request returned status %d", e.StatusCode)
	}
	return fmt.Sprintf("openai request returned status %d: %s", e.StatusCode, e.Body)
}

type Client struct {
	apiKey   string
	model    string
	baseURL  string
	username string
	password string
	http     *http.Client
}

// BaseURL returns the base URL this client targets (used to distinguish the
// MLflow gateway from direct OpenAI).
func (c *Client) BaseURL() string {
	return c.baseURL
}

// WithBasicAuth switches authentication from Bearer (OpenAI) to HTTP Basic
// auth, used when talking to the MLflow gateway rather than OpenAI directly.
func (c *Client) WithBasicAuth(username, password string) *Client {
	c.username = username
	c.password = password
	return c
}

// WithBaseURL overrides the OpenAI API base URL (e.g. a self-hosted OpenAI
// gateway server).
func (c *Client) WithBaseURL(baseURL string) *Client {
	c.baseURL = baseURL
	return c
}

// WithHTTPClient sets the underlying HTTP client (mainly for tests).
func (c *Client) WithHTTPClient(h *http.Client) *Client {
	c.http = h
	return c
}

// NewClient creates an OpenAI client with the given API key and model.
// Model defaults to "gpt-4o-mini".
func NewClient(apiKey string, model string) *Client {
	if model == "" {
		model = "gpt-4o-mini"
	}

	return &Client{
		apiKey:  apiKey,
		model:   model,
		baseURL: defaultBaseURL,
		http:    &http.Client{Timeout: 30 * time.Second},
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

	u, err := url.JoinPath(c.baseURL, "chat", "completions")
	if err != nil {
		return "", fmt.Errorf("building URL: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("building request: %w", err)
	}
	if c.username != "" || c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	} else {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", &ConnectivityError{Err: err}
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	// Check the status code first so decoder/decoding errors never mask a
	// server that rejected the request. Non-2xx responses are surfaced with
	// the server's body for diagnostics.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", &HTTPStatusError{StatusCode: resp.StatusCode, Body: strings.TrimSpace(string(bodyBytes))}
	}

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
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
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
