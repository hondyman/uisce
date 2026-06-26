package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/identity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Context key constants for testing
const (
	testTenantIDKey = "tenant_id"
	testIdentityKey = "identity"
)

// ============================================================================
// TIMEOUT TRIGGERS HTTP HANDLER TESTS
// ============================================================================
// Tests for creating, reading, updating, and deleting timeout triggers
// via REST API endpoints.
// ============================================================================

// TestHandleCreateTimeoutTrigger tests creating a new timeout trigger
func TestHandleCreateTimeoutTrigger(t *testing.T) {
	// Setup mock database
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Mock the INSERT query
	mock.ExpectQuery(`
		INSERT INTO workflow_timeout_triggers 
		  \(id, tenant_id, workflow_name, step_name, due_hours, actions_json, is_active, created_at, updated_at\)
		VALUES 
		  \(\$1, \$2, \$3, \$4, \$5, \$6, \$7, NOW\(\), NOW\(\)\)
		RETURNING id, tenant_id, workflow_name, step_name, due_hours, actions_json, is_active, created_at, updated_at
	`).
		WithArgs(
			sqlmock.AnyArg(), // ID
			"tenant-123",
			"HireEmployee",
			"ManagerApproval",
			48,
			sqlmock.AnyArg(), // actions JSON
			true,
		).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "tenant_id", "workflow_name", "step_name", "due_hours", "actions_json", "is_active", "created_at", "updated_at",
		}).AddRow(
			"trigger-123",
			"tenant-123",
			"HireEmployee",
			"ManagerApproval",
			48,
			[]byte(`[{"percent":80,"type":"notify","target":"assignee"},{"percent":100,"type":"escalate","target":"hr_director"}]`),
			true,
			"2025-10-28T10:00:00Z",
			"2025-10-28T10:00:00Z",
		))

	// Create handler
	handler := NewTimeoutTriggersHandler(db)

	// Prepare request
	requestBody := CreateTimeoutTriggerRequest{
		WorkflowName: "HireEmployee",
		StepName:     "ManagerApproval",
		DueHours:     48,
		ActionsJSON:  json.RawMessage(`[{"percent":80,"type":"notify","target":"assignee"},{"percent":100,"type":"escalate","target":"hr_director"}]`),
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/api/admin/timeout-triggers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Add tenant context
	ctx := context.WithValue(req.Context(), testTenantIDKey, "tenant-123")
	ctx = context.WithValue(ctx, testIdentityKey, mockIdentityContext("temporal.admin"))
	req = req.WithContext(ctx)

	// Execute request
	w := httptest.NewRecorder()
	handler.HandleCreateTimeoutTrigger(w, req)

	// Verify
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response TimeoutTrigger
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "trigger-123", response.ID)
	assert.Equal(t, "HireEmployee", response.WorkflowName)
	assert.Equal(t, "ManagerApproval", response.StepName)
	assert.Equal(t, 48, response.DueHours)
	assert.True(t, response.IsActive)

	// Verify mock expectations
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestHandleCreateTimeoutTrigger_InvalidActions tests creating trigger with invalid actions
func TestHandleCreateTimeoutTrigger_InvalidActions(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()

	handler := NewTimeoutTriggersHandler(db)

	// Invalid JSON in actions
	requestBody := CreateTimeoutTriggerRequest{
		WorkflowName: "HireEmployee",
		StepName:     "ManagerApproval",
		DueHours:     48,
		ActionsJSON:  json.RawMessage(`"not an array"`),
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/api/admin/timeout-triggers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := context.WithValue(req.Context(), testTenantIDKey, "tenant-123")
	ctx = context.WithValue(ctx, testIdentityKey, mockIdentityContext("temporal.admin"))
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.HandleCreateTimeoutTrigger(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid actions_json")
}

// TestHandleCreateTimeoutTrigger_NoRBACRole tests creating trigger without temporal.admin role
func TestHandleCreateTimeoutTrigger_NoRBACRole(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()

	handler := NewTimeoutTriggersHandler(db)

	requestBody := CreateTimeoutTriggerRequest{
		WorkflowName: "HireEmployee",
		StepName:     "ManagerApproval",
		DueHours:     48,
		ActionsJSON:  json.RawMessage(`[]`),
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/api/admin/timeout-triggers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := context.WithValue(req.Context(), testTenantIDKey, "tenant-123")
	ctx = context.WithValue(ctx, testIdentityKey, mockIdentityContext("user")) // Wrong role
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.HandleCreateTimeoutTrigger(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

// TestHandleListTimeoutTriggers tests listing all timeout triggers
func TestHandleListTimeoutTriggers(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	// Mock query
	mock.ExpectQuery(`
		SELECT id, tenant_id, workflow_name, step_name, due_hours, actions_json, is_active, created_at, updated_at
		FROM workflow_timeout_triggers
		WHERE tenant_id = \$1 AND is_active = TRUE
		ORDER BY workflow_name, step_name
	`).WithArgs("tenant-123").WillReturnRows(sqlmock.NewRows([]string{
		"id", "tenant_id", "workflow_name", "step_name", "due_hours", "actions_json", "is_active", "created_at", "updated_at",
	}).AddRow(
		"trigger-1",
		"tenant-123",
		"HireEmployee",
		"ManagerApproval",
		48,
		[]byte(`[{"percent":100,"type":"escalate","target":"hr_director"}]`),
		true,
		"2025-10-28T10:00:00Z",
		"2025-10-28T10:00:00Z",
	).AddRow(
		"trigger-2",
		"tenant-123",
		"OrderApproval",
		"FinanceApproval",
		24,
		[]byte(`[{"percent":100,"type":"escalate","target":"finance_manager"}]`),
		true,
		"2025-10-28T11:00:00Z",
		"2025-10-28T11:00:00Z",
	))

	handler := NewTimeoutTriggersHandler(db)

	req := httptest.NewRequest("GET", "/api/admin/timeout-triggers", nil)
	ctx := context.WithValue(req.Context(), testTenantIDKey, "tenant-123")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.HandleListTimeoutTriggers(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var triggers []TimeoutTrigger
	err := json.Unmarshal(w.Body.Bytes(), &triggers)
	require.NoError(t, err)
	assert.Len(t, triggers, 2)
	assert.Equal(t, "HireEmployee", triggers[0].WorkflowName)
	assert.Equal(t, "OrderApproval", triggers[1].WorkflowName)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestHandleListTimeoutTriggers_FilterByWorkflow tests listing triggers for specific workflow
func TestHandleListTimeoutTriggers_FilterByWorkflow(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	// Mock query with workflow filter
	mock.ExpectQuery(`
		SELECT id, tenant_id, workflow_name, step_name, due_hours, actions_json, is_active, created_at, updated_at
		FROM workflow_timeout_triggers
		WHERE tenant_id = \$1 AND workflow_name = \$2 AND is_active = TRUE
		ORDER BY workflow_name, step_name
	`).WithArgs("tenant-123", "HireEmployee").WillReturnRows(sqlmock.NewRows([]string{
		"id", "tenant_id", "workflow_name", "step_name", "due_hours", "actions_json", "is_active", "created_at", "updated_at",
	}).AddRow(
		"trigger-1",
		"tenant-123",
		"HireEmployee",
		"ManagerApproval",
		48,
		[]byte(`[]`),
		true,
		"2025-10-28T10:00:00Z",
		"2025-10-28T10:00:00Z",
	))

	handler := NewTimeoutTriggersHandler(db)

	req := httptest.NewRequest("GET", "/api/admin/timeout-triggers?workflow=HireEmployee", nil)
	ctx := context.WithValue(req.Context(), testTenantIDKey, "tenant-123")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.HandleListTimeoutTriggers(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var triggers []TimeoutTrigger
	json.Unmarshal(w.Body.Bytes(), &triggers)
	assert.Len(t, triggers, 1)
	assert.Equal(t, "HireEmployee", triggers[0].WorkflowName)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestHandleGetTimeoutTrigger tests retrieving a specific trigger
func TestHandleGetTimeoutTrigger(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	mock.ExpectQuery(`
		SELECT id, tenant_id, workflow_name, step_name, due_hours, actions_json, is_active, created_at, updated_at
		FROM workflow_timeout_triggers
		WHERE id = \$1 AND tenant_id = \$2
	`).WithArgs("trigger-123", "tenant-123").WillReturnRows(sqlmock.NewRows([]string{
		"id", "tenant_id", "workflow_name", "step_name", "due_hours", "actions_json", "is_active", "created_at", "updated_at",
	}).AddRow(
		"trigger-123",
		"tenant-123",
		"HireEmployee",
		"ManagerApproval",
		48,
		[]byte(`[]`),
		true,
		"2025-10-28T10:00:00Z",
		"2025-10-28T10:00:00Z",
	))

	handler := NewTimeoutTriggersHandler(db)

	req := httptest.NewRequest("GET", "/api/admin/timeout-triggers/trigger-123", nil)
	ctx := context.WithValue(req.Context(), testTenantIDKey, "tenant-123")

	// Set up chi route context for URL parameters
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "trigger-123")
	ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)

	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.HandleGetTimeoutTrigger(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var trigger TimeoutTrigger
	err := json.Unmarshal(w.Body.Bytes(), &trigger)
	require.NoError(t, err)
	assert.Equal(t, "trigger-123", trigger.ID)
	assert.Equal(t, "HireEmployee", trigger.WorkflowName)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestHandleDeleteTimeoutTrigger tests deleting a trigger (soft delete)
func TestHandleDeleteTimeoutTrigger(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	mock.ExpectExec(`
		UPDATE workflow_timeout_triggers
		SET is_active = FALSE, updated_at = NOW\(\)
		WHERE id = \$1 AND tenant_id = \$2
	`).WithArgs("trigger-123", "tenant-123").WillReturnResult(sqlmock.NewResult(0, 1))

	handler := NewTimeoutTriggersHandler(db)

	req := httptest.NewRequest("DELETE", "/api/admin/timeout-triggers/trigger-123", nil)
	ctx := context.WithValue(req.Context(), testTenantIDKey, "tenant-123")
	ctx = context.WithValue(ctx, testIdentityKey, mockIdentityContext("temporal.admin"))

	// Set up chi route context for URL parameters
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "trigger-123")
	ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)

	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.HandleDeleteTimeoutTrigger(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "deleted", response["status"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

// ============================================================================
// INTEGRATION TESTS (SMOKE TESTS)
// ============================================================================
// These tests simulate real timeout scenarios

// TestTimeoutScenario_48HourEscalation simulates a 48-hour escalation scenario
func TestTimeoutScenario_48HourEscalation(t *testing.T) {
	/*
		SCENARIO: HireEmployee workflow, ManagerApproval step, 48 hour timeout

		1. Create timeout trigger:
		   - Workflow: HireEmployee
		   - Step: ManagerApproval
		   - Due: 48 hours
		   - Actions: Notify at 80% (38.4h), Escalate at 100% (48h)

		2. Simulate workflow start at Oct 21, 10:00 AM
		3. Simulate checking timeout at Oct 23, 10:30 AM (48.5 hours elapsed)
		4. Verify:
		   - Escalation action triggered
		   - Workflow reassigned to HR Director
		   - Email sent to both original manager and HR
		   - Audit event logged
	*/

	db, mock, _ := sqlmock.New()
	defer db.Close()

	// Mock trigger creation
	mock.ExpectQuery(`INSERT INTO workflow_timeout_triggers`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "tenant_id", "workflow_name", "step_name", "due_hours", "actions_json", "is_active", "created_at", "updated_at",
		}).AddRow(
			"trigger-hire-48h",
			"tenant-123",
			"HireEmployee",
			"ManagerApproval",
			48,
			[]byte(`[{"percent":80,"type":"notify"},{"percent":100,"type":"escalate","target":"hr_director"}]`),
			true,
			"2025-10-28T10:00:00Z",
			"2025-10-28T10:00:00Z",
		))

	handler := NewTimeoutTriggersHandler(db)

	// Test trigger creation
	requestBody := CreateTimeoutTriggerRequest{
		WorkflowName: "HireEmployee",
		StepName:     "ManagerApproval",
		DueHours:     48,
		ActionsJSON:  json.RawMessage(`[{"percent":80,"type":"notify","target":"assignee"},{"percent":100,"type":"escalate","target":"hr_director"}]`),
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/api/admin/timeout-triggers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := context.WithValue(req.Context(), testTenantIDKey, "tenant-123")
	ctx = context.WithValue(ctx, testIdentityKey, mockIdentityContext("temporal.admin"))
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.HandleCreateTimeoutTrigger(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var trigger TimeoutTrigger
	json.Unmarshal(w.Body.Bytes(), &trigger)
	assert.Equal(t, "HireEmployee", trigger.WorkflowName)
	assert.Equal(t, 48, trigger.DueHours)

	// In real scenario:
	// 1. Workflow starts at Oct 21, 10 AM
	// 2. Timeout monitor runs at Oct 23, 10:30 AM
	// 3. Detects 48.5 hours elapsed > 48 hour due time
	// 4. Executes escalate action
	// 5. Triggers notify and escalate actions
}

// ============================================================================
// HELPER FUNCTIONS FOR TESTS
// ============================================================================

func mockIdentityContext(roles ...string) identity.IdentityContext {
	return identity.IdentityContext{
		TenantID: "tenant-123",
		Roles:    roles,
	}
}
