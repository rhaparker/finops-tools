package aws

import (
	"context"
	"testing"

)

func TestResolveAccountNamesDescribePerID(t *testing.T) {
	client := fakeOrganizations{accounts: map[string]string{
		"111111111111": "Quay Production",
		"222222222222": "Staging",
	}}
	names, err := resolveAccountNamesWithClient(context.Background(), client, []string{"111111111111"})
	if err != nil {
		t.Fatal(err)
	}
	if names["111111111111"] != "Quay Production" {
		t.Fatalf("names = %+v", names)
	}
}

func TestResolveAccountNamesUniqueIDs(t *testing.T) {
	ids := uniqueAccountIDs([]string{" 111 ", "111", ""})
	if len(ids) != 1 || ids[0] != "111" {
		t.Fatalf("ids = %v", ids)
	}
}

func resolveAccountNamesWithClient(ctx context.Context, client OrganizationsAccountsAPI, accountIDs []string) (map[string]string, error) {
	ids := uniqueAccountIDs(accountIDs)
	if len(ids) == 0 {
		return map[string]string{}, nil
	}
	out := make(map[string]string, len(ids))
	for _, id := range ids {
		name, err := accountName(ctx, client, id)
		if err != nil {
			continue
		}
		out[id] = name
	}
	return out, nil
}
