// report_list.go implements "finops report list".
package cmd

import (
	reportpkg "github.com/openshift-online/finops-tools/cli/internal/report"
	"github.com/openshift-online/finops-tools/cli/internal/output"
	"github.com/spf13/cobra"
)

var reportListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available report templates",
	Long: `List report templates that can be passed as the first argument to "finops report generate".

Example:
  finops report list`,
	Args: cobra.NoArgs,
	RunE: runReportList,
}

func init() {
	reportCmd.AddCommand(reportListCmd)
}

func runReportList(cmd *cobra.Command, _ []string) error {
	return output.WriteReportTemplateList(cmd.OutOrStdout(), reportpkg.Templates())
}
