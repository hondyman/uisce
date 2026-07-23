package validation

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

func TestTriggerValidate_Pass(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create mock: %v", err)
	}
	defer db.Close()

	engine := NewTriggerValidationEngine(db, &SimpleLogger{})
	ctx := context.Background()
	tenantID := uuid.New()

	// Mock fetchTriggers query
	mock.ExpectQuery("SELECT id, tenant_id, trigger_type, target_entity, step_name, rule_ids").
		WithArgs(tenantID.String(), "save", "orders", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "trigger_type", "target_entity", "step_name", "rule_ids", "meta"}).
			AddRow("trigger-1", tenantID.String(), "save", "orders", nil, pq.Array([]string{"rule-1"}), "{}"))

	// Mock fetchRuleByID query
	ruleCondition := map[string]interface{}{
		"field":    "total",
		"operator": ">",
		"value":    0,
	}
	conditionJSON, _ := json.Marshal(ruleCondition)

	mock.ExpectQuery("SELECT id, tenant_id, rule_name, rule_type, target_entities, condition_json, error_message,\\s*core_rule_id, inherit_mode, core_version_pin").
		WithArgs("rule-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "rule_name", "rule_type", "target_entities", "condition_json", "error_message", "core_rule_id", "inherit_mode", "core_version_pin"}).
			AddRow("rule-1", tenantID.String(), "OrderTotalPositive", "cardinality", pq.Array([]string{"orders"}), conditionJSON, "Total must be positive", nil, nil, nil))

	// Call TriggerValidate with valid data
	data := map[string]interface{}{
		"total":       100,
		"customer_id": 1,
	}

	err = engine.TriggerValidate(ctx, tenantID, "save", "orders", "", data)
	if err != nil {
		t.Errorf("TriggerValidate failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("mock expectations not met: %v", err)
	}
}

func TestTriggerValidate_Fail(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create mock: %v", err)
	}
	defer db.Close()

	engine := NewTriggerValidationEngine(db, &SimpleLogger{})
	ctx := context.Background()
	tenantID := uuid.New()

	// Mock fetchTriggers query
	mock.ExpectQuery("SELECT id, tenant_id, trigger_type, target_entity, step_name, rule_ids").
		WithArgs(tenantID.String(), "save", "orders", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "trigger_type", "target_entity", "step_name", "rule_ids", "meta"}).
			AddRow("trigger-1", tenantID.String(), "save", "orders", nil, pq.Array([]string{"rule-1"}), "{}"))

	// Mock fetchRuleByID query - cardinality rule that should fail
	ruleCondition := map[string]interface{}{
		"field":    "total",
		"operator": ">",
		"value":    0,
	}
	conditionJSON, _ := json.Marshal(ruleCondition)

	mock.ExpectQuery("SELECT id, tenant_id, rule_name, rule_type, target_entities, condition_json, error_message,\\s*core_rule_id, inherit_mode, core_version_pin").
		WithArgs("rule-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "rule_name", "rule_type", "target_entities", "condition_json", "error_message", "core_rule_id", "inherit_mode", "core_version_pin"}).
			AddRow("rule-1", tenantID.String(), "OrderTotalPositive", "cardinality", pq.Array([]string{"orders"}), conditionJSON, "Total must be positive", nil, nil, nil))

	// Call TriggerValidate with invalid data (total <= 0)
	data := map[string]interface{}{
		"total":       -50,
		"customer_id": 1,
	}

	err = engine.TriggerValidate(ctx, tenantID, "save", "orders", "", data)
	if err == nil {
		t.Errorf("TriggerValidate should have failed but did not")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("mock expectations not met: %v", err)
	}
}

func TestValidateField_Pass(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create mock: %v", err)
	}
	defer db.Close()

	engine := NewTriggerValidationEngine(db, &SimpleLogger{})
	ctx := context.Background()
	tenantID := uuid.New()

	// Mock query for field_format rules
	ruleCondition := map[string]interface{}{
		"field":   "phone",
		"pattern": `^\+?1?\d{9,15}$`,
	}
	conditionJSON, _ := json.Marshal(ruleCondition)

	mock.ExpectQuery("SELECT id, tenant_id, rule_name, rule_type, target_entities, condition_json, error_message").
		WithArgs(tenantID.String(), "customers", "phone").
		WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "rule_name", "rule_type", "target_entities", "condition_json", "error_message"}).
			AddRow("rule-1", tenantID.String(), "PhoneFormat", "field_format", pq.Array([]string{"customers"}), conditionJSON, "Phone must be valid E.164"))

	err = engine.ValidateField(ctx, tenantID, "customers", "phone", "+15551234567")
	if err != nil {
		t.Errorf("ValidateField failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("mock expectations not met: %v", err)
	}
}

func TestValidateField_Fail(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create mock: %v", err)
	}
	defer db.Close()

	engine := NewTriggerValidationEngine(db, &SimpleLogger{})
	ctx := context.Background()
	tenantID := uuid.New()

	// Mock query for field_format rules
	ruleCondition := map[string]interface{}{
		"field":   "phone",
		"pattern": `^\+?1?\d{9,15}$`,
	}
	conditionJSON, _ := json.Marshal(ruleCondition)

	mock.ExpectQuery("SELECT id, tenant_id, rule_name, rule_type, target_entities, condition_json, error_message").
		WithArgs(tenantID.String(), "customers", "phone").
		WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "rule_name", "rule_type", "target_entities", "condition_json", "error_message"}).
			AddRow("rule-1", tenantID.String(), "PhoneFormat", "field_format", pq.Array([]string{"customers"}), conditionJSON, "Phone must be valid E.164"))

	err = engine.ValidateField(ctx, tenantID, "customers", "phone", "abc123")
	if err == nil {
		t.Errorf("ValidateField should have failed but did not")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("mock expectations not met: %v", err)
	}
}
