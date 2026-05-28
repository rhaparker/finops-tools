// account_list_tags.go fetches AWS Organizations tags for a specific account.
package aws

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
)

// AccountTag is one AWS Organizations account tag.
type AccountTag struct {
	Key   string
	Value string
}

// OrganizationsAccountTagsAPI is the subset of Organizations used for account tags.
type OrganizationsAccountTagsAPI interface {
	ListTagsForResource(
		ctx context.Context,
		params *organizations.ListTagsForResourceInput,
		optFns ...func(*organizations.Options),
	) (*organizations.ListTagsForResourceOutput, error)
}

// AccountTags returns AWS Organizations tags for accountID.
func AccountTags(ctx context.Context, cfg aws.Config, accountID string) ([]AccountTag, error) {
	client := organizations.NewFromConfig(cfg, func(o *organizations.Options) {
		o.Region = organizationsRegion
	})
	return accountTagsWithClient(ctx, client, accountID)
}

func accountTagsWithClient(ctx context.Context, client OrganizationsAccountTagsAPI, accountID string) ([]AccountTag, error) {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return nil, fmt.Errorf("account ID is required")
	}

	tags := make([]AccountTag, 0)
	var token *string
	for {
		out, err := client.ListTagsForResource(ctx, &organizations.ListTagsForResourceInput{
			ResourceId: aws.String(accountID),
			NextToken:  token,
		})
		if err != nil {
			return nil, err
		}
		for _, tag := range out.Tags {
			key := strings.TrimSpace(aws.ToString(tag.Key))
			if key == "" {
				continue
			}
			tags = append(tags, AccountTag{
				Key:   key,
				Value: strings.TrimSpace(aws.ToString(tag.Value)),
			})
		}
		if out.NextToken == nil || aws.ToString(out.NextToken) == "" {
			break
		}
		token = out.NextToken
	}

	slices.SortFunc(tags, func(a, b AccountTag) int {
		if cmp := strings.Compare(a.Key, b.Key); cmp != 0 {
			return cmp
		}
		return strings.Compare(a.Value, b.Value)
	})
	return tags, nil
}
