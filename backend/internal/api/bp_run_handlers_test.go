package api

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBPRunHandler_HandleListRuns(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handler := NewBPRunHandler(db, nil)
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	t.Run("list runs filtered by entity and entity_id", func(t *testing.T) {
		id := uuid.New()
		now := time.Now()

		rows := sqlmock.NewRows([]string{
			"id", "workflow_id", "run_id", "tenant_id", "process_id", "process_name",
			"trigger_type", "trigger_name", "entity", "entity_id", "status",
			"input_payload", "output_payload", "error_message",
			"started_at", "completed_at", "duration_ms", "created_at", "updated_at",
		}).AddRow(
			id, "wf-1", "run-1", "tenant-test", "proc-1", "Order Ingestion",
			"event", "order.created", "Order", "ORD-12345", "COMPLETED",
			[]byte(`{}`), []byte(`{"status":"approved"}`), sql.NullString{},
			now, &now, 1200, now, now,
		)

		mock.ExpectQuery(`SELECT (.+) FROM public\.bp_workflow_run WHERE tenant_id = \$1 AND entity = \$2 AND entity_id = \$3`).
			WithArgs("tenant-test", "Order", "ORD-12345", 50).
			WillReturnRows(rows)

		req := httptest.NewRequest(http.MethodGet, "/api/bp/runs?entity=Order&entity_id=ORD-12345", nil)
		req.Header.Set("X-Tenant-ID", "tenant-test")
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "ORD-12345")
		assert.Contains(t, rec.Body.String(), "COMPLETED")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("missing tenant returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/bp/runs", nil)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

func TestBPRunHandler_HandleGetRun(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handler := NewBPRunHandler(db, nil)
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	t.Run("get run by run_id success", func(t *testing.T) {
		id := uuid.New()
		now := time.Now()

		rows := sqlmock.NewRows([]string{
			"id", "workflow_id", "run_id", "tenant_id", "process_id", "process_name",
			"trigger_type", "trigger_name", "entity", "entity_id", "status",
			"input_payload", "output_payload", "error_message",
			"started_at", "completed_at", "duration_ms", "created_at", "updated_at",
		}).AddRow(
			id, "wf-1", "run-abc-123", "tenant-test", "proc-1", "Order Ingestion",
			"manual", "api.run", "Order", "ORD-12345", "RUNNING",
			[]byte(`{}`), []byte(nil), sql.NullString{},
			now, nil, nil, now, now,
		)

		mock.ExpectQuery(`SELECT (.+) FROM public\.bp_workflow_run WHERE tenant_id = \$1 AND \(run_id = \$2 OR workflow_id = \$2 OR id::text = \$2\)`).
			WithArgs("tenant-test", "run-abc-123").
			WillReturnRows(rows)

		req := httptest.NewRequest(http.MethodGet, "/api/bp/runs/run-abc-123", nil)
		req.Header.Set("X-Tenant-ID", "tenant-test")
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "run-abc-123")
		assert.Contains(t, rec.Body.String(), "RUNNING")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
