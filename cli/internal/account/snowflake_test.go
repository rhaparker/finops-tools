package account

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/openshift-online/finops-tools/cli/internal/snowflakeoauth"
)

func TestAddSnowflake_ReusesValidCachedToken(t *testing.T) {
	t.Parallel()

	loginCalled := false
	token := testSnowflakeAccessToken(t, "cached-user")

	_, err := AddSnowflake(context.Background(), AddSnowflakeOptions{
		Account: SnowflakeAccountSettings{
			Account:   "ORG-ACCT",
			Warehouse: "MY_WH",
		},
		Alias: "rhprod",
		OAuth: snowflakeoauth.ClientConfig{
			ClientID: "client-id",
			Audience: snowflakeoauth.DefaultAudience,
		},
		ExistingToken: snowflakeoauth.TokenSet{
			AccessToken: token,
			Expiry:      time.Now().Add(10 * time.Minute),
		},
		TokensPath: t.TempDir() + "/tokens.yaml",
		Login: func(context.Context, snowflakeoauth.ClientConfig, snowflakeoauth.TokenSet) (snowflakeoauth.TokenSet, error) {
			loginCalled = true
			return snowflakeoauth.TokenSet{}, errors.New("login should not run")
		},
	})
	if loginCalled {
		t.Fatal("login should not be called for a valid cached token")
	}
	if err == nil {
		t.Fatal("expected connection error after reusing cached token")
	}
	if !strings.Contains(err.Error(), "snowflake connection check") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func testSnowflakeAccessToken(t *testing.T, username string) string {
	t.Helper()
	payload, err := json.Marshal(map[string]any{
		"aud":                snowflakeoauth.DefaultAudience,
		"scope":              snowflakeoauth.ScopeSessionRoleAny,
		"preferred_username": username,
	})
	if err != nil {
		t.Fatal(err)
	}
	segment := base64.RawURLEncoding.EncodeToString(payload)
	return "header." + segment + ".signature"
}
