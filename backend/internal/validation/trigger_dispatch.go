package validation

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// ============================================================================
// PHASE 6A: TRIGGER DISPATCH SYSTEM
// ============================================================================
// Application-layer trigger dispatch that fires validation rules based on
// entity actions (Create, Save, Delete, FieldChange, etc).
//
// This is Workday's secret: NOT database triggers, but APPLICATION-LAYER
// smart events that execute before data persistence.
//
// Usage Pattern:
//   Before INSERT: engine.DispatchTrigger(ctx, tenantID, "create", "orders", orderData)
//   Before UPDATE: engine.DispatchTrigger(ctx, tenantID, "save", "orders", orderData)
//   Before DELETE: engine.DispatchTrigger(ctx, tenantID, "delete", "orders", orderData)
//   On field change: engine.DispatchFieldChange(ctx, tenantID, "orders", "total", oldVal, newVal, record)
//
// All trigger types defined in const below (matches Workday's 13 types):
//   - save, create, delete, field_change, workflow_step, bulk_load, integration,
//     time_based, sub_entity_change, relationship_change, status_change, calculated_field, security_role
//
// Returns nil if all triggers pass; otherwise returns error describing first failure.
// ============================================================================

// TriggerType represents the different types of triggers (Workday's 13 types)
type TriggerType string

const (
	TriggerTypeSave               TriggerType = "save"                // Entity saved (UPDATE)
	TriggerTypeCreate             TriggerType = "create"              // New entity (INSERT)
	TriggerTypeDelete             TriggerType = "delete"              // Entity deleted
	TriggerTypeFieldChange        TriggerType = "field_change"        // Single field modified
	TriggerTypeWorkflowStep       TriggerType = "workflow_step"       // BP step complete
	TriggerTypeBulkLoad           TriggerType = "bulk_load"           // Batch import
	TriggerTypeIntegration        TriggerType = "integration"         // External system event
	TriggerTypeTimeBased          TriggerType = "time_based"          // Scheduled/cron
	TriggerTypeSubEntityChange    TriggerType = "sub_entity_change"   // Child entity modified
	TriggerTypeRelationshipChange TriggerType = "relationship_change" // Link/FK modified
	TriggerTypeStatusChange       TriggerType = "status_change"       // Status field updated
	TriggerTypeCalculatedField    TriggerType = "calculated_field"    // Formula recalc
	TriggerTypeSecurityRole       TriggerType = "security_role"       // Role assignment
)

// TriggerDispatchContext holds metadata about the trigger dispatch event
type TriggerDispatchContext struct {
	TenantID      uuid.UUID              // Tenant scope
	Entity        string                 // Target entity (orders, customers, etc)
	Action        TriggerType            // What action fired (create, save, delete)
	Data          map[string]interface{} // New data being saved
	OldData       map[string]interface{} // Previous data (for updates)
	FieldName     string                 // For field_change: which field changed
	UserID        string                 // Who performed action (optional)
	CorrelationID string                 // Request correlation for tracing
}

// DispatchTrigger is the main entry point for entity-action triggers.
// Call this BEFORE database INSERT/UPDATE/DELETE to validate the operation.
//
// Returns nil if all triggers pass; returns error if any rule fails.
//
// Example usage:
//
//	if err := engine.DispatchTrigger(ctx, tenantID, TriggerTypeCreate, "orders", orderData); err != nil {
//	    return fmt.Errorf("validation failed: %w", err)  // Block the insert
//	}
//	// Safe to insert into DB now
//	db.ExecContext(ctx, "INSERT INTO orders...", orderData)
func (tve *TriggerValidationEngine) DispatchTrigger(
	ctx context.Context,
	tenantID uuid.UUID,
	action TriggerType,
	entity string,
	data map[string]interface{},
) error {
	// Use existing TriggerValidate which already handles this pattern
	return tve.TriggerValidate(ctx, tenantID, string(action), entity, "", data)
}

// DispatchTriggerWithStep is like DispatchTrigger but includes a step name
// for workflow step triggers.
//
// Example:
//
//	if err := engine.DispatchTriggerWithStep(ctx, tenantID, TriggerTypeWorkflowStep,
//	    "hire_employee", "manager_approval", approvalData); err != nil {
//	    return fmt.Errorf("approval validation failed: %w", err)
//	}
func (tve *TriggerValidationEngine) DispatchTriggerWithStep(
	ctx context.Context,
	tenantID uuid.UUID,
	action TriggerType,
	entity string,
	stepName string,
	data map[string]interface{},
) error {
	return tve.TriggerValidate(ctx, tenantID, string(action), entity, stepName, data)
}

