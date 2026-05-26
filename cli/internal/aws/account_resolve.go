package aws

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// accountNameListThreshold is the max IDs to resolve via DescribeAccount before falling back to ListAccounts.
const accountNameListThreshold = 50

// ResolveAccountNames returns display names for the given account IDs.
// For small sets it uses DescribeAccount per ID; for larger sets it lists the organization once.
func ResolveAccountNames(ctx context.Context, cfg aws.Config, accountIDs []string) (map[string]string, error) {
	ids := uniqueAccountIDs(accountIDs)
	if len(ids) == 0 {
		return map[string]string{}, nil
	}
	if len(ids) > accountNameListThreshold {
		all, err := ListAccountNames(ctx, cfg)
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

	client := newOrganizationsClient(cfg)
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
