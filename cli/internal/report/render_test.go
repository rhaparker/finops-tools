package report

import (
	"bytes"
	"strings"
	"testing"
	"time"

	corereport "github.com/openshift-online/finops-tools/core/report"
	"github.com/openshift-online/finops-tools/core/cost"
)

func TestRenderCostsHTML(t *testing.T) {
	var buf bytes.Buffer
	err := RenderCostsHTML(&buf, corereport.CostsReport{
		GeneratedAt: time.Date(2026, 5, 26, 10, 0, 0, 0, time.UTC),
		StartDate:   "2026-04-25",
		EndDate:     "2026-05-24",
		Currency:    "USD",
		Metric:      "NetAmortizedCost",
		Total:       1000,
		ByAccount: []cost.CostBreakdownItem{
			{Account: "111111111111", AccountName: "Member", Amount: 600},
		},
		ByService: []cost.CostBreakdownItem{
			{Service: "Amazon EC2", Amount: 700},
		},
		Daily: []cost.DailyCostItem{
			{Date: "2026-05-23", Amount: 30},
			{Date: "2026-05-24", Amount: 40},
		},
		Accounts: []cost.AccountTarget{{
			AccountID:   "123456789012",
			DisplayName: "rh-control",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{
		"Costs Report",
		"rh-control",
		"USD 1,000.00",
		"Member",
		"Amazon EC2",
		`<svg class="daily-chart"`,
		"2026-05-23",
		"2026-05-24",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q", want)
		}
	}
}

func TestFormatAccountSummary(t *testing.T) {
	s := formatAccountSummary([]cost.AccountTarget{{
		DisplayAlias: "quay",
		AccountID:    "111",
	}})
	if s != "quay" {
		t.Errorf("got %q", s)
	}
}
