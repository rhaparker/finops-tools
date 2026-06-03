// Package report aggregates cost data for multi-section reports.
package report

import (
	"context"
	"fmt"
	"time"

	"github.com/openshift-online/finops-tools/core/cost"
)

// CostsReport is aggregated cost data for the costs HTML template.
type CostsReport struct {
	GeneratedAt time.Time
	StartDate   string
	EndDate     string
	Currency    string
	Metric      string
	Total       float64
	ByAccount   []cost.CostBreakdownItem
	ByService   []cost.CostBreakdownItem
	Daily       []cost.DailyCostItem
	Accounts    []cost.AccountTarget
}

// EmptyCostsReport returns a zero-filled report for the query period with no accounts.
func EmptyCostsReport(q cost.CostQuery, now time.Time) CostsReport {
	dr := cost.EffectiveRange(q, now)
	endInclusive := dr.End.AddDate(0, 0, -1)
	return CostsReport{
		GeneratedAt: now,
		StartDate:   dr.Start.Format("2006-01-02"),
		EndDate:     endInclusive.Format("2006-01-02"),
		Metric:      cost.MetricNetAmortized,
	}
}

// BuildCostsReport fetches total, per-account, per-service, and daily net amortized costs.
// progress may be nil to disable status updates.
func BuildCostsReport(ctx context.Context, q cost.CostQuery, progress Progress) (CostsReport, error) {
	if len(q.Accounts) == 0 {
		return CostsReport{}, fmt.Errorf("at least one account is required")
	}
	if progress == nil {
		progress = noopProgress{}
	}

	dr := cost.EffectiveRange(q, time.Now().UTC())
	period := fmt.Sprintf("%s – %s", dr.Start.Format("2006-01-02"), dr.End.AddDate(0, 0, -1).Format("2006-01-02"))
	progress.Step(fmt.Sprintf("Fetching total costs from AWS Cost Explorer (%s)…", period))
	totalQ := q
	totalQ.SplitBy = cost.SplitByNone
	totalRes, err := cost.Fetch(ctx, totalQ)
	if err != nil {
		return CostsReport{}, fmt.Errorf("total costs: %w", err)
	}

	progress.Step("Fetching costs by linked account…")
	byAccountQ := q
	byAccountQ.SplitBy = cost.SplitByAccount
	byAccountRes, err := cost.Fetch(ctx, byAccountQ)
	if err != nil {
		return CostsReport{}, fmt.Errorf("costs by account: %w", err)
	}

	progress.Step("Fetching costs by service…")
	byServiceQ := q
	byServiceQ.SplitBy = cost.SplitByService
	byServiceRes, err := cost.Fetch(ctx, byServiceQ)
	if err != nil {
		return CostsReport{}, fmt.Errorf("costs by service: %w", err)
	}

	progress.Step("Fetching daily cost trend…")
	daily, dailyCurrency, err := cost.FetchDaily(ctx, q)
	if err != nil {
		return CostsReport{}, fmt.Errorf("daily costs: %w", err)
	}
	if dailyCurrency != "" && dailyCurrency != totalRes.Currency {
		return CostsReport{}, fmt.Errorf("daily currency %s does not match total currency %s", dailyCurrency, totalRes.Currency)
	}

	return CostsReport{
		GeneratedAt: time.Now().UTC(),
		StartDate:   totalRes.StartDate,
		EndDate:     totalRes.EndDate,
		Currency:    totalRes.Currency,
		Metric:      totalRes.Metric,
		Total:       totalRes.Amount,
		ByAccount:   byAccountRes.Breakdown,
		ByService:   byServiceRes.Breakdown,
		Daily:       daily,
		Accounts:    cost.FilterOverlappingTargets(q.Accounts),
	}, nil
}

// PercentOfTotal returns the percentage of total represented by amount.
func PercentOfTotal(amount, total float64) float64 {
	if total == 0 {
		return 0
	}
	return amount / total * 100
}
