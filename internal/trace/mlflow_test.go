package trace

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	otlpcollectortrace "go.opentelemetry.io/proto/otlp/collector/trace/v1"

	"tubectl/internal/ai"
)

func TestStartTrace(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/ajax-api/2.0/mlflow/traces" {
			t.Errorf("path = %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		if err := json.Unmarshal(body, &req); err != nil {
			t.Errorf("json: %v", err)
		}
		if req["experiment_id"] != "0" {
			t.Errorf("experiment_id = %v", req["experiment_id"])
		}
		if _, ok := req["timestamp_ms"]; !ok {
			t.Error("missing timestamp_ms")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"trace_info": map[string]any{
				"request_id": "tr-test-request-id",
			},
		})
	}))
	defer srv.Close()

	tr := NewMLflowTracer(srv.URL, "user", "pass")
	requestID, err := tr.StartTrace(context.Background(), "")
	if err != nil {
		t.Fatalf("StartTrace: %v", err)
	}
	if requestID != "tr-test-request-id" {
		t.Errorf("requestID = %q", requestID)
	}
}

func TestStartTrace_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("unauthorized"))
	}))
	defer srv.Close()

	tr := NewMLflowTracer(srv.URL, "", "")
	_, err := tr.StartTrace(context.Background(), "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error = %v, want 401", err)
	}
}

func TestStartTrace_EmptyRequestID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"trace_info": map[string]any{
				"request_id": "",
			},
		})
	}))
	defer srv.Close()

	tr := NewMLflowTracer(srv.URL, "u", "p")
	_, err := tr.StartTrace(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty request_id")
	}
}

