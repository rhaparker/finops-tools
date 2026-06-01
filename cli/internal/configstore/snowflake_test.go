package configstore

import (
	"path/filepath"
	"testing"
)

func TestRegisterSnowflakeAccount(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	acct := SnowflakeAccount{
		Account: "ORG-ACCT",
		Role:    "PUBLIC",
		SSO:     "prod",
	}
	if err := RegisterSnowflakeAccount(path, "rhprod", acct); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := cfg.SnowflakeAccountForAlias("rhprod")
	if !ok || got.Account != "ORG-ACCT" || got.Role != "PUBLIC" {
		t.Fatalf("got %+v ok=%v", got, ok)
	}
}

func TestResolveSnowflakeOAuthClientUsesDefaultPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "snowflake-oauth.yaml")
	if err := SaveSnowflakeOAuthSecrets(path, SnowflakeOAuthSecrets{
		ClientID:     "from-file",
		ClientSecret: "secret",
	}); err != nil {
		t.Fatal(err)
	}

	clientID, clientSecret, err := ResolveSnowflakeOAuthClient(path)
	if err != nil {
		t.Fatal(err)
	}
	if clientID != "from-file" || clientSecret != "secret" {
		t.Fatalf("got id=%q secret=%q", clientID, clientSecret)
	}
}

func TestResolveSnowflakeOAuthClientEmptyFlagUsesDefaultPath(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	defaultPath, err := DefaultSnowflakeOAuthSecretsPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := SaveSnowflakeOAuthSecrets(defaultPath, SnowflakeOAuthSecrets{
		ClientID:     "default-path-id",
		ClientSecret: "s3cret",
	}); err != nil {
		t.Fatal(err)
	}

	clientID, _, err := ResolveSnowflakeOAuthClient("")
	if err != nil {
		t.Fatal(err)
	}
	if clientID != "default-path-id" {
		t.Fatalf("client_id = %q, want default-path-id", clientID)
	}
}

func TestSnowflakeOAuthScopesDefault(t *testing.T) {
	cfg := Default()
	if scopes := cfg.SnowflakeOAuthScopes(); scopes != nil {
		t.Fatalf("default scopes = %v, want nil (SSO client defaults)", scopes)
	}
}

func TestSnowflakeOAuthScopesFromConfig(t *testing.T) {
	cfg := Default()
	var err error
	cfg, err = cfg.SetDefault(DefaultFQNSnowflakeOAuthScopes, "openid,session:role-any")
	if err != nil {
		t.Fatal(err)
	}
	scopes := cfg.SnowflakeOAuthScopes()
	if len(scopes) != 2 || scopes[1] != "session:role-any" {
		t.Fatalf("got %v", scopes)
	}
}

func TestSnowflakeOAuthSecretsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "snowflake-oauth.yaml")

	secrets := SnowflakeOAuthSecrets{ClientID: "my-client", ClientSecret: "s3cret"}
	if err := SaveSnowflakeOAuthSecrets(path, secrets); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadSnowflakeOAuthSecrets(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ClientID != secrets.ClientID || loaded.ClientSecret != secrets.ClientSecret {
		t.Fatalf("got %+v", loaded)
	}
}
