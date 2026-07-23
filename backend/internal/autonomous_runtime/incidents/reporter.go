package incidents

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type IncidentReport struct {
	IncidentID                uuid.UUID       `json:"incident_id"`
	Timestamp                 time.Time       `json:"timestamp"`
	RootCause                 string          `json:"root_cause"`
	Timeline                  []TimelineEvent `json:"timeline"`
	ImpactedTenants           []string        `json:"impacted_tenants"`
	ImpactedPages             []string        `json:"impacted_pages"`
	ImpactedAPIs              []string        `json:"impacted_apis"`
	SLOViolations             []string        `json:"slo_violations"`
	PreAggFailures            []string        `json:"preagg_failures"`
	QueryPlanRegressions      []string        `json:"query_plan_regressions"`
	SuggestedRemediation      string          `json:"suggested_remediation"`
	PreventiveRecommendations []string        `json:"preventive_recommendations"`
}

type TimelineEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Event     string    `json:"event"`
	Severity  string    `json:"severity"`
}

type IncidentReporter struct{}

func NewIncidentReporter() *IncidentReporter {
	return &IncidentReporter{}
}

func (ir *IncidentReporter) Generate(ctx context.Context, incidentID uuid.UUID) (*IncidentReport, error) {
	// Mock: Generate incident report
	// Real: Analyze logs, metrics, traces, lineage

	report := &IncidentReport{
		IncidentID: incidentID,
		Timestamp:  time.Now(),
		RootCause:  "Pre-aggregation positions_daily failed to refresh due to upstream BO schema change",
		Timeline: []TimelineEvent{
			{Timestamp: time.Now().Add(-30 * time.Minute), Event: "BO Position schema change deployed", Severity: "info"},
			{Timestamp: time.Now().Add(-25 * time.Minute), Event: "Pre-agg positions_daily refresh started", Severity: "info"},
			{Timestamp: time.Now().Add(-22 * time.Minute), Event: "Pre-agg refresh failed: column 'market_value_usd' not found", Severity: "error"},
			{Timestamp: time.Now().Add(-20 * time.Minute), Event: "Positions API p95 latency spike: 95ms → 450ms", Severity: "critical"},
			{Timestamp: time.Now().Add(-18 * time.Minute), Event: "14 tenants impacted", Severity: "critical"},
		},
		ImpactedTenants:      []string{"tenant-123", "tenant-456", "tenant-789"},
		ImpactedPages:        []string{"Positions Dashboard", "Account Overview", "Portfolio Summary"},
		ImpactedAPIs:         []string{"positions_api", "portfolio_api"},
		SLOViolations:        []string{"Positions API p95 latency > 100ms (450ms observed)"},
		PreAggFailures:       []string{"positions_daily"},
		QueryPlanRegressions: []string{"Fallback to full table scan"},
		SuggestedRemediation: "1. Add schema guard to pre-agg refresh logic. 2. Implement fallback pre-agg. 3. Add column existence check before refresh.",
		PreventiveRecommendations: []string{
			"Add schema change detection to CRS",
			"Implement pre-agg schema validation",
			"Add automated rollback for failed pre-agg refreshes",
		},
	}

	return report, nil
}

func (ir *IncidentReporter) ExportMarkdown(ctx context.Context, report *IncidentReport) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# Incident Report: %s\n\n", report.IncidentID.String()))
	sb.WriteString(fmt.Sprintf("**Time**: %s\n\n", report.Timestamp.Format("2006-01-02 15:04:05")))

	sb.WriteString(fmt.Sprintf("## Root Cause\n%s\n\n", report.RootCause))

	sb.WriteString("## Timeline\n")
	for _, event := range report.Timeline {
		sb.WriteString(fmt.Sprintf("- **%s** [%s]: %s\n", event.Timestamp.Format("15:04:05"), event.Severity, event.Event))
	}
	sb.WriteString("\n")

	sb.WriteString(fmt.Sprintf("## Impact\n"))
	sb.WriteString(fmt.Sprintf("- **Tenants**: %s\n", strings.Join(report.ImpactedTenants, ", ")))
	sb.WriteString(fmt.Sprintf("- **Pages**: %s\n", strings.Join(report.ImpactedPages, ", ")))
	sb.WriteString(fmt.Sprintf("- **APIs**: %s\n\n", strings.Join(report.ImpactedAPIs, ", ")))

	sb.WriteString(fmt.Sprintf("## Suggested Remediation\n%s\n\n", report.SuggestedRemediation))

	sb.WriteString("## Preventive Recommendations\n")
	for _, rec := range report.PreventiveRecommendations {
		sb.WriteString(fmt.Sprintf("- %s\n", rec))
	}

	return sb.String()
}
