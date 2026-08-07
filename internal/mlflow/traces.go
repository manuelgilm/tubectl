package mlflow

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// TraceInfo is the V3 trace info record returned by the MLflow trace API.
type TraceInfo struct {
	TraceID           string            `json:"trace_id"`
	ClientRequestID   string            `json:"client_request_id"`
	TraceLocation     TraceLocation     `json:"trace_location"`
	RequestPreview    string            `json:"request_preview"`
	ResponsePreview   string            `json:"response_preview"`
	RequestTime       string            `json:"request_time"`
	ExecutionDuration string            `json:"execution_duration"`
	State             string            `json:"state"`
	TraceMetadata     map[string]string `json:"trace_metadata"`
	Tags              map[string]string `json:"tags"`
}

type TraceLocation struct {
	Type             string `json:"type"`
	MlflowExperiment struct {
		ExperimentID string `json:"experiment_id"`
	} `json:"mlflow_experiment"`
}

// Trace is the full record returned by the trace get API: info plus OTel spans.
type Trace struct {
	TraceInfo TraceInfo `json:"trace_info"`
	Spans     []Span    `json:"spans"`
}

// Span mirrors the JSON representation of an OpenTelemetry span proto as
// serialized by the MLflow server.
type Span struct {
	Name              string      `json:"name"`
	SpanID            string      `json:"span_id"`
	ParentSpanID      string      `json:"parent_span_id"`
	StartTimeUnixNano int64       `json:"start_time_unix_nano"`
	EndTimeUnixNano   int64       `json:"end_time_unix_nano"`
	Attributes        []Attribute `json:"attributes"`
	Status            *SpanStatus `json:"status"`
}

type Attribute struct {
	Key   string    `json:"key"`
	Value SpanValue `json:"value"`
}

type SpanValue struct {
	StringValue *string         `json:"stringValue"`
	IntValue    *int64          `json:"intValue"`
	DoubleValue *float64        `json:"doubleValue"`
	BoolValue   *bool           `json:"boolValue"`
	ArrayValue  *SpanArrayValue `json:"arrayValue"`
}

type SpanArrayValue struct {
	Values []SpanValue `json:"values"`
}

// String renders a span attribute value for display.
func (v SpanValue) String() string {
	switch {
	case v.ArrayValue != nil:
		parts := make([]string, 0, len(v.ArrayValue.Values))
		for _, e := range v.ArrayValue.Values {
			parts = append(parts, e.String())
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case v.IntValue != nil:
		return fmt.Sprint(*v.IntValue)
	case v.DoubleValue != nil:
		return fmt.Sprint(*v.DoubleValue)
	case v.BoolValue != nil:
		return fmt.Sprint(*v.BoolValue)
	case v.StringValue != nil:
		return *v.StringValue
	}
	return ""
}

type SpanStatus struct {
	Code    int32  `json:"code"`
	Message string `json:"message"`
}

// GetTrace fetches a single trace (info + spans) by its request id (e.g. "tr-...").
//
// It uses the v3 trace API (/api/3.0/mlflow/traces/get) because that endpoint
// returns full span data plus request/response previews and tags (map form).
func (c *Client) GetTrace(ctx context.Context, traceID string) (*Trace, error) {
	params := url.Values{}
	params.Set("trace_id", traceID)
	params.Set("allow_partial", "true")

	var resp struct {
		Trace Trace `json:"trace"`
	}
	if err := c.get(ctx, "/api/3.0/mlflow/traces/get", params, &resp); err != nil {
		return nil, err
	}
	if resp.Trace.TraceInfo.TraceID == "" {
		return nil, fmt.Errorf("trace %q not found", traceID)
	}
	return &resp.Trace, nil
}

// TraceListItem is the legacy V1/V2 trace info returned by the search traces API.
type TraceListItem struct {
	RequestID       string `json:"request_id"`
	ExperimentID    string `json:"experiment_id"`
	TimestampMS     int64  `json:"timestamp_ms"`
	ExecutionTimeMS int64  `json:"execution_time_ms"`
	Status          string `json:"status"`
	RequestMetadata []KV   `json:"request_metadata"`
	Tags            []KV   `json:"tags"`
}

type KV struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// ListTraces returns trace info records for the given experiment IDs, most
// recent first, up to maxResults.
//
// It uses the legacy v2 search API (/api/2.0/mlflow/traces) since that endpoint
// accepts plain experiment_ids query params; the v3 search endpoint requires a
// serialized locations payload and adds nothing for listing.
func (c *Client) ListTraces(ctx context.Context, experimentIDs []string, maxResults int) ([]TraceListItem, error) {
	if len(experimentIDs) == 0 {
		experimentIDs = []string{"0"}
	}
	if maxResults <= 0 {
		maxResults = 20
	}
	if maxResults > 500 {
		maxResults = 500
	}

	params := url.Values{}
	for _, id := range experimentIDs {
		params.Add("experiment_ids", id)
	}
	params.Set("max_results", fmt.Sprint(maxResults))
	params.Add("order_by", "timestamp_ms DESC")

	var resp struct {
		Traces []TraceListItem `json:"traces"`
	}
	if err := c.get(ctx, "/api/2.0/mlflow/traces", params, &resp); err != nil {
		return nil, err
	}
	return resp.Traces, nil
}
