// provider.go defines supported cloud account providers and parses provider flag values.
package account

import (
	"fmt"
	"strings"
)

// Provider identifies a cloud account provider.
type Provider string

const (
	ProviderAWS       Provider = "aws"
	ProviderGCP       Provider = "gcp"
	ProviderSnowflake Provider = "snowflake"
)

// ParseProvider parses a provider name (case-insensitive).
func ParseProvider(s string) (Provider, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case string(ProviderAWS):
		return ProviderAWS, nil
	case string(ProviderGCP):
		return ProviderGCP, nil
	case string(ProviderSnowflake):
		return ProviderSnowflake, nil
	default:
		return "", fmt.Errorf("unknown provider %q (supported: aws, gcp, snowflake)", s)
	}
}