// DispatchFieldChange fires field_change triggers when a single field is modified.
// This is optimized for onChange events in forms.
//
// Arguments:
//   - fieldName: name of field that changed (e.g., "total", "status")
//   - newValue: the new value for the field
//   - record: the full record being edited (for context)
//
// Example:
//
//	// User changes order total from 100 to -50
//	if err := engine.DispatchFieldChange(ctx, tenantID, "orders", "total",
//	    100, -50, orderRecord); err != nil {
//	    return fmt.Errorf("order total validation: %w", err)  // Show error to user
//	}
//	// Field change is valid, update UI/DB
func (tve *TriggerValidationEngine) DispatchFieldChange(
	ctx context.Context,
	tenantID uuid.UUID,
	entity string,
	fieldName string,
	oldValue interface{},
	newValue interface{},
	record map[string]interface{},
) error {
	// Allow test-only in-memory overrides to be used without a DB connection.
	if tve.db == nil && tve.testRules == nil && tve.testTriggers == nil {
		return fmt.Errorf("field change: db not configured")
	}

	// 1. Quick field-level check (fast path)
	if err := tve.ValidateField(ctx, tenantID, entity, fieldName, newValue); err != nil {
		return fmt.Errorf("%s change validation failed: %w", fieldName, err)
	}

	// 2. Fetch and run field_change triggers for this entity
	//    (these can check cross-field logic, e.g., "if total < 0, customer must be VIP")
	data := record
	if data == nil {
		data = make(map[string]interface{})
	}
	data[fieldName] = newValue

	triggers, err := tve.fetchTriggers(ctx, tenantID.String(), string(TriggerTypeFieldChange), entity, "")
	if err != nil {
		return fmt.Errorf("fetch field_change triggers: %w", err)
	}

	for _, t := range triggers {
		for _, rid := range t.RuleIDs {
			rule, err := tve.fetchRuleByID(ctx, rid)
			if err != nil {
				tve.logger.Warn("DispatchFieldChange: rule not found", "rule_id", rid)
				continue
			}

			// Unmarshal condition
			var condition map[string]interface{}
			if err := json.Unmarshal(rule.ConditionJSON, &condition); err != nil {
				continue
			}

			// Execute rule
			result := tve.Execute(ExecutionContext{
				RuleID:       rid,
				RuleType:     rule.RuleType,
				TargetEntity: entity,
				Condition:    condition,
				Data:         data,
			})

			if !result.Passed {
				msg := rule.ErrorMessage
				if msg == "" {
					msg = result.Message
				}
				return fmt.Errorf("%s: %s", rule.RuleName, msg)
			}
		}
	}

	return nil
}

// DispatchSubEntityChange fires sub_entity_change triggers when a child/nested
// entity is modified. Used for line item changes in orders, positions in trades, etc.
//
// Arguments:
//   - parentEntity: parent entity type (e.g., "orders")
//   - parentID: ID of parent record (e.g., order UUID)
//   - childEntity: child entity type (e.g., "order_items")
//   - childData: the child record data being inserted/updated
//
// Example:
//
//	// User adds a new line item to an order
//	if err := engine.DispatchSubEntityChange(ctx, tenantID, "orders", orderID, "order_items",
//	    lineItemData); err != nil {
//	    return fmt.Errorf("line item validation failed: %w", err)
//	}
func (tve *TriggerValidationEngine) DispatchSubEntityChange(
	ctx context.Context,
	tenantID uuid.UUID,
	parentEntity string,
	parentID uuid.UUID,
	childEntity string,
	childData map[string]interface{},
) error {
	// Allow test-only in-memory overrides to be used without a DB connection.
	if tve.db == nil && tve.testRules == nil && tve.testTriggers == nil {
		return fmt.Errorf("sub_entity change: db not configured")
	}

	// Dispatch sub_entity_change trigger with composite key
	data := childData
	if data == nil {
		data = make(map[string]interface{})
	}
	data["_parent_id"] = parentID.String()
	data["_parent_entity"] = parentEntity
	data["_child_entity"] = childEntity

	return tve.TriggerValidate(ctx, tenantID, string(TriggerTypeSubEntityChange), childEntity, "", data)
}

