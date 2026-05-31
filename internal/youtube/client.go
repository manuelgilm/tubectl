package youtube

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const defaultBaseURL = "https://www.googleapis.com/youtube/v3"

type Client struct {
	httpClient *http.Client
	token      *Token
	cfg        *OAuthConfig
	baseURL    string
}

// NewClientWithToken creates a client that auto-refreshes the token when expired.
func NewClientWithToken(cfg *OAuthConfig, token *Token) *Client {
	return &Client{
		httpClient: &http.Client{},
		token:      token,
		cfg:        cfg,
		baseURL:    defaultBaseURL,
	}
}

// ensureToken refreshes the access token if it has expired and a config is set.
func (c *Client) ensureToken(ctx context.Context) error {
	if c.token.Valid() {
		return nil
	}
	if c.cfg == nil || c.token.RefreshToken == "" {
		return fmt.Errorf("access token expired and no refresh token available")
	}
	tok, err := c.cfg.Refresh(ctx, c.token.RefreshToken)
	if err != nil {
		return fmt.Errorf("refreshing token: %w", err)
	}
	c.token = tok
	return nil
}


func (c *Client) get(ctx context.Context, path string, params map[string]string, out any) error {
	if err := c.ensureToken(ctx); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}

	q := req.URL.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Authorization", "Bearer "+c.token.AccessToken)

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("executing request: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        var apiErr struct {
            Error struct {
                Message string `json:"message"`
                Code    int    `json:"code"`
            } `json:"error"`
        }
        _ = json.NewDecoder(resp.Body).Decode(&apiErr)
        return fmt.Errorf("youtube api error %d: %s", apiErr.Error.Code, apiErr.Error.Message)
    }

    return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) post(ctx context.Context, path string, params map[string]string, body any, out any) error {
	if err := c.ensureToken(ctx); err != nil {
		return err
	}

	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshaling request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}

	q := req.URL.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Authorization", "Bearer "+c.token.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr struct {
			Error struct {
				Message string `json:"message"`
				Code    int    `json:"code"`
			} `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&apiErr)
		return fmt.Errorf("youtube api error %d: %s", apiErr.Error.Code, apiErr.Error.Message)
	}

	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}