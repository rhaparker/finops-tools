// snowflake_oauth.go loads and saves Snowflake OAuth client credentials outside the main config file.
package configstore

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// SnowflakeOAuthSecrets holds OAuth client credentials (never commit to git).
type SnowflakeOAuthSecrets struct {
	ClientID     string `yaml:"client_id,omitempty"`
	ClientSecret string `yaml:"client_secret,omitempty"`
}

// DefaultSnowflakeOAuthSecretsPath returns ~/.config/finops/snowflake-oauth.yaml (or OS equivalent).
func DefaultSnowflakeOAuthSecretsPath() (string, error) {
	configPath, err := DefaultPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(configPath), "snowflake-oauth.yaml"), nil
}

// ResolveSnowflakeOAuthSecretsPath returns flagPath when set, otherwise DefaultSnowflakeOAuthSecretsPath.
func ResolveSnowflakeOAuthSecretsPath(flagPath string) (string, error) {
	if strings.TrimSpace(flagPath) != "" {
		return flagPath, nil
	}
	return DefaultSnowflakeOAuthSecretsPath()
}

// LoadSnowflakeOAuthSecrets reads secrets from path. Missing files yield empty secrets.
func LoadSnowflakeOAuthSecrets(path string) (SnowflakeOAuthSecrets, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return SnowflakeOAuthSecrets{}, nil
		}
		return SnowflakeOAuthSecrets{}, err
	}
	var secrets SnowflakeOAuthSecrets
	if err := yaml.Unmarshal(data, &secrets); err != nil {
		return SnowflakeOAuthSecrets{}, fmt.Errorf("parse snowflake oauth secrets %s: %w", path, err)
	}
	return secrets, nil
}

// SaveSnowflakeOAuthSecrets writes secrets atomically with mode 0600.
func SaveSnowflakeOAuthSecrets(path string, secrets SnowflakeOAuthSecrets) error {
	secrets.ClientID = strings.TrimSpace(secrets.ClientID)
	secrets.ClientSecret = strings.TrimSpace(secrets.ClientSecret)
	if secrets.ClientID == "" {
		return fmt.Errorf("client_id is required")
	}

	data, err := yaml.Marshal(secrets)
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".snowflake-oauth-*.yaml")
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

// ResolveSnowflakeOAuthClient resolves client ID and secret from secrets file and environment.
func ResolveSnowflakeOAuthClient(secretsPathFlag string) (clientID, clientSecret string, err error) {
	path, err := ResolveSnowflakeOAuthSecretsPath(secretsPathFlag)
	if err != nil {
		return "", "", err
	}
	secrets, err := LoadSnowflakeOAuthSecrets(path)
	if err != nil {
		return "", "", err
	}
	clientID = strings.TrimSpace(secrets.ClientID)
	clientSecret = strings.TrimSpace(secrets.ClientSecret)
	if v := strings.TrimSpace(os.Getenv("FINOPS_SNOWFLAKE_OAUTH_CLIENT_ID")); v != "" {
		clientID = v
	}
	if v := strings.TrimSpace(os.Getenv("FINOPS_SNOWFLAKE_OAUTH_CLIENT_SECRET")); v != "" {
		clientSecret = v
	}
	return clientID, clientSecret, nil
}
