// account.go registers the "finops account" noun command for payer and linked account management.
package cmd

import (
	"github.com/spf13/cobra"
)

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Manage cloud payer accounts and aliases",
}

func init() {
	accountCmd.GroupID = "setup"
	bindAWSPersistentFlags(accountCmd)
	rootCmd.AddCommand(accountCmd)
}
