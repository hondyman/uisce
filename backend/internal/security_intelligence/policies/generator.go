package policies

import (
	"context"

	"github.com/google/uuid"
)

type PolicyType string

const (
	PolicyTypeMasking        PolicyType = "data_masking"
	PolicyTypeResidency      PolicyType = "residency"
	PolicyTypeEntitlement    PolicyType = "entitlement"
	PolicyTypeAPIAccess      PolicyType = "api_access"
	PolicyTypePageVisibility PolicyType = "page_visibility"
	PolicyTypeWorkflow       PolicyType = "workflow_security"
)

type PolicySuggestion struct {
	ID          uuid.UUID      `json:"id"`
	Type        PolicyType     `json:"type"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Rationale   string         `json:"rationale"`
	Impact      ImpactAnalysis `json:"impact"`
	ChangeSet   string         `json:"changeset_id,omitempty"`
}

type ImpactAnalysis struct {
	AffectedTenants int      `json:"affected_tenants"`
	AffectedUsers   int      `json:"affected_users"`
	AffectedPages   []string `json:"affected_pages"`
	AffectedAPIs    []string `json:"affected_apis"`
	RiskLevel       string   `json:"risk_level"`
}

type PolicyGenerator struct{}

func NewPolicyGenerator() *PolicyGenerator {
	return &PolicyGenerator{}
}

func (pg *PolicyGenerator) Suggest(ctx context.Context) ([]PolicySuggestion, error) {
	// Mock: Generate policy suggestions
	// Real: Analyze BO definitions, PII classifications, residency metadata, compliance templates

	suggestions := []PolicySuggestion{
		{
			ID:          uuid.New(),
			Type:        PolicyTypeMasking,
			Name:        "Mask client_ssn for non-advisor roles",
			Description: "Apply data masking to field 'client_ssn' for all roles except Advisor",
			Rationale:   "Field contains PII (SSN). Only advisors require unmasked access for KYC compliance.",
			Impact: ImpactAnalysis{
				AffectedTenants: 14,
				AffectedUsers:   247,
				AffectedPages:   []string{"Client Profile", "Account Overview", "KYC Dashboard"},
				AffectedAPIs:    []string{"clients_api", "accounts_api"},
				RiskLevel:       "low",
			},
		},
		{
			ID:          uuid.New(),
			Type:        PolicyTypeResidency,
			Name:        "Restrict positions_detail API to US tenants",
			Description: "Block access to positions_detail API for non-US tenants due to data residency requirements",
			Rationale:   "API exposes US-domiciled securities data. EU tenants must use positions_summary API instead.",
			Impact: ImpactAnalysis{
				AffectedTenants: 8,
				AffectedUsers:   92,
				AffectedPages:   []string{"Positions Dashboard"},
				AffectedAPIs:    []string{"positions_detail"},
				RiskLevel:       "medium",
			},
		},
		{
			ID:          uuid.New(),
			Type:        PolicyTypeWorkflow,
			Name:        "Require MFA for trade_approval workflow step 3",
			Description: "Add multi-factor authentication requirement for final approval step in trade workflow",
			Rationale:   "High-value trades (>$100k) require additional security. MFA reduces unauthorized approval risk.",
			Impact: ImpactAnalysis{
				AffectedTenants: 22,
				AffectedUsers:   45,
				AffectedPages:   []string{"Trade Approval Dashboard"},
				AffectedAPIs:    []string{"workflow_api"},
				RiskLevel:       "low",
			},
		},
		{
			ID:          uuid.New(),
			Type:        PolicyTypeEntitlement,
			Name:        "Remove delete_audit_logs permission from Compliance role",
			Description: "Revoke delete_audit_logs permission as it violates audit integrity requirements",
			Rationale:   "Audit logs must be immutable per SOX compliance. No role should have delete permissions.",
			Impact: ImpactAnalysis{
				AffectedTenants: 31,
				AffectedUsers:   18,
				AffectedPages:   []string{},
				AffectedAPIs:    []string{"audit_api"},
				RiskLevel:       "critical",
			},
		},
	}

	return suggestions, nil
}
