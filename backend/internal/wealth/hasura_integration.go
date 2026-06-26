package wealth

import (
	"context"
	"fmt"

	"github.com/hondyman/semlayer/backend/pkg/meta"
)

// WealthTransferHasuraIntegration generates Hasura metadata for wealth transfer
type WealthTransferHasuraIntegration struct {
	hasuraGenerator *meta.HasuraMetadataGenerator
	metaService     *meta.Service
}

// NewWealthTransferHasuraIntegration creates the integration handler
func NewWealthTransferHasuraIntegration(
	hasuraGenerator *meta.HasuraMetadataGenerator,
	metaService *meta.Service,
) *WealthTransferHasuraIntegration {
	return &WealthTransferHasuraIntegration{
		hasuraGenerator: hasuraGenerator,
		metaService:     metaService,
	}
}

// GenerateAndApplyMetadata generates and applies all Hasura metadata for wealth transfer
func (w *WealthTransferHasuraIntegration) GenerateAndApplyMetadata(ctx context.Context, tenantID string) error {
	// Load all wealth transfer business objects
	businessObjects := []string{
		"Family_Office",
		"Family_Member",
		"Estate_Plan_Scenario",
		"Gift_History",
		"Trust_Entity",
	}

	for _, boName := range businessObjects {
		// Get business object definition
		bo, err := w.GetBusinessObjectByName(ctx, tenantID, boName)
		if err != nil {
			return fmt.Errorf("failed to get business object %s: %w", boName, err)
		}

		// Generate and apply Hasura metadata
		if err := w.hasuraGenerator.GenerateAndApply(ctx, bo); err != nil {
			return fmt.Errorf("failed to generate Hasura metadata for %s: %w", boName, err)
		}
	}

	// Generate custom Hasura actions for wealth transfer
	if err := w.generateCustomActions(ctx); err != nil {
		return fmt.Errorf("failed to generate custom actions: %w", err)
	}

	return nil
}

// generateCustomActions creates custom Hasura actions for complex operations
func (w *WealthTransferHasuraIntegration) generateCustomActions(ctx context.Context) error {
	actions := []HasuraAction{
		{
			Name:        "generate_estate_plan",
			Handler:     "{{BACKEND_URL}}/api/wealth-transfer/hasura/generate-estate-plan",
			Kind:        "synchronous",
			Permissions: []string{"advisor", "family_admin"},
			InputType: map[string]string{
				"family_id":     "uuid!",
				"max_scenarios": "Int",
			},
			OutputType: "GenerateEstatePlanOutput",
		},
		{
			Name:        "calculate_estate_tax",
			Handler:     "{{BACKEND_URL}}/api/wealth-transfer/hasura/calculate-estate-tax",
			Kind:        "synchronous",
			Permissions: []string{"advisor", "family_admin", "family_member"},
			InputType: map[string]string{
				"gross_estate_value": "numeric!",
				"state_code":         "String!",
			},
			OutputType: "EstateTaxOutput",
		},
		{
			Name:        "optimize_gifting_strategy",
			Handler:     "{{BACKEND_URL}}/api/wealth-transfer/hasura/optimize-gifting",
			Kind:        "synchronous",
			Permissions: []string{"advisor", "family_admin"},
			InputType: map[string]string{
				"family_id":   "uuid!",
				"constraints": "jsonb",
			},
			OutputType: "OptimizedGiftingOutput",
		},
		{
			Name:        "record_gift_with_calc",
			Handler:     "{{BACKEND_URL}}/api/wealth-transfer/hasura/record-gift",
			Kind:        "synchronous",
			Permissions: []string{"advisor", "family_admin"},
			InputType: map[string]string{
				"family_id":         "uuid!",
				"donor_member_id":   "uuid!",
				"fair_market_value": "numeric!",
				"gift_date":         "date!",
			},
			OutputType: "GiftRecordOutput",
		},
	}

	for _, action := range actions {
		if err := w.applyHasuraAction(ctx, action); err != nil {
			return fmt.Errorf("failed to apply action %s: %w", action.Name, err)
		}
	}

	return nil
}

// HasuraAction represents a custom Hasura action
type HasuraAction struct {
	Name        string
	Handler     string
	Kind        string
	Permissions []string
	InputType   map[string]string
	OutputType  string
}

func (w *WealthTransferHasuraIntegration) applyHasuraAction(ctx context.Context, action HasuraAction) error {
	// This would call Hasura's metadata API to create the action
	// Implementation would use HTTP client to POST to Hasura metadata endpoint
	return nil
}

// GetBusinessObjectByName is a helper (this should be in metaService)
func (w *WealthTransferHasuraIntegration) GetBusinessObjectByName(ctx context.Context, tenantID, name string) (*meta.BusinessObjectDefinition, error) {
	// This would query the business_object_definition table by name
	// For now, placeholder
	return &meta.BusinessObjectDefinition{
		TenantID: tenantID,
		Name:     name,
	}, nil
}
