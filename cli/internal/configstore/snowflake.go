// snowflake.go holds Snowflake account aliases in the finops config file.
package configstore

import (
	"fmt"
	"strings"
)

// SnowflakeAccount holds non-secret Snowflake connection settings for an alias.
type SnowflakeAccount struct {
	Account   string `yaml:"account"`
	Role      string `yaml:"role,omitempty"`
	Warehouse string `yaml:"warehouse,omitempty"`
	Database  string `yaml:"database,omitempty"`
	Schema    string `yaml:"schema,omitempty"`
	// SSO selects the Red Hat SSO issuer: "prod" (default) or "stage".
	SSO string `yaml:"sso,omitempty"`
}

// SnowflakeConfig holds Snowflake-specific settings.
type SnowflakeConfig struct {
	AccountAliases map[string]SnowflakeAccount `yaml:"account_aliases,omitempty"`
}

// SetSnowflakeAlias records alias → account settings and returns the updated config.
func (f File) SetSnowflakeAlias(alias string, acct SnowflakeAccount) (File, error) {
	alias = strings.TrimSpace(alias)
	acct.Account = strings.TrimSpace(acct.Account)
	if alias == "" {
		return File{}, fmt.Errorf("alias is required")
	}
	if acct.Account == "" {
		return File{}, fmt.Errorf("snowflake account identifier is required")
	}
	if f.Snowflake.AccountAliases == nil {
		f.Snowflake.AccountAliases = make(map[string]SnowflakeAccount)
	}
	f.Snowflake.AccountAliases[alias] = acct
	return f, nil
}

// SnowflakeAccountForAlias returns settings for alias, if configured.
func (f File) SnowflakeAccountForAlias(alias string) (SnowflakeAccount, bool) {
	entry, ok := f.Snowflake.AccountAliases[strings.TrimSpace(alias)]
	if !ok {
		return SnowflakeAccount{}, false
	}
	return entry, true
}

// HasSnowflakeAccount reports whether the Snowflake account identifier is registered.
func (f File) HasSnowflakeAccount(account string) bool {
	account = strings.TrimSpace(account)
	for _, entry := range f.Snowflake.AccountAliases {
		if strings.TrimSpace(entry.Account) == account {
			return true
		}
	}
	return false
}

// RegisterSnowflakeAccount ensures the config file exists and records alias → account.
func RegisterSnowflakeAccount(path, alias string, acct SnowflakeAccount) error {
	cfg, err := Ensure(path)
	if err != nil {
		return err
	}
	if alias == "" {
		alias = acct.Account
	}
	cfg, err = cfg.SetSnowflakeAlias(alias, acct)
	if err != nil {
		return err
	}
	return Save(path, cfg)
}
