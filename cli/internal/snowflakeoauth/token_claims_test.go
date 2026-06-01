package snowflakeoauth

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestValidateDataverseToken(t *testing.T) {
	token := testJWT(t, map[string]any{
		"iss":                "https://auth.redhat.com/auth/realms/EmployeeIDP",
		"aud":                "dataverse-snowflake",
		"scope":              "openid session:role-any",
		"preferred_username": "joe@redhat.com",
	})
	claims, err := ValidateDataverseToken(token, DefaultAudience)
	if err != nil {
		t.Fatal(err)
	}
	if claims.SnowflakeLoginName() != "joe@redhat.com" {
		t.Fatalf("login name = %q", claims.SnowflakeLoginName())
	}
}

func TestValidateDataverseTokenMissingScope(t *testing.T) {
	token := testJWT(t, map[string]any{
		"aud":   "dataverse-snowflake",
		"scope": "openid",
	})
	_, err := ValidateDataverseToken(token, DefaultAudience)
	if err == nil {
		t.Fatal("expected error")
	}
}

func testJWT(t *testing.T, claims map[string]any) string {
	t.Helper()
	b, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	payload := base64.RawURLEncoding.EncodeToString(b)
	return "header." + payload + ".sig"
}
