// cost_get.go implements "finops cost get": resolves targets, ensures credentials, fetches costs, and prints output.
package cmd

import (
	"fmt"
	"strings"

	awsconfig "github.com/openshift-online/finops-tools/cli/internal/aws"
	"github.com/openshift-online/finops-tools/cli/internal/configstore"
	"github.com/openshift-online/finops-tools/cli/internal/output"
	"github.com/openshift-online/finops-tools/core/cost"
	"github.com/spf13/cobra"
)

var (
	costGetAccount           string
	costGetAccountAliases    string
	costGetFormat          string
	costGetProvider        string
	costGetSplitBy         string
)

var costGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get net amortized cost for the last 30 days",
	Long: `Fetch the sum of AWS Cost Explorer NetAmortizedCost over the last 30 days
for one or more payer or linked accounts. Provide --account with 12-digit AWS account IDs
and/or --account-alias with configured aliases (see finops account add aws).

For linked accounts, credentials are obtained from the registered payer account.
Authentication uses --auth-method when set, otherwise defaults.aws.auth_method in config (saml by default).

Only AWS is supported today; GCP will be added later.`,
	Args: cobra.NoArgs,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(costGetAccount) == "" && strings.TrimSpace(costGetAccountAliases) == "" {
			return fmt.Errorf("at least one of --account or --account-alias is required")
		}
		if _, err := output.ParseFormat(costGetFormat); err != nil {
			return err
		}
		if _, err := cost.ParseProvider(costGetProvider); err != nil {
			return err
		}
		if _, err := cost.ParseSplitBy(costGetSplitBy); err != nil {
			return err
		}
		return nil
	},
	RunE: runCostGet,
}

func init() {
	costCmd.AddCommand(costGetCmd)
	costGetCmd.Flags().StringVar(&costGetAccount, "account", "", "Payer AWS account ID(s), comma-separated 12-digit IDs")
	costGetCmd.Flags().StringVar(&costGetAccountAliases, "account-alias", "", "Configured account alias(es), comma-separated (e.g. rh-control)")
	costGetCmd.Flags().StringVar(&costGetFormat, "format", string(output.FormatPrettyPrint),
		"Output format: pretty-print, json, csv")
	costGetCmd.Flags().StringVar(&costGetProvider, "provider", string(cost.ProviderAWS),
		"Cloud provider: aws or gcp")
	costGetCmd.Flags().StringVar(&costGetSplitBy, "split-by", "",
		"Split results by dimension (supported: service, account)")
}

func runCostGet(cmd *cobra.Command, _ []string) error {
	format, err := output.ParseFormat(costGetFormat)
	if err != nil {
		return err
	}
	provider, err := cost.ParseProvider(costGetProvider)
	if err != nil {
		return err
	}
	splitBy, err := cost.ParseSplitBy(costGetSplitBy)
	if err != nil {
		return err
	}

	cfgPath, err := configstore.ResolvePath(awsFlags.ConfigPath)
	if err != nil {
		return err
	}
	cfg, err := configstore.Load(cfgPath)
	if err != nil {
		return err
	}

	var accountIDs, aliases []string
	if strings.TrimSpace(costGetAccount) != "" {
		accountIDs, err = configstore.ParseAWSAccountIDs(costGetAccount)
		if err != nil {
			return err
		}
	}
	if strings.TrimSpace(costGetAccountAliases) != "" {
		aliases, err = configstore.ParseAccountAliases(costGetAccountAliases)
		if err != nil {
			return err
		}
	}

	targets, err := configstore.ResolveCostTargets(cfg, accountIDs, aliases)
	if err != nil {
		return err
	}

	if provider == cost.ProviderAWS {
		if err := ensureCostCredentials(cmd.Context(), cmd, cfg, targets, awsFlags.ConfigPath, awsFlags.CredentialsFile, awsFlags.AuthMethod); err != nil {
			return err
		}
		targets, err = prepareCostTargets(cmd.Context(), cfg, targets, awsFlags.CredentialsFile)
		if err != nil {
			return err
		}
	}

	costQuery := cost.CostQuery{
		Provider: provider,
		Accounts: targets,
		Days:     cost.DefaultDays,
		SplitBy:  splitBy,
	}
	if provider == cost.ProviderAWS && splitBy == cost.SplitByAccount {
		costQuery.AWSFetch = &cost.AWSFetchOptions{
			ResolveAccountNames: awsconfig.ResolveAccountNames,
		}
	}

	result, err := cost.Fetch(cmd.Context(), costQuery)
	if err != nil {
		return err
	}

	return output.WriteCostResult(cmd.OutOrStdout(), format, result)
}
