// report_generate.go implements "finops report generate".
package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/openshift-online/finops-tools/cli/internal/configstore"
	"github.com/openshift-online/finops-tools/cli/internal/progress"
	reportpkg "github.com/openshift-online/finops-tools/cli/internal/report"
	coreaccount "github.com/openshift-online/finops-tools/core/account"
	"github.com/openshift-online/finops-tools/core/cost"
	corereport "github.com/openshift-online/finops-tools/core/report"
	"github.com/spf13/cobra"
)

var (
	reportGenerateAccount         string
	reportGenerateAccountAliases  string
	reportGenerateFormat          string
	reportGenerateOU              string
	reportGenerateOUDirect        bool
	reportGenerateOutput          string
	reportGeneratePayer           string
	reportGenerateQuiet           bool
	reportGenerateTagKey          string
	reportGenerateTagValue        string
	reportGenerateSkipOrgCache    bool
	reportGenerateRefreshOrgCache bool
)

var reportGenerateCmd = &cobra.Command{
	Use:   "generate [template]",
	Short: "Generate a report from a template",
	Long: `Generate a report for configured cloud accounts.

Example:
  finops report list
  finops report generate costs --account-alias rh-control
  finops report generate costs --account-alias rh-control -o costs.html
  finops report generate costs --account 333333333333 --payer rhc -o member.html
  finops report generate costs --ou ou-abcd-1234 --payer rh-control -o ou-costs.html
  finops report generate costs --payer rh-control --tag-key env --tag-value prod -o prod.html`,
	Args: cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		sel, err := parseCostTargetSelector(
			reportGenerateAccount, reportGenerateAccountAliases, reportGenerateOU, reportGeneratePayer,
			reportGenerateTagKey, reportGenerateTagValue, reportGenerateOUDirect,
			reportGenerateSkipOrgCache, reportGenerateRefreshOrgCache,
		)
		if err != nil {
			return err
		}
		if _, err := validateCostTargetSelector(sel); err != nil {
			return err
		}
		if _, err := reportpkg.ParseTemplate(args[0]); err != nil {
			return err
		}
		if err := validatePeriodFlags(cmd); err != nil {
			return err
		}
		if _, err := reportpkg.ParseFormat(reportGenerateFormat); err != nil {
			return err
		}
		return validateOrgCacheFlags(reportGenerateSkipOrgCache, reportGenerateRefreshOrgCache)
	},
	RunE: runReportGenerate,
}

func init() {
	reportCmd.AddCommand(reportGenerateCmd)
	reportGenerateCmd.Flags().StringVar(&reportGenerateFormat, "format", reportpkg.FormatHTML, "Output format (supported: html)")
	reportGenerateCmd.Flags().StringVar(&reportGenerateAccount, "account", "", "Payer AWS account ID(s), comma-separated 12-digit IDs")
	reportGenerateCmd.Flags().StringVar(&reportGenerateAccountAliases, "account-alias", "", "Configured account alias(es), comma-separated (e.g. rh-control)")
	reportGenerateCmd.Flags().StringVar(&reportGenerateOU, "ou", "", "AWS OU ID(s), comma-separated (requires --payer; recursive by default)")
	reportGenerateCmd.Flags().BoolVar(&reportGenerateOUDirect, "ou-direct", false, "Include only accounts directly in --ou, not descendant OUs")
	reportGenerateCmd.Flags().StringVar(&reportGeneratePayer, "payer", "", "Registered payer alias for --account member IDs, --ou, or --tag-key (e.g. rhc)")
	reportGenerateCmd.Flags().StringVar(&reportGenerateTagKey, "tag-key", "", "Select accounts by AWS Organizations tag key")
	reportGenerateCmd.Flags().StringVar(&reportGenerateTagValue, "tag-value", "", "Optional tag value (omit to match any value for --tag-key)")
	reportGenerateCmd.Flags().StringVarP(&reportGenerateOutput, "output", "o", "", "Write HTML to this file instead of stdout")
	reportGenerateCmd.Flags().BoolVar(&reportGenerateQuiet, "quiet", false, "Suppress progress messages on stderr")
	reportGenerateCmd.Flags().BoolVar(&reportGenerateSkipOrgCache, "skip-org-cache", false, "Bypass cached organization account/tag data (always fetch live from AWS)")
	reportGenerateCmd.Flags().BoolVar(&reportGenerateRefreshOrgCache, "refresh-org-cache", false, "Ignore cached organization data and refresh the cache from AWS")
	addPeriodFlags(reportGenerateCmd)
}

