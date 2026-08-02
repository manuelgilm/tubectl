package trace

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	otlpcollectortrace "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	otlpcommon "go.opentelemetry.io/proto/otlp/common/v1"
	otlptrace "go.opentelemetry.io/proto/otlp/trace/v1"

	"github.com/manuelgilm/tubectl/internal/ai"
)

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
		gotAttrs := map[string]*otlpcommon.AnyValue{}
		for _, attr := range span.Attributes {
			gotAttrs[attr.Key] = attr.Value
		}
		for key, want := range map[string]any{
			"mlflow.traceName":               "openai_chat",
			"mlflow.spanInputs":              `{"messages":[{"role":"user","content":"Hi"}]}`,
			"mlflow.spanOutputs":             `{"messages":[{"role":"assistant","content":"Hello!"}]}`,
			"gen_ai.operation.name":          "chat",
			"gen_ai.usage.input_tokens":      int64(10),
			"gen_ai.usage.output_tokens":     int64(20),
			"gen_ai.request.model":           "gpt-4o-mini",
			"gen_ai.provider.name":           "openai",
			"gen_ai.response.finish_reasons": []string{"stop"},
			"mlflow.traceTag.source":         "youtube-comment",
			"mlflow.traceTag.comment_id":     "abc123",
		} {
			val, ok := gotAttrs[key]
			if !ok {
				t.Errorf("missing attribute %q", key)
				continue
			}
			switch want := want.(type) {
			case string:
				if got := val.GetStringValue(); got != want {
					t.Errorf("attribute %q = %q, want %q", key, got, want)
				}
			case int64:
				if got := val.GetIntValue(); got != want {
					t.Errorf("attribute %q = %d, want %d", key, got, want)
				}
			case []string:
				arr := val.GetArrayValue()
				if arr == nil || len(arr.Values) != len(want) {
					t.Errorf("attribute %q array length = %d, want %d", key, len(arr.Values), len(want))
					continue
				}
				for i, s := range want {
					if got := arr.Values[i].GetStringValue(); got != s {
						t.Errorf("attribute %q[%d] = %q, want %q", key, i, got, s)
					}
				}
			}
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tr := NewMLflowTracer(srv.URL, "user", "pass")
	now := time.Now()
	err := tr.CreateSpan(context.Background(), ai.SpanRequest{
		TraceID:          "4bf92f3577b34da6a3ce929d0e0e4736",
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
		Tags: map[string]string{
			"source":     "youtube-comment",
			"comment_id": "abc123",
		},
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
		TraceID:   "4bf92f3577b34da6a3ce929d0e0e4736",
		Name:      "test",
		StartTime: time.Now(),
		EndTime:   time.Now(),
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCreateSpan_TrailingSlashBaseURL(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tr := NewMLflowTracer(srv.URL+"/", "", "")
	err := tr.CreateSpan(context.Background(), ai.SpanRequest{
		TraceID:   "4bf92f3577b34da6a3ce929d0e0e4736",
		Name:      "test",
		StartTime: time.Now(),
		EndTime:   time.Now(),
	})
	if err != nil {
		t.Fatalf("CreateSpan: %v", err)
	}
	if gotPath != "/v1/traces" {
		t.Errorf("path = %q, want /v1/traces", gotPath)
	}
}

func TestCreateSpan_ErrorBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error_code": "PERMISSION_DENIED", "message": "denied"}`))
	}))
	defer srv.Close()

	tr := NewMLflowTracer(srv.URL, "", "")
	err := tr.CreateSpan(context.Background(), ai.SpanRequest{
		TraceID:   "4bf92f3577b34da6a3ce929d0e0e4736",
		Name:      "test",
		StartTime: time.Now(),
		EndTime:   time.Now(),
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "denied") {
		t.Errorf("error does not include response body: %v", err)
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
		TraceID:   "4bf92f3577b34da6a3ce929d0e0e4736",
		Name:      "openai_chat",
		StartTime: now,
		EndTime:   now,
		Error:     "something went wrong",
	})
	if err != nil {
		t.Fatalf("CreateSpan: %v", err)
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
		{"a", []string{"stop", "length"}},
	}
	for _, tc := range tests {
		kv := kv(tc.key, tc.value)
		if kv.Key != tc.key {
			t.Errorf("key = %q", kv.Key)
		}
		if arr, ok := tc.value.([]string); ok {
			got := kv.Value.GetArrayValue()
			if got == nil || len(got.Values) != len(arr) {
				t.Errorf("key %q: array = %v, want %v", tc.key, got, arr)
				continue
			}
			for i, s := range arr {
				if g := got.Values[i].GetStringValue(); g != s {
					t.Errorf("key %q[%d] = %q, want %q", tc.key, i, g, s)
				}
			}
		}
	}
}

