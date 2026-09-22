package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

func TestHandleValidateField_Pass(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create mock: %v", err)
	}
	defer db.Close()

	handler := NewValidationTriggersHandler(db, nil)

	// Mock the field query
	ruleCondition := map[string]interface{}{
		"field":    "total",
		"operator": ">",
		"value":    0,
	}
	conditionJSON, _ := json.Marshal(ruleCondition)

	mock.ExpectQuery("SELECT id, tenant_id, rule_name, rule_type, target_entities, condition_json, error_message").
		WithArgs(sqlmock.AnyArg(), "orders", "total").
		WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "rule_name", "rule_type", "target_entities", "condition_json", "error_message"}).
			AddRow("rule-1", "tenant-1", "OrderTotal", "cardinality", pq.Array([]string{"orders"}), conditionJSON, "Total must be positive"))

	payload := map[string]interface{}{
		"entity": "orders",
		"field":  "total",
		"value":  100,
		"record": map[string]interface{}{},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/validate/field", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "910638ba-a459-4a3f-bb2d-78391b0595f6")
	req = withAuthClaims(req, "user1", "910638ba-a459-4a3f-bb2d-78391b0595f6") // tenant comes from the authenticated claims, not the header

	w := httptest.NewRecorder()
	handler.HandleValidateField(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "pass" {
		t.Errorf("expected status 'pass', got %v", resp["status"])
	}
}

func TestHandleValidateField_Fail(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create mock: %v", err)
	}
	defer db.Close()

	handler := NewValidationTriggersHandler(db, nil)

	// Mock the field query
	ruleCondition := map[string]interface{}{
		"field":    "total",
		"operator": ">",
		"value":    0,
	}
	conditionJSON, _ := json.Marshal(ruleCondition)

	mock.ExpectQuery("SELECT id, tenant_id, rule_name, rule_type, target_entities, condition_json, error_message").
		WithArgs(sqlmock.AnyArg(), "orders", "total").
		WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "rule_name", "rule_type", "target_entities", "condition_json", "error_message"}).
			AddRow("rule-1", "tenant-1", "OrderTotal", "cardinality", pq.Array([]string{"orders"}), conditionJSON, "Total must be positive"))

	payload := map[string]interface{}{
		"entity": "orders",
		"field":  "total",
		"value":  -50,
		"record": map[string]interface{}{},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/validate/field", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "910638ba-a459-4a3f-bb2d-78391b0595f6")
	req = withAuthClaims(req, "user1", "910638ba-a459-4a3f-bb2d-78391b0595f6") // tenant comes from the authenticated claims, not the header

	w := httptest.NewRecorder()
	handler.HandleValidateField(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] == nil {
		t.Errorf("expected error in response")
	}
}

func TestHandleValidateField_MissingHeaders(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create mock: %v", err)
	}
	defer db.Close()

	handler := NewValidationTriggersHandler(db, nil)

	payload := map[string]interface{}{
		"entity": "orders",
		"field":  "total",
		"value":  100,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/validate/field", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// Missing X-Tenant-ID
	req = withAuthClaims(req, "user1", "") // authenticated, but the claims carry no tenant

	w := httptest.NewRecorder()
	handler.HandleValidateField(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 when tenant ID missing, got %d", w.Code)
	}
}

// A request with no authenticated claims must be rejected outright. The tenant is taken from
// the claims, never from the X-Tenant-ID header, so a header alone cannot select a tenant.
func TestHandleValidateField_Unauthenticated(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	handler := NewValidationTriggersHandler(db, nil)

	body, _ := json.Marshal(map[string]interface{}{"entity": "orders", "field": "total", "value": 100})
	req := httptest.NewRequest("POST", "/api/validate/field", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "910638ba-a459-4a3f-bb2d-78391b0595f6") // header only: must not authorise

	w := httptest.NewRecorder()
	handler.HandleValidateField(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without claims even with an X-Tenant-ID header, got %d", w.Code)
	}
}

func TestTriggerValidate_Integration(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create mock: %v", err)
	}
	defer db.Close()

	handler := NewValidationTriggersHandler(db, nil)
	ctx := context.Background()
	tenantID := uuid.New()

	// Mock fetchTriggers
	mock.ExpectQuery("SELECT id, tenant_id, trigger_type, target_entity, step_name, rule_ids").
		WithArgs(tenantID.String(), "create", "orders", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "trigger_type", "target_entity", "step_name", "rule_ids", "meta"}).
			AddRow("trigger-1", tenantID.String(), "create", "orders", nil, pq.Array([]string{"rule-1"}), "{}"))

	// Mock fetchRuleByID
	ruleCondition := map[string]interface{}{
		"field":    "customer_id",
		"operator": ">",
		"value":    0,
	}
	conditionJSON, _ := json.Marshal(ruleCondition)

	mock.ExpectQuery("SELECT id, tenant_id, rule_name, rule_type, target_entities, condition_json, error_message, core_rule_id, inherit_mode, core_version_pin").
		WithArgs("rule-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "rule_name", "rule_type", "target_entities", "condition_json", "error_message", "core_rule_id", "inherit_mode", "core_version_pin"}).
			AddRow("rule-1", tenantID.String(), "OrderTotal", "cardinality", pq.Array([]string{"orders"}), conditionJSON, "Total must be positive", nil, nil, nil))

	data := map[string]interface{}{
		"customer_id": 1,
		"total":       100,
	}

	err = handler.TriggerValidate(ctx, tenantID, "create", "orders", "", data)
	if err != nil {
		t.Errorf("TriggerValidate failed: %v", err)
	}
}
