package api

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/identity"
	"github.com/hondyman/semlayer/backend/internal/services"
	"github.com/hondyman/semlayer/backend/internal/temporal"
)

func TestHandleListExecutions_ReturnsRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	// Expect the query used in ListExecutions
	rows := sqlmock.NewRows([]string{"id", "workflow_id", "status", "input", "result"}).
		AddRow("1", "wf-1", "running", `{"x":1}`, "null").
		AddRow("2", "wf-2", "completed", "null", `{"ok":true}`)

	mock.ExpectQuery(`SELECT id, workflow_id, status, input, result FROM temporal_workflows ORDER BY id DESC LIMIT \$1`).
		WithArgs(200).
		WillReturnRows(rows)

	// Create a real WorkflowAdminService pointing at sqlmock DB
	was := temporal.NewWorkflowAdminService(nil, "default", db, nil)

	// Security manager with core_admin allowed
	sec := services.NewSecurityManager(nil, nil, []byte("test"))

	handler := &TemporalAdminHandler{adminService: was, secMgr: sec}

	r := chi.NewRouter()
	r.Get("/api/temporal/executions", handler.HandleListExecutions)

	req := httptest.NewRequest("GET", "/api/temporal/executions", nil)
	// inject actor into context (core_admin bypass)
	req = req.WithContext(identity.WithActorTenant(req.Context(), "core_admin", "t1"))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var out []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(out))
	}

	// ensure sql expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock expectations: %v", err)
	}
}
