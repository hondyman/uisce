package workflows

import (
	"context"
	"fmt"

	"github.com/hondyman/semlayer/backend/pkg/governance"
)

type ComplianceActivities struct {
	engine *governance.GovernanceEngine
}

func NewComplianceActivities(engine *governance.GovernanceEngine) *ComplianceActivities {
	return &ComplianceActivities{
		engine: engine,
	}
}

// ActivityCheckCompliance validates a payload against the trade compliance policy
// It returns an error if the compliance check fails, effectively blocking the workflow.
func (a *ComplianceActivities) ActivityCheckCompliance(ctx context.Context, config map[string]interface{}, payload map[string]interface{}) (map[string]interface{}, error) {
	// activity.RecordHeartbeat(ctx, "Validating compliance policy...")

	// Extract Tenant ID
	var tenantID string
	if tid, ok := payload["tenant_id"].(string); ok {
		tenantID = tid
	} else if tid, ok := config["tenant_id"].(string); ok {
		tenantID = tid
	}

	result, err := a.engine.ValidateTransaction(ctx, tenantID, payload)
	if err != nil {
		return nil, fmt.Errorf("compliance engine error: %w", err)
	}

	if !result.Allowed {
		// We return a specific error so the workflow can handle it (or fail)
		return nil, fmt.Errorf("COMPLIANCE_VIOLATION: %v", result.Reasons)
	}

	return map[string]interface{}{
		"compliance_status": "APPROVED",
		"timestamp":         "now", // simplifying
	}, nil
}
