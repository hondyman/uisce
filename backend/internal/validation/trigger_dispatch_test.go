package validation

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

// test helpers
func jsonRaw(s string) json.RawMessage { return json.RawMessage([]byte(s)) }
func ptrString(s string) *string       { return &s }

// ============================================================================
// PHASE 6A: TRIGGER DISPATCH UNIT TESTS
// ============================================================================
// Tests for the trigger dispatch system covering all 8 implemented trigger types.
//
// Run with: go test ./backend/internal/validation -v -run TestDispatch
// ============================================================================

// TestDispatchTrigger_Create tests the CREATE trigger dispatch
func TestDispatchTrigger_Create(t *testing.T) {
	engine := NewTriggerValidationEngine(nil, &SimpleLogger{})
	tenantID := uuid.New()
	ctx := context.Background()
	// prepare in-memory triggers and rules for the test
	trig := ValidationTrigger{
		ID:           uuid.New().String(),
		TenantID:     tenantID.String(),
		TriggerType:  "create",
		TargetEntity: "orders",
		StepName:     nil,
		RuleIDs:      pq.StringArray{"rule-1", "rule-2"},
		Meta:         jsonRaw(`{}`),
	}

	rule2 := ValidationRule{
		ID:             "rule-2",
		TenantID:       tenantID.String(),
		RuleName:       "Test Rule 2",
		RuleType:       "cardinality",
		TargetEntities: pq.StringArray{"orders"},
		ConditionJSON:  jsonRaw(`{"field":"total","operator":">","value":0}`),
		ErrorMessage:   "Total must be positive",
	}

	rule1 := ValidationRule{
		ID:             "rule-1",
		TenantID:       tenantID.String(),
		RuleName:       "Positive Total",
		RuleType:       "cardinality",
		TargetEntities: pq.StringArray{"orders"},
		ConditionJSON:  jsonRaw(`{"field":"total","operator":">","value":0}`),
		ErrorMessage:   "Total must be > 0",
	}

	engine.WithTestTriggers([]ValidationTrigger{trig}).WithTestRules(map[string]ValidationRule{"rule-1": rule1, "rule-2": rule2})

	orderData := map[string]interface{}{
		"customer_id": "cust-123",
		"total":       100,
	}

	// ===== TEST: Create trigger should pass with valid data =====
	err := engine.DispatchTrigger(ctx, tenantID, TriggerTypeCreate, "orders", orderData)
	assert.NoError(t, err, "Create trigger should pass with valid data")
}

// TestDispatchTrigger_Create_ValidationFails tests CREATE trigger when validation fails
func TestDispatchTrigger_Create_ValidationFails(t *testing.T) {
	engine := NewTriggerValidationEngine(nil, &SimpleLogger{})
	tenantID := uuid.New()
	ctx := context.Background()

	trig := ValidationTrigger{
		ID:           uuid.New().String(),
		TenantID:     tenantID.String(),
		TriggerType:  "create",
		TargetEntity: "orders",
		StepName:     nil,
		RuleIDs:      pq.StringArray{"rule-1"},
		Meta:         jsonRaw(`{}`),
	}

	rule := ValidationRule{
		ID:             "rule-1",
		TenantID:       tenantID.String(),
		RuleName:       "Positive Total",
		RuleType:       "cardinality",
		TargetEntities: pq.StringArray{"orders"},
		ConditionJSON:  jsonRaw(`{"field":"total","operator":">","value":0}`),
		ErrorMessage:   "Total must be > 0",
	}

	engine.WithTestTriggers([]ValidationTrigger{trig}).WithTestRules(map[string]ValidationRule{"rule-1": rule})

	orderData := map[string]interface{}{
		"customer_id": "cust-123",
		"total":       -50, // INVALID: negative total
	}

	// ===== TEST: Create trigger should fail with negative total =====
	err := engine.DispatchTrigger(ctx, tenantID, TriggerTypeCreate, "orders", orderData)
	assert.Error(t, err, "Create trigger should fail with negative total")
	assert.Contains(t, err.Error(), "Positive Total", "Error should mention the rule name")
}

// TestDispatchTrigger_Save tests the SAVE trigger dispatch
func TestDispatchTrigger_Save(t *testing.T) {
	engine := NewTriggerValidationEngine(nil, &SimpleLogger{})
	tenantID := uuid.New()
	ctx := context.Background()

	// No triggers/rules configured
	engine.WithTestTriggers([]ValidationTrigger{}).WithTestRules(map[string]ValidationRule{})

	orderData := map[string]interface{}{
		"customer_id": "cust-456",
		"total":       250,
	}

	// ===== TEST: Save with no triggers should pass =====
	err := engine.DispatchTrigger(ctx, tenantID, TriggerTypeSave, "orders", orderData)
	assert.NoError(t, err, "Save trigger should pass with no rules")
}

