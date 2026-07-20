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
	"strings"
	"time"

	"google.golang.org/protobuf/proto"

	otlpcollectortrace "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	otlpcommon "go.opentelemetry.io/proto/otlp/common/v1"
	otlptrace "go.opentelemetry.io/proto/otlp/trace/v1"

	"tubectl/internal/ai"
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

func (t *MLflowTracer) StartTrace(ctx context.Context, experimentID string) (string, error) {
	eid := t.experimentID
	if experimentID != "" {
		eid = experimentID
	}
	body, err := json.Marshal(map[string]any{
		"experiment_id": eid,
		"timestamp_ms":  time.Now().UnixMilli(),
	})
	if err != nil {
		return "", fmt.Errorf("marshal start trace: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST",
		t.baseURL+"/ajax-api/2.0/mlflow/traces", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build start trace request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if t.username != "" {
		req.SetBasicAuth(t.username, t.password)
	}
	resp, err := t.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("start trace: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		data, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("start trace: %s %s", resp.Status, strings.TrimSpace(string(data)))
	}
	var result struct {
		TraceInfo struct {
			RequestID string `json:"request_id"`
		} `json:"trace_info"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode trace info: %w", err)
	}
	if result.TraceInfo.RequestID == "" {
		return "", fmt.Errorf("start trace: empty request_id in response")
	}
	return result.TraceInfo.RequestID, nil
}

func (t *MLflowTracer) CreateSpan(ctx context.Context, req ai.SpanRequest) error {
	rawTraceID := req.TraceID
	traceID, err := hex.DecodeString(strings.TrimPrefix(rawTraceID, "tr-"))
	if err != nil {
		return fmt.Errorf("decode trace id %q: %w", rawTraceID, err)
	}
	spanID := make([]byte, 8)
	if _, err := rand.Read(spanID); err != nil {
		return fmt.Errorf("generate span id: %w", err)
	}

	attrs := []*otlpcommon.KeyValue{
		kv("mlflow.traceName", "openai_completion"),
		kv("model", req.Model),
		kv("prompt", formatMessages(req.Messages)),
		kv("response", req.Response),
		kv("finish_reason", req.FinishReason),
		kv("prompt_tokens", int64(req.PromptTokens)),
		kv("completion_tokens", int64(req.CompletionTokens)),
		kv("total_tokens", int64(req.TotalTokens)),
		kv("latency_ms", req.LatencyMs),
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

func (t *MLflowTracer) EndTrace(ctx context.Context, requestID, status string) error {
	body, err := json.Marshal(map[string]string{"status": status})
	if err != nil {
		return fmt.Errorf("marshal end trace: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, "PATCH",
		fmt.Sprintf("%s/ajax-api/2.0/mlflow/traces/%s", t.baseURL, requestID),
		bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build end trace request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if t.username != "" {
		httpReq.SetBasicAuth(t.username, t.password)
	}
	resp, err := t.http.Do(httpReq)
	if err != nil {
		return fmt.Errorf("end trace: %w", err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("end trace: %s", resp.Status)
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
	default:
		v = &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: fmt.Sprint(value)}}
	}
	return &otlpcommon.KeyValue{Key: key, Value: v}
}

func formatMessages(msgs []ai.Message) string {
	data, err := json.Marshal(msgs)
	if err != nil {
		return "[]"
	}
	return string(data)
}
