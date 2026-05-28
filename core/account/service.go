package account

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	orgtypes "github.com/aws/aws-sdk-go-v2/service/organizations/types"
)

// OrganizationsAPI is the subset of Organizations used by core account operations.
type OrganizationsAPI interface {
	DescribeAccount(
		ctx context.Context,
		params *organizations.DescribeAccountInput,
		optFns ...func(*organizations.Options),
	) (*organizations.DescribeAccountOutput, error)
	ListAccounts(
		ctx context.Context,
		params *organizations.ListAccountsInput,
		optFns ...func(*organizations.Options),
	) (*organizations.ListAccountsOutput, error)
	ListTagsForAccount(
		ctx context.Context,
		accountID string,
		nextToken *string,
	) ([]Tag, *string, error)
	SetAccountTag(
		ctx context.Context,
		accountID, tagKey, tagValue string,
	) error
	DescribeOrganization(
		ctx context.Context,
		params *organizations.DescribeOrganizationInput,
		optFns ...func(*organizations.Options),
	) (*organizations.DescribeOrganizationOutput, error)
}

type organizationsClientFactory func(aws.Config) OrganizationsAPI

func newOrganizationsClient(cfg aws.Config) OrganizationsAPI {
	client := organizations.NewFromConfig(cfg, func(o *organizations.Options) {
		o.Region = organizationsRegion
	})
	return organizationsClient{client: client}
}

type organizationsClient struct {
	client *organizations.Client
}

func (c organizationsClient) DescribeAccount(
	ctx context.Context,
	params *organizations.DescribeAccountInput,
	optFns ...func(*organizations.Options),
) (*organizations.DescribeAccountOutput, error) {
	return c.client.DescribeAccount(ctx, params, optFns...)
}

func (c organizationsClient) ListAccounts(
	ctx context.Context,
	params *organizations.ListAccountsInput,
	optFns ...func(*organizations.Options),
) (*organizations.ListAccountsOutput, error) {
	return c.client.ListAccounts(ctx, params, optFns...)
}

func (c organizationsClient) ListTagsForAccount(
	ctx context.Context,
	accountID string,
	nextToken *string,
) ([]Tag, *string, error) {
	out, err := c.client.ListTagsForResource(ctx, &organizations.ListTagsForResourceInput{
		ResourceId: aws.String(accountID),
		NextToken:  nextToken,
	})
	if err != nil {
		return nil, nil, err
	}
	tags := make([]Tag, 0, len(out.Tags))
	for _, tag := range out.Tags {
		key := strings.TrimSpace(aws.ToString(tag.Key))
		if key == "" {
			continue
		}
		tags = append(tags, Tag{
			Key:   key,
			Value: strings.TrimSpace(aws.ToString(tag.Value)),
		})
	}
	return tags, out.NextToken, nil
}

func (c organizationsClient) SetAccountTag(
	ctx context.Context,
	accountID, tagKey, tagValue string,
) error {
	_, err := c.client.TagResource(ctx, &organizations.TagResourceInput{
		ResourceId: aws.String(accountID),
		Tags: []orgtypes.Tag{
			{
				Key:   aws.String(tagKey),
				Value: aws.String(tagValue),
			},
		},
	})
	return err
}

func (c organizationsClient) DescribeOrganization(
	ctx context.Context,
	params *organizations.DescribeOrganizationInput,
	optFns ...func(*organizations.Options),
) (*organizations.DescribeOrganizationOutput, error) {
	return c.client.DescribeOrganization(ctx, params, optFns...)
}
