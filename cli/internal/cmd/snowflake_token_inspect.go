package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/openshift-online/finops-tools/cli/internal/configstore"
	"github.com/openshift-online/finops-tools/cli/internal/snowflakeoauth"
	"github.com/spf13/cobra"
)

var snowflakeTokenInspectCmd = &cobra.Command{
	Use:   "inspect-token",
	Short: "Inspect the stored OAuth access token for a Snowflake account alias",
	Long: `Decode JWT claims from the cached Red Hat SSO access token (no Snowflake connection).

Use this after account add fails with "Invalid OAuth access token" to verify aud and scope.`,
	RunE: runSnowflakeTokenInspect,
}

func init() {
	snowflakeCmd.AddCommand(snowflakeTokenInspectCmd)
	snowflakeTokenInspectCmd.Flags().StringVar(&snowflakeFlags.AccountAlias, "account-alias", "", "Registered Snowflake account alias (required)")
	_ = snowflakeTokenInspectCmd.MarkFlagRequired("account-alias")
}

func runSnowflakeTokenInspect(cmd *cobra.Command, _ []string) error {
	path, err := configstore.ResolvePath(snowflakeFlags.ConfigPath)
	if err != nil {
		return err
	}
	cfg, err := configstore.Load(path)
	if err != nil {
		return err
	}

	alias := strings.TrimSpace(snowflakeFlags.AccountAlias)
	tok, err := loadSnowflakeToken(alias, snowflakeFlags.TokensPath)
	if err != nil {
		return err
	}
	if !tok.Valid() && strings.TrimSpace(tok.AccessToken) == "" {
		return fmt.Errorf("no access token for alias %q; run finops account add snowflake --alias %s --force", alias, alias)
	}

	claims, err := snowflakeoauth.ParseTokenClaims(tok.AccessToken)
	if err != nil {
		return err
	}

	audience := cfg.SnowflakeOAuthAudience()
	validationErr := ""
	if _, err := snowflakeoauth.ValidateDataverseToken(tok.AccessToken, audience); err != nil {
		validationErr = err.Error()
	}

	out := map[string]any{
		"issuer":             claims.Issuer,
		"audience":           claims.Audience,
		"scopes":             claims.Scopes,
		"preferred_username": claims.PreferredUsername,
		"email":              claims.Email,
		"snowflake_login":    claims.SnowflakeLoginName(),
		"required_audience":  audience,
		"required_scope":     snowflakeoauth.ScopeSessionRoleAny,
		"dataverse_ok":       validationErr == "",
	}
	if validationErr != "" {
		out["validation_error"] = validationErr
	}

	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
