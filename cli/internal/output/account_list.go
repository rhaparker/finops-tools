// account_list.go formats registered cloud accounts for terminal output.
package output

import "io"

// AccountListRow is one registered account for list display.
type AccountListRow struct {
	Alias      string
	AccountID  string
	Kind       string // "payer", "linked", or empty (e.g. GCP)
	PayerAlias string
	Role       string
}

// WriteAWSAccountList renders registered AWS accounts (payer and linked).
func WriteAWSAccountList(w io.Writer, entries []AccountListRow) error {
	return writeAWSAccountListPretty(w, entries)
}

// WriteGCPAccountList renders registered GCP account aliases.
func WriteGCPAccountList(w io.Writer, entries []AccountListRow) error {
	return writeGCPAccountListPretty(w, entries)
}

// WriteSnowflakeAccountList renders registered Snowflake account aliases.
func WriteSnowflakeAccountList(w io.Writer, entries []AccountListRow) error {
	return writeSnowflakeAccountListPretty(w, entries)
}
