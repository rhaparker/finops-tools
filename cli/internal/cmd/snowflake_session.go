package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/openshift-online/finops-tools/cli/internal/configstore"
	"github.com/openshift-online/finops-tools/cli/internal/snowflakecred"
	"github.com/openshift-online/finops-tools/cli/internal/snowflakeoauth"
)

func resolveSnowflakeOAuthClient(secretsPath string) (clientID, clientSecret string, err error) {
	return configstore.ResolveSnowflakeOAuthClient(secretsPath)
}

func snowflakeOAuthConfig(cfg configstore.File, clientID, clientSecret, ssoEnv string) (snowflakeoauth.ClientConfig, error) {
	issuer, err := snowflakeoauth.IssuerForEnv(ssoEnv)
	if err != nil {
		return snowflakeoauth.ClientConfig{}, err
	}
	if strings.TrimSpace(clientID) == "" {
		return snowflakeoauth.ClientConfig{}, fmt.Errorf(
			"snowflake oauth client_id not configured; run finops config snowflake oauth set --client-id <id> or set FINOPS_SNOWFLAKE_OAUTH_CLIENT_ID",
		)
	}
	return snowflakeoauth.ClientConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Audience:     cfg.SnowflakeOAuthAudience(),
		Scopes:       cfg.SnowflakeOAuthScopes(),
		Issuer:       issuer,
	}, nil
}

func loadSnowflakeToken(alias, tokensPath string) (snowflakeoauth.TokenSet, error) {
	path := tokensPath
	if path == "" {
		var err error
		path, err = snowflakecred.DefaultTokensPath()
		if err != nil {
			return snowflakeoauth.TokenSet{}, err
		}
	}
	file, err := snowflakecred.Load(path)
	if err != nil {
		return snowflakeoauth.TokenSet{}, err
	}
	tok, ok := file.Get(alias)
	if !ok {
		return snowflakeoauth.TokenSet{}, fmt.Errorf("no oauth tokens for snowflake alias %q; run finops account add snowflake", alias)
	}
	return tok, nil
}

func ensureSnowflakeAccessToken(ctx context.Context, cfg configstore.File, alias, secretsPath, tokensPath string, acct configstore.SnowflakeAccount) (snowflakeoauth.TokenSet, error) {
	clientID, clientSecret, err := resolveSnowflakeOAuthClient(secretsPath)
	if err != nil {
		return snowflakeoauth.TokenSet{}, err
	}
	sso := acct.SSO
	if strings.TrimSpace(sso) == "" {
		sso = cfg.SnowflakeSSOIssuer()
	}
	oauthCfg, err := snowflakeOAuthConfig(cfg, clientID, clientSecret, sso)
	if err != nil {
		return snowflakeoauth.TokenSet{}, err
	}

	tok, err := loadSnowflakeToken(alias, tokensPath)
	if err != nil {
		return snowflakeoauth.TokenSet{}, err
	}
	if tok.Valid() {
		return tok, nil
	}
	if strings.TrimSpace(tok.RefreshToken) != "" {
		refreshed, err := snowflakeoauth.Refresh(ctx, oauthCfg, tok.RefreshToken)
		if err == nil && refreshed.Valid() {
			if err := persistSnowflakeToken(alias, tokensPath, refreshed); err != nil {
				return snowflakeoauth.TokenSet{}, err
			}
			return refreshed, nil
		}
	}
	return snowflakeoauth.TokenSet{}, fmt.Errorf("snowflake oauth token expired for alias %q; run finops account add snowflake %s --alias %s --force",
		alias, acct.Account, alias)
}

func persistSnowflakeToken(alias, tokensPath string, tok snowflakeoauth.TokenSet) error {
	path := tokensPath
	if path == "" {
		var err error
		path, err = snowflakecred.DefaultTokensPath()
		if err != nil {
			return err
		}
	}
	file, err := snowflakecred.Load(path)
	if err != nil {
		return err
	}
	file.Set(alias, tok)
	return snowflakecred.Save(path, file)
}
