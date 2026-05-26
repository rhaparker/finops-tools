package report

import (
	"context"
	"testing"

	"github.com/openshift-online/finops-tools/core/cost"
)

type recordingProgress struct {
	steps []string
}

func (r *recordingProgress) Step(message string) {
	r.steps = append(r.steps, message)
}

func TestBuildCostsReportProgressRequiresAccountsOnly(t *testing.T) {
	rec := &recordingProgress{}
	_, err := BuildCostsReport(context.Background(), cost.CostQuery{Provider: cost.ProviderAWS}, rec)
	if err == nil {
		t.Fatal("expected error")
	}
	if len(rec.steps) != 0 {
		t.Fatalf("steps = %v, want none before fetch", rec.steps)
	}
}
