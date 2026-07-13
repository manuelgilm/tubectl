package prompt

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const defaultMlflowServer = "https://sandbox-mlflow.gilmanuel.com"

type Client struct {
	httpClient *http.Client
	baseURL    string
	username   string
	password   string
}

func NewClient(username, password string) *Client {
	return &Client{
		httpClient: &http.Client{},
		baseURL:    defaultMlflowServer,
		username:   username,
		password:   password,
	}
}

func (c *Client) get(ctx context.Context, path string, params url.Values, out any) error {
	u, _ := url.JoinPath(c.baseURL, path)
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
		return fmt.Errorf("MLflow API error (status %d)", resp.StatusCode)
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