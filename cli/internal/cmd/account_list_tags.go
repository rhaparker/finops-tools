// account_list_tags.go implements "finops account list-tags" to list AWS Organizations tags for one account.
package cmd

import (
	"fmt"
	"strings"

	"github.com/openshift-online/finops-tools/cli/internal/account"
	"github.com/openshift-online/finops-tools/cli/internal/awsauth"
	"github.com/openshift-online/finops-tools/cli/internal/configstore"
	"github.com/openshift-online/finops-tools/cli/internal/output"
	coreaccount "github.com/openshift-online/finops-tools/core/account"
	"github.com/spf13/cobra"
)

var accountTagsCmd = &cobra.Command{
	Use:   "list-tags",
	Short: "List AWS Organizations tags for an account",
	Long: `List all AWS Organizations tags for an account.

Pass either --account-alias or --account-id.

Examples:
  finops account list-tags --account-alias rh-control
  finops account list-tags --account-alias osd-tenant-1 --format json
  finops account list-tags --account-id 123456789012 --format csv
  finops account list-tags --account-id 111111111111 --payer rh-control`,
	Args: cobra.NoArgs,
	PreRunE: func(_ *cobra.Command, _ []string) error {
		_, err := output.ParseFormat(accountTagsFormat)
		return err
	},
	RunE: runAccountTags,
}

var (
	accountTagsEnsureCredentials  = awsauth.EnsureAccountCredentials
	accountTagsLoadConfigForCreds = loadAWSConfigForCredentialsAccount
	accountTagsFetch              = coreaccount.ListTags
	accountTagsFormat             string
	accountTagsPayer              string
	accountTagsAlias              string
	accountTagsAccountID          string
)

type accountTagsTarget struct {
	AccountID            string
	CredentialsAccountID string
	Alias                string
}

func init() {
	accountCmd.AddCommand(accountTagsCmd)
	accountTagsCmd.Flags().StringVar(&accountTagsFormat, "format", string(output.FormatPrettyPrint),
		"Output format: pretty-print, json, csv")
	accountTagsCmd.Flags().StringVar(&accountTagsPayer, "payer", "", "Registered payer alias to use for credentials when listing account tags")
	accountTagsCmd.Flags().StringVar(&accountTagsAlias, "account-alias", "", "Registered account alias")
	accountTagsCmd.Flags().StringVar(&accountTagsAccountID, "account-id", "", "12-digit AWS account ID")
}

func runAccountTags(cmd *cobra.Command, args []string) error {
	format, err := output.ParseFormat(accountTagsFormat)
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

	target, err := resolveAccountTagsTargetExplicit(cfg, accountTagsAlias, accountTagsAccountID)
	if err != nil {
		return err
	}
	if payerAlias := strings.TrimSpace(accountTagsPayer); payerAlias != "" {
		payerID, ok := cfg.PayerAccountIDForAlias(payerAlias)
		if !ok {
			return errUnknownPayerAlias(payerAlias)
		}
		target.CredentialsAccountID = payerID
	}

	profiles := account.AWSProfileNames(
		target.CredentialsAccountID,
		cfg.PayerAliasForAccountID(target.CredentialsAccountID),
		nil,
	)

	ensureOpts, err := newAWSEnsureOptions(cmd, awsEnsureConfig{
		configPath:      awsFlags.ConfigPath,
		authMethodFlag:  awsFlags.AuthMethod,
		credentialsFile: awsFlags.CredentialsFile,
	})
	if err != nil {
		return err
	}
	ensureOpts.AccountName = target.CredentialsAccountID
	ensureOpts.ProfileNames = profiles
	if _, err := accountTagsEnsureCredentials(cmd.Context(), ensureOpts); err != nil {
		return fmt.Errorf("%s: %w", target.CredentialsAccountID, mapCredentialError(target.CredentialsAccountID, err))
	}

	awsCfg, err := accountTagsLoadConfigForCreds(cmd.Context(), cfg, target.CredentialsAccountID, awsFlags.CredentialsFile)
	if err != nil {
		return err
	}
	tags, err := accountTagsFetch(cmd.Context(), awsCfg, target.AccountID)
	if err != nil {
		return fmt.Errorf("list tags for account %s: %w", target.AccountID, err)
	}

	rows := make([]output.AccountTagRow, len(tags))
	for i, tag := range tags {
		rows[i] = output.AccountTagRow{
			Key:   tag.Key,
			Value: tag.Value,
		}
	}
	return output.WriteAWSAccountTagsResult(cmd.OutOrStdout(), format, output.AccountTagsView{
		AccountID: target.AccountID,
		Alias:     target.Alias,
		Tags:      rows,
	})
}

func resolveAccountTagsTargetExplicit(cfg configstore.File, accountAlias, accountID string) (accountTagsTarget, error) {
	accountAlias = strings.TrimSpace(accountAlias)
	accountID = strings.TrimSpace(accountID)
	if accountAlias != "" && accountID != "" {
		return accountTagsTarget{}, fmt.Errorf("provide exactly one of --account-alias or --account-id")
	}
	if accountAlias == "" && accountID == "" {
		return accountTagsTarget{}, fmt.Errorf("provide exactly one of --account-alias or --account-id")
	}

	if accountAlias != "" {
		if linked, ok := cfg.LinkedAccountForAlias(accountAlias); ok {
			payerID, ok := cfg.PayerAccountIDForAlias(linked.PayerAlias)
			if !ok {
				return accountTagsTarget{}, fmt.Errorf("unknown payer alias %q for linked account %q", linked.PayerAlias, accountAlias)
			}
			return accountTagsTarget{
				AccountID:            linked.AccountID,
				CredentialsAccountID: payerID,
				Alias:                accountAlias,
			}, nil
		}

		if payerID, ok := cfg.PayerAccountIDForAlias(accountAlias); ok {
			return accountTagsTarget{
				AccountID:            payerID,
				CredentialsAccountID: payerID,
				Alias:                accountAlias,
			}, nil
		}
		return accountTagsTarget{}, fmt.Errorf("unknown account alias %q", accountAlias)
	}

	if err := account.ValidateAWSAccountID(accountID); err != nil {
		return accountTagsTarget{}, err
	}

	credsID := accountID
	if payerID, ok := cfg.PayerAccountIDForLinkedAccountID(accountID); ok {
		credsID = payerID
	}

	alias := cfg.AliasForAccountID(accountID)
	if alias == accountID {
		alias = ""
	}

	return accountTagsTarget{
		AccountID:            accountID,
		CredentialsAccountID: credsID,
		Alias:                alias,
	}, nil
}
