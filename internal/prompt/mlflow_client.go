package prompt

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const DefaultMlflowServer = "https://sandbox-mlflow.gilmanuel.com"

type Client struct {
	httpClient *http.Client
	baseURL    string
	username   string
	password   string
}

func NewClient(username, password, serverURL string) *Client {
	if serverURL == "" {
		serverURL = DefaultMlflowServer
	}
	return &Client{
		httpClient: &http.Client{},
		baseURL:    serverURL,
		username:   username,
		password:   password,
	}
}

func (c *Client) get(ctx context.Context, path string, params url.Values, out any) error {
	u, err := url.JoinPath(c.baseURL, path)
	if err != nil {
		return fmt.Errorf("building URL: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.URL.RawQuery = params.Encode()
	req.SetBasicAuth(c.username, c.password)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("MLflow API error (status %d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if out != nil {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("reading response body: %w", err)
		}
		return json.Unmarshal(body, out)
	}
	return nil
}

func (c *Client) GetPrompt(ctx context.Context, name string) (*RegisteredModel, error) {
	params := url.Values{}
	params.Set("name", name)

	var resp RegisteredModelResponse
	if err := c.get(ctx, "/api/2.0/mlflow/registered-models/get", params, &resp); err != nil {
		return nil, fmt.Errorf("fetching prompt %q: %w", name, err)
	}

	return &resp.RegisteredModel, nil
}

func (c *Client) ListPrompts(ctx context.Context) ([]ModelVersion, error) {
    params := url.Values{}
    params.Set("filter", `tag.mlflow.prompt.is_prompt = "true"`)

    var resp ModelVersionSearchResponse
    if err := c.get(ctx, "/api/2.0/mlflow/model-versions/search", params, &resp); err != nil {
        return nil, fmt.Errorf("listing prompts: %w", err)
    }
    return resp.ModelVersions, nil
}