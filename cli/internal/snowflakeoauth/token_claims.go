package snowflakeoauth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// TokenClaims holds JWT claims needed for Dataverse Snowflake access.
type TokenClaims struct {
	Issuer            string
	Audience          []string
	Scopes            []string
	PreferredUsername string
	Email             string
}

// ParseTokenClaims decodes the JWT payload without signature verification (local inspection only).
func ParseTokenClaims(accessToken string) (TokenClaims, error) {
	payload, err := jwtPayloadJSON(accessToken)
	if err != nil {
		return TokenClaims{}, err
	}
	return claimsFromPayload(payload), nil
}

// ValidateDataverseToken checks claims required by Dataverse Snowflake external OAuth.
func ValidateDataverseToken(accessToken, audience string) (TokenClaims, error) {
	if audience == "" {
		audience = DefaultAudience
	}
	claims, err := ParseTokenClaims(accessToken)
	if err != nil {
		return TokenClaims{}, err
	}
	if !claims.hasAudience(audience) {
		return claims, fmt.Errorf(
			"access token audience must include %q (got %v); ensure the SSO client adds an audience mapper for Dataverse Snowflake",
			audience, claims.Audience,
		)
	}
	if !claims.hasScope(ScopeSessionRoleAny) {
		return claims, fmt.Errorf(
			"access token scope must include %q (got %v); ask IAM to assign %q as a default client scope on your SSO client",
			ScopeSessionRoleAny, claims.Scopes, ScopeSessionRoleAny,
		)
	}
	return claims, nil
}

// SnowflakeLoginName returns the username to send to Snowflake (gosnowflake LoginName).
func (c TokenClaims) SnowflakeLoginName() string {
	if u := strings.TrimSpace(c.PreferredUsername); u != "" {
		return u
	}
	if e := strings.TrimSpace(c.Email); e != "" {
		return e
	}
	return ""
}

func (c TokenClaims) hasAudience(want string) bool {
	want = strings.TrimSpace(want)
	for _, aud := range c.Audience {
		if aud == want {
			return true
		}
	}
	return false
}

func (c TokenClaims) hasScope(want string) bool {
	for _, s := range c.Scopes {
		if s == want {
			return true
		}
	}
	return false
}

func jwtPayloadJSON(accessToken string) (map[string]any, error) {
	parts := strings.Split(accessToken, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("not a JWT access token")
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode jwt payload: %w", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("parse jwt payload: %w", err)
	}
	return payload, nil
}

func claimsFromPayload(payload map[string]any) TokenClaims {
	var out TokenClaims
	out.Issuer = stringClaim(payload, "iss")
	out.PreferredUsername = stringClaim(payload, "preferred_username")
	out.Email = stringClaim(payload, "email")
	out.Audience = audienceClaim(payload["aud"])
	out.Scopes = scopeClaim(payload["scope"])
	return out
}

func stringClaim(payload map[string]any, key string) string {
	v, ok := payload[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

func audienceClaim(v any) []string {
	switch t := v.(type) {
	case string:
		if t != "" {
			return []string{t}
		}
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func scopeClaim(v any) []string {
	s, ok := v.(string)
	if !ok || strings.TrimSpace(s) == "" {
		return nil
	}
	return strings.Fields(s)
}
