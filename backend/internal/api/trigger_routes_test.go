package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
)

func TestListWorkflowEvents_MissingTenant(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")

	r := chi.NewRouter()
	RegisterTriggerRoutesChi(r, sqlxDB, nil)

	req := httptest.NewRequest("GET", "/api/v1/triggers/events", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when tenant missing, got %d", w.Code)
	}
}

func TestListWorkflowEvents_WithTenantReturnsList(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"id", "key", "label", "description", "event_type", "config", "created_at", "updated_at"}).
		AddRow("id-1", "key1", "Label 1", "desc", "on_submit", "{}", time.Now(), time.Now())

	mock.ExpectQuery("SELECT id, key, label, description, event_type, config, created_at, updated_at").
		WithArgs("tenant-123").WillReturnRows(rows)

	sqlxDB := sqlx.NewDb(db, "postgres")

	r := chi.NewRouter()
	RegisterTriggerRoutesChi(r, sqlxDB, nil)

	req := httptest.NewRequest("GET", "/api/v1/triggers/events", nil)
	req.Header.Set("X-Tenant-ID", "tenant-123")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body=%s", w.Code, w.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock expectations not met: %v", err)
	}
}
