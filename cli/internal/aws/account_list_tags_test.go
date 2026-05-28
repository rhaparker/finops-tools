package aws

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/organizations/types"
)

type fakeOrganizationsTags struct {
	pages []*organizations.ListTagsForResourceOutput
	err   error
	call  int
}

func (f *fakeOrganizationsTags) ListTagsForResource(
	_ context.Context,
	_ *organizations.ListTagsForResourceInput,
	_ ...func(*organizations.Options),
) (*organizations.ListTagsForResourceOutput, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.call >= len(f.pages) {
		return &organizations.ListTagsForResourceOutput{}, nil
	}
	out := f.pages[f.call]
	f.call++
	return out, nil
}

func TestAccountTagsWithClient(t *testing.T) {
	client := &fakeOrganizationsTags{
		pages: []*organizations.ListTagsForResourceOutput{
			{
				Tags: []types.Tag{
					{Key: aws.String("owner"), Value: aws.String("team-b")},
					{Key: aws.String("env"), Value: aws.String("prod")},
				},
				NextToken: aws.String("page-2"),
			},
			{
				Tags: []types.Tag{
					{Key: aws.String("owner"), Value: aws.String("team-a")},
				},
			},
		},
	}

	tags, err := accountTagsWithClient(context.Background(), client, "123456789012")
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 3 {
		t.Fatalf("expected 3 tags, got %d", len(tags))
	}
	if tags[0].Key != "env" || tags[0].Value != "prod" {
		t.Fatalf("unexpected first tag: %+v", tags[0])
	}
	if tags[1].Key != "owner" || tags[1].Value != "team-a" {
		t.Fatalf("unexpected second tag: %+v", tags[1])
	}
	if tags[2].Key != "owner" || tags[2].Value != "team-b" {
		t.Fatalf("unexpected third tag: %+v", tags[2])
	}
}

func TestAccountTagsWithClientValidationAndErrors(t *testing.T) {
	client := &fakeOrganizationsTags{}
	if _, err := accountTagsWithClient(context.Background(), client, " "); err == nil {
		t.Fatal("expected account ID validation error")
	}

	wantErr := errors.New("boom")
	client.err = wantErr
	_, err := accountTagsWithClient(context.Background(), client, "123456789012")
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped error %v, got %v", wantErr, err)
	}
}
