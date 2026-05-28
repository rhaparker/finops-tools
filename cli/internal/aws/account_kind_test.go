package aws

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/organizations/types"
)

type fakeDescribeOrganizationClient struct {
	output *organizations.DescribeOrganizationOutput
	err    error
}

func (f fakeDescribeOrganizationClient) DescribeOrganization(
	_ context.Context,
	_ *organizations.DescribeOrganizationInput,
	_ ...func(*organizations.Options),
) (*organizations.DescribeOrganizationOutput, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.output, nil
}

func TestDetectAccountKindWithClientPayer(t *testing.T) {
	client := fakeDescribeOrganizationClient{
		output: &organizations.DescribeOrganizationOutput{
			Organization: organizationWithManagementAccountID("123456789012"),
		},
	}
	kind, err := detectAccountKindWithClient(context.Background(), client, "123456789012")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if kind != AccountKindPayer {
		t.Fatalf("kind = %q, want %q", kind, AccountKindPayer)
	}
}

func TestDetectAccountKindWithClientLinked(t *testing.T) {
	client := fakeDescribeOrganizationClient{
		output: &organizations.DescribeOrganizationOutput{
			Organization: organizationWithManagementAccountID("999999999999"),
		},
	}
	kind, err := detectAccountKindWithClient(context.Background(), client, "123456789012")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if kind != AccountKindLinked {
		t.Fatalf("kind = %q, want %q", kind, AccountKindLinked)
	}
}

func TestDetectAccountKindWithClientUnknownOnDescribeFailure(t *testing.T) {
	client := fakeDescribeOrganizationClient{err: errors.New("access denied")}
	kind, err := detectAccountKindWithClient(context.Background(), client, "123456789012")
	if err == nil {
		t.Fatal("expected error")
	}
	if kind != AccountKindUnknown {
		t.Fatalf("kind = %q, want %q", kind, AccountKindUnknown)
	}
}

func TestOrganizationManagementAccountIDFallsBackToOrganizationID(t *testing.T) {
	org := &types.Organization{Id: aws.String("123456789012")}
	if got := organizationManagementAccountID(org); got != "123456789012" {
		t.Fatalf("got %q want %q", got, "123456789012")
	}
}

func organizationWithManagementAccountID(accountID string) *types.Organization {
	org := &types.Organization{}
	v := reflect.ValueOf(org).Elem()
	for _, fieldName := range []string{"ManagementAccountId", "MasterAccountId"} {
		field := v.FieldByName(fieldName)
		if !field.IsValid() || !field.CanSet() || field.Kind() != reflect.Ptr || field.Type().Elem().Kind() != reflect.String {
			continue
		}
		id := accountID
		field.Set(reflect.ValueOf(&id))
		return org
	}
	org.Id = aws.String(accountID)
	return org
}
