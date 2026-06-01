package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/openshift-online/finops-tools/cli/internal/configstore"
	"github.com/openshift-online/finops-tools/cli/internal/snowflakeoauth"
	coresnowflake "github.com/openshift-online/finops-tools/core/snowflake"
	"github.com/openshift-online/finops-tools/cli/internal/output"
	"github.com/spf13/cobra"
)

var (
	snowflakeQuerySQL    string
	snowflakeQueryFormat string
)

var snowflakeQueryCmd = &cobra.Command{
	Use:   "query",
	Short: "Run a SQL query against a registered Snowflake account",
	Long: `Execute SQL using Red Hat SSO OAuth tokens stored by finops account add snowflake.

Examples:
  finops snowflake query --account-alias rhprod --sql "SELECT CURRENT_USER(), CURRENT_ROLE()"
  finops snowflake query --account-alias rhprod --sql "SELECT 1" --format json`,
	RunE: runSnowflakeQuery,
}

func init() {
	snowflakeCmd.AddCommand(snowflakeQueryCmd)
	snowflakeQueryCmd.Flags().StringVar(&snowflakeQuerySQL, "sql", "", "SQL statement to execute (required)")
	snowflakeQueryCmd.Flags().StringVar(&snowflakeFlags.AccountAlias, "account-alias", "", "Registered Snowflake account alias (required)")
	snowflakeQueryCmd.Flags().StringVar(&snowflakeQueryFormat, "format", string(output.FormatPrettyPrint),
		"Output format: pretty-print, json, csv")
	_ = snowflakeQueryCmd.MarkFlagRequired("sql")
	_ = snowflakeQueryCmd.MarkFlagRequired("account-alias")
}

func runSnowflakeQuery(cmd *cobra.Command, _ []string) error {
	format, err := output.ParseFormat(snowflakeQueryFormat)
	if err != nil {
		return err
	}

	path, err := configstore.ResolvePath(snowflakeFlags.ConfigPath)
	if err != nil {
		return err
	}
	cfg, err := configstore.Load(path)
	if err != nil {
		return err
	}

	alias := strings.TrimSpace(snowflakeFlags.AccountAlias)
	acct, ok := cfg.SnowflakeAccountForAlias(alias)
	if !ok {
		return fmt.Errorf("unknown snowflake account alias %q", alias)
	}

	tok, err := ensureSnowflakeAccessToken(cmd.Context(), cfg, alias, snowflakeFlags.SecretsPath, snowflakeFlags.TokensPath, acct)
	if err != nil {
		return err
	}

	clientID, clientSecret, err := resolveSnowflakeOAuthClient(snowflakeFlags.SecretsPath)
	if err != nil {
		return err
	}
	sso := acct.SSO
	if strings.TrimSpace(sso) == "" {
		sso = cfg.SnowflakeSSOIssuer()
	}
	oauthCfg, err := snowflakeOAuthConfig(cfg, clientID, clientSecret, sso)
	if err != nil {
		return err
	}
	claims, err := snowflakeoauth.ValidateDataverseToken(tok.AccessToken, oauthCfg.Audience)
	if err != nil {
		return err
	}

	db, err := coresnowflake.OpenDB(coresnowflake.ConnectParams{
		Account:   acct.Account,
		User:      claims.SnowflakeLoginName(),
		Token:     tok.AccessToken,
		Role:      acct.Role,
		Warehouse: acct.Warehouse,
		Database:  acct.Database,
		Schema:    acct.Schema,
	})
	if err != nil {
		return err
	}
	defer db.Close()

	result, err := coresnowflake.Query(cmd.Context(), db, snowflakeQuerySQL)
	if err != nil {
		return err
	}

	switch format {
	case output.FormatJSON:
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	case output.FormatCSV:
		return writeSnowflakeCSV(cmd.OutOrStdout(), result)
	default:
		return writeSnowflakePretty(cmd.OutOrStdout(), result)
	}
}

func writeSnowflakePretty(w interface{ Write([]byte) (int, error) }, result coresnowflake.QueryResult) error {
	if len(result.Rows) == 0 {
		_, err := fmt.Fprintln(w, "(no rows)")
		return err
	}
	for i, row := range result.Rows {
		if _, err := fmt.Fprintf(w, "row %d:\n", i+1); err != nil {
			return err
		}
		for _, col := range result.Columns {
			if _, err := fmt.Fprintf(w, "  %s: %s\n", col, row[col]); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeSnowflakeCSV(w interface{ Write([]byte) (int, error) }, result coresnowflake.QueryResult) error {
	if len(result.Columns) == 0 {
		return nil
	}
	if _, err := fmt.Fprintln(w, strings.Join(result.Columns, ",")); err != nil {
		return err
	}
	for _, row := range result.Rows {
		vals := make([]string, len(result.Columns))
		for i, col := range result.Columns {
			vals[i] = csvEscape(row[col])
		}
		if _, err := fmt.Fprintln(w, strings.Join(vals, ",")); err != nil {
			return err
		}
	}
	return nil
}

func csvEscape(s string) string {
	if strings.ContainsAny(s, ",\"\n") {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}
