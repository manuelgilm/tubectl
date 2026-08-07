# MLflow tracing (archived)

This directory holds the previous manual OTLP trace uploader for the MLflow
tracking server.

- `otel.go` — `MLflowTracer` / `NewMLflowTracer` / `CreateSpan`: serialized OTel
  spans (via `go.opentelemetry.io/proto/otlp`) and POSTed them to the MLflow
  tracking server.
- `otel_test.go` — tests for the above.

## Why it was removed

Tubectl now routes LLM completions through the MLflow **gateway**
(`/gateway/mlflow/v1`, OpenAI-compatible). The gateway automatically records
traces on the MLflow server, so the manual `WithTracer` span upload became
redundant and was removed to avoid double-tracing.

If manual tracing is ever needed again (e.g. logging non-gateway LLM calls),
restore these files under `internal/mlflow/` and re-add
`go.opentelemetry.io/proto/otlp` + `google.golang.org/protobuf` to `go.mod`.

Note: both files carry a `//go:build ignore` tag and are excluded from the
module build.