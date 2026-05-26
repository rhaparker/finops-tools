package report

import (
	"context"
	"testing"

	"github.com/openshift-online/finops-tools/core/cost"
)

func TestPercentOfTotal(t *testing.T) {
	if got := PercentOfTotal(25, 100); got != 25 {
		t.Errorf("got %v, want 25", got)
	}
	if got := PercentOfTotal(1, 0); got != 0 {
		t.Errorf("got %v, want 0", got)
	}
}

func TestBuildCostsReportRequiresAccounts(t *testing.T) {
	_, err := BuildCostsReport(context.Background(), cost.CostQuery{Provider: cost.ProviderAWS}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}
