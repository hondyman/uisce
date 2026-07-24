package audit

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type RegulatoryFramework string

const (
	FrameworkSEC     RegulatoryFramework = "SEC"
	FrameworkFINRA   RegulatoryFramework = "FINRA"
	FrameworkFCA     RegulatoryFramework = "FCA"
	FrameworkGDPR    RegulatoryFramework = "GDPR"
	FrameworkMiFIDII RegulatoryFramework = "MiFID_II"
)

type AuditTrail struct {
	ChangeSetID      uuid.UUID                      `json:"changeset_id"`
	Timestamp        time.Time                      `json:"timestamp"`
	Summary          string                         `json:"summary"`
	WhatChanged      string                         `json:"what_changed"`
	WhyChanged       string                         `json:"why_changed"`
	Impact           ImpactSummary                  `json:"impact"`
	RegulatoryNotes  map[RegulatoryFramework]string `json:"regulatory_notes"`
	ReviewerApproval string                         `json:"reviewer_approval"`
}

type ImpactSummary struct {
	BOsAffected       []string `json:"bos_affected"`
	APIsAffected      []string `json:"apis_affected"`
	PagesAffected     []string `json:"pages_affected"`
	WorkflowsAffected []string `json:"workflows_affected"`
	SLOImpact         string   `json:"slo_impact"`
	DataPolicyImpact  string   `json:"data_policy_impact"`
	TenantImpact      string   `json:"tenant_impact"`
}

type AuditTrailGenerator struct{}

func NewAuditTrailGenerator() *AuditTrailGenerator {
	return &AuditTrailGenerator{}
}

func (g *AuditTrailGenerator) Generate(ctx context.Context, changesetID uuid.UUID) (*AuditTrail, error) {
	// Mock: Generate audit trail
	// Real: Analyze changeset, generate regulatory-aligned documentation

	trail := &AuditTrail{
		ChangeSetID: changesetID,
		Timestamp:   time.Now(),
		Summary:     "ChangeSet CS-4821 updated the Positions BO to include market_value_usd field",
		WhatChanged: "Added field 'market_value_usd' (type: decimal) to Positions Business Object. Updated 4 pages and 2 APIs to display this field.",
		WhyChanged:  "Business requirement to display market value in USD for multi-currency portfolios",
		Impact: ImpactSummary{
			BOsAffected:       []string{"Position"},
			APIsAffected:      []string{"positions_api", "portfolio_api"},
			PagesAffected:     []string{"Positions Dashboard", "Account Overview", "Portfolio Summary", "Risk Dashboard"},
			WorkflowsAffected: []string{},
			SLOImpact:         "No SLO violations. All pages remain within p95 render time thresholds.",
			DataPolicyImpact:  "No PII exposure. Field is non-sensitive financial data. No residency rules affected.",
			TenantImpact:      "Change applies to all tenants. No tenant-specific overrides required.",
		},
		RegulatoryNotes: map[RegulatoryFramework]string{
			FrameworkSEC:     "Field addition complies with SEC reporting requirements for portfolio valuation.",
			FrameworkGDPR:    "No personal data involved. GDPR compliance maintained.",
			FrameworkMiFIDII: "Market value reporting aligns with MiFID II transparency requirements.",
		},
		ReviewerApproval: "Approved by Sarah Chen (Senior Engineer, Positions Domain) with no exceptions.",
	}

	return trail, nil
}

func (g *AuditTrailGenerator) ExportPDF(ctx context.Context, trail *AuditTrail) (string, error) {
	// Mock: Export to PDF
	// Real: Generate PDF using template engine
	return fmt.Sprintf("audit_trail_%s.pdf", trail.ChangeSetID.String()), nil
}

func (g *AuditTrailGenerator) ExportMarkdown(ctx context.Context, trail *AuditTrail) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# Audit Trail: %s\n\n", trail.ChangeSetID.String()))
	sb.WriteString(fmt.Sprintf("**Date**: %s\n\n", trail.Timestamp.Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("## Summary\n%s\n\n", trail.Summary))
	sb.WriteString(fmt.Sprintf("## What Changed\n%s\n\n", trail.WhatChanged))
	sb.WriteString(fmt.Sprintf("## Why Changed\n%s\n\n", trail.WhyChanged))

	sb.WriteString("## Impact\n")
	sb.WriteString(fmt.Sprintf("- **BOs Affected**: %s\n", strings.Join(trail.Impact.BOsAffected, ", ")))
	sb.WriteString(fmt.Sprintf("- **APIs Affected**: %s\n", strings.Join(trail.Impact.APIsAffected, ", ")))
	sb.WriteString(fmt.Sprintf("- **Pages Affected**: %s\n", strings.Join(trail.Impact.PagesAffected, ", ")))
	sb.WriteString(fmt.Sprintf("- **SLO Impact**: %s\n", trail.Impact.SLOImpact))
	sb.WriteString(fmt.Sprintf("- **Data Policy Impact**: %s\n\n", trail.Impact.DataPolicyImpact))

	sb.WriteString("## Regulatory Compliance\n")
	for framework, note := range trail.RegulatoryNotes {
		sb.WriteString(fmt.Sprintf("- **%s**: %s\n", framework, note))
	}

	sb.WriteString(fmt.Sprintf("\n## Approval\n%s\n", trail.ReviewerApproval))

	return sb.String()
}
