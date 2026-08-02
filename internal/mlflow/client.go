package mlflow

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/manuelgilm/tubectl/internal"
)

// Client is a REST client for an MLflow tracking server: the prompt registry
// and trace metadata/read+write operations.
type Client struct {
	httpClient *http.Client
	url        *url.URL
	username   string
	password   string
}

// NewClient creates an MLflow REST client. An empty serverURL falls back to the
// default MLflow server.
func NewClient(username, password, serverURL string) *Client {
	if serverURL == "" {
		serverURL = internal.DefaultMlflowServer
	}
	u, err := url.Parse(serverURL)
	if err != nil {
		u, _ = url.Parse("http://localhost:5000")
	}
	return &Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		url:        u,
		username:   username,
		password:   password,
	}
}

func (c *Client) get(ctx context.Context, path string, params url.Values, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url.JoinPath(path).String(), nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.URL.RawQuery = params.Encode()
	if c.username != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
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

// Ping verifies the MLflow server is reachable. MLflow's /health endpoint
// returns "OK" with status 200 and requires no authentication.
func (c *Client) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url.JoinPath("/health").String(), nil)
	if err != nil {
		return fmt.Errorf("building health request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("MLflow server unreachable: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 128))
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "OK") {
		return fmt.Errorf("MLflow server unhealthy (status %d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}
