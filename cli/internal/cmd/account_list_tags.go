// account_list_tags.go implements "finops account list-tags" to list AWS Organizations tags for one account.
package cmd

import (
	"fmt"
	"strings"

	"github.com/openshift-online/finops-tools/cli/internal/account"
	awsconfig "github.com/openshift-online/finops-tools/cli/internal/aws"
	"github.com/openshift-online/finops-tools/cli/internal/awsauth"
	"github.com/openshift-online/finops-tools/cli/internal/configstore"
	"github.com/openshift-online/finops-tools/cli/internal/output"
	"github.com/spf13/cobra"
)

var accountTagsCmd = &cobra.Command{
	Use:   "list-tags <account>",
	Short: "List AWS Organizations tags for an account",
	Long: `List all AWS Organizations tags for an account.

<account> can be a registered alias or a 12-digit AWS account ID.

Examples:
  finops account list-tags rh-control
  finops account list-tags osd-tenant-1 --format json
  finops account list-tags 123456789012 --format csv`,
	Args: cobra.ExactArgs(1),
	PreRunE: func(_ *cobra.Command, _ []string) error {
		_, err := output.ParseFormat(accountTagsFormat)
		return err
	},
	RunE: runAccountTags,
}

var (
	accountTagsEnsureCredentials  = awsauth.EnsureAccountCredentials
	accountTagsResolveCredentials = awsconfig.ResolveCredentials
	accountTagsLoadProfile        = awsconfig.LoadSharedConfigProfile
	accountTagsFetch              = awsconfig.AccountTags
	accountTagsFormat             string
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

	target, err := resolveAccountTagsTarget(cfg, args[0])
	if err != nil {
		return err
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

	res, status, err := accountTagsResolveCredentials(cmd.Context(), awsconfig.ResolveOptions{
		AccountName:     target.CredentialsAccountID,
		ProfileNames:    profiles,
		CredentialsPath: awsFlags.CredentialsFile,
	})
	if err != nil {
		return fmt.Errorf("%s: %w", target.CredentialsAccountID, err)
	}
	if status != awsconfig.CredentialsValid {
		return fmt.Errorf("%s: %w", target.CredentialsAccountID, mapCredentialStatusError(target.CredentialsAccountID, status))
	}

	profile := res.Profile
	if profile == "" {
		profile = awsconfig.SanitizeProfileName(target.CredentialsAccountID)
	}
	awsCfg, err := accountTagsLoadProfile(cmd.Context(), profile)
	if err != nil {
		return fmt.Errorf("%s: load AWS profile %q: %w", target.CredentialsAccountID, profile, err)
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

func resolveAccountTagsTarget(cfg configstore.File, accountRef string) (accountTagsTarget, error) {
	accountRef = strings.TrimSpace(accountRef)
	if accountRef == "" {
		return accountTagsTarget{}, fmt.Errorf("account is required")
	}

	if linked, ok := cfg.LinkedAccountForAlias(accountRef); ok {
		payerID, ok := cfg.PayerAccountIDForAlias(linked.PayerAlias)
		if !ok {
			return accountTagsTarget{}, fmt.Errorf("unknown payer alias %q for linked account %q", linked.PayerAlias, accountRef)
		}
		return accountTagsTarget{
			AccountID:            linked.AccountID,
			CredentialsAccountID: payerID,
			Alias:                accountRef,
		}, nil
	}

	if payerID, ok := cfg.PayerAccountIDForAlias(accountRef); ok {
		return accountTagsTarget{
			AccountID:            payerID,
			CredentialsAccountID: payerID,
			Alias:                accountRef,
		}, nil
	}

	if err := account.ValidateAWSAccountID(accountRef); err != nil {
		return accountTagsTarget{}, fmt.Errorf(
			"unknown account alias %q (use a registered alias or 12-digit AWS account ID)",
			accountRef,
		)
	}

	credsID := accountRef
	if payerID, ok := cfg.PayerAccountIDForLinkedAccountID(accountRef); ok {
		credsID = payerID
	}

	alias := cfg.AliasForAccountID(accountRef)
	if alias == accountRef {
		alias = ""
	}

	return accountTagsTarget{
		AccountID:            accountRef,
		CredentialsAccountID: credsID,
		Alias:                alias,
	}, nil
}
