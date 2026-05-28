package account

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/organizations/types"
)

// ListTags returns AWS Organizations tags for accountID.
func ListTags(ctx context.Context, cfg aws.Config, accountID string) ([]Tag, error) {
	return listTagsWithClient(ctx, newOrganizationsClient(cfg), accountID)
}

// SetAccountTag adds or updates one AWS Organizations tag on accountID.
func SetAccountTag(ctx context.Context, cfg aws.Config, accountID, tagKey, tagValue string) error {
	return setAccountTagWithClient(ctx, newOrganizationsClient(cfg), accountID, tagKey, tagValue)
}

func listTagsWithClient(ctx context.Context, client OrganizationsAPI, accountID string) ([]Tag, error) {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return nil, fmt.Errorf("account ID is required")
	}

	tags := make([]Tag, 0)
	var token *string
	for {
		pageTags, nextToken, err := client.ListTagsForAccount(ctx, accountID, token)
		if err != nil {
			return nil, err
		}
		tags = append(tags, pageTags...)
		if nextToken == nil || aws.ToString(nextToken) == "" {
			break
		}
		token = nextToken
	}

	slices.SortFunc(tags, func(a, b Tag) int {
		if cmp := strings.Compare(a.Key, b.Key); cmp != 0 {
			return cmp
		}
		return strings.Compare(a.Value, b.Value)
	})
	return tags, nil
}

func setAccountTagWithClient(ctx context.Context, client OrganizationsAPI, accountID, tagKey, tagValue string) error {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return fmt.Errorf("account ID is required")
	}
	tagKey = strings.TrimSpace(tagKey)
	if tagKey == "" {
		return fmt.Errorf("tag key is required")
	}
	tagValue = strings.TrimSpace(tagValue)
	if tagValue == "" {
		return fmt.Errorf("tag value is required")
	}

	return client.SetAccountTag(ctx, accountID, tagKey, tagValue)
}

// AccountName returns the AWS Organizations account name for accountID.
func AccountName(ctx context.Context, cfg aws.Config, accountID string) (string, error) {
	return accountNameWithClient(ctx, newOrganizationsClient(cfg), accountID)
}

func accountNameWithClient(ctx context.Context, client OrganizationsAPI, accountID string) (string, error) {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return "", fmt.Errorf("account ID is required")
	}
	out, err := client.DescribeAccount(ctx, &organizations.DescribeAccountInput{
		AccountId: aws.String(accountID),
	})
	if err != nil {
		return "", err
	}
	return accountNameFromOrganizationAccount(out.Account, accountID)
}

// ListAccountNames returns a map of account ID to AWS Organizations account name.
func ListAccountNames(ctx context.Context, cfg aws.Config) (map[string]string, error) {
	return listAccountNamesWithClient(ctx, newOrganizationsClient(cfg))
}

func listAccountNamesWithClient(ctx context.Context, client OrganizationsAPI) (map[string]string, error) {
	names := make(map[string]string)
	var token *string
	for {
		out, err := client.ListAccounts(ctx, &organizations.ListAccountsInput{NextToken: token})
		if err != nil {
			return nil, err
		}
		for _, acct := range out.Accounts {
			if name, err := accountNameFromOrganizationAccount(&acct, aws.ToString(acct.Id)); err == nil {
				names[aws.ToString(acct.Id)] = name
			}
		}
		if out.NextToken == nil || aws.ToString(out.NextToken) == "" {
			break
		}
		token = out.NextToken
	}
	return names, nil
}

// ResolveAccountNames returns display names for the given account IDs.
// For small sets it uses DescribeAccount per ID; for larger sets it lists the organization once.
func ResolveAccountNames(ctx context.Context, cfg aws.Config, accountIDs []string) (map[string]string, error) {
	return resolveAccountNamesWithClient(ctx, newOrganizationsClient(cfg), accountIDs)
}