// DispatchStatusChange fires status_change triggers when the status field is updated.
// Used to validate state transitions (e.g., "pending" -> "approved" -> "completed").
//
// Arguments:
//   - statusField: name of the status field (e.g., "order_status", "approval_status")
//   - oldStatus: previous status value
//   - newStatus: new status value
//   - record: full record for context
//
// Example:
//
//	// Approver changes order status from "pending" to "approved"
//	if err := engine.DispatchStatusChange(ctx, tenantID, "orders", "order_status",
//	    "pending", "approved", orderRecord); err != nil {
//	    return fmt.Errorf("status transition not allowed: %w", err)
//	}
func (tve *TriggerValidationEngine) DispatchStatusChange(
	ctx context.Context,
	tenantID uuid.UUID,
	entity string,
	statusField string,
	oldStatus string,
	newStatus string,
	record map[string]interface{},
) error {
	// Allow test-only in-memory overrides to be used without a DB connection.
	if tve.db == nil && tve.testRules == nil && tve.testTriggers == nil {
		return fmt.Errorf("status change: db not configured")
	}

	data := record
	if data == nil {
		data = make(map[string]interface{})
	}
	data[statusField] = newStatus
	data["_old_status"] = oldStatus
	data["_status_field"] = statusField

	triggers, err := tve.fetchTriggers(ctx, tenantID.String(), string(TriggerTypeStatusChange), entity, "")
	if err != nil {
		return fmt.Errorf("fetch status_change triggers: %w", err)
	}

	for _, t := range triggers {
		for _, rid := range t.RuleIDs {
			rule, err := tve.fetchRuleByID(ctx, rid)
			if err != nil {
				tve.logger.Warn("DispatchStatusChange: rule not found", "rule_id", rid)
				continue
			}

			var condition map[string]interface{}
			if err := json.Unmarshal(rule.ConditionJSON, &condition); err != nil {
				continue
			}

			result := tve.Execute(ExecutionContext{
				RuleID:       rid,
				RuleType:     rule.RuleType,
				TargetEntity: entity,
				Condition:    condition,
				Data:         data,
			})

			if !result.Passed {
				msg := rule.ErrorMessage
				if msg == "" {
					msg = result.Message
				}
				return fmt.Errorf("%s: %s", rule.RuleName, msg)
			}
		}
	}

	return nil
}

// DispatchRelationshipChange fires relationship_change triggers when a foreign
// key or link is modified. Used for cascading validations when entities are linked.
//
// Arguments:
//   - entity: entity being modified
//   - relationshipName: name of the relationship (e.g., "customer_id", "assigned_to")
//   - oldRelatedID: previous related entity ID
//   - newRelatedID: new related entity ID
//   - record: full record for context
//
// Example:
//
//	// User reassigns order to different customer
//	if err := engine.DispatchRelationshipChange(ctx, tenantID, "orders", "customer_id",
//	    oldCustomerID, newCustomerID, orderRecord); err != nil {
//	    return fmt.Errorf("cannot change customer: %w", err)
//	}
func (tve *TriggerValidationEngine) DispatchRelationshipChange(
	ctx context.Context,
	tenantID uuid.UUID,
	entity string,
	relationshipName string,
	oldRelatedID string,
	newRelatedID string,
	record map[string]interface{},
) error {
	// Allow test-only in-memory overrides to be used without a DB connection.
	if tve.db == nil && tve.testRules == nil && tve.testTriggers == nil {
		return fmt.Errorf("relationship change: db not configured")
	}

	data := record
	if data == nil {
		data = make(map[string]interface{})
	}
	data[relationshipName] = newRelatedID
	data["_old_related_id"] = oldRelatedID
	data["_relationship"] = relationshipName

	triggers, err := tve.fetchTriggers(ctx, tenantID.String(), string(TriggerTypeRelationshipChange), entity, "")
	if err != nil {
		return fmt.Errorf("fetch relationship_change triggers: %w", err)
	}

	for _, t := range triggers {
		for _, rid := range t.RuleIDs {
			rule, err := tve.fetchRuleByID(ctx, rid)
			if err != nil {
				tve.logger.Warn("DispatchRelationshipChange: rule not found", "rule_id", rid)
				continue
			}

			var condition map[string]interface{}
			if err := json.Unmarshal(rule.ConditionJSON, &condition); err != nil {
				continue
			}

			result := tve.Execute(ExecutionContext{
				RuleID:       rid,
				RuleType:     rule.RuleType,
				TargetEntity: entity,
				Condition:    condition,
				Data:         data,
			})

			if !result.Passed {
				msg := rule.ErrorMessage
				if msg == "" {
					msg = result.Message
				}
				return fmt.Errorf("%s: %s", rule.RuleName, msg)
			}
		}
	}

	return nil
}

// ============================================================================
// TRIGGER SUMMARY & COVERAGE
// ============================================================================
// Trigger types implemented in this dispatch system:
//
// ✅ 1. Save           (DispatchTrigger with TriggerTypeSave)
// ✅ 2. Field Change   (DispatchFieldChange)
// ✅ 3. Delete         (DispatchTrigger with TriggerTypeDelete)
// ✅ 4. Create         (DispatchTrigger with TriggerTypeCreate)
// ✅ 5. Workflow Step  (DispatchTriggerWithStep)
// ✅ 6. Sub-Entity     (DispatchSubEntityChange)
// ✅ 7. Relationship   (DispatchRelationshipChange)
// ✅ 8. Status Change  (DispatchStatusChange)
// 🔄 9. Bulk Load      (Future - batch validation)
// 🔄 10. Integration   (Future - external event triggers)
// 🔄 11. Time-Based    (Future - scheduled/cron)
// 🔄 12. Calculated    (Future - formula recalc)
// 🔄 13. Security Role (Future - role assignment)
//
// Current: 8/13 (62%) ✅
// ============================================================================
