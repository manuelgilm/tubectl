package youtube

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/manuelgilm/tubectl/internal"
)

const (
	// ScopeYoutube manages a YouTube account (broadest scope).
	ScopeYoutube = "https://www.googleapis.com/auth/youtube"
	// ScopeYoutubeReadonly grants read-only access to public data.
	ScopeYoutubeReadonly = "https://www.googleapis.com/auth/youtube.readonly"
	// ScopeYoutubeForceSsl allows reading and writing videos, comments, and captions.
	// Required for posting replies.
	ScopeYoutubeForceSsl = "https://www.googleapis.com/auth/youtube.force-ssl"
)

type YouTubeProvider struct {
	tokenPath string         // $HOME/.tubectl/auth/youtube
	config    *oauth2.Config // cached for refresh
}

func NewYoutubeProvider(tokenPath string) *YouTubeProvider {
	return &YouTubeProvider{
		tokenPath: tokenPath,
		config:    nil,
	}
}

func (p *YouTubeProvider) Name() string { return "youtube" }

func (p *YouTubeProvider) Login(ctx context.Context, opts internal.Options) error {
	config, listener, err := youtubeConfigFromEnv()
	if err != nil {
		return err
	}
	defer listener.Close()
	// no valid token - do full OAuth flow
	p.config = config // To use later

	//Channel to receive the auth code from the http handler
	codeCh := make(chan string)

	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		return fmt.Errorf("generating state: %w", err)
	}
	state := hex.EncodeToString(stateBytes)

	// Reads ?code= from query
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			http.Error(w, "state mismatch", http.StatusForbidden)
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "code not found!", http.StatusBadRequest)
			return
		}
		codeCh <- code
		fmt.Fprint(w, "Success!")
	})

	srv := &http.Server{Handler: mux}
	go srv.Serve(listener)

	authURL := config.AuthCodeURL(state)
	fmt.Println("Open this URL in your browser:", authURL)

	select {
	case code := <-codeCh:
		token, err := config.Exchange(ctx, code)
		srv.Shutdown(ctx)
		if err != nil {
			return fmt.Errorf("token exchange: %w", err)
		}
		return SaveToken(p.tokenPath, tokenFromOAuth2(token))
	case <-ctx.Done():
		srv.Shutdown(ctx)
		return ctx.Err()
	}

}
func (p *YouTubeProvider) Logout() error {
	var exist bool = exists(p.tokenPath)
	if exist {
		err := os.Remove(p.tokenPath)
		if err != nil {
			return err
		}
	}
	// The file does not exist. Nothing to logout
	return nil
}

func (p *YouTubeProvider) Status() (internal.Status, error) {
	var exist bool = exists(p.tokenPath)
	if !exist {
		return internal.Status{}, errors.New("Token not found")
	}

	//read token
	token, err := LoadToken(p.tokenPath)
	if err != nil {
		return internal.Status{}, fmt.Errorf("loading token: %w", err)
	}

	return internal.Status{
		Authenticated: time.Now().Before(token.ExpiresAt),
		ExpiresAt:     token.ExpiresAt,
	}, nil
}
func tokenFromOAuth2(t *oauth2.Token) *Token {
	return &Token{
		AccessToken:  t.AccessToken,
		RefreshToken: t.RefreshToken,
		TokenType:    t.TokenType,
		ExpiresAt:    t.Expiry,
	}
}

// RefreshToken attempts to refresh an expired OAuth token using the stored
// refresh token. The YOUTUBE_CLIENT_ID and YOUTUBE_CLIENT_SECRET environment
// variables must be set for the refresh to succeed.
func RefreshToken(ctx context.Context, tokenPath string) (*Token, error) {
	clientID := os.Getenv("YOUTUBE_CLIENT_ID")
	clientSecret := os.Getenv("YOUTUBE_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("YOUTUBE_CLIENT_ID and YOUTUBE_CLIENT_SECRET must be set to refresh the token")
	}

	token, err := LoadToken(tokenPath)
	if err != nil {
		return nil, fmt.Errorf("loading token for refresh: %w", err)
	}

	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
	}

	ts := config.TokenSource(ctx, &oauth2.Token{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		Expiry:       token.ExpiresAt,
	})

	newToken, err := ts.Token()
	if err != nil {
		return nil, fmt.Errorf("token refresh failed: %w", err)
	}

	saved := tokenFromOAuth2(newToken)
	if err := SaveToken(tokenPath, saved); err != nil {
		return nil, fmt.Errorf("saving refreshed token: %w", err)
	}

	return saved, nil
}

func youtubeConfigFromEnv() (*oauth2.Config, net.Listener, error) {
	clientID := os.Getenv("YOUTUBE_CLIENT_ID")
	clientSecret := os.Getenv("YOUTUBE_CLIENT_SECRET")

	if clientID == "" || clientSecret == "" {
		return nil, nil, fmt.Errorf("YOUTUBE_CLIENT_ID and YOUTUBE_CLIENT_SECRET must be set")
	}

	//pick random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to pick port: %w", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port
	redirectURL := fmt.Sprintf("http://127.0.0.1:%d/callback", port)

	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{ScopeYoutubeForceSsl},
		Endpoint:     google.Endpoint,
	}, listener, nil
}
func LoadToken(path string) (*Token, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading token file: %w", err)
	}
	var token Token
	err = json.Unmarshal(data, &token)
	if err != nil {
		return nil, fmt.Errorf("decoding token file: %w", err)
	}
	return &token, nil
}

// Valid reports whether the token is set and not expired.
func (t *Token) Valid() bool {
	return t != nil && t.AccessToken != "" && time.Now().Before(t.ExpiresAt)
}

type Token struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// Utilities
func exists(filepath string) bool {
	_, err := os.Stat(filepath)
	if err == nil {
		return true // File exists
	}

	if errors.Is(err, os.ErrNotExist) {
		return false // File explicitily does not exist
	}
	// It may exist but it cannot be accessed
	return false
}
