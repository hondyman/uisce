package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// InitializeDefaultWorkflowPoliciesSQLX inserts default workflow ABAC policies using sqlx.
func InitializeDefaultWorkflowPoliciesSQLX(db *sqlx.DB, tenantID, datasourceID string) error {
	defaultPolicies := []WorkflowABACPolicy{
		{
			ID:               uuid.New().String(),
			TenantID:         tenantID,
			DatasourceID:     datasourceID,
			WorkflowType:     "investment",
			Action:           "create",
			ResourcePattern:  "*",
			SubjectRules:     map[string]interface{}{"role": []string{"advisor", "portfolio_manager"}},
			EnvironmentRules: map[string]interface{}{"business_hours": true},
			RiskLevel:        "medium",
			RequiresApproval: false,
			TimeRestrictions: map[string]interface{}{"business_hours": true},
			Enabled:          true,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		},
		{
			ID:               uuid.New().String(),
			TenantID:         tenantID,
			DatasourceID:     datasourceID,
			WorkflowType:     "compliance",
			Action:           "execute",
			ResourcePattern:  "*",
			SubjectRules:     map[string]interface{}{"role": []string{"compliance_officer", "senior_compliance"}},
			EnvironmentRules: map[string]interface{}{},
			RiskLevel:        "high",
			RequiresApproval: true,
			ApprovalRoles:    []string{"senior_compliance", "chief_compliance_officer"},
			TimeRestrictions: map[string]interface{}{},
			Enabled:          true,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		},
		{
			ID:               uuid.New().String(),
			TenantID:         tenantID,
			DatasourceID:     datasourceID,
			WorkflowType:     "onboarding",
			Action:           "modify",
			ResourcePattern:  "*",
			SubjectRules:     map[string]interface{}{"role": []string{"client_services", "relationship_manager"}},
			EnvironmentRules: map[string]interface{}{},
			RiskLevel:        "low",
			RequiresApproval: false,
			TimeRestrictions: map[string]interface{}{},
			Enabled:          true,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		},
	}

	const q = `INSERT INTO workflow_abac_policies
        (id, tenant_id, datasource_id, workflow_type, action, resource_pattern, subject_rules, environment_rules, risk_level, requires_approval, approval_roles, time_restrictions, enabled, created_at, updated_at)
        VALUES ($1,$2,$3,$4,$5,$6,$7::jsonb,$8::jsonb,$9,$10,$11::jsonb,$12::jsonb,$13,$14,$15)
        ON CONFLICT (id) DO NOTHING`

	for _, p := range defaultPolicies {
		subj, _ := json.Marshal(p.SubjectRules)
		env, _ := json.Marshal(p.EnvironmentRules)
		approvalRoles, _ := json.Marshal(p.ApprovalRoles)
		timeRestr, _ := json.Marshal(p.TimeRestrictions)

		if _, err := db.Exec(q,
			p.ID,
			p.TenantID,
			p.DatasourceID,
			p.WorkflowType,
			p.Action,
			p.ResourcePattern,
			string(subj),
			string(env),
			p.RiskLevel,
			p.RequiresApproval,
			string(approvalRoles),
			string(timeRestr),
			p.Enabled,
			p.CreatedAt,
			p.UpdatedAt,
		); err != nil {
			return fmt.Errorf("failed to insert policy %s: %w", p.ID, err)
		}
	}

	return nil
}
