// snowflake_accounts_list.go lists registered Snowflake accounts from the finops config.
package configstore

import (
	"slices"
	"strings"
)

// SnowflakeAccountListEntry is one registered Snowflake account alias.
type SnowflakeAccountListEntry struct {
	Alias     string
	Account   string
	Role      string
	Warehouse string
	Database  string
	SSO       string
}

// ListSnowflakeAccounts returns registered Snowflake account aliases, sorted by alias.
func (f File) ListSnowflakeAccounts() []SnowflakeAccountListEntry {
	if len(f.Snowflake.AccountAliases) == 0 {
		return nil
	}
	out := make([]SnowflakeAccountListEntry, 0, len(f.Snowflake.AccountAliases))
	for alias, entry := range f.Snowflake.AccountAliases {
		out = append(out, SnowflakeAccountListEntry{
			Alias:     alias,
			Account:   entry.Account,
			Role:      entry.Role,
			Warehouse: entry.Warehouse,
			Database:  entry.Database,
			SSO:       snowflakeSSOLabel(entry.SSO),
		})
	}
	slices.SortFunc(out, func(a, b SnowflakeAccountListEntry) int {
		return strings.Compare(a.Alias, b.Alias)
	})
	return out
}

func snowflakeSSOLabel(sso string) string {
	switch strings.ToLower(strings.TrimSpace(sso)) {
	case "", "prod", "production":
		return "prod"
	case "stage", "staging":
		return "stage"
	default:
		return strings.TrimSpace(sso)
	}
}
