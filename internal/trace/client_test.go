package trace

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClient_GetTrace(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/3.0/mlflow/traces/get" {
			t.Errorf("path = %s", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("trace_id") != "tr-abc" {
			t.Errorf("trace_id = %q", q.Get("trace_id"))
		}
		if q.Get("allow_partial") != "true" {
			t.Errorf("allow_partial = %q", q.Get("allow_partial"))
		}
		user, pass, ok := r.BasicAuth()
		if !ok || user != "user" || pass != "pass" {
			t.Errorf("basic auth = %q/%q (ok=%v)", user, pass, ok)
		}
		spanID := base64.StdEncoding.EncodeToString([]byte{0xde, 0xad})
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"trace": {
				"trace_info": {
					"trace_id": "tr-abc",
					"client_request_id": "x",
					"trace_location": {"type": "TRACE_LOCATION_TYPE_MLFLOW_EXPERIMENT", "mlflow_experiment": {"experiment_id": "0"}},
					"request_preview": "Hi",
					"response_preview": "Hello!",
					"request_time": "2024-07-03T09:46:40.123Z",
					"execution_duration": "2.500s",
					"state": "OK",
					"trace_metadata": {"mlflow.artifact_uri": "s3://x"},
					"tags": {"source": "youtube-comment"}
				},
				"spans": [
					{
						"name": "openai_chat",
						"span_id": "` + spanID + `",
						"parent_span_id": "",
						"start_time_unix_nano": 1720000000123000000,
						"end_time_unix_nano": 1720000002600000000,
						"attributes": [
							{"key": "gen_ai.request.model", "value": {"stringValue": "gpt-4o-mini"}},
							{"key": "gen_ai.usage.input_tokens", "value": {"intValue": 10}}
						],
						"status": {"code": 1, "message": ""}
					}
				]
			}
		}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "user", "pass")
	tr, err := c.GetTrace(context.Background(), "tr-abc")
	if err != nil {
		t.Fatalf("GetTrace: %v", err)
	}
	if tr.TraceInfo.TraceID != "tr-abc" {
		t.Errorf("trace_id = %q", tr.TraceInfo.TraceID)
	}
	if tr.TraceInfo.State != "OK" {
		t.Errorf("state = %q", tr.TraceInfo.State)
	}
	if tr.TraceInfo.RequestPreview != "Hi" || tr.TraceInfo.ResponsePreview != "Hello!" {
		t.Errorf("previews = %q / %q", tr.TraceInfo.RequestPreview, tr.TraceInfo.ResponsePreview)
	}
	if tr.TraceInfo.Tags["source"] != "youtube-comment" {
		t.Errorf("tags = %v", tr.TraceInfo.Tags)
	}
	if len(tr.Spans) != 1 {
		t.Fatalf("spans = %d", len(tr.Spans))
	}
	span := tr.Spans[0]
	if span.Name != "openai_chat" {
		t.Errorf("span name = %q", span.Name)
	}
	if span.StartTimeUnixNano == 0 || span.EndTimeUnixNano == 0 {
		t.Error("missing span timestamps")
	}
	if len(span.Attributes) != 2 || span.Attributes[0].Key != "gen_ai.request.model" {
		t.Errorf("span attributes = %v", span.Attributes)
	}
	if span.Status == nil || span.Status.Code != 1 {
		t.Errorf("span status = %+v", span.Status)
	}
}

func TestClient_ListTraces(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/2.0/mlflow/traces" {
			t.Errorf("path = %s", r.URL.Path)
		}
		q := r.URL.Query()
		if got := q.Get("experiment_ids"); got != "0" {
			t.Errorf("experiment_ids = %q", got)
		}
		if got := q.Get("max_results"); got != "10" {
			t.Errorf("max_results = %q", got)
		}
		if got := q.Get("order_by"); got != "timestamp_ms DESC" {
			t.Errorf("order_by = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"traces": [
				{
					"request_id": "tr-abc",
					"experiment_id": "0",
					"timestamp_ms": 1720000000000,
					"execution_time_ms": 2500,
					"status": "OK",
					"request_metadata": [{"key": "k", "value": "v"}],
					"tags": [{"key": "source", "value": "youtube-comment"}]
				}
			],
			"next_page_token": ""
		}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "")
	items, err := c.ListTraces(context.Background(), nil, 10)
	if err != nil {
		t.Fatalf("ListTraces: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items = %d", len(items))
	}
	it := items[0]
	if it.RequestID != "tr-abc" {
		t.Errorf("request_id = %q", it.RequestID)
	}
	if it.Status != "OK" {
		t.Errorf("status = %q", it.Status)
	}
	if len(it.Tags) != 1 || it.Tags[0].Key != "source" || it.Tags[0].Value != "youtube-comment" {
		t.Errorf("tags = %v", it.Tags)
	}
}

func TestClient_ListTraces_Defaults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if got := q.Get("max_results"); got != "20" {
			t.Errorf("max_results = %q, want 20", got)
		}
		w.Write([]byte(`{"traces": []}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "")
	if _, err := c.ListTraces(context.Background(), nil, 0); err != nil {
		t.Fatalf("ListTraces: %v", err)
	}
}

func TestClient_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error_code": "RESOURCE_DOES_NOT_EXIST", "message": "Trace tr-xyz not found"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "")
	_, err := c.GetTrace(context.Background(), "tr-xyz")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "404") {
		// status text uses the numeric status code from the wire
		t.Logf("error = %v", err)
	}
}
