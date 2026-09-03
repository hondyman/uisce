package workflows

import (
	"context"
	"fmt"

	"go.temporal.io/sdk/activity"

	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/hondyman/uisce/backend/models"
)

// ActivityDeps holds dependencies real (non-stub) BP activities need. Today
// just the centralized calc engine — ActivityCalculation below calls the
// exact same analytics.SemanticCalculationService.ExecuteFormulaCalculation
// that backs POST /calculations/{id}/execute, so a "Calculation" workflow
// step and a direct API call to run the same calc produce identical
// results; there is one engine, not a workflow-specific reimplementation.
type ActivityDeps struct {
	CalcService *analytics.SemanticCalculationService
}

// ActivityCalculation runs a centralized calculation
// (analytics.SemanticCalculationService.ExecuteFormulaCalculation) as a
// workflow step. This was previously a gap: dynamic_bp_workflow.go
// dispatches "Calculation"/"SemanticRollup" nodes to an activity literally
// named "Activity"+node.Type by convention, but no ActivityCalculation was
// ever implemented or registered — a real "Calculation" node would fail at
// runtime with "unable to find activityType". See bp_worker.go for
// registration.
//
// config must include "tenant_id" and either "calculation_id" or
// "calculation_name" (matching models.Calculation.Name).
func (d *ActivityDeps) ActivityCalculation(ctx context.Context, config map[string]interface{}, state map[string]interface{}) (map[string]interface{}, error) {
	logger := activity.GetLogger(ctx)

	tenantID, _ := config["tenant_id"].(string)
	if tenantID == "" {
		return nil, fmt.Errorf("ActivityCalculation: config.tenant_id is required")
	}

	calc, err := d.resolveCalculation(config)
	if err != nil {
		return nil, err
	}

	logger.Info("Executing centralized calculation", "calculation", calc.Name, "tenant", tenantID)

	results, tier, err := d.CalcService.ExecuteFormulaCalculation(ctx, tenantID, calc)
	if err != nil {
		return nil, fmt.Errorf("calculation %q failed: %w", calc.Name, err)
	}

	return map[string]interface{}{
		"calculation_status": "completed",
		"calculation_name":   calc.Name,
		"calculation_tier":   tier,
		"result":             results,
	}, nil
}

func (d *ActivityDeps) resolveCalculation(config map[string]interface{}) (*models.Calculation, error) {
	if id, ok := config["calculation_id"].(string); ok && id != "" {
		return d.CalcService.GetCalculationByID(id)
	}
	if name, ok := config["calculation_name"].(string); ok && name != "" {
		return d.CalcService.GetCalculationByName(name)
	}
	return nil, fmt.Errorf("ActivityCalculation: config must include calculation_id or calculation_name")
}
