// account_list.go implements "finops account list" to show registered account aliases.
package cmd

import (
	"fmt"
	"slices"

	"github.com/openshift-online/finops-tools/cli/internal/account"
	"github.com/openshift-online/finops-tools/cli/internal/configstore"
	"github.com/openshift-online/finops-tools/cli/internal/output"
	"github.com/spf13/cobra"
)

var accountListCmd = &cobra.Command{
	Use:   "list [provider]",
	Short: "List registered cloud accounts and aliases",
	Long: `List accounts saved in the finops config file.

AWS entries show whether each alias is a payer account (org billing / Cost Explorer)
or a linked member account (role assumption from a registered payer).

Examples:
  finops account list
  finops account list aws
  finops account list gcp
  finops account list snowflake`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAccountList,
}

func init() {
	accountCmd.AddCommand(accountListCmd)
}

func runAccountList(cmd *cobra.Command, args []string) error {
	provider := account.ProviderAWS
	if len(args) == 1 {
		p, err := account.ParseProvider(args[0])
		if err != nil {
			return err
		}
		provider = p
	}

	path, err := configstore.ResolvePath(awsFlags.ConfigPath)
	if err != nil {
		return err
	}
	cfg, err := configstore.Load(path)
	if err != nil {
		return err
	}

	switch provider {
	case account.ProviderAWS:
		return printAWSAccountList(cmd, cfg)
	case account.ProviderGCP:
		return printGCPAccountList(cmd, cfg)
	case account.ProviderSnowflake:
		return printSnowflakeAccountList(cmd, cfg)
	default:
		return fmt.Errorf("unsupported provider %q", provider)
	}
}

func printAWSAccountList(cmd *cobra.Command, cfg configstore.File) error {
	rows := awsAccountListRows(cfg.ListAWSAccounts())
	return output.WriteAWSAccountList(cmd.OutOrStdout(), rows)
}

func printGCPAccountList(cmd *cobra.Command, cfg configstore.File) error {
	aliases := make([]string, 0, len(cfg.GCP.AccountAliases))
	for alias := range cfg.GCP.AccountAliases {
		aliases = append(aliases, alias)
	}
	slices.Sort(aliases)

	rows := make([]output.AccountListRow, 0, len(aliases))
	for _, alias := range aliases {
		rows = append(rows, output.AccountListRow{
			Alias:     alias,
			AccountID: cfg.GCP.AccountAliases[alias],
		})
	}
	return output.WriteGCPAccountList(cmd.OutOrStdout(), rows)
}

func printSnowflakeAccountList(cmd *cobra.Command, cfg configstore.File) error {
	entries := cfg.ListSnowflakeAccounts()
	rows := make([]output.AccountListRow, len(entries))
	for i, e := range entries {
		rows[i] = output.AccountListRow{
			Alias:     e.Alias,
			AccountID: e.Account,
			Kind:      "snowflake",
			Role:      e.Role,
		}
	}
	return output.WriteSnowflakeAccountList(cmd.OutOrStdout(), rows)
}

func awsAccountListRows(entries []configstore.AWSAccountListEntry) []output.AccountListRow {
	rows := make([]output.AccountListRow, len(entries))
	for i, e := range entries {
		rows[i] = output.AccountListRow{
			Alias:      e.Alias,
			AccountID:  e.AccountID,
			Kind:       e.Kind,
			PayerAlias: e.PayerAlias,
			Role:       e.Role,
		}
	}
	return rows
}
