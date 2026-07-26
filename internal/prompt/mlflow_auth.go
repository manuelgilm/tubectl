package prompt

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/manuelgilm/tubectl/internal"
)

type Credentials struct {
	ServerURL string `json:"server_url"`
	Username  string `json:"username"`
	Password  string `json:"password"`
}

func SaveCredentials(path string, c *Credentials) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func LoadCredentials(path string) (*Credentials, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Credentials
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

type MLflowProvider struct {
	credentialsPath string
}

func NewMLflowProvider(credentialsPath string) *MLflowProvider {
	return &MLflowProvider{credentialsPath: credentialsPath}
}

func (p *MLflowProvider) Name() string { return "mlflow" }

func (p *MLflowProvider) Login(_ context.Context, opts internal.Options) error {
	if opts.Username == "" || opts.Password == "" {
		return fmt.Errorf("username and password are required")
	}
	return SaveCredentials(p.credentialsPath, &Credentials{
		ServerURL: opts.ServerURL,
		Username:  opts.Username,
		Password:  opts.Password,
	})
}

func (p *MLflowProvider) Logout() error {
	err := os.Remove(p.credentialsPath)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (p *MLflowProvider) Status() (internal.Status, error) {
	if _, err := os.Stat(p.credentialsPath); os.IsNotExist(err) {
		return internal.Status{}, fmt.Errorf("not authenticated")
	}
	return internal.Status{Authenticated: true}, nil
}
