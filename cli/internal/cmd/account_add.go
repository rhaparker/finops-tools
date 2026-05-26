// account_add.go implements "finops account add" to log in and register payer or linked AWS accounts.
package cmd

import (
	"fmt"
	"strings"

	"github.com/openshift-online/finops-tools/cli/internal/account"
	"github.com/openshift-online/finops-tools/cli/internal/awsauth"
	awsconfig "github.com/openshift-online/finops-tools/cli/internal/aws"
	"github.com/spf13/cobra"
)

var (
	accountAddForce           bool
	accountAddAlias           string
	accountAddPayer           string
	accountAddRole            string
)

var accountAddCmd = &cobra.Command{
	Use:   "add <provider> <account>",
	Short: "Log in and register a payer or linked AWS account",
	Long: `Log into a cloud account and save it in the finops config file.

For AWS, <account> must be a 12-digit account ID.

Payer account (Cost Explorer and org billing):
  finops account add aws 123456789012 --alias rh-control

Linked account (assume a role from a registered payer):
  finops account add aws 111111111111 --alias osd-tenant-1 --payer rh-control
  finops account add aws 111111111111 --payer rh-control --role CustomRole

The role name defaults to OrganizationAccountAccessRole, or the value of
defaults.aws.linked_role in the finops config (finops config default set).

With --auth-method saml (default): runs rh-aws-saml-login for the payer when credentials are missing.
With --auth-method profile: uses an existing ~/.aws profile when valid.

cost get supports payer and linked account aliases. Linked accounts use payer credentials
and the configured auth method (defaults.aws.auth_method or --auth-method).`,
	Args: cobra.ExactArgs(2),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		provider, err := account.ParseProvider(args[0])
		if err != nil {
			return err
		}
		if provider != account.ProviderAWS {
			return nil
		}
		if err := account.ValidateAWSAccountID(args[1]); err != nil {
			return err
		}
		if strings.TrimSpace(accountAddPayer) != "" {
			if _, err := resolveLinkedRoleName(cmd, awsFlags.ConfigPath, accountAddRole); err != nil {
				return err
			}
		}
		method, err := resolveAuthMethod(cmd, awsFlags.ConfigPath, awsFlags.AuthMethod)
		if err != nil {
			return err
		}
		if accountAddForce && method == awsauth.MethodProfile {
			return fmt.Errorf("cannot use --force with --auth-method profile")
		}
		return nil
	},
	RunE: runAccountAdd,
}

func init() {
	accountCmd.AddCommand(accountAddCmd)
	accountAddCmd.Flags().BoolVar(&accountAddForce, "force", false, "Re-run SAML login even if existing payer credentials are valid (AWS only)")
	accountAddCmd.Flags().StringVar(&accountAddAlias, "alias", "", "Friendly alias for the account (e.g. rh-control)")
	accountAddCmd.Flags().StringVar(&accountAddPayer, "payer", "", "Registered payer alias (linked account: assume role in <account> from this payer)")
	accountAddCmd.Flags().StringVar(&accountAddRole, "role", "", "IAM role name in the linked account (default: config aws.linked_role or OrganizationAccountAccessRole)")
}

func runAccountAdd(cmd *cobra.Command, args []string) error {
	provider, err := account.ParseProvider(args[0])
	if err != nil {
		return err
	}
	accountID := args[1]

	if provider == account.ProviderAWS && strings.TrimSpace(accountAddPayer) != "" {
		return runAccountAddLinked(cmd, accountID)
	}

	ensureOpts, err := buildAWSEnsureOptions(cmd)
	if err != nil {
		return err
	}

	res, err := account.Add(cmd.Context(), account.AddOptions{
		Provider:      provider,
		AccountID:     accountID,
		Alias:         accountAddAlias,
		AWSEnsureOpts: ensureOpts,
	})
	if err != nil {
		return err
	}

	if err := registerAWSAccount(awsFlags.ConfigPath, res.AccountID, accountAddAlias); err != nil {
		return err
	}

	return printAccountAddResult(cmd, res, accountAddAlias, "payer")
}

func runAccountAddLinked(cmd *cobra.Command, linkedAccountID string) error {
	payerAlias := strings.TrimSpace(accountAddPayer)
	payerAccountID, err := payerAccountIDFromConfig(awsFlags.ConfigPath, payerAlias)
	if err != nil {
		return err
	}

	roleARN, err := resolveLinkedRoleARN(cmd, awsFlags.ConfigPath, linkedAccountID, accountAddRole)
	if err != nil {
		return err
	}
	roleName, err := resolveLinkedRoleName(cmd, awsFlags.ConfigPath, accountAddRole)
	if err != nil {
		return err
	}

	payerEnsure, err := buildAWSEnsureOptions(cmd)
	if err != nil {
		return err
	}

	res, err := account.AddAWSLinked(cmd.Context(), account.AddAWSLinkedOptions{
		LinkedAccountID: linkedAccountID,
		Alias:           accountAddAlias,
		PayerAccountID:  payerAccountID,
		PayerAlias:      payerAlias,
		RoleARN:         roleARN,
		PayerEnsureOpts: payerEnsure,
		EnsureLinkedOpts: awsconfig.EnsureLinkedOptions{
			CredentialsPath: awsFlags.CredentialsFile,
		},
	})
	if err != nil {
		return err
	}

	if err := registerAWSLinkedAccount(awsFlags.ConfigPath, res.AccountID, accountAddAlias, payerAlias, roleName); err != nil {
		return err
	}

	return printAccountAddResult(cmd, res, accountAddAlias, "linked")
}

func buildAWSEnsureOptions(cmd *cobra.Command) (awsauth.EnsureOptions, error) {
	return newAWSEnsureOptions(cmd, awsEnsureConfig{
		configPath:      awsFlags.ConfigPath,
		authMethodFlag:  awsFlags.AuthMethod,
		force:           accountAddForce,
		credentialsFile: awsFlags.CredentialsFile,
	})
}

func printAccountAddResult(cmd *cobra.Command, res account.AddResult, alias, kind string) error {
	label := alias
	if label == "" {
		label = res.AccountID
	}
	if !res.Refreshed {
		_, err := fmt.Fprintf(cmd.OutOrStdout(),
			"%s account %s added (provider=%s profile=%s account=%s arn=%s)\n",
			kind, label, res.Provider, res.Profile, res.AccountID, res.ARN,
		)
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(),
		"%s account %s added and credentials stored (provider=%s profile=%s account=%s arn=%s)\n",
		kind, label, res.Provider, res.Profile, res.AccountID, res.ARN,
	)
	return err
}
