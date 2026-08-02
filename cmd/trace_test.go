package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/manuelgilm/tubectl/internal/trace"
	"github.com/spf13/cobra"
)

func TestPrintTrace_writesToCommandOutput(t *testing.T) {
	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&out)

	tr := &trace.Trace{
		TraceInfo: trace.TraceInfo{
			TraceID: "tr-abc",
			State:   "OK",
			Tags:    map[string]string{"source": "youtube-comment"},
		},
		Spans: []trace.Span{
			{
				Name:              "openai_chat",
				StartTimeUnixNano: 1720000000123000000,
				EndTimeUnixNano:   1720000002600000000,
				Status:            &trace.SpanStatus{Code: 1},
			},
		},
	}

	printTrace(cmd, tr)

	output := out.String()
	for _, want := range []string{"Trace ID:   tr-abc", "State:      OK", "source=youtube-comment", "openai_chat"} {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %q, got:\n%s", want, output)
		}
	}
}
