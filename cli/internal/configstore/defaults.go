// defaults.go reads and writes fully qualified configuration defaults (e.g. aws.auth_method).
package configstore

import (
	"fmt"
	"strings"

	"github.com/openshift-online/finops-tools/cli/internal/awsauth"
	"github.com/openshift-online/finops-tools/cli/internal/awsrole"
)

// DefaultFQNAWSAuthMethod is the fully qualified default name for AWS authentication.
const DefaultFQNAWSAuthMethod = "aws.auth_method"

// DefaultFQNGCPAuthMethod is the fully qualified default name for GCP authentication (reserved).
const DefaultFQNGCPAuthMethod = "gcp.auth_method"

// DefaultFQNAWSLinkedRole is the fully qualified default IAM role name for linked account access.
const DefaultFQNAWSLinkedRole = "aws.linked_role"

// Cost period defaults use DefaultFQNCost* in cost_defaults.go.

// SetDefault sets a fully qualified default and returns the updated config.
func (f File) SetDefault(fqn, value string) (File, error) {
	key, err := normalizeDefaultFQN(fqn)
	if err != nil {
		return File{}, err
	}
	if err := validateDefaultValue(key, value); err != nil {
		return File{}, err
	}
	value = strings.TrimSpace(value)
	if key == DefaultFQNSnowflakeAccountAlias {
		if _, ok := f.SnowflakeAccountForAlias(value); !ok {
			return File{}, fmt.Errorf("snowflake account alias %q is not registered", value)
		}
	}
	if f.Defaults == nil {
		f.Defaults = make(map[string]string)
	}
	f.Defaults[key] = value
	return f, nil
}

// Default returns a configured value for a fully qualified default name.
func (f File) Default(fqn string) (string, bool) {
	key, err := normalizeDefaultFQN(fqn)
	if err != nil {
		return "", false
	}
	if f.Defaults == nil {
		return "", false
	}
	v := strings.TrimSpace(f.Defaults[key])
	return v, v != ""
}

func normalizeDefaultFQN(name string) (string, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	if !strings.Contains(name, ".") {
		return "", fmt.Errorf("default name %q must be fully qualified (e.g. %s)", name, DefaultFQNAWSAuthMethod)
	}
	provider, setting, ok := strings.Cut(name, ".")
	if !ok || provider == "" || setting == "" {
		return "", fmt.Errorf("invalid default name %q (expected <provider>.<setting>)", name)
	}
	setting = strings.ReplaceAll(setting, "-", "_")
	return provider + "." + setting, nil
}

func validateDefaultValue(fqn, value string) error {
	switch fqn {
	case DefaultFQNAWSAuthMethod:
		_, err := awsauth.ParseMethod(value)
		return err
	case DefaultFQNAWSLinkedRole:
		return awsrole.ValidateName(value)
	case DefaultFQNGCPAuthMethod:
		return fmt.Errorf("default %s is not supported yet", fqn)
	case DefaultFQNCostDays, DefaultFQNCostMonths, DefaultFQNCostFrom, DefaultFQNCostTo, DefaultFQNCostExcludeRecentDays:
		return validateCostDefaultValue(fqn, value)
	case DefaultFQNSnowflakeSSOIssuer, DefaultFQNSnowflakeOAuthAudience, DefaultFQNSnowflakeAccountAlias:
		return validateSnowflakeDefaultValue(fqn, value)
	default:
		return fmt.Errorf("unknown default %q (supported: %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)",
			fqn,
			DefaultFQNAWSAuthMethod, DefaultFQNAWSLinkedRole,
			DefaultFQNCostDays, DefaultFQNCostMonths, DefaultFQNCostFrom, DefaultFQNCostTo, DefaultFQNCostExcludeRecentDays,
			DefaultFQNSnowflakeSSOIssuer, DefaultFQNSnowflakeOAuthAudience,
			DefaultFQNSnowflakeAccountAlias)
	}
}

// SetDefault persists a fully qualified default in the config file at path.
func SetDefault(path, fqn, value string) error {
	cfg, err := Ensure(path)
	if err != nil {
		return err
	}
	cfg, err = cfg.SetDefault(fqn, value)
	if err != nil {
		return err
	}
	return Save(path, cfg)
}

// migrateDefaults moves legacy aws.defaults entries into the top-level defaults map.
func (f *File) migrateDefaults() {
	if len(f.AWS.Defaults) == 0 {
		return
	}
	if f.Defaults == nil {
		f.Defaults = make(map[string]string)
	}
	for k, v := range f.AWS.Defaults {
		fqn, err := legacyAWSDefaultFQN(k)
		if err != nil {
			continue
		}
		if _, exists := f.Defaults[fqn]; !exists {
			f.Defaults[fqn] = v
		}
	}
	f.AWS.Defaults = nil
}

func legacyAWSDefaultFQN(shortKey string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(shortKey)) {
	case "auth-method", "auth_method":
		return DefaultFQNAWSAuthMethod, nil
	default:
		return "", fmt.Errorf("unknown legacy default %q", shortKey)
	}
}
