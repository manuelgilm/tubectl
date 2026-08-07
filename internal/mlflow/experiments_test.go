package mlflow

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_ListExperiments(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/2.0/mlflow/experiments/search" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("view_type"); got != "ACTIVE_ONLY" {
			t.Errorf("view_type = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"experiments": [
				{"id": "0", "name": "Default"},
				{"id": "7", "name": "gateway/chat"}
			]
		}`))
	}))
	defer srv.Close()

	c := NewClient("", "", srv.URL)
	experiments, err := c.ListExperiments(context.Background())
	if err != nil {
		t.Fatalf("ListExperiments: %v", err)
	}
	if len(experiments) != 2 || experiments[1].ID != "7" || experiments[1].Name != "gateway/chat" {
		t.Errorf("experiments = %+v", experiments)
	}

	id, err := c.ExperimentIDByName(context.Background(), "gateway/chat")
	if err != nil {
		t.Fatalf("ExperimentIDByName: %v", err)
	}
	if id != "7" {
		t.Errorf("id = %q", id)
	}

	missing, err := c.ExperimentIDByName(context.Background(), "nope")
	if err != nil {
		t.Fatalf("ExperimentIDByName missing: %v", err)
	}
	if missing != "" {
		t.Errorf("missing id = %q", missing)
	}
}
