package entitlements

import (
	"context"
)

type EntitlementIssue struct {
	Type           string   `json:"type"`     // over_privileged, privilege_creep, cross_tenant_inconsistency, high_risk, unused
	Severity       string   `json:"severity"` // critical, high, medium, low
	RoleID         string   `json:"role_id"`
	RoleName       string   `json:"role_name"`
	Description    string   `json:"description"`
	Evidence       []string `json:"evidence"`
	Recommendation string   `json:"recommendation"`
}

type EntitlementReport struct {
	TenantID string             `json:"tenant_id"`
	Issues   []EntitlementIssue `json:"issues"`
	Summary  string             `json:"summary"`
}

type EntitlementAnalyzer struct{}

func NewEntitlementAnalyzer() *EntitlementAnalyzer {
	return &EntitlementAnalyzer{}
}

func (ea *EntitlementAnalyzer) Analyze(ctx context.Context, tenantID string) (*EntitlementReport, error) {
	// Mock: Generate entitlement analysis
	// Real: Analyze role → BO/API/page usage, tenant overlays, data policies, access logs

	issues := []EntitlementIssue{
		{
			Type:        "over_privileged",
			Severity:    "high",
			RoleID:      "role-advisor",
			RoleName:    "Advisor",
			Description: "Role has access to 14 APIs but only uses 6",
			Evidence: []string{
				"Access granted: positions_api, accounts_api, trades_api, portfolio_api, risk_api, compliance_api, reporting_api, analytics_api, market_data_api, reference_data_api, workflow_api, notification_api, audit_api, admin_api",
				"Actually used (last 30 days): positions_api, accounts_api, portfolio_api, risk_api, reporting_api, analytics_api",
			},
			Recommendation: "Remove unused API permissions: trades_api, compliance_api, market_data_api, reference_data_api, workflow_api, notification_api, audit_api, admin_api",
		},
		{
			Type:        "privilege_creep",
			Severity:    "medium",
			RoleID:      "role-analyst",
			RoleName:    "Analyst",
			Description: "Role accumulated 8 new permissions in last 90 days",
			Evidence: []string{
				"Permissions added: view_pii, edit_positions, approve_trades, access_audit_logs, manage_workflows, configure_alerts, export_data, delete_records",
				"Original role scope: read-only data access",
			},
			Recommendation: "Review and remove unnecessary permissions added over time",
		},
		{
			Type:        "cross_tenant_inconsistency",
			Severity:    "medium",
			RoleID:      "role-advisor",
			RoleName:    "Advisor",
			Description: "Tenant-123 has weaker PII protections than similar tenants",
			Evidence: []string{
				"Tenant-123: PII masking disabled for 3 fields",
				"Peer tenants (456, 789): PII masking enabled for all fields",
			},
			Recommendation: "Apply policy bundle 'wealth_standard_pii' to align with peer tenants",
		},
		{
			Type:        "high_risk",
			Severity:    "critical",
			RoleID:      "role-trader",
			RoleName:    "Trader",
			Description: "Role has unrestricted access to trading actions and PII",
			Evidence: []string{
				"Can execute trades without approval",
				"Can access client SSN and account numbers",
				"No MFA requirement",
			},
			Recommendation: "Add MFA requirement + approval workflow for high-value trades + mask PII fields",
		},
		{
			Type:        "unused",
			Severity:    "low",
			RoleID:      "role-compliance",
			RoleName:    "Compliance Officer",
			Description: "Permission 'delete_audit_logs' granted but never used",
			Evidence: []string{
				"Permission granted: 2024-03-15",
				"Last used: never",
				"Days since grant: 307",
			},
			Recommendation: "Remove unused permission",
		},
	}

	report := &EntitlementReport{
		TenantID: tenantID,
		Issues:   issues,
		Summary:  "Found 5 entitlement issues: 1 critical, 2 high, 2 medium, 0 low. Recommend immediate review of high-risk permissions.",
	}

	return report, nil
}
