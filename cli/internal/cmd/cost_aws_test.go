package cmd

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/openshift-online/finops-tools/cli/internal/configstore"
	"github.com/openshift-online/finops-tools/core/cost"
)

func TestEnrichCostTargetDisplayNameSkipsWhenSet(t *testing.T) {
	target := &cost.AccountTarget{
		AccountID:   "111111111111",
		AWSConfig:   aws.Config{},
		DisplayName: "Already Known",
	}
	if err := enrichCostTargetDisplayName(context.Background(), target, configstore.File{}); err != nil {
		t.Fatal(err)
	}
	if target.DisplayName != "Already Known" {
		t.Fatalf("DisplayName = %q", target.DisplayName)
	}
}
