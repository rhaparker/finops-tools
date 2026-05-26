// report_generate.go implements "finops report generate".
package cmd

import (
	"fmt"
	"os"
	"strings"

	awsconfig "github.com/openshift-online/finops-tools/cli/internal/aws"
	"github.com/openshift-online/finops-tools/cli/internal/configstore"
	"github.com/openshift-online/finops-tools/cli/internal/progress"
	reportpkg "github.com/openshift-online/finops-tools/cli/internal/report"
	"github.com/openshift-online/finops-tools/core/cost"
	corereport "github.com/openshift-online/finops-tools/core/report"
	"github.com/spf13/cobra"
)

var (
	reportGenerateAccount        string
	reportGenerateAccountAliases string
	reportGenerateFormat         string
	reportGenerateOutput         string
	reportGeneratePayer          string
	reportGenerateQuiet          bool
)

var reportGenerateCmd = &cobra.Command{
	Use:   "generate [template]",
	Short: "Generate a report from a template",
	Long: `Generate a report for configured cloud accounts.

Example:
  finops report list
  finops report generate costs --account-alias rh-control
  finops report generate costs --account-alias rh-control -o costs.html
  finops report generate costs --account 710019948333 --payer rhc -o member.html`,
	Args: cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(reportGenerateAccount) == "" && strings.TrimSpace(reportGenerateAccountAliases) == "" {
			return fmt.Errorf("at least one of --account or --account-alias is required")
		}
		if _, err := reportpkg.ParseTemplate(args[0]); err != nil {
			return err
		}
		if _, err := reportpkg.ParseFormat(reportGenerateFormat); err != nil {
			return err
		}
		if strings.TrimSpace(reportGeneratePayer) != "" && strings.TrimSpace(reportGenerateAccount) == "" {
			return fmt.Errorf("--payer requires --account")
		}
		return nil
	},
	RunE: runReportGenerate,
}

func init() {
	reportCmd.AddCommand(reportGenerateCmd)
	reportGenerateCmd.Flags().StringVar(&reportGenerateFormat, "format", reportpkg.FormatHTML, "Output format (supported: html)")
	reportGenerateCmd.Flags().StringVar(&reportGenerateAccount, "account", "", "Payer AWS account ID(s), comma-separated 12-digit IDs")
	reportGenerateCmd.Flags().StringVar(&reportGenerateAccountAliases, "account-alias", "", "Configured account alias(es), comma-separated (e.g. rh-control)")
	reportGenerateCmd.Flags().StringVar(&reportGeneratePayer, "payer", "", "Registered payer alias for --account member IDs not in config (e.g. rhc)")
	reportGenerateCmd.Flags().StringVarP(&reportGenerateOutput, "output", "o", "", "Write HTML to this file instead of stdout")
	reportGenerateCmd.Flags().BoolVar(&reportGenerateQuiet, "quiet", false, "Suppress progress messages on stderr")
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

	var accountIDs, aliases []string
	if strings.TrimSpace(reportGenerateAccount) != "" {
		accountIDs, err = configstore.ParseAWSAccountIDs(reportGenerateAccount)
		if err != nil {
			return err
		}
	}
	if strings.TrimSpace(reportGenerateAccountAliases) != "" {
		aliases, err = configstore.ParseAccountAliases(reportGenerateAccountAliases)
		if err != nil {
			return err
		}
	}

	targets, err := configstore.ResolveCostTargets(cfg, accountIDs, aliases, reportGeneratePayer)
	if err != nil {
		return err
	}

	status := progress.New(cmd.ErrOrStderr(), reportGenerateQuiet)

	status.Step("Ensuring AWS credentials…")
	if err := ensureCostCredentials(cmd.Context(), cmd, cfg, targets, awsFlags.ConfigPath, awsFlags.CredentialsFile, awsFlags.AuthMethod); err != nil {
		return err
	}
	status.Step("Preparing account configuration…")
	targets, err = prepareCostTargets(cmd.Context(), cfg, targets, awsFlags.CredentialsFile)
	if err != nil {
		return err
	}

	costQuery := cost.CostQuery{
		Provider: cost.ProviderAWS,
		Accounts: targets,
		Days:     cost.DefaultDays,
		AWSFetch: &cost.AWSFetchOptions{
			ResolveAccountNames: awsconfig.ResolveAccountNames,
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
