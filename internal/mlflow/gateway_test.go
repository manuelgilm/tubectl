package mlflow

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_ListGatewayEndpoints(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/2.0/mlflow/gateway/endpoints/list" {
			t.Errorf("path = %s", r.URL.Path)
		}
		user, pass, ok := r.BasicAuth()
		if !ok || user != "user" || pass != "pass" {
			t.Errorf("basic auth = %q/%q (ok=%v)", user, pass, ok)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"endpoints": [
				{
					"endpointId": "ep-1",
					"name": "chat",
					"routingStrategy": "REQUEST_BASED_TRAFFIC_SPLIT",
					"usageTracking": true,
					"experimentId": "2"
				},
				{
					"endpointId": "ep-2",
					"name": "embed",
					"routingStrategy": "REQUEST_BASED_TRAFFIC_SPLIT",
					"usageTracking": false,
					"experimentId": ""
				}
			]
		}`))
	}))
	defer srv.Close()

	c := NewClient("user", "pass", srv.URL)
	endpoints, err := c.ListGatewayEndpoints(context.Background())
	if err != nil {
		t.Fatalf("ListGatewayEndpoints: %v", err)
	}
	if len(endpoints) != 2 {
		t.Fatalf("len = %d", len(endpoints))
	}
	if endpoints[0].EndpointID != "ep-1" || endpoints[0].Name != "chat" {
		t.Errorf("endpoint[0] = %+v", endpoints[0])
	}
	if !endpoints[0].UsageTracking || endpoints[0].ExperimentID != "2" {
		t.Errorf("endpoint[0] tracking = %+v", endpoints[0])
	}
	if endpoints[1].UsageTracking {
		t.Errorf("endpoint[1] should not track usage: %+v", endpoints[1])
	}
}

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
