// Package snowflake runs SQL queries against Snowflake using an OAuth access token.
package snowflake

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/snowflakedb/gosnowflake"
)

// ConnectParams configures a Snowflake connection with Red Hat SSO OAuth.
type ConnectParams struct {
	Account   string
	User      string // Snowflake login name (from SSO token); required by gosnowflake for OAUTH
	Token     string
	Role      string
	Warehouse string
	Database  string
	Schema    string
}

// OpenDB opens a database/sql handle using OAuth token authentication.
func OpenDB(params ConnectParams) (*sql.DB, error) {
	account := strings.TrimSpace(params.Account)
	token := strings.TrimSpace(params.Token)
	if account == "" {
		return nil, fmt.Errorf("snowflake account is required")
	}
	if token == "" {
		return nil, fmt.Errorf("oauth access token is required")
	}

	cfg := &gosnowflake.Config{
		Account:       account,
		User:          strings.TrimSpace(params.User),
		Authenticator: gosnowflake.AuthTypeOAuth,
		Token:         token,
		Role:          strings.TrimSpace(params.Role),
		Warehouse:     strings.TrimSpace(params.Warehouse),
		Database:      strings.TrimSpace(params.Database),
		Schema:        strings.TrimSpace(params.Schema),
	}
	dsn, err := gosnowflake.DSN(cfg)
	if err != nil {
		return nil, fmt.Errorf("build snowflake DSN: %w", err)
	}
	db, err := sql.Open("snowflake", dsn)
	if err != nil {
		return nil, fmt.Errorf("open snowflake: %w", err)
	}
	return db, nil
}

// QueryRow runs a single-row query and scans into dest.
func QueryRow(ctx context.Context, db *sql.DB, query string, dest ...any) error {
	return db.QueryRowContext(ctx, query).Scan(dest...)
}

// Ping verifies the connection is usable.
func Ping(ctx context.Context, db *sql.DB) error {
	return db.PingContext(ctx)
}
