package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// This test validates that the preview and link endpoints are wired and call the DB.
// We do not assert SQL, just ensure handler paths return 200/400 as expected.
func TestAlphaLinkHandler_PreviewAndLink_Wiring(t *testing.T) {

	dbSQL, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer dbSQL.Close()
	dbx := sqlx.NewDb(dbSQL, "sqlmock")

	h := NewAlphaLinkHandler(dbx)
	r := chi.NewRouter()
	h.RegisterRoutes(r)

	tenant := uuid.New()

	// Preview: expect a simple SELECT and return empty rows
	mock.ExpectQuery("SELECT ").WillReturnRows(sqlmock.NewRows([]string{"id", "node_type_id", "qualified_path", "candidate_core_id", "gold_tenant_id"}))

	req := httptest.NewRequest(http.MethodGet, "/api/tenants/"+tenant.String()+"/catalog/link-alpha/preview", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("preview status = %d, body=%s", w.Code, w.Body.String())
	}

	// Link: expect an UPDATE and no error
	mock.ExpectExec("UPDATE public.catalog_node").WillReturnResult(sqlmock.NewResult(0, 0))
	req2 := httptest.NewRequest(http.MethodPost, "/api/tenants/"+tenant.String()+"/catalog/link-alpha", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("link status = %d, body=%s", w2.Code, w2.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("mock expectations: %v", err)
	}
}