// TestDispatchTrigger_Delete tests the DELETE trigger dispatch
func TestDispatchTrigger_Delete(t *testing.T) {
	engine := NewTriggerValidationEngine(nil, &SimpleLogger{})
	tenantID := uuid.New()
	ctx := context.Background()

	trig := ValidationTrigger{
		ID:           uuid.New().String(),
		TenantID:     tenantID.String(),
		TriggerType:  "delete",
		TargetEntity: "orders",
		StepName:     nil,
		RuleIDs:      pq.StringArray{"rule-cannot-delete-shipped"},
		Meta:         jsonRaw(`{}`),
	}

	rule := ValidationRule{
		ID:             "rule-cannot-delete-shipped",
		TenantID:       tenantID.String(),
		RuleName:       "Cannot Delete Shipped",
		RuleType:       "business_logic",
		TargetEntities: pq.StringArray{"orders"},
		// Pass when status != shipped
		ConditionJSON: jsonRaw(`{"field":"status","operator":"!=","value":"shipped"}`),
		ErrorMessage:  "Cannot delete shipped orders",
	}

	engine.WithTestTriggers([]ValidationTrigger{trig}).WithTestRules(map[string]ValidationRule{"rule-cannot-delete-shipped": rule})

	orderData := map[string]interface{}{
		"status": "pending",
	}

	// ===== TEST: Delete of pending order should pass =====
	err := engine.DispatchTrigger(ctx, tenantID, TriggerTypeDelete, "orders", orderData)
	assert.NoError(t, err)
}

// TestDispatchFieldChange tests field change trigger dispatch
func TestDispatchFieldChange(t *testing.T) {
	engine := NewTriggerValidationEngine(nil, &SimpleLogger{})
	tenantID := uuid.New()
	ctx := context.Background()

	// No field_format rules for this tenant/entity/field
	engine.WithTestTriggers([]ValidationTrigger{}).WithTestRules(map[string]ValidationRule{})

	record := map[string]interface{}{
		"id":    "order-1",
		"total": 200,
	}

	// ===== TEST: Field change with valid value =====
	err := engine.DispatchFieldChange(ctx, tenantID, "orders", "total", 100, 200, record)
	assert.NoError(t, err, "Field change should pass with valid value")
}

// TestDispatchStatusChange tests status change trigger dispatch
func TestDispatchStatusChange(t *testing.T) {
	engine := NewTriggerValidationEngine(nil, &SimpleLogger{})
	tenantID := uuid.New()
	ctx := context.Background()

	trig := ValidationTrigger{
		ID:           uuid.New().String(),
		TenantID:     tenantID.String(),
		TriggerType:  "status_change",
		TargetEntity: "orders",
		StepName:     nil,
		RuleIDs:      pq.StringArray{"rule-status-transition"},
		Meta:         jsonRaw(`{}`),
	}

	rule := ValidationRule{
		ID:             "rule-status-transition",
		TenantID:       tenantID.String(),
		RuleName:       "Valid Status",
		RuleType:       "business_logic",
		TargetEntities: pq.StringArray{"orders"},
		ConditionJSON:  jsonRaw(`{"field":"status","operator":"==","value":"approved"}`),
		ErrorMessage:   "Status transition not allowed",
	}

	engine.WithTestTriggers([]ValidationTrigger{trig}).WithTestRules(map[string]ValidationRule{"rule-status-transition": rule})

	record := map[string]interface{}{
		"id": "order-1",
	}

	// ===== TEST: Status change from pending to approved =====
	err := engine.DispatchStatusChange(ctx, tenantID, "orders", "status", "pending", "approved", record)
	assert.NoError(t, err, "Status change should pass")
}

// TestDispatchSubEntityChange tests sub-entity change trigger dispatch
func TestDispatchSubEntityChange(t *testing.T) {
	engine := NewTriggerValidationEngine(nil, &SimpleLogger{})
	tenantID := uuid.New()
	parentID := uuid.New()
	ctx := context.Background()

	trig := ValidationTrigger{
		ID:           uuid.New().String(),
		TenantID:     tenantID.String(),
		TriggerType:  "sub_entity_change",
		TargetEntity: "order_items",
		StepName:     nil,
		RuleIDs:      pq.StringArray{"rule-item-qty"},
		Meta:         jsonRaw(`{}`),
	}

	rule := ValidationRule{
		ID:             "rule-item-qty",
		TenantID:       tenantID.String(),
		RuleName:       "Item Qty Must Be Positive",
		RuleType:       "cardinality",
		TargetEntities: pq.StringArray{"order_items"},
		ConditionJSON:  jsonRaw(`{"field":"qty","operator":">","value":0}`),
		ErrorMessage:   "Qty must be > 0",
	}

	engine.WithTestTriggers([]ValidationTrigger{trig}).WithTestRules(map[string]ValidationRule{"rule-item-qty": rule})

	itemData := map[string]interface{}{
		"sku": "SKU-123",
		"qty": 5,
	}

	// ===== TEST: Sub-entity change with valid data =====
	err := engine.DispatchSubEntityChange(ctx, tenantID, "orders", parentID, "order_items", itemData)
	assert.NoError(t, err, "Sub-entity change should pass with valid qty")
}

