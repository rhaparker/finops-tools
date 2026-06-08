package cmd

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/openshift-online/finops-tools/cli/internal/configstore"
	"github.com/openshift-online/finops-tools/cli/internal/snowflakeoauth"
)

func TestEnsureSnowflakeAccessToken_UsesValidCachedToken(t *testing.T) {
	reset := stubSnowflakeSessionDeps()
	defer reset()

	resolveSnowflakeOAuthClientFn = func(string) (string, string, error) { return "client-id", "secret", nil }
	loadSnowflakeTokenFn = func(string, string) (snowflakeoauth.TokenSet, error) {
		return snowflakeoauth.TokenSet{
			AccessToken: "cached",
			Expiry:      time.Now().Add(10 * time.Minute),
		}, nil
	}

	refreshCalled := false
	refreshSnowflakeTokenFn = func(context.Context, snowflakeoauth.ClientConfig, string) (snowflakeoauth.TokenSet, error) {
		refreshCalled = true
		return snowflakeoauth.TokenSet{}, nil
	}
	loginCalled := false
	loginSnowflakeTokenFn = func(context.Context, snowflakeoauth.ClientConfig) (snowflakeoauth.TokenSet, error) {
		loginCalled = true
		return snowflakeoauth.TokenSet{}, nil
	}
	persistSnowflakeTokenFn = func(string, string, snowflakeoauth.TokenSet) error {
		t.Fatal("persist should not be called")
		return nil
	}

	got, _, err := ensureSnowflakeAccessToken(context.Background(), configstore.Default(), "rhprod", "", "", configstore.SnowflakeAccount{Account: "acct"})
	if err != nil {
		t.Fatalf("ensureSnowflakeAccessToken error: %v", err)
	}
	if got.AccessToken != "cached" {
		t.Fatalf("expected cached token, got %q", got.AccessToken)
	}
	if refreshCalled {
		t.Fatal("refresh should not be called for valid token")
	}
	if loginCalled {
		t.Fatal("login should not be called for valid token")
	}
}

func TestEnsureSnowflakeAccessToken_RefreshesExpiredToken(t *testing.T) {
	reset := stubSnowflakeSessionDeps()
	defer reset()

	resolveSnowflakeOAuthClientFn = func(string) (string, string, error) { return "client-id", "secret", nil }
	loadSnowflakeTokenFn = func(string, string) (snowflakeoauth.TokenSet, error) {
		return snowflakeoauth.TokenSet{
			AccessToken:  "expired",
			RefreshToken: "refresh",
			Expiry:       time.Now().Add(-10 * time.Minute),
		}, nil
	}
	refreshSnowflakeTokenFn = func(context.Context, snowflakeoauth.ClientConfig, string) (snowflakeoauth.TokenSet, error) {
		return snowflakeoauth.TokenSet{
			AccessToken: "refreshed",
			Expiry:      time.Now().Add(10 * time.Minute),
		}, nil
	}
	loginSnowflakeTokenFn = func(context.Context, snowflakeoauth.ClientConfig) (snowflakeoauth.TokenSet, error) {
		t.Fatal("login should not be called when refresh succeeds")
		return snowflakeoauth.TokenSet{}, nil
	}
	persisted := snowflakeoauth.TokenSet{}
	persistSnowflakeTokenFn = func(_, _ string, tok snowflakeoauth.TokenSet) error {
		persisted = tok
		return nil
	}

	got, _, err := ensureSnowflakeAccessToken(context.Background(), configstore.Default(), "rhprod", "", "", configstore.SnowflakeAccount{Account: "acct"})
	if err != nil {
		t.Fatalf("ensureSnowflakeAccessToken error: %v", err)
	}
	if got.AccessToken != "refreshed" {
		t.Fatalf("expected refreshed token, got %q", got.AccessToken)
	}
	if persisted.AccessToken != "refreshed" {
		t.Fatalf("expected refreshed token to persist, got %q", persisted.AccessToken)
	}
}

