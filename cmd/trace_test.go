package cmd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/manuelgilm/tubectl/internal/mlflow"
	"github.com/spf13/cobra"
)

func TestIsAllDigits(t *testing.T) {
	cases := map[string]bool{
		"":     false,
		"0":    true,
		"123":  true,
		"12a":  false,
		"ep-1": false,
	}
	for in, want := range cases {
		if got := isAllDigits(in); got != want {
			t.Errorf("isAllDigits(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestResolveExperimentID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"experiments": [
			{"id": "0", "name": "Default"},
			{"id": "7", "name": "gateway/chat"}
		]}`))
	}))
	defer srv.Close()

	client := mlflow.NewClient("", "", srv.URL)
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	t.Run("empty uses fallback", func(t *testing.T) {
		got, err := resolveExperimentID(cmd, client, "", "5")
		if err != nil || got != "5" {
			t.Fatalf("got %q err %v", got, err)
		}
	})
	t.Run("numeric treated as ID", func(t *testing.T) {
		got, err := resolveExperimentID(cmd, client, "42", "5")
		if err != nil || got != "42" {
			t.Fatalf("got %q err %v", got, err)
		}
	})
	t.Run("name resolved", func(t *testing.T) {
		got, err := resolveExperimentID(cmd, client, "gateway/chat", "5")
		if err != nil || got != "7" {
			t.Fatalf("got %q err %v", got, err)
		}
	})
	t.Run("missing name errors", func(t *testing.T) {
		if _, err := resolveExperimentID(cmd, client, "nope", "5"); err == nil {
			t.Fatal("expected error for unknown experiment name")
		}
	})
}
