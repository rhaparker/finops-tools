// account_list_ous.go implements "finops account list-ous".
package cmd

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/openshift-online/finops-tools/cli/internal/account"
	"github.com/openshift-online/finops-tools/cli/internal/awsauth"
	"github.com/openshift-online/finops-tools/cli/internal/configstore"
	"github.com/openshift-online/finops-tools/cli/internal/output"
	coreaccount "github.com/openshift-online/finops-tools/core/account"
	"github.com/spf13/cobra"
)

var (
	accountListOUsEnsureCredentials  = awsauth.EnsureAccountCredentials
	accountListOUsLoadConfigForCreds = loadAWSConfigForCredentialsAccount
	accountListOUsFetch              = coreaccount.ListOrganizationalUnits
	accountListOUsFormat             string
	accountListOUsParent             string
	accountListOUsPayer              string
)

var parentIDPattern = regexp.MustCompile(`^(ou-[0-9a-z]{4,32}-[0-9a-z]{4,32}|r-[0-9a-z]{4,32}-[0-9a-z]{4,32})$`)

var accountListOUsCmd = &cobra.Command{
	Use:   "list-ous",
	Short: "List AWS Organizational Units under a payer",
	Long: `List child AWS Organizational Units for discovery before using --ou on cost/report commands.

Examples:
  finops account list-ous --payer rh-control
  finops account list-ous --payer rh-control --parent ou-abcd-1234
  finops account list-ous --payer rh-control --format json`,
	Args: cobra.NoArgs,
	PreRunE: func(_ *cobra.Command, _ []string) error {
		if strings.TrimSpace(accountListOUsPayer) == "" {
			return fmt.Errorf("--payer is required")
		}
		if parent := strings.TrimSpace(accountListOUsParent); parent != "" && !parentIDPattern.MatchString(parent) {
			return fmt.Errorf("invalid parent ID %q (expected ou-xxxx-yyyyy or r-xxxx-yyyyy)", parent)
		}
		_, err := output.ParseFormat(accountListOUsFormat)
		return err
	},
	RunE: runAccountListOUs,
}

func init() {
	accountCmd.AddCommand(accountListOUsCmd)
	accountListOUsCmd.Flags().StringVar(&accountListOUsFormat, "format", string(output.FormatPrettyPrint),
		"Output format: pretty-print, json, csv")
	accountListOUsCmd.Flags().StringVar(&accountListOUsPayer, "payer", "", "Registered payer alias (required)")
	accountListOUsCmd.Flags().StringVar(&accountListOUsParent, "parent", "", "Parent OU or root ID (default: organization root)")
}

func runAccountListOUs(cmd *cobra.Command, _ []string) error {
	format, err := output.ParseFormat(accountListOUsFormat)
	if err != nil {
		return err
	}

	configPath, err := configstore.ResolvePath(awsFlags.ConfigPath)
	if err != nil {
		return err
	}
	cfg, err := configstore.Load(configPath)
	if err != nil {
		return err
	}

	payerAlias := strings.TrimSpace(accountListOUsPayer)
	payerID, ok := cfg.PayerAccountIDForAlias(payerAlias)
	if !ok {
		return errUnknownPayerAlias(payerAlias)
	}

	profiles := account.AWSProfileNames(payerID, payerAlias, nil)
	ensureOpts, err := newAWSEnsureOptions(cmd, awsEnsureConfig{
		configPath:      awsFlags.ConfigPath,
		authMethodFlag:  awsFlags.AuthMethod,
		credentialsFile: awsFlags.CredentialsFile,
	})
	if err != nil {
		return err
	}
	ensureOpts.AccountName = payerID
	ensureOpts.ProfileNames = profiles
	if _, err := accountListOUsEnsureCredentials(cmd.Context(), ensureOpts); err != nil {
		return fmt.Errorf("%s: %w", payerID, mapCredentialError(payerID, err))
	}

	awsCfg, err := accountListOUsLoadConfigForCreds(cmd.Context(), cfg, payerID, awsFlags.CredentialsFile)
	if err != nil {
		return err
	}

	parentID := strings.TrimSpace(accountListOUsParent)
	ous, err := accountListOUsFetch(cmd.Context(), awsCfg, parentID)
	if err != nil {
		return fmt.Errorf("list organizational units: %w", err)
	}

	rows := make([]output.OrganizationalUnitRow, len(ous))
	for i, ou := range ous {
		rows[i] = output.OrganizationalUnitRow{
			ID:   ou.ID,
			Name: ou.Name,
		}
	}
	return output.WriteOrganizationalUnitListResult(cmd.OutOrStdout(), format, output.OrganizationalUnitListView{
		PayerAlias: payerAlias,
		ParentID:   parentID,
		Units:      rows,
	})
}
