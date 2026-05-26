// report.go registers the "finops report" noun command for report generation.
package cmd

import (
	"github.com/spf13/cobra"
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate FinOps reports",
	Long:  "Build reports from configured cloud accounts and cost data sources.",
}

func init() {
	reportCmd.GroupID = "core"
	bindAWSPersistentFlags(reportCmd)
	rootCmd.AddCommand(reportCmd)
}
