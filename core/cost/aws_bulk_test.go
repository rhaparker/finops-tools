package cost

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
)

func TestFetchBulkManyLinkedAccounts(t *testing.T) {
	now := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	ce := &fakeCECaptureFilter{
		fakeCE: fakeCE{
			pages: [][]types.ResultByTime{{
				{
					Groups: []types.Group{
						{
							Keys: []string{"111111111111"},
							Metrics: map[string]types.MetricValue{
								MetricNetAmortized: {Amount: aws.String("10"), Unit: aws.String("USD")},
							},
						},
						{
							Keys: []string{"222222222222"},
							Metrics: map[string]types.MetricValue{
								MetricNetAmortized: {Amount: aws.String("20"), Unit: aws.String("USD")},
							},
						},
						{
							Keys: []string{"999999999999"},
							Metrics: map[string]types.MetricValue{
								MetricNetAmortized: {Amount: aws.String("1000"), Unit: aws.String("USD")},
							},
						},
					},
				},
			}},
		},
	}

	targets := []AccountTarget{
		{AccountID: "111111111111", PayerAccountID: "123456789012", ScopeAccountOnly: true, AWSConfig: aws.Config{}, DisplayName: "A"},
		{AccountID: "222222222222", PayerAccountID: "123456789012", ScopeAccountOnly: true, AWSConfig: aws.Config{}, DisplayName: "B"},
	}

	res, err := fetchAWSNetAmortizedBulk(context.Background(), CostQuery{
		Provider: ProviderAWS,
		Range:    LastNDaysRange(30, now),
	}, targets, fetchAWSOptions{
		Now:             now,
		NewCostExplorer: func(aws.Config) CostExplorerAPI { return ce },
	})
	if err != nil {
		t.Fatal(err)
	}
	if ce.calls != 1 {
		t.Fatalf("expected 1 Cost Explorer call, got %d", ce.calls)
	}
	if ce.lastFilter == nil || ce.lastFilter.Dimensions == nil {
		t.Fatal("expected linked account filter on Cost Explorer call")
	}
	if got := ce.lastFilter.Dimensions.Values; len(got) != 2 || got[0] != "111111111111" || got[1] != "222222222222" {
		t.Fatalf("filter values = %v, want [111111111111 222222222222]", got)
	}
	if res.Amount != 30 {
		t.Fatalf("Amount = %v, want 30", res.Amount)
	}
}

func TestPlanBulkFetchRequiresSharedPayer(t *testing.T) {
	_, ok := planBulkFetch([]AccountTarget{
		{AccountID: "111111111111", PayerAccountID: "123456789012", ScopeAccountOnly: true},
		{AccountID: "222222222222", PayerAccountID: "987654321098", ScopeAccountOnly: true},
	})
	if ok {
		t.Fatal("expected mixed payers to disable bulk fetch")
	}
}

func TestBatchStrings(t *testing.T) {
	batches := batchStrings([]string{"a", "b", "c", "d", "e"}, 2)
	if len(batches) != 3 || len(batches[0]) != 2 || len(batches[2]) != 1 {
		t.Fatalf("batches = %+v", batches)
	}
}