func TestEnsureSnowflakeAccessToken_FallsBackToLoginWhenRefreshFails(t *testing.T) {
	reset := stubSnowflakeSessionDeps()
	defer reset()

	resolveSnowflakeOAuthClientFn = func(string) (string, string, error) { return "client-id", "secret", nil }
	loadSnowflakeTokenFn = func(string, string) (snowflakeoauth.TokenSet, error) {
		return snowflakeoauth.TokenSet{
			AccessToken:  "expired",
			RefreshToken: "refresh",
			Expiry:       time.Now().Add(-5 * time.Minute),
		}, nil
	}
	refreshSnowflakeTokenFn = func(context.Context, snowflakeoauth.ClientConfig, string) (snowflakeoauth.TokenSet, error) {
		return snowflakeoauth.TokenSet{}, errors.New("refresh failed")
	}
	loginSnowflakeTokenFn = func(context.Context, snowflakeoauth.ClientConfig) (snowflakeoauth.TokenSet, error) {
		return snowflakeoauth.TokenSet{
			AccessToken: "interactive",
			Expiry:      time.Now().Add(5 * time.Minute),
		}, nil
	}
	persisted := snowflakeoauth.TokenSet{}
	persistSnowflakeTokenFn = func(_, _ string, tok snowflakeoauth.TokenSet) error {
		persisted = tok
		return nil
	}

	got, _, err := ensureSnowflakeAccessToken(context.Background(), configstore.Default(), "rhprod", "", "", configstore.SnowflakeAccount{Account: "acct"})
	if err != nil {
		t.Fatalf("ensureSnowflakeAccessToken error: %v", err)
	}
	if got.AccessToken != "interactive" {
		t.Fatalf("expected login token, got %q", got.AccessToken)
	}
	if persisted.AccessToken != "interactive" {
		t.Fatalf("expected login token to persist, got %q", persisted.AccessToken)
	}
}

func TestEnsureSnowflakeAccessToken_LogsInWhenNoCachedToken(t *testing.T) {
	reset := stubSnowflakeSessionDeps()
	defer reset()

	resolveSnowflakeOAuthClientFn = func(string) (string, string, error) { return "client-id", "secret", nil }
	loadSnowflakeTokenFn = func(string, string) (snowflakeoauth.TokenSet, error) {
		return snowflakeoauth.TokenSet{}, errSnowflakeTokensNotFound
	}
	refreshSnowflakeTokenFn = func(context.Context, snowflakeoauth.ClientConfig, string) (snowflakeoauth.TokenSet, error) {
		t.Fatal("refresh should not be called without a cached token")
		return snowflakeoauth.TokenSet{}, nil
	}
	loginSnowflakeTokenFn = func(context.Context, snowflakeoauth.ClientConfig) (snowflakeoauth.TokenSet, error) {
		return snowflakeoauth.TokenSet{
			AccessToken: "interactive",
			Expiry:      time.Now().Add(5 * time.Minute),
		}, nil
	}
	persistCalled := false
	persistSnowflakeTokenFn = func(string, string, snowflakeoauth.TokenSet) error {
		persistCalled = true
		return nil
	}

	got, _, err := ensureSnowflakeAccessToken(context.Background(), configstore.Default(), "rhprod", "", "", configstore.SnowflakeAccount{Account: "acct"})
	if err != nil {
		t.Fatalf("ensureSnowflakeAccessToken error: %v", err)
	}
	if got.AccessToken != "interactive" {
		t.Fatalf("expected login token, got %q", got.AccessToken)
	}
	if !persistCalled {
		t.Fatal("expected login token to be persisted")
	}
}

func TestEnsureSnowflakeAccessToken_ReturnsClearLoginFailure(t *testing.T) {
	reset := stubSnowflakeSessionDeps()
	defer reset()

	resolveSnowflakeOAuthClientFn = func(string) (string, string, error) { return "client-id", "secret", nil }
	loadSnowflakeTokenFn = func(string, string) (snowflakeoauth.TokenSet, error) {
		return snowflakeoauth.TokenSet{}, errSnowflakeTokensNotFound
	}
	loginSnowflakeTokenFn = func(context.Context, snowflakeoauth.ClientConfig) (snowflakeoauth.TokenSet, error) {
		return snowflakeoauth.TokenSet{}, errors.New("browser canceled")
	}
	persistSnowflakeTokenFn = func(string, string, snowflakeoauth.TokenSet) error {
		t.Fatal("persist should not be called when login fails")
		return nil
	}

	_, _, err := ensureSnowflakeAccessToken(context.Background(), configstore.Default(), "rhprod", "", "", configstore.SnowflakeAccount{Account: "acct"})
	if err == nil {
		t.Fatal("expected login failure")
	}
	if !strings.Contains(err.Error(), "snowflake oauth login failed for alias \"rhprod\"") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "browser canceled") {
		t.Fatalf("expected wrapped login error, got: %v", err)
	}
}

func stubSnowflakeSessionDeps() func() {
	resolveOrig := resolveSnowflakeOAuthClientFn
	loadOrig := loadSnowflakeTokenFn
	persistOrig := persistSnowflakeTokenFn
	refreshOrig := refreshSnowflakeTokenFn
	loginOrig := loginSnowflakeTokenFn
	return func() {
		resolveSnowflakeOAuthClientFn = resolveOrig
		loadSnowflakeTokenFn = loadOrig
		persistSnowflakeTokenFn = persistOrig
		refreshSnowflakeTokenFn = refreshOrig
		loginSnowflakeTokenFn = loginOrig
	}
}
