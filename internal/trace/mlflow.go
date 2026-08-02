package trace

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"google.golang.org/protobuf/proto"

	otlpcollectortrace "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	otlpcommon "go.opentelemetry.io/proto/otlp/common/v1"
	otlptrace "go.opentelemetry.io/proto/otlp/trace/v1"

	"github.com/manuelgilm/tubectl/internal/ai"
)

type MLflowTracer struct {
	baseURL      string
	username     string
	password     string
	experimentID string
	http         *http.Client
}

func NewMLflowTracer(baseURL, username, password string) *MLflowTracer {
	return &MLflowTracer{
		baseURL:      baseURL,
		username:     username,
		password:     password,
		experimentID: "0",
		http:         &http.Client{Timeout: 10 * time.Second},
	}
}

func (t *MLflowTracer) WithExperimentID(id string) *MLflowTracer {
	t.experimentID = id
	return t
}

func (t *MLflowTracer) CreateSpan(ctx context.Context, req ai.SpanRequest) error {
	traceID, err := hex.DecodeString(req.TraceID)
	if err != nil {
		return fmt.Errorf("decode trace id %q: %w", req.TraceID, err)
	}
	spanID := make([]byte, 8)
	if _, err := rand.Read(spanID); err != nil {
		return fmt.Errorf("generate span id: %w", err)
	}

	attrs := []*otlpcommon.KeyValue{
		kv("mlflow.traceName", req.Name),
		kv("mlflow.spanInputs", formatSpanInputs(req.Messages)),
		kv("mlflow.spanOutputs", formatSpanOutputs(req.Response)),
		kv("gen_ai.operation.name", "chat"),
		kv("gen_ai.usage.input_tokens", int64(req.PromptTokens)),
		kv("gen_ai.usage.output_tokens", int64(req.CompletionTokens)),
		kv("gen_ai.request.model", req.Model),
		kv("gen_ai.provider.name", "openai"),
		kv("gen_ai.response.finish_reasons", []string{req.FinishReason}),
		kv("latency_ms", req.LatencyMs),
	}
	for k, v := range req.Tags {
		attrs = append(attrs, kv("mlflow.traceTag."+k, v))
	}
	if req.Error != "" {
		attrs = append(attrs, kv("error", req.Error))
	}

	statusCode := otlptrace.Status_STATUS_CODE_OK
	if req.Error != "" {
		statusCode = otlptrace.Status_STATUS_CODE_ERROR
	}

	span := &otlptrace.Span{
		TraceId:           traceID,
		SpanId:            spanID,
		Name:              req.Name,
		StartTimeUnixNano: uint64(req.StartTime.UnixNano()),
		EndTimeUnixNano:   uint64(req.EndTime.UnixNano()),
		Attributes:        attrs,
		Status:            &otlptrace.Status{Code: statusCode},
	}

	exportReq := &otlpcollectortrace.ExportTraceServiceRequest{
		ResourceSpans: []*otlptrace.ResourceSpans{
			{
				ScopeSpans: []*otlptrace.ScopeSpans{
					{Spans: []*otlptrace.Span{span}},
				},
			},
		},
	}

	protoData, err := proto.Marshal(exportReq)
	if err != nil {
		return fmt.Errorf("marshal span: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		t.baseURL+"/v1/traces", bytes.NewReader(protoData))
	if err != nil {
		return fmt.Errorf("build span request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/x-protobuf")
	httpReq.Header.Set("x-mlflow-experiment-id", t.experimentID)
	if t.username != "" {
		httpReq.SetBasicAuth(t.username, t.password)
	}
	resp, err := t.http.Do(httpReq)
	if err != nil {
		return fmt.Errorf("create span: %w", err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("create span: %s", resp.Status)
	}
	return nil
}

func kv(key string, value any) *otlpcommon.KeyValue {
	var v *otlpcommon.AnyValue
	switch val := value.(type) {
	case string:
		v = &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: val}}
	case int64:
		v = &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_IntValue{IntValue: val}}
	case float64:
		v = &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_DoubleValue{DoubleValue: val}}
	case []string:
		values := make([]*otlpcommon.AnyValue, 0, len(val))
		for _, s := range val {
			values = append(values, &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: s}})
		}
		v = &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_ArrayValue{ArrayValue: &otlpcommon.ArrayValue{Values: values}}}
	default:
		v = &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: fmt.Sprint(value)}}
	}
	return &otlpcommon.KeyValue{Key: key, Value: v}
}

func formatSpanInputs(msgs []ai.Message) string {
	data, err := json.Marshal(map[string]any{"messages": msgs})
	if err != nil {
		return `{"messages":[]}`
	}
	return string(data)
}

func formatSpanOutputs(response string) string {
	data, err := json.Marshal(map[string]any{
		"messages": []ai.Message{{Role: "assistant", Content: response}},
	})
	if err != nil {
		return `{"messages":[]}`
	}
	return string(data)
}
