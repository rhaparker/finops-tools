// account_kind.go classifies an AWS account session as payer, linked, or unknown.
package aws

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/organizations/types"
)

// AccountKind describes whether a validated account session is payer or linked.
type AccountKind string

const (
	AccountKindPayer   AccountKind = "payer"
	AccountKindLinked  AccountKind = "linked"
	AccountKindUnknown AccountKind = "unknown"
)

// OrganizationsDescribeOrganizationAPI is the subset of Organizations used for payer detection.
type OrganizationsDescribeOrganizationAPI interface {
	DescribeOrganization(
		ctx context.Context,
		params *organizations.DescribeOrganizationInput,
		optFns ...func(*organizations.Options),
	) (*organizations.DescribeOrganizationOutput, error)
}

// DetectAccountKind loads AWS config for profile and classifies callerAccountID.
// Returns AccountKindUnknown when classification cannot be determined.
func DetectAccountKind(ctx context.Context, profile, callerAccountID string) (AccountKind, error) {
	profile = strings.TrimSpace(profile)
	if profile == "" {
		return AccountKindUnknown, fmt.Errorf("profile is required")
	}
	callerAccountID = strings.TrimSpace(callerAccountID)
	if callerAccountID == "" {
		return AccountKindUnknown, fmt.Errorf("caller account ID is required")
	}

	cfg, err := LoadSharedConfigProfile(ctx, profile)
	if err != nil {
		return AccountKindUnknown, fmt.Errorf("load shared config profile %q: %w", profile, err)
	}
	return detectAccountKindWithClient(ctx, newOrganizationsClient(cfg), callerAccountID)
}

func detectAccountKindWithClient(
	ctx context.Context,
	client OrganizationsDescribeOrganizationAPI,
	callerAccountID string,
) (AccountKind, error) {
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