func resolveAccountNamesWithClient(ctx context.Context, client OrganizationsAPI, accountIDs []string) (map[string]string, error) {
	ids := uniqueAccountIDs(accountIDs)
	if len(ids) == 0 {
		return map[string]string{}, nil
	}
	if len(ids) > accountNameListThreshold {
		all, err := listAccountNamesWithClient(ctx, client)
		if err != nil {
			return nil, err
		}
		out := make(map[string]string, len(ids))
		for _, id := range ids {
			if name, ok := all[id]; ok {
				out[id] = name
			}
		}
		return out, nil
	}

	out := make(map[string]string, len(ids))
	for _, id := range ids {
		name, err := accountNameWithClient(ctx, client, id)
		if err != nil {
			continue
		}
		out[id] = name
	}
	return out, nil
}

// ListOrganizationAccounts returns all organization accounts.
func ListOrganizationAccounts(ctx context.Context, cfg aws.Config) ([]OrganizationAccount, error) {
	client := newOrganizationsClient(cfg)
	var token *string
	out := make([]OrganizationAccount, 0)
	for {
		resp, err := client.ListAccounts(ctx, &organizations.ListAccountsInput{NextToken: token})
		if err != nil {
			return nil, err
		}
		for _, acct := range resp.Accounts {
			name, err := accountNameFromOrganizationAccount(&acct, aws.ToString(acct.Id))
			if err != nil {
				continue
			}
			out = append(out, OrganizationAccount{
				ID:   strings.TrimSpace(aws.ToString(acct.Id)),
				Name: name,
			})
		}
		if resp.NextToken == nil || aws.ToString(resp.NextToken) == "" {
			break
		}
		token = resp.NextToken
	}
	return out, nil
}

// DetectAccountKind classifies callerAccountID against organization management account.
func DetectAccountKind(ctx context.Context, cfg aws.Config, callerAccountID string) (AccountKind, error) {
	callerAccountID = strings.TrimSpace(callerAccountID)
	if callerAccountID == "" {
		return AccountKindUnknown, fmt.Errorf("caller account ID is required")
	}
	return detectAccountKindWithClient(ctx, newOrganizationsClient(cfg), callerAccountID)
}

func detectAccountKindWithClient(ctx context.Context, client OrganizationsAPI, callerAccountID string) (AccountKind, error) {
	out, err := client.DescribeOrganization(ctx, &organizations.DescribeOrganizationInput{})
	if err != nil {
		return AccountKindUnknown, fmt.Errorf("describe organization: %w", err)
	}

	managementAccountID := organizationManagementAccountID(out.Organization)
	if managementAccountID == "" {
		return AccountKindUnknown, fmt.Errorf("describe organization: missing management account ID")
	}
	if managementAccountID == callerAccountID {
		return AccountKindPayer, nil
	}
	return AccountKindLinked, nil
}

func accountNameFromOrganizationAccount(acct *types.Account, accountID string) (string, error) {
	if acct == nil {
		return "", fmt.Errorf("account %s not found", accountID)
	}
	name := strings.TrimSpace(aws.ToString(acct.Name))
	if name == "" {
		return "", fmt.Errorf("account %s has no name", accountID)
	}
	return name, nil
}

func uniqueAccountIDs(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func organizationManagementAccountID(org *types.Organization) string {
	if org == nil {
		return ""
	}
	v := reflect.ValueOf(*org)
	for _, field := range []string{"ManagementAccountId", "MasterAccountId"} {
		accountID := stringFieldValue(v, field)
		if accountID != "" {
			return accountID
		}
	}
	return strings.TrimSpace(aws.ToString(org.Id))
}

func stringFieldValue(v reflect.Value, fieldName string) string {
	field := v.FieldByName(fieldName)
	if !field.IsValid() || field.Kind() != reflect.Ptr || field.IsNil() {
		return ""
	}
	str, ok := field.Interface().(*string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(aws.ToString(str))
}
