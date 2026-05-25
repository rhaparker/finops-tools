package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "finops",
	Short: "FinOps command-line tools",
	Long:  "FinOps command-line tools for cost, cluster, and related operations.",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)
}
