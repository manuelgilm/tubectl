package cmd

import (
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/manuelgilm/tubectl/internal/mlflow"
	"github.com/spf13/cobra"
)

var (
	traceGetArgs struct {
		traceID string
	}
	traceListArgs struct {
		experimentID string
		maxResults   int
	}
)

func loadTraceClient() (*mlflow.Client, error) {
	creds, err := resolveMLflowCreds()
	if err != nil {
		return nil, err
	}
	return mlflow.NewClient(creds.serverURL, creds.username, creds.password), nil
}

var traceGetCmd = &cobra.Command{
	Use:   "get <traceID>",
	Short: "Get a trace by ID",
	Long: `Fetches a trace (span info, inputs/outputs, tags) from the MLflow
server. The trace ID is the "request id" shown by the MLflow UI
(e.g. tr-4bf92f3577b34da6a3ce929d0e0e4736).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		traceID := traceGetArgs.traceID
		if len(args) > 0 {
			traceID = args[0]
		}
		if traceID == "" {
			return fmt.Errorf("trace ID is required (positional argument or --trace-id)")
		}

		client, err := loadTraceClient()
		if err != nil {
			return fmt.Errorf("loading MLflow client: %w", err)
		}
		tr, err := client.GetTrace(cmd.Context(), traceID)
		if err != nil {
			return fmt.Errorf("getting trace %s: %w", traceID, err)
		}
		printTrace(cmd, tr)
		return nil
	},
}

var traceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent traces",
	Long: `Lists recent traces recorded on the MLflow server, most recent
first. Use --experiment-id to filter by experiment (default 0) and
--max-results to control the page size (default 20, max 500).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := loadTraceClient()
		if err != nil {
			return fmt.Errorf("loading MLflow client: %w", err)
		}
		items, err := client.ListTraces(cmd.Context(), []string{traceListArgs.experimentID}, traceListArgs.maxResults)
		if err != nil {
			return fmt.Errorf("listing traces: %w", err)
		}
		if len(items) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No traces found.")
			return nil
		}
		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "TRACE ID\tSTATE\tCREATED\tDURATION\tTAGS")
		for _, it := range items {
			created := time.UnixMilli(it.TimestampMS).Format(time.RFC3339)
			duration := ""
			if it.ExecutionTimeMS > 0 {
				duration = fmt.Sprintf("%dms", it.ExecutionTimeMS)
			}
			tagStr := joinTags(it.Tags)
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", it.RequestID, it.Status, created, duration, tagStr)
		}
		return w.Flush()
	},
}

var traceCmd = &cobra.Command{
	Use:   "trace",
	Short: "Inspect traces recorded on the MLflow server",
}

func printTrace(cmd *cobra.Command, tr *mlflow.Trace) {
	info := tr.TraceInfo
	out := cmd.OutOrStdout()

	fmt.Fprintf(out, "Trace ID:   %s\n", info.TraceID)
	fmt.Fprintf(out, "State:      %s\n", info.State)
	if exp := info.TraceLocation.MlflowExperiment.ExperimentID; exp != "" {
		fmt.Fprintf(out, "Experiment: %s\n", exp)
	}
	if info.ExecutionDuration != "" {
		fmt.Fprintf(out, "Duration:   %s\n", info.ExecutionDuration)
	}
	if info.RequestTime != "" {
		fmt.Fprintf(out, "Created:    %s\n", info.RequestTime)
	}
	if len(info.Tags) > 0 {
		fmt.Fprintf(out, "Tags:       %s\n", joinTagsMap(info.Tags))
	}

	if info.RequestPreview != "" {
		fmt.Fprintf(out, "\nRequest preview:\n---\n%s\n---\n", info.RequestPreview)
	}
	if info.ResponsePreview != "" {
		fmt.Fprintf(out, "\nResponse preview:\n---\n%s\n---\n", info.ResponsePreview)
	}

	if len(tr.Spans) > 0 {
		fmt.Fprintln(out, "\nSpans:")
		w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tSTATUS\tDURATION")
		for _, s := range tr.Spans {
			status := spanStatus(s)
			duration := formatNanoDuration(s.StartTimeUnixNano, s.EndTimeUnixNano)
			fmt.Fprintf(w, "%s\t%s\t%s\n", s.Name, status, duration)
		}
		w.Flush()
		for _, s := range tr.Spans {
			if len(s.Attributes) == 0 {
				continue
			}
			fmt.Fprintf(out, "\n  %s attributes:\n", s.Name)
			sort.Slice(s.Attributes, func(i, j int) bool { return s.Attributes[i].Key < s.Attributes[j].Key })
			for _, a := range s.Attributes {
				fmt.Fprintf(out, "    %s = %s\n", a.Key, a.Value.String())
			}
		}
	}
}

func spanStatus(s mlflow.Span) string {
	if s.Status == nil {
		return "UNSET"
	}
	switch s.Status.Code {
	case 1:
		return "OK"
	case 2:
		return "ERROR"
	default:
		return "UNSET"
	}
}

func formatNanoDuration(start, end int64) string {
	ms := (end - start) / int64(time.Millisecond)
	if ms < 0 {
		return ""
	}
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	return fmt.Sprintf("%.2fs", float64(ms)/1000)
}

func joinTags(tags []mlflow.KV) string {
	if len(tags) == 0 {
		return ""
	}
	sort.Slice(tags, func(i, j int) bool { return tags[i].Key < tags[j].Key })
	parts := make([]string, 0, len(tags))
	for _, t := range tags {
		parts = append(parts, t.Key+"="+t.Value)
	}
	return strings.Join(parts, ", ")
}

func joinTagsMap(tags map[string]string) string {
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+tags[k])
	}
	return strings.Join(parts, ", ")
}

func init() {
	rootCmd.AddCommand(traceCmd)
	traceCmd.AddCommand(traceGetCmd)
	traceGetCmd.Flags().StringVar(&traceGetArgs.traceID, "trace-id", "", "Trace ID to fetch (e.g. tr-...)")

	traceCmd.AddCommand(traceListCmd)
	traceListCmd.Flags().StringVar(&traceListArgs.experimentID, "experiment-id", "0", "Experiment ID to list traces for")
	traceListCmd.Flags().IntVar(&traceListArgs.maxResults, "max-results", 20, "Maximum number of traces to return (max 500)")
}
