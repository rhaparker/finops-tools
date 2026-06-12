package report

import (
	"fmt"
	"html/template"
	"io"
	"sync"

	"github.com/openshift-online/finops-tools/cli/internal/format"
	coreca "github.com/openshift-online/finops-tools/core/report/costanomalies"
)

var (
	caTplOnce sync.Once
	caTpl     *template.Template
	caTplErr  error
)

func costAnomaliesTemplateCompiled() (*template.Template, error) {
	caTplOnce.Do(func() {
		caTpl, caTplErr = template.ParseFS(templateFS,
			"templates/layout.html",
			"templates/cost-anomalies.html",
		)
	})
	return caTpl, caTplErr
}

// RootCauseView is one root-cause row for the template.
type RootCauseView struct {
	DisplayAccount        string
	Region                string
	Service               string
	UsageType             string
	Contribution          float64
	ContributionFormatted string
}

// AnomalyView is one anomaly row for the template.
type AnomalyView struct {
	Service                string
	PeriodDisplay          string
	CurrentScore           float64
	CurrentScoreFormatted  string
	ScoreBarWidth          int
	TotalImpact            float64
	TotalImpactFormatted   string
	ActualSpend            float64
	ActualSpendFormatted   string
	ExpectedSpend          float64
	ExpectedSpendFormatted string
	ImpactPct              float64
	ImpactPctFormatted     string
	Severity               string
	SeverityClass          string
	RootCauseSummary       string
	RootCauses             []RootCauseView
}

// CostAnomaliesReportView is the template context for cost-anomalies.html.
type CostAnomaliesReportView struct {
	GeneratedAt          string
	AccountSummary       string
	StartDate            string
	EndDate              string
	AnomalyCount         int
	AnomalyCountLabel    string
	TotalImpactFormatted string
	Anomalies            []AnomalyView
}

// NewCostAnomaliesReportView maps a core costanomalies.Report to the template context.
func NewCostAnomaliesReportView(r coreca.Report, accountSummary string) CostAnomaliesReportView {
	currency := "USD"

	totalImpact := 0.0
	for _, a := range r.Anomalies {
		totalImpact += a.TotalImpact
	}

	views := make([]AnomalyView, 0, len(r.Anomalies))
	for _, a := range r.Anomalies {
		views = append(views, anomalyToView(a, currency))
	}

	countLabel := "ies"
	if len(r.Anomalies) == 1 {
		countLabel = "y"
	}

	return CostAnomaliesReportView{
		GeneratedAt:          r.GeneratedAt.UTC().Format("2006-01-02 15:04:05 UTC"),
		AccountSummary:       accountSummary,
		StartDate:            r.StartDate,
		EndDate:              r.EndDate,
		AnomalyCount:         len(r.Anomalies),
		AnomalyCountLabel:    countLabel,
		TotalImpactFormatted: format.FormatMoney(totalImpact, currency),
		Anomalies:            views,
	}
}

func anomalyToView(a coreca.Anomaly, currency string) AnomalyView {
	severity, severityClass := anomalySeverity(a.TotalImpact)

	rcViews := make([]RootCauseView, 0, len(a.RootCauses))
	for _, rc := range a.RootCauses {
		displayAccount := rc.AccountName
		if displayAccount == "" {
			displayAccount = rc.Account
		}
		rcViews = append(rcViews, RootCauseView{
			DisplayAccount:        displayAccount,
			Region:                rc.Region,
			Service:               rc.Service,
			UsageType:             rc.UsageType,
			Contribution:          rc.Contribution,
			ContributionFormatted: format.FormatMoney(rc.Contribution, currency),
		})
	}

	barWidth := int(a.CurrentScore * 80)
	if barWidth < 2 {
		barWidth = 2
	}

	rootCauseSummary := ""
	if n := len(a.RootCauses); n > 0 {
		word := "causes"
		if n == 1 {
			word = "cause"
		}
		rootCauseSummary = fmt.Sprintf("%d root %s", n, word)
	}

	periodDisplay := a.StartDate
	if a.EndDate != "" {
		periodDisplay = a.StartDate + " — " + a.EndDate
	} else {
		periodDisplay += " (open)"
	}

	return AnomalyView{
		Service:                a.Service,
		PeriodDisplay:          periodDisplay,
		CurrentScore:           a.CurrentScore,
		CurrentScoreFormatted:  fmt.Sprintf("%.2f", a.CurrentScore),
		ScoreBarWidth:          barWidth,
		TotalImpact:            a.TotalImpact,
		TotalImpactFormatted:   format.FormatMoney(a.TotalImpact, currency),
		ActualSpend:            a.ActualSpend,
		ActualSpendFormatted:   format.FormatMoney(a.ActualSpend, currency),
		ExpectedSpend:          a.ExpectedSpend,
		ExpectedSpendFormatted: format.FormatMoney(a.ExpectedSpend, currency),
		ImpactPct:              a.ImpactPct,
		ImpactPctFormatted:     fmt.Sprintf("%.1f%%", a.ImpactPct),
		Severity:               severity,
		SeverityClass:          severityClass,
		RootCauseSummary:       rootCauseSummary,
		RootCauses:             rcViews,
	}
}

// anomalySeverity returns a human label and CSS class based on dollar impact.
func anomalySeverity(impact float64) (string, string) {
	switch {
	case impact >= 10000:
		return "Critical", "badge-critical"
	case impact >= 1000:
		return "High", "badge-high"
	case impact >= 100:
		return "Medium", "badge-medium"
	default:
		return "Low", "badge-low"
	}
}

// RenderCostAnomaliesHTML renders the cost anomalies report as HTML to w.
func RenderCostAnomaliesHTML(w io.Writer, r coreca.Report, accountSummary string) error {
	t, err := costAnomaliesTemplateCompiled()
	if err != nil {
		return fmt.Errorf("compile cost-anomalies template: %w", err)
	}
	view := NewCostAnomaliesReportView(r, accountSummary)
	if err := t.ExecuteTemplate(w, TemplateCostAnomalies, view); err != nil {
		return fmt.Errorf("render cost-anomalies template: %w", err)
	}
	return nil
}
