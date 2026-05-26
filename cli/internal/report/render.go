package report

import (
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/nikolalohinski/gonja/v2"
	"github.com/nikolalohinski/gonja/v2/config"
	"github.com/nikolalohinski/gonja/v2/exec"
	"github.com/nikolalohinski/gonja/v2/loaders"
	"github.com/openshift-online/finops-tools/cli/internal/format"
	corereport "github.com/openshift-online/finops-tools/core/report"
	"github.com/openshift-online/finops-tools/core/cost"
)

const costsTemplate = "costs.html.j2"

var (
	initOnce sync.Once
	tpl      *exec.Template
	initErr  error
)

func costsTemplateCompiled() (*exec.Template, error) {
	initOnce.Do(func() {
		embedLoader, err := loaders.NewEmbedFSLoader("templates", &templateFS)
		if err != nil {
			initErr = err
			return
		}
		loader := newStableLoader(embedLoader)
		cfg := config.New()
		environment := &exec.Environment{
			Context:           exec.EmptyContext(),
			Filters:           gonja.DefaultEnvironment.Filters,
			Tests:             gonja.DefaultEnvironment.Tests,
			ControlStructures: gonja.DefaultEnvironment.ControlStructures,
			Methods:           gonja.DefaultEnvironment.Methods,
		}
		tpl, initErr = exec.NewTemplate(costsTemplate, cfg, loader, environment)
	})
	return tpl, initErr
}

// CostsReportView is the template context for costs.html.j2.
type CostsReportView struct {
	GeneratedAt     string
	AccountSummary  string
	StartDate       string
	EndDate         string
	Currency        string
	Metric          string
	Total           float64
	TotalFormatted  string
	ByAccount       []BreakdownRowView
	ByService       []BreakdownRowView
	Daily           []cost.DailyCostItem
}

// BreakdownRowView is one breakdown table row for templates.
type BreakdownRowView struct {
	Label             string
	Amount            float64
	AmountFormatted   string
	Percent           float64
	PercentFormatted  string
}

// NewCostsReportView maps a core CostsReport for HTML rendering.
func NewCostsReportView(r corereport.CostsReport) CostsReportView {
	return CostsReportView{
		GeneratedAt:    r.GeneratedAt.UTC().Format("2006-01-02 15:04:05 UTC"),
		AccountSummary: formatAccountSummary(r.Accounts),
		StartDate:      r.StartDate,
		EndDate:        r.EndDate,
		Currency:       r.Currency,
		Metric:         r.Metric,
		Total:          r.Total,
		TotalFormatted: format.FormatMoney(r.Total, r.Currency),
		ByAccount:      breakdownRows(r.ByAccount, cost.SplitByAccount, r.Total, r.Currency),
		ByService:      breakdownRows(r.ByService, cost.SplitByService, r.Total, r.Currency),
		Daily:          r.Daily,
	}
}

func breakdownRows(items []cost.CostBreakdownItem, split cost.SplitBy, total float64, currency string) []BreakdownRowView {
	rows := make([]BreakdownRowView, 0, len(items))
	for _, item := range items {
		pct := corereport.PercentOfTotal(item.Amount, total)
		rows = append(rows, BreakdownRowView{
			Label:            item.DisplayLabel(split),
			Amount:           item.Amount,
			AmountFormatted:  format.FormatMoney(item.Amount, currency),
			Percent:          pct,
			PercentFormatted: fmt.Sprintf("%.1f%%", pct),
		})
	}
	return rows
}

func formatAccountSummary(accounts []cost.AccountTarget) string {
	if len(accounts) == 0 {
		return ""
	}
	names := make([]string, 0, len(accounts))
	for _, a := range accounts {
		if name := strings.TrimSpace(a.DisplayName); name != "" {
			names = append(names, name)
			continue
		}
		if alias := strings.TrimSpace(a.DisplayAlias); alias != "" {
			names = append(names, alias)
			continue
		}
		names = append(names, a.AccountID)
	}
	return strings.Join(names, ", ")
}

// RenderCostsHTML renders the costs report as HTML to w.
func RenderCostsHTML(w io.Writer, r corereport.CostsReport) error {
	t, err := costsTemplateCompiled()
	if err != nil {
		return fmt.Errorf("compile costs template: %w", err)
	}
	view := NewCostsReportView(r)
	ctx := exec.NewContext(map[string]any{
		"generated_at":     view.GeneratedAt,
		"account_summary":  view.AccountSummary,
		"start_date":       view.StartDate,
		"end_date":         view.EndDate,
		"currency":         view.Currency,
		"metric":           view.Metric,
		"total":            view.Total,
		"total_formatted":  view.TotalFormatted,
		"by_account":       view.ByAccount,
		"by_service":       view.ByService,
		"daily_chart_svg":  dailyChartSVG(view.Daily, view.Currency),
	})
	if err := t.Execute(w, ctx); err != nil {
		return fmt.Errorf("render costs template: %w", err)
	}
	return nil
}
