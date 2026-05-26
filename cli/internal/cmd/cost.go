// cost.go registers the "finops cost" noun command for cloud cost queries.
package cmd

import (
	"github.com/spf13/cobra"
)

var costCmd = &cobra.Command{
	Use:   "cost",
	Short: "Cloud cost queries",
	Long:  "Fetch cost and usage summaries from cloud providers (AWS today; GCP planned).",
}

func init() {
	costCmd.GroupID = "core"
	bindAWSPersistentFlags(costCmd)
	rootCmd.AddCommand(costCmd)
}