func TestCreateSpan(t *testing.T) {
	var gotProto bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v1/traces" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/x-protobuf" {
			t.Errorf("Content-Type = %q", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("x-mlflow-experiment-id") != "0" {
			t.Errorf("x-mlflow-experiment-id = %q", r.Header.Get("x-mlflow-experiment-id"))
		}
		body, _ := io.ReadAll(r.Body)
		var exportReq otlpcollectortrace.ExportTraceServiceRequest
		if err := proto.Unmarshal(body, &exportReq); err != nil {
			t.Errorf("unmarshal protobuf: %v", err)
		}
		gotProto = true
		if len(exportReq.ResourceSpans) != 1 {
			t.Fatalf("ResourceSpans count = %d", len(exportReq.ResourceSpans))
		}
		rs := exportReq.ResourceSpans[0]
		if len(rs.ScopeSpans) != 1 {
			t.Fatalf("ScopeSpans count = %d", len(rs.ScopeSpans))
		}
		ss := rs.ScopeSpans[0]
		if len(ss.Spans) != 1 {
			t.Fatalf("Spans count = %d", len(ss.Spans))
		}
		span := ss.Spans[0]
		if span.Name != "openai_chat" {
			t.Errorf("span name = %q", span.Name)
		}
		if span.StartTimeUnixNano == 0 || span.EndTimeUnixNano == 0 {
			t.Error("missing timestamps")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tr := NewMLflowTracer(srv.URL, "user", "pass")
	now := time.Now()
	err := tr.CreateSpan(context.Background(), ai.SpanRequest{
		TraceID:          "tr-4bf92f3577b34da6a3ce929d0e0e4736",
		Name:             "openai_chat",
		StartTime:        now.Add(-time.Second),
		EndTime:          now,
		Model:            "gpt-4o-mini",
		Messages:         []ai.Message{{Role: "user", Content: "Hi"}},
		Response:         "Hello!",
		FinishReason:     "stop",
		PromptTokens:     10,
		CompletionTokens: 20,
		TotalTokens:      30,
		LatencyMs:        1000,
	})
	if err != nil {
		t.Fatalf("CreateSpan: %v", err)
	}
	if !gotProto {
		t.Error("server did not receive protobuf request")
	}
}

func TestCreateSpan_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	tr := NewMLflowTracer(srv.URL, "", "")
	err := tr.CreateSpan(context.Background(), ai.SpanRequest{
		TraceID:   "tr-4bf92f3577b34da6a3ce929d0e0e4736",
		Name:      "test",
		StartTime: time.Now(),
		EndTime:   time.Now(),
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEndTrace(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Errorf("method = %s, want PATCH", r.Method)
		}
		if r.URL.Path != "/ajax-api/2.0/mlflow/traces/tr-test-id" {
			t.Errorf("path = %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var req map[string]string
		if err := json.Unmarshal(body, &req); err != nil {
			t.Errorf("json: %v", err)
		}
		if req["status"] != "OK" {
			t.Errorf("status = %q", req["status"])
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tr := NewMLflowTracer(srv.URL, "", "")
	err := tr.EndTrace(context.Background(), "tr-test-id", "OK")
	if err != nil {
		t.Fatalf("EndTrace: %v", err)
	}
}

func TestEndTrace_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	tr := NewMLflowTracer(srv.URL, "", "")
	err := tr.EndTrace(context.Background(), "tr-missing", "OK")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWithExperimentID(t *testing.T) {
	tr := NewMLflowTracer("http://localhost:5000", "", "")
	if tr.experimentID != "0" {
		t.Errorf("default experimentID = %q", tr.experimentID)
	}
	tr.WithExperimentID("42")
	if tr.experimentID != "42" {
		t.Errorf("experimentID = %q after WithExperimentID", tr.experimentID)
	}
}

func TestStartTrace_WithExperimentID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		json.Unmarshal(body, &req)
		if req["experiment_id"] != "99" {
			t.Errorf("experiment_id = %v", req["experiment_id"])
		}
		json.NewEncoder(w).Encode(map[string]any{
			"trace_info": map[string]any{"request_id": "tr-xxx"},
		})
	}))
	defer srv.Close()

	tr := NewMLflowTracer(srv.URL, "", "")
	tr.WithExperimentID("99")
	requestID, err := tr.StartTrace(context.Background(), "")
	if err != nil {
		t.Fatalf("StartTrace: %v", err)
	}
	if requestID != "tr-xxx" {
		t.Errorf("requestID = %q", requestID)
	}
}

func TestCreateSpan_WithError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var exportReq otlpcollectortrace.ExportTraceServiceRequest
		if err := proto.Unmarshal(body, &exportReq); err != nil {
			t.Errorf("unmarshal: %v", err)
		}
		span := exportReq.ResourceSpans[0].ScopeSpans[0].Spans[0]
		if span.Status.Code != 2 { // STATUS_CODE_ERROR
			t.Errorf("status code = %d, want 2 (ERROR)", span.Status.Code)
		}
		// Check error attribute
		var found bool
		for _, attr := range span.Attributes {
			if attr.Key == "error" {
				found = true
				if attr.Value.GetStringValue() != "something went wrong" {
					t.Errorf("error attribute = %q", attr.Value.GetStringValue())
				}
			}
		}
		if !found {
			t.Error("missing error attribute")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tr := NewMLflowTracer(srv.URL, "", "")
	now := time.Now()
	err := tr.CreateSpan(context.Background(), ai.SpanRequest{
		TraceID:   "tr-4bf92f3577b34da6a3ce929d0e0e4736",
		Name:      "openai_chat",
		StartTime: now,
		EndTime:   now,
		Error:     "something went wrong",
	})
	if err != nil {
		t.Fatalf("CreateSpan: %v", err)
	}
}

func TestBasicAuth(t *testing.T) {
	var authHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		if r.URL.Path == "/ajax-api/2.0/mlflow/traces" {
			json.NewEncoder(w).Encode(map[string]any{
				"trace_info": map[string]any{"request_id": "tr-xxx"},
			})
		}
	}))
	defer srv.Close()

	tr := NewMLflowTracer(srv.URL, "myuser", "mypass")
	_, err := tr.StartTrace(context.Background(), "")
	if err != nil {
		t.Fatalf("StartTrace: %v", err)
	}
	if authHeader == "" {
		t.Fatal("missing Authorization header")
	}
	if !strings.HasPrefix(authHeader, "Basic ") {
		t.Errorf("Authorization header = %q", authHeader)
	}
}

func TestKVHelper(t *testing.T) {
	tests := []struct {
		key   string
		value any
	}{
		{"s", "hello"},
		{"i", int64(42)},
		{"f", float64(3.14)},
	}
	for _, tc := range tests {
		kv := kv(tc.key, tc.value)
		if kv.Key != tc.key {
			t.Errorf("key = %q", kv.Key)
		}
	}
}

func TestFormatMessages(t *testing.T) {
	msgs := []ai.Message{
		{Role: "user", Content: "Hi"},
		{Role: "assistant", Content: "Hello!"},
	}
	got := formatMessages(msgs)
	if !strings.Contains(got, "user") || !strings.Contains(got, "Hi") {
		t.Errorf("formatMessages = %q", got)
	}
}

func TestCreateSpan_InvalidTraceID(t *testing.T) {
	tr := NewMLflowTracer("http://localhost:5000", "", "")
	err := tr.CreateSpan(context.Background(), ai.SpanRequest{
		TraceID: "invalid-hex",
	})
	if err == nil {
		t.Fatal("expected error for invalid trace ID")
	}
}
