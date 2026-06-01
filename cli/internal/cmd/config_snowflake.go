package cmd

import (
	"github.com/spf13/cobra"
)

var configSnowflakeCmd = &cobra.Command{
	Use:   "snowflake",
	Short: "Manage Snowflake-related configuration",
}

func init() {
	configCmd.AddCommand(configSnowflakeCmd)
}
