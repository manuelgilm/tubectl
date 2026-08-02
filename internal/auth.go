package internal

import (
	"context"
	"time"
)

type Provider interface {
	Name() string
	Login(ctx context.Context, opts Options) error
	Logout() error
	Status() (Status, error)
}

type Options struct {
	ClientID     string
	ClientSecret string
	Scopes       []string
	Username     string // for Mlflow basic auth
	Password     string // for Mlflow basic auth
	ServerURL    string // for Mlflow server
}

type Status struct {
	Authenticated bool
	ExpiresAt     time.Time
}
