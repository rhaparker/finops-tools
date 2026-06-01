package cmd

import (
	"fmt"
	"os"

	"github.com/openshift-online/finops-tools/cli/internal/configstore"
	"github.com/spf13/cobra"
)

var (
	snowflakeOAuthSetClientID     string
	snowflakeOAuthSetClientSecret string
	snowflakeOAuthSetSecretsPath  string
)

var configSnowflakeOAuthSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Store Snowflake OAuth client credentials",
	Long: `Store OAuth client ID and secret for Red Hat SSO → Snowflake access.

Credentials are saved outside the main finops config file (default:
~/.config/finops/snowflake-oauth.yaml, mode 0600). Do not commit this file.

You can also set FINOPS_SNOWFLAKE_OAUTH_CLIENT_ID and FINOPS_SNOWFLAKE_OAUTH_CLIENT_SECRET.

Example:
  finops config snowflake oauth set --client-id finops-tools-dataverse --client-secret "$SECRET"`,
	RunE: runConfigSnowflakeOAuthSet,
}

var configSnowflakeOAuthCmd = &cobra.Command{
	Use:   "oauth",
	Short: "Manage Snowflake OAuth client credentials",
}

func init() {
	configSnowflakeCmd.AddCommand(configSnowflakeOAuthCmd)
	configSnowflakeOAuthCmd.AddCommand(configSnowflakeOAuthSetCmd)
	configSnowflakeOAuthSetCmd.Flags().StringVar(&snowflakeOAuthSetClientID, "client-id", "", "Red Hat SSO OAuth client ID (required)")
	configSnowflakeOAuthSetCmd.Flags().StringVar(&snowflakeOAuthSetClientSecret, "client-secret", "", "Red Hat SSO OAuth client secret")
	configSnowflakeOAuthSetCmd.Flags().StringVar(&snowflakeOAuthSetSecretsPath, "secrets-file", "",
		"Path to snowflake OAuth secrets file (default: alongside finops config)")
	_ = configSnowflakeOAuthSetCmd.MarkFlagRequired("client-id")
}

func runConfigSnowflakeOAuthSet(cmd *cobra.Command, _ []string) error {
	secret := snowflakeOAuthSetClientSecret
	if secret == "" {
		secret = os.Getenv("FINOPS_SNOWFLAKE_OAUTH_CLIENT_SECRET")
	}
	if secret == "" {
		return fmt.Errorf("--client-secret or FINOPS_SNOWFLAKE_OAUTH_CLIENT_SECRET is required")
	}

	path, err := configstore.ResolveSnowflakeOAuthSecretsPath(snowflakeOAuthSetSecretsPath)
	if err != nil {
		return err
	}
	if err := configstore.SaveSnowflakeOAuthSecrets(path, configstore.SnowflakeOAuthSecrets{
		ClientID:     snowflakeOAuthSetClientID,
		ClientSecret: secret,
	}); err != nil {
		return err
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "Snowflake OAuth credentials saved to %s\n", path)
	return err
}
