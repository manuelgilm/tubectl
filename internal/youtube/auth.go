package youtube

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	authURL  = "https://accounts.google.com/o/oauth2/v2/auth"
	tokenURL = "https://oauth2.googleapis.com/token"

	// ScopeYoutube manages a YouTube account (broadest scope).
	ScopeYoutube = "https://www.googleapis.com/auth/youtube"
	// ScopeYoutubeReadonly grants read-only access to public data.
	ScopeYoutubeReadonly = "https://www.googleapis.com/auth/youtube.readonly"
	// ScopeYoutubeForceSsl allows reading and writing videos, comments, and captions.
	// Required for posting replies.
	ScopeYoutubeForceSsl = "https://www.googleapis.com/auth/youtube.force-ssl"
)

// OAuthConfig holds the credentials from your Google Cloud Console OAuth2 client.
type OAuthConfig struct {
	ClientID      string
	ClientSecret  string
	RedirectURI   string
	Scopes        []string
	// tokenEndpoint overrides the Google token URL; used in tests.
	tokenEndpoint string
}

// Token represents an OAuth2 token pair.
type Token struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// Valid reports whether the token is set and not expired.
func (t *Token) Valid() bool {
	return t != nil && t.AccessToken != "" && time.Now().Before(t.ExpiresAt)
}

// AuthCodeURL returns the Google consent URL. Direct the user to open this in
// their browser. After granting access, Google redirects to RedirectURI with a
// "code" query parameter (or displays it for the OOB flow).
func (cfg *OAuthConfig) AuthCodeURL(state string) string {
	v := url.Values{
		"client_id":     {cfg.ClientID},
		"redirect_uri":  {cfg.RedirectURI},
		"response_type": {"code"},
		"scope":         {strings.Join(cfg.Scopes, " ")},
		"access_type":   {"offline"},
		"prompt":        {"consent"},
		"state":         {state},
	}
	return authURL + "?" + v.Encode()
}

// tokenResponse is the raw JSON body returned by the token endpoint.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Error        string `json:"error"`
	ErrorDesc    string `json:"error_description"`
}

// Exchange trades an authorization code for an access+refresh token pair.
func (cfg *OAuthConfig) Exchange(ctx context.Context, code string) (*Token, error) {
	return cfg.exchange(ctx, url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code},
		"redirect_uri": {cfg.RedirectURI},
	})
}

// Refresh uses a refresh token to obtain a new access token.
func (cfg *OAuthConfig) Refresh(ctx context.Context, refreshToken string) (*Token, error) {
	return cfg.exchange(ctx, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	})
}

// exchangeWithURL is the testable core of Exchange/Refresh; endpoint is overridable.
func exchangeWithURL(ctx context.Context, cfg *OAuthConfig, endpoint string, extra url.Values) (*Token, error) {
	body := url.Values{
		"client_id":     {cfg.ClientID},
		"client_secret": {cfg.ClientSecret},
	}
	for k, vs := range extra {
		body[k] = vs
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint,
		strings.NewReader(body.Encode()))
	if err != nil {
		return nil, fmt.Errorf("building token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	var tr tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return nil, fmt.Errorf("decoding token response: %w", err)
	}
	if tr.Error != "" {
		return nil, fmt.Errorf("oauth error %s: %s", tr.Error, tr.ErrorDesc)
	}

	tok := &Token{
		AccessToken: tr.AccessToken,
		TokenType:   tr.TokenType,
		ExpiresAt:   time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second),
	}
	if tr.RefreshToken != "" {
		tok.RefreshToken = tr.RefreshToken
	} else {
		tok.RefreshToken = extra.Get("refresh_token")
	}
	return tok, nil
}

func (cfg *OAuthConfig) exchange(ctx context.Context, extra url.Values) (*Token, error) {
	endpoint := tokenURL
	if cfg.tokenEndpoint != "" {
		endpoint = cfg.tokenEndpoint
	}
	return exchangeWithURL(ctx, cfg, endpoint, extra)
}

// SaveToken persists a token to a JSON file.
func SaveToken(path string, tok *Token) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("opening token file: %w", err)
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(tok)
}

// LoadToken reads a token from a JSON file.
func LoadToken(path string) (*Token, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening token file: %w", err)
	}
	defer f.Close()

	var tok Token
	if err := json.NewDecoder(f).Decode(&tok); err != nil {
		return nil, fmt.Errorf("decoding token file: %w", err)
	}
	return &tok, nil
}