func runReportGenerate(cmd *cobra.Command, args []string) error {
	templateName, err := reportpkg.ParseTemplate(args[0])
	if err != nil {
		return err
	}
	format, err := reportpkg.ParseFormat(reportGenerateFormat)
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

	status := progress.New(cmd.ErrOrStderr(), reportGenerateQuiet)

	sel, err := parseCostTargetSelector(
		reportGenerateAccount, reportGenerateAccountAliases, reportGenerateOU, reportGeneratePayer,
		reportGenerateTagKey, reportGenerateTagValue, reportGenerateOUDirect,
		reportGenerateSkipOrgCache, reportGenerateRefreshOrgCache,
	)
	if err != nil {
		return err
	}

	targets, err := resolveCostTargets(
		cmd.Context(), cmd, cfg, sel,
		awsFlags.ConfigPath, awsFlags.CredentialsFile, awsFlags.AuthMethod,
		status,
	)
	if err != nil {
		return err
	}

	dateRange, err := resolveCostPeriod(time.Now().UTC())
	if err != nil {
		return err
	}

	if len(targets) == 0 {
		var out *os.File
		if path := strings.TrimSpace(reportGenerateOutput); path != "" {
			f, err := os.Create(path)
			if err != nil {
				return fmt.Errorf("create output file: %w", err)
			}
			defer f.Close()
			out = f
		} else {
			out = os.Stdout
		}
		report := corereport.EmptyCostsReport(cost.CostQuery{
			Provider: cost.ProviderAWS,
			Range:    dateRange,
		}, time.Now().UTC())
		status.Step("Rendering HTML report…")
		if err := reportpkg.RenderCostsHTML(out, report); err != nil {
			return err
		}
		if !reportGenerateQuiet {
			if path := strings.TrimSpace(reportGenerateOutput); path != "" {
				status.Step(fmt.Sprintf("Wrote report to %s", path))
			} else {
				status.Step("Report written to stdout")
			}
		}
		return nil
	}

	status.Step("Ensuring AWS credentials…")
	if err := ensureCostCredentials(cmd.Context(), cmd, cfg, targets, awsFlags.ConfigPath, awsFlags.CredentialsFile, awsFlags.AuthMethod); err != nil {
		return err
	}
	if len(targets) <= 1 {
		status.Step("Preparing account configuration…")
	}
	targets, err = prepareCostTargets(cmd.Context(), cfg, targets, awsFlags.CredentialsFile, status)
	if err != nil {
		return err
	}

	if len(targets) > 1 {
		status.Step(fmt.Sprintf("Fetching net amortized costs for %d account(s) from AWS Cost Explorer…", len(targets)))
	}

	costQuery := cost.CostQuery{
		Provider: cost.ProviderAWS,
		Accounts: targets,
		Range:    dateRange,
		Progress: status,
		AWSFetch: &cost.AWSFetchOptions{
			ResolveAccountNames: coreaccount.ResolveAccountNames,
		},
	}

	var out *os.File
	if path := strings.TrimSpace(reportGenerateOutput); path != "" {
		f, err := os.Create(path)
		if err != nil {
			return fmt.Errorf("create output file: %w", err)
		}
		defer f.Close()
		out = f
	} else {
		out = os.Stdout
	}

	switch templateName {
	case reportpkg.TemplateCosts:
		if format != reportpkg.FormatHTML {
			return fmt.Errorf("template %q does not support format %q", templateName, format)
		}
		report, err := corereport.BuildCostsReport(cmd.Context(), costQuery, status)
		if err != nil {
			return err
		}
		status.Step("Rendering HTML report…")
		if err := reportpkg.RenderCostsHTML(out, report); err != nil {
			return err
		}
		if !reportGenerateQuiet {
			if path := strings.TrimSpace(reportGenerateOutput); path != "" {
				status.Step(fmt.Sprintf("Wrote report to %s", path))
			} else {
				status.Step("Report written to stdout")
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported template %q", templateName)
	}
}
