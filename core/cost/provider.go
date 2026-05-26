// Package cost fetches and aggregates cloud cost data from provider APIs using caller-supplied credentials.
package cost

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// Provider identifies a cloud cost data source.
type Provider string

const (
	ProviderAWS Provider = "aws"
	ProviderGCP Provider = "gcp"
)

// DefaultDays is the lookback window for cost get.
const DefaultDays = 30

// MetricNetAmortized is the AWS Cost Explorer metric name.
const MetricNetAmortized = "NetAmortizedCost"

// SplitBy identifies how cost results are grouped.
type SplitBy string

const (
	SplitByNone    SplitBy = ""
	SplitByService SplitBy = "service"
	SplitByAccount SplitBy = "account"
)

var errProviderNotImplemented = errors.New("cost provider not implemented")

// ParseSplitBy parses a --split-by flag value (case-insensitive). Empty means no split.
func ParseSplitBy(s string) (SplitBy, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "":
		return SplitByNone, nil
	case string(SplitByService):
		return SplitByService, nil
	case string(SplitByAccount):
		return SplitByAccount, nil
	default:
		return "", fmt.Errorf("unknown split-by %q (supported: service, account)", s)
	}
}

// ParseProvider parses a provider flag value (case-insensitive).
func ParseProvider(s string) (Provider, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case string(ProviderAWS), "":
		return ProviderAWS, nil
	case string(ProviderGCP):
		return ProviderGCP, nil
	default:
		return "", fmt.Errorf("unknown provider %q (supported: aws, gcp)", s)
	}
}

// ResolveAWSAccountNamesFunc maps specific account IDs to display names (Organizations).
type ResolveAWSAccountNamesFunc func(context.Context, aws.Config, []string) (map[string]string, error)

// AWSFetchOptions configures optional AWS-specific behavior for cost fetch.
type AWSFetchOptions struct {
	// ListAccountNames loads all organization accounts (slow on large orgs).
	// Prefer ResolveAccountNames when available.
	ListAccountNames ListAWSAccountNamesFunc
	// ResolveAccountNames looks up only the given account IDs (fast for small sets).
	ResolveAccountNames ResolveAWSAccountNamesFunc
}

// CostQuery describes a cost fetch request.
type CostQuery struct {
	Provider  Provider
	Accounts  []AccountTarget
	Days      int
	SplitBy   SplitBy
	AWSFetch  *AWSFetchOptions
}

// AccountTarget identifies an AWS account whose costs are fetched.
type AccountTarget struct {
	// AccountID is the 12-digit account ID whose costs are reported.
	AccountID string
	// PayerAccountID is set when AccountID is a linked (member) account.
	PayerAccountID string
	// AWSConfig holds authenticated payer credentials for Cost Explorer (set by the CLI).
	AWSConfig aws.Config
	// DisplayName is a human-readable account name (alias or Organizations name from CLI).
	DisplayName string
	// DisplayAlias is the configured finops alias when the target was selected by alias.
	DisplayAlias string
}

// CredentialsAccountID returns the account ID whose credentials are in AWSConfig.
func (t AccountTarget) CredentialsAccountID() string {
	if id := strings.TrimSpace(t.PayerAccountID); id != "" {
		return id
	}
	return strings.TrimSpace(t.AccountID)
}

// IsLinked reports whether costs are scoped to a linked (member) account.
func (t AccountTarget) IsLinked() bool {
	payer := strings.TrimSpace(t.PayerAccountID)
	return payer != "" && payer != strings.TrimSpace(t.AccountID)
}

// DailyCostItem is net amortized cost for one calendar day.
type DailyCostItem struct {
	Date   string  `json:"date"`
	Amount float64 `json:"amount"`
}

// CostBreakdownItem is one row when costs are split by service or linked account.
type CostBreakdownItem struct {
	Service     string  `json:"service,omitempty"`
	Account     string  `json:"account,omitempty"`
	AccountName string  `json:"account_name,omitempty"`
	Amount      float64 `json:"amount"`
}

