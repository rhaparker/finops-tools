// Package snowflakecred stores Snowflake OAuth tokens per finops account alias.
package snowflakecred

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/openshift-online/finops-tools/cli/internal/snowflakeoauth"
	"gopkg.in/yaml.v3"
)

// TokensFile maps alias → token set.
type TokensFile map[string]storedToken

type storedToken struct {
	AccessToken  string    `yaml:"access_token"`
	RefreshToken string    `yaml:"refresh_token,omitempty"`
	Expiry       time.Time `yaml:"expiry,omitempty"`
}

// DefaultTokensPath returns ~/.config/finops/snowflake-tokens.yaml.
func DefaultTokensPath() (string, error) {
	dir, err := finopsConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "snowflake-tokens.yaml"), nil
}

func finopsConfigDir() (string, error) {
	if runtime.GOOS == "windows" {
		dir, err := os.UserConfigDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(dir, "finops"), nil
	}
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configHome = filepath.Join(home, ".config")
	}
	return filepath.Join(configHome, "finops"), nil
}

// Load reads tokens from path. Missing files yield an empty map.
func Load(path string) (TokensFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return TokensFile{}, nil
		}
		return nil, err
	}
	var file TokensFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse snowflake tokens %s: %w", path, err)
	}
	if file == nil {
		file = TokensFile{}
	}
	return file, nil
}

// Save writes tokens atomically with mode 0600.
func Save(path string, file TokensFile) error {
	data, err := yaml.Marshal(file)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".snowflake-tokens-*.yaml")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, path)
}

// Get returns the token set for alias.
func (f TokensFile) Get(alias string) (snowflakeoauth.TokenSet, bool) {
	entry, ok := f[strings.TrimSpace(alias)]
	if !ok {
		return snowflakeoauth.TokenSet{}, false
	}
	return snowflakeoauth.TokenSet{
		AccessToken:  entry.AccessToken,
		RefreshToken: entry.RefreshToken,
		Expiry:       entry.Expiry,
	}, true
}

// Set stores tokens for alias.
func (f TokensFile) Set(alias string, tok snowflakeoauth.TokenSet) {
	if f == nil {
		return
	}
	f[strings.TrimSpace(alias)] = storedToken{
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		Expiry:       tok.Expiry,
	}
}
