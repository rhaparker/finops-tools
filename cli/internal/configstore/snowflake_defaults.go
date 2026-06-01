// snowflake_defaults.go defines fully qualified Snowflake configuration defaults.
package configstore

import (
	"fmt"
	"strings"
)

const (
	// DefaultFQNSnowflakeSSOIssuer selects Red Hat SSO: prod or stage.
	DefaultFQNSnowflakeSSOIssuer = "snowflake.sso_issuer"
	// DefaultFQNSnowflakeOAuthAudience is the required JWT audience for Dataverse Snowflake.
	DefaultFQNSnowflakeOAuthAudience = "snowflake.oauth_audience"
	// DefaultFQNSnowflakeOAuthScopes is a comma-separated OAuth scope list for Red Hat SSO login.
	DefaultFQNSnowflakeOAuthScopes = "snowflake.oauth_scopes"
)

const (
	defaultSnowflakeOAuthAudience = "dataverse-snowflake"
	defaultSnowflakeSSOIssuer     = "prod"
)

func validateSnowflakeDefaultValue(fqn, value string) error {
	switch fqn {
	case DefaultFQNSnowflakeSSOIssuer:
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "", "prod", "production", "stage", "staging":
			return nil
		default:
			return fmt.Errorf("snowflake.sso_issuer must be prod or stage")
		}
	case DefaultFQNSnowflakeOAuthAudience:
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("snowflake.oauth_audience cannot be empty")
		}
		return nil
	case DefaultFQNSnowflakeOAuthScopes:
		// Empty value is allowed: use SSO client default scopes only.
		return nil
	default:
		return fmt.Errorf("unknown snowflake default %q", fqn)
	}
}

// SnowflakeSSOIssuer returns the configured Red Hat SSO environment (prod or stage).
func (f File) SnowflakeSSOIssuer() string {
	if v, ok := f.Default(DefaultFQNSnowflakeSSOIssuer); ok && strings.TrimSpace(v) != "" {
		return strings.ToLower(strings.TrimSpace(v))
	}
	return defaultSnowflakeSSOIssuer
}

// SnowflakeOAuthAudience returns the JWT audience required by Dataverse Snowflake.
func (f File) SnowflakeOAuthAudience() string {
	if v, ok := f.Default(DefaultFQNSnowflakeOAuthAudience); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return defaultSnowflakeOAuthAudience
}

// SnowflakeOAuthScopes returns OAuth scopes to request from Red Hat SSO.
// When unset, returns nil so the authorize request omits scope and the SSO client
// default scopes apply.
func (f File) SnowflakeOAuthScopes() []string {
	v, ok := f.Default(DefaultFQNSnowflakeOAuthScopes)
	if !ok || strings.TrimSpace(v) == "" {
		return nil
	}
	return parseSnowflakeOAuthScopes(v)
}

func parseSnowflakeOAuthScopes(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
