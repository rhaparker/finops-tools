// cost_get.go implements "finops cost get": resolves targets, ensures credentials, fetches costs, and prints output.
package cmd

import (
	"time"

	"github.com/openshift-online/finops-tools/cli/internal/configstore"
	"github.com/openshift-online/finops-tools/cli/internal/output"
	coreaccount "github.com/openshift-online/finops-tools/core/account"
	"github.com/openshift-online/finops-tools/core/cost"
	"github.com/spf13/cobra"
)

var (
	costGetAccount        string
	costGetAccountAliases string
	costGetFormat         string
	costGetOU             string
	costGetOUDirect       bool
	costGetPayer          string
	costGetProvider       string
	costGetSplitBy        string
)

var costGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get net amortized cost for a date range",
	Long: `Fetch the sum of AWS Cost Explorer NetAmortizedCost for one or more payer or linked accounts.
Provide --account with 12-digit AWS account IDs and/or --account-alias with configured aliases (see finops account add aws).

Period (default: last 30 calendar days, or defaults.cost.* in config):
  --days, --months, --from/--to, --exclude-recent-days (omit recent incomplete CE days)

For linked accounts, credentials are obtained from the registered payer account.
Use --payer with --account to query a member account that is not registered (the payer alias must be registered).

Use --ou with --payer to query all accounts in an AWS Organizational Unit (recursive by default).
Add --ou-direct to include only accounts directly in the OU, not descendant OUs.

Examples:
  finops cost get --ou ou-abcd-1234 --payer rh-control
  finops cost get --ou ou-abcd-1234 --payer rh-control --ou-direct --days 7

Authentication uses --auth-method when set, otherwise defaults.aws.auth_method in config (saml by default).

Only AWS is supported today; GCP will be added later.`,
	Args: cobra.NoArgs,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		sel, err := parseCostTargetSelector(costGetAccount, costGetAccountAliases, costGetOU, costGetPayer, costGetOUDirect)
		if err != nil {
			return err
		}
		if err := validateCostTargetSelector(sel); err != nil {
			return err
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
		return validatePeriodFlags(cmd)
	},
	RunE: runCostGet,
}

func init() {
	costCmd.AddCommand(costGetCmd)
	costGetCmd.Flags().StringVar(&costGetAccount, "account", "", "Payer AWS account ID(s), comma-separated 12-digit IDs")
	costGetCmd.Flags().StringVar(&costGetAccountAliases, "account-alias", "", "Configured account alias(es), comma-separated (e.g. rh-control)")
	costGetCmd.Flags().StringVar(&costGetOU, "ou", "", "AWS OU ID(s), comma-separated (requires --payer; recursive by default)")
	costGetCmd.Flags().BoolVar(&costGetOUDirect, "ou-direct", false, "Include only accounts directly in --ou, not descendant OUs")
	costGetCmd.Flags().StringVar(&costGetPayer, "payer", "", "Registered payer alias for --account member IDs or --ou (e.g. rhc)")
	costGetCmd.Flags().StringVar(&costGetFormat, "format", string(output.FormatPrettyPrint),
		"Output format: pretty-print, json, csv")
	costGetCmd.Flags().StringVar(&costGetProvider, "provider", string(cost.ProviderAWS),
		"Cloud provider: aws or gcp")
	costGetCmd.Flags().StringVar(&costGetSplitBy, "split-by", "",
		"Split results by dimension (supported: service, account)")
	addPeriodFlags(costGetCmd)
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
	if err := applyCostPeriodDefaults(cmd, cfg); err != nil {
		return err
	}

	sel, err := parseCostTargetSelector(costGetAccount, costGetAccountAliases, costGetOU, costGetPayer, costGetOUDirect)
	if err != nil {
		return err
	}

	targets, err := resolveCostTargetsWithOU(
		cmd.Context(), cmd, cfg, sel,
		awsFlags.ConfigPath, awsFlags.CredentialsFile, awsFlags.AuthMethod,
	)
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

	dateRange, err := resolveCostPeriod(time.Now().UTC())
	if err != nil {
		return err
	}

	costQuery := cost.CostQuery{
		Provider: provider,
		Accounts: targets,
		Range:    dateRange,
		SplitBy:  splitBy,
	}
	if provider == cost.ProviderAWS && splitBy == cost.SplitByAccount {
		costQuery.AWSFetch = &cost.AWSFetchOptions{
			ResolveAccountNames: coreaccount.ResolveAccountNames,
		}
	}

	result, err := cost.Fetch(cmd.Context(), costQuery)
	if err != nil {
		return err
	}

	return output.WriteCostResult(cmd.OutOrStdout(), format, result)
}