// Label returns the merge/group key for this breakdown row (always the raw dimension value).
func (b CostBreakdownItem) Label(splitBy SplitBy) string {
	switch splitBy {
	case SplitByAccount:
		return b.Account
	default:
		return b.Service
	}
}

// DisplayLabel returns the formatted label for output (includes account ID when a name is known).
func (b CostBreakdownItem) DisplayLabel(splitBy SplitBy) string {
	switch splitBy {
	case SplitByAccount:
		if name := strings.TrimSpace(b.AccountName); name != "" && name != b.Account {
			return name + " (" + b.Account + ")"
		}
		return b.Label(splitBy)
	default:
		return b.Label(splitBy)
	}
}

// CostResult is the aggregated cost summary returned to callers.
type CostResult struct {
	Provider    Provider            `json:"provider"`
	AccountName string              `json:"account_name"`
	AccountID   string              `json:"account_id"`
	Metric      string              `json:"metric"`
	SplitBy     SplitBy             `json:"split_by,omitempty"`
	StartDate   string              `json:"start_date"`
	EndDate     string              `json:"end_date"`
	Amount      float64             `json:"amount"`
	Currency    string              `json:"currency"`
	Breakdown   []CostBreakdownItem `json:"breakdown,omitempty"`
	// Linked is true when costs are scoped to linked (member) accounts rather than payers.
	Linked bool `json:"linked,omitempty"`
}

// Fetch retrieves cost data for one or more accounts and returns a combined summary.
func Fetch(ctx context.Context, q CostQuery) (CostResult, error) {
	if len(q.Accounts) == 0 {
		return CostResult{}, errors.New("at least one account is required")
	}
	days := q.Days
	if days <= 0 {
		days = DefaultDays
	}

	targets := FilterOverlappingTargets(q.Accounts)
	results := make([]CostResult, 0, len(targets))
	for _, acct := range targets {
		single := q
		single.Days = days
		single.Accounts = []AccountTarget{acct}

		var r CostResult
		var err error
		switch q.Provider {
		case ProviderAWS, "":
			r, err = fetchAWSNetAmortized(ctx, single)
		case ProviderGCP:
			err = fmt.Errorf("%w: gcp", errProviderNotImplemented)
		default:
			err = fmt.Errorf("unknown provider %q", q.Provider)
		}
		if err != nil {
			return CostResult{}, fmt.Errorf("%s: %w", acct.AccountID, err)
		}
		results = append(results, r)
	}
	return MergeResults(results)
}

// FetchDaily retrieves per-day net amortized costs for one or more accounts.
func FetchDaily(ctx context.Context, q CostQuery) ([]DailyCostItem, string, error) {
	if len(q.Accounts) == 0 {
		return nil, "", errors.New("at least one account is required")
	}
	days := q.Days
	if days <= 0 {
		days = DefaultDays
	}

	switch q.Provider {
	case ProviderAWS, "":
		targets := FilterOverlappingTargets(q.Accounts)
		series := make([][]DailyCostItem, 0, len(targets))
		var currency string
		for _, acct := range targets {
			single := q
			single.Days = days
			single.Accounts = []AccountTarget{acct}
			daily, cur, err := fetchAWSDailyNetAmortized(ctx, single)
			if err != nil {
				return nil, "", fmt.Errorf("%s: %w", acct.AccountID, err)
			}
			if currency == "" {
				currency = cur
			} else if cur != currency {
				return nil, "", fmt.Errorf("cannot merge accounts with different currencies (%s vs %s)", currency, cur)
			}
			series = append(series, daily)
		}
		return MergeDaily(series), currency, nil
	case ProviderGCP:
		return nil, "", fmt.Errorf("%w: gcp", errProviderNotImplemented)
	default:
		return nil, "", fmt.Errorf("unknown provider %q", q.Provider)
	}
}
