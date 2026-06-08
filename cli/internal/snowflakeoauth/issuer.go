// Package snowflakeoauth performs Red Hat SSO OAuth for Dataverse Snowflake access.
package snowflakeoauth

import (
	"fmt"
	"strings"
)

const (
	// DefaultAudience is the JWT audience required by Dataverse Snowflake (configurable via finops defaults).
	DefaultAudience = "dataverse-snowflake"
	// ScopeSessionRoleAny allows Snowflake to assume any role granted to the user.
	ScopeSessionRoleAny = "session:role-any"
)

// IssuerURLs holds OIDC endpoints for a Red Hat SSO realm.
type IssuerURLs struct {
	Issuer        string
	AuthorizeURL  string
	TokenURL      string
	DeviceAuthURL string
}

// IssuerForEnv returns Red Hat SSO issuer URLs for prod or stage.
func IssuerForEnv(env string) (IssuerURLs, error) {
	switch strings.ToLower(strings.TrimSpace(env)) {
	case "", "prod", "production":
		return prodIssuer(), nil
	case "stage", "staging":
		return stageIssuer(), nil
	default:
		return IssuerURLs{}, fmt.Errorf("unknown snowflake SSO environment %q (use prod or stage)", env)
	}
}

func prodIssuer() IssuerURLs {
	issuer := "https://auth.redhat.com/auth/realms/EmployeeIDP"
	return issuerURLs(issuer)
}

func stageIssuer() IssuerURLs {
	issuer := "https://auth.stage.redhat.com/auth/realms/EmployeeIDP"
	return issuerURLs(issuer)
}

func issuerURLs(issuer string) IssuerURLs {
	base := strings.TrimSuffix(issuer, "/")
	return IssuerURLs{
		Issuer:        base,
		AuthorizeURL:  base + "/protocol/openid-connect/auth",
		TokenURL:      base + "/protocol/openid-connect/token",
		DeviceAuthURL: base + "/protocol/openid-connect/auth/device",
	}
}
