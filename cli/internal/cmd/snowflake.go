package cmd

import (
	"github.com/spf13/cobra"
)

var snowflakeCmd = &cobra.Command{
	Use:   "snowflake",
	Short: "Run Snowflake SQL queries",
}

func init() {
	snowflakeCmd.GroupID = "core"
	bindSnowflakePersistentFlags(snowflakeCmd)
	rootCmd.AddCommand(snowflakeCmd)
}

var snowflakeFlags struct {
	ConfigPath     string
	SecretsPath    string
	TokensPath     string
	AccountAlias   string
}

func bindSnowflakePersistentFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&snowflakeFlags.ConfigPath, "config", "",
		"Path to finops config file (default: OS-specific config dir)")
	cmd.PersistentFlags().StringVar(&snowflakeFlags.SecretsPath, "oauth-secrets-file", "",
		"Path to Snowflake OAuth client secrets file")
	cmd.PersistentFlags().StringVar(&snowflakeFlags.TokensPath, "tokens-file", "",
		"Path to Snowflake OAuth tokens file")
	cmd.PersistentFlags().StringVar(&snowflakeFlags.AccountAlias, "account-alias", "",
		"Registered Snowflake account alias (default: snowflake.account_alias config or first registered account)")
}
