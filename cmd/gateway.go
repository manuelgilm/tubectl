package cmd

import (
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var gatewayListCmd = &cobra.Command{
	Use:   "list",
	Short: "List MLflow gateway endpoints",
	Long: `Lists MLflow gateway endpoints and their tracing configuration. The
Usage Tracking column indicates whether the endpoint emits gateway
traces (auto-created in an experiment named "gateway/<name>"), which
can then be inspected with: tubectl trace list --experiment <name>.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := loadTraceClient()
		if err != nil {
			return fmt.Errorf("loading MLflow client: %w", err)
		}
		endpoints, err := client.ListGatewayEndpoints(cmd.Context())
		if err != nil {
			return fmt.Errorf("listing gateway endpoints: %w", err)
		}
		if len(endpoints) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No gateway endpoints found.")
			return nil
		}
		sort.Slice(endpoints, func(i, j int) bool { return endpoints[i].Name < endpoints[j].Name })

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ENDPOINT ID\tNAME\tROUTING\tUSAGE TRACKING\tEXPERIMENT")
		for _, e := range endpoints {
			experiment := e.ExperimentID
			if !e.UsageTracking {
				experiment = "-"
			}
			usage := strings.Title(boolToStr(e.UsageTracking))
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", e.EndpointID, e.Name, e.RoutingStrategy, usage, experiment)
		}
		return w.Flush()
	},
}

func boolToStr(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

var gatewayCmd = &cobra.Command{
	Use:   "gateway",
	Short: "Inspect MLflow gateway endpoints",
}

func init() {
	rootCmd.AddCommand(gatewayCmd)
	gatewayCmd.AddCommand(gatewayListCmd)
}