func TestFormatSpanInputs(t *testing.T) {
	msgs := []ai.Message{
		{Role: "user", Content: "Hi"},
	}
	got := formatSpanInputs(msgs)
	if !strings.Contains(got, "messages") || !strings.Contains(got, "Hi") {
		t.Errorf("formatSpanInputs = %q", got)
	}
}

func TestFormatSpanOutputs(t *testing.T) {
	got := formatSpanOutputs("Hello!")
	if !strings.Contains(got, "assistant") || !strings.Contains(got, "Hello!") {
		t.Errorf("formatSpanOutputs = %q", got)
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

func TestCreateSpan_ExportsWithCancelledContext(t *testing.T) {
	received := make(chan struct{}, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received <- struct{}{}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	tr := NewMLflowTracer(srv.URL, "", "")
	err := tr.CreateSpan(ctx, ai.SpanRequest{
		TraceID:   "4bf92f3577b34da6a3ce929d0e0e4736",
		Name:      "test",
		StartTime: time.Now(),
		EndTime:   time.Now(),
	})
	if err != nil {
		t.Fatalf("CreateSpan with cancelled context: %v", err)
	}
	select {
	case <-received:
	case <-time.After(5 * time.Second):
		t.Fatal("server did not receive span despite cancelled context")
	}
}

func TestCreateSpan_ZeroTimestamps(t *testing.T) {
	var gotSpan *otlptrace.Span
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var exportReq otlpcollectortrace.ExportTraceServiceRequest
		if err := proto.Unmarshal(body, &exportReq); err != nil {
			t.Errorf("unmarshal: %v", err)
		}
		gotSpan = exportReq.ResourceSpans[0].ScopeSpans[0].Spans[0]
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tr := NewMLflowTracer(srv.URL, "", "")
	err := tr.CreateSpan(context.Background(), ai.SpanRequest{
		TraceID: "4bf92f3577b34da6a3ce929d0e0e4736",
		Name:    "test",
	})
	if err != nil {
		t.Fatalf("CreateSpan: %v", err)
	}
	if gotSpan == nil {
		t.Fatal("server did not receive span")
	}
	if gotSpan.StartTimeUnixNano == 0 || gotSpan.EndTimeUnixNano == 0 {
		t.Fatalf("timestamps must be set: start=%d end=%d", gotSpan.StartTimeUnixNano, gotSpan.EndTimeUnixNano)
	}
	if gotSpan.EndTimeUnixNano < gotSpan.StartTimeUnixNano {
		t.Fatalf("end %d < start %d", gotSpan.EndTimeUnixNano, gotSpan.StartTimeUnixNano)
	}
}

func TestCreateSpan_NoFinishReasonsWhenEmpty(t *testing.T) {
	var gotSpan *otlptrace.Span
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var exportReq otlpcollectortrace.ExportTraceServiceRequest
		if err := proto.Unmarshal(body, &exportReq); err != nil {
			t.Errorf("unmarshal: %v", err)
		}
		gotSpan = exportReq.ResourceSpans[0].ScopeSpans[0].Spans[0]
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tr := NewMLflowTracer(srv.URL, "", "")
	now := time.Now()
	err := tr.CreateSpan(context.Background(), ai.SpanRequest{
		TraceID:   "4bf92f3577b34da6a3ce929d0e0e4736",
		Name:      "openai_chat",
		StartTime: now,
		EndTime:   now,
		Error:     "boom",
	})
	if err != nil {
		t.Fatalf("CreateSpan: %v", err)
	}
	for _, attr := range gotSpan.Attributes {
		if attr.Key == "gen_ai.response.finish_reasons" {
			t.Errorf("unexpected finish_reasons attribute with empty FinishReason")
		}
	}
}