// TestDispatchRelationshipChange tests relationship change trigger dispatch
func TestDispatchRelationshipChange(t *testing.T) {
	engine := NewTriggerValidationEngine(nil, &SimpleLogger{})
	tenantID := uuid.New()
	ctx := context.Background()

	// No triggers configured for relationship_change
	engine.WithTestTriggers([]ValidationTrigger{}).WithTestRules(map[string]ValidationRule{})

	record := map[string]interface{}{
		"id": "order-1",
	}

	// ===== TEST: Relationship change with no rules =====
	err := engine.DispatchRelationshipChange(ctx, tenantID, "orders", "customer_id", "cust-1", "cust-2", record)
	assert.NoError(t, err, "Relationship change should pass with no rules")
}

// TestDispatchWithStep tests dispatch with step name (workflow step trigger)
func TestDispatchWithStep(t *testing.T) {
	engine := NewTriggerValidationEngine(nil, &SimpleLogger{})
	tenantID := uuid.New()
	ctx := context.Background()

	// Mock fetchTriggers with step name via in-memory overrides
	trig := ValidationTrigger{
		ID:           uuid.New().String(),
		TenantID:     tenantID.String(),
		TriggerType:  "workflow_step",
		TargetEntity: "hire_employee",
		StepName:     ptrString("manager_approval"),
		RuleIDs:      pq.StringArray{"rule-manager-auth"},
		Meta:         jsonRaw(`{}`),
	}

	rule := ValidationRule{
		ID:             "rule-manager-auth",
		TenantID:       tenantID.String(),
		RuleName:       "Manager Authorization",
		RuleType:       "business_logic",
		TargetEntities: pq.StringArray{"hire_employee"},
		ConditionJSON:  jsonRaw(`{"field":"role","operator":"==","value":"manager"}`),
		ErrorMessage:   "Only managers can approve hires",
	}

	engine.WithTestTriggers([]ValidationTrigger{trig}).WithTestRules(map[string]ValidationRule{"rule-manager-auth": rule})

	hireData := map[string]interface{}{
		"employee_name": "John Doe",
		"role":          "manager",
	}

	// ===== TEST: Workflow step trigger with step name =====
	err := engine.DispatchTriggerWithStep(ctx, tenantID, TriggerTypeWorkflowStep, "hire_employee", "manager_approval", hireData)
	assert.NoError(t, err, "Workflow step dispatch should pass")
}

// TestDispatchTrigger_NoDatabase tests error handling when DB is nil
func TestDispatchTrigger_NoDatabase(t *testing.T) {
	engine := &TriggerValidationEngine{
		ValidationEngine: NewValidationEngine(),
		db:               nil,
		logger:           &SimpleLogger{},
	}

	ctx := context.Background()
	tenantID := uuid.New()
	data := map[string]interface{}{"total": 100}

	// ===== TEST: Should return error when DB is nil =====
	err := engine.DispatchTrigger(ctx, tenantID, TriggerTypeCreate, "orders", data)
	assert.Error(t, err, "Should return error when DB is nil")
	assert.Contains(t, err.Error(), "db not configured", "Error should mention DB configuration")
}

// TestDispatchFieldChange_NoDatabase tests field change error when DB is nil
func TestDispatchFieldChange_NoDatabase(t *testing.T) {
	engine := &TriggerValidationEngine{
		ValidationEngine: NewValidationEngine(),
		db:               nil,
		logger:           &SimpleLogger{},
	}

	ctx := context.Background()
	tenantID := uuid.New()

	// ===== TEST: Should return error when DB is nil =====
	err := engine.DispatchFieldChange(ctx, tenantID, "orders", "total", 100, 200, nil)
	assert.Error(t, err, "Should return error when DB is nil")
	assert.Contains(t, err.Error(), "db not configured", "Error should mention DB configuration")
}

// TestDispatchTriggerTypes tests all trigger type constants
func TestDispatchTriggerTypes(t *testing.T) {
	// ===== TEST: Verify all 8 implemented trigger types =====
	assert.Equal(t, TriggerTypeSave, TriggerType("save"))
	assert.Equal(t, TriggerTypeCreate, TriggerType("create"))
	assert.Equal(t, TriggerTypeDelete, TriggerType("delete"))
	assert.Equal(t, TriggerTypeFieldChange, TriggerType("field_change"))
	assert.Equal(t, TriggerTypeWorkflowStep, TriggerType("workflow_step"))
	assert.Equal(t, TriggerTypeSubEntityChange, TriggerType("sub_entity_change"))
	assert.Equal(t, TriggerTypeRelationshipChange, TriggerType("relationship_change"))
	assert.Equal(t, TriggerTypeStatusChange, TriggerType("status_change"))

	// 5 future types
	assert.Equal(t, TriggerTypeBulkLoad, TriggerType("bulk_load"))
	assert.Equal(t, TriggerTypeIntegration, TriggerType("integration"))
	assert.Equal(t, TriggerTypeTimeBased, TriggerType("time_based"))
	assert.Equal(t, TriggerTypeCalculatedField, TriggerType("calculated_field"))
	assert.Equal(t, TriggerTypeSecurityRole, TriggerType("security_role"))
}
