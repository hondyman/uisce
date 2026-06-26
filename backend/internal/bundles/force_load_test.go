package bundles

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
)

func TestForceLoadGuardrailsHandler_DB(t *testing.T) {
	// setup sqlmock
	dbSQL, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer dbSQL.Close()
	dbx := sqlx.NewDb(dbSQL, "postgres")

	// expected rows
	cols := []string{"type", "data"}
	rowData1, _ := json.Marshal(map[string]interface{}{"pairs": [][]string{{"a", "b"}}})
	rowData2, _ := json.Marshal(map[string]interface{}{"claims": []string{"c"}})
	mockRows := sqlmock.NewRows(cols).
		AddRow("sod", rowData1).
		AddRow("certified", rowData2)
	mock.ExpectQuery("SELECT type, data FROM guardrail_rules").WillReturnRows(mockRows)

	// inject initDBFunc to return our mock
	initDBFunc = func(_ string) (*sqlx.DB, error) { return dbx, nil }
	defer func() { initDBFunc = InitDBFromConfig }()

	// create router
	r := chi.NewRouter()
	r.Get("/admin/guardrails/load", AdminAuthMiddleware()(http.HandlerFunc(ForceLoadGuardrailsHandler)).ServeHTTP)
	// Alternatively, if RegisterRoutes exists and handles this:
	// h := &Handler{...}
	// h.RegisterRoutes(r)
	// But let's stick to the manual setup if it's simpler for this test

	// call the endpoint
	req := httptest.NewRequest("GET", "/admin/guardrails/load?source=db", nil)
	req.Header.Set("X-User-Role", "admin")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// ensure expectations met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock expectations not met: %v", err)
	}
}
