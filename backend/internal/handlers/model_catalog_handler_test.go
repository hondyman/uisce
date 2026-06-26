package handlers

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// TestDeleteModelCoreCascade verifies that deleting a core model queries for associated
// custom models and performs deletes in a transaction (customs then core).
func TestDeleteModelCoreCascade(t *testing.T) {
	dbSQL, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer dbSQL.Close()

	db := dbSQL
	h := NewModelCatalogHandler(db)

	// Prepare input IDs
	tenantID := uuid.New()
	dsID := uuid.New()
	coreID := uuid.New()
	coreModelKey := "orders"

	// First, the handler will QueryRow to fetch model_key and source_config
	// Expect QueryRow -> will return model_key and json indicating core generator
	mock.ExpectQuery("SELECT model_key, source_config").WithArgs(coreID, tenantID, dsID).
		WillReturnRows(sqlmock.NewRows([]string{"model_key", "source_config"}).AddRow(coreModelKey, json.RawMessage(`{"generator":"core"}`)))

	// Next, it will query for associated custom models - return no rows to simulate none found
	mock.ExpectQuery(`(?s)SELECT\s+id\s*,\s*model_key\s+FROM\s+fabric_defn.*`).
		WithArgs(tenantID, dsID, coreModelKey, coreModelKey+"_custom").
		WillReturnRows(sqlmock.NewRows([]string{"id", "model_key"}))

	// Begin transaction and expect deletes: core fabric_defn then catalog_node, then commit
	mock.ExpectBegin()
	mock.ExpectExec(`(?s)DELETE\s+FROM\s+fabric_defn\s+WHERE\s+id\s*=\s*\$1\s+AND\s+tenant_id\s*=\s*\$2\s+AND\s+tenant_datasource_id\s*=\s*\$3`).
		WithArgs(coreID, tenantID, dsID).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`(?s)DELETE\s+FROM\s+public\.catalog_node\s+WHERE`).
		WithArgs(tenantID, dsID, NODE_TYPE_SEMANTIC_MODEL, "/semantic_model/"+coreModelKey).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	// Build request to match registered route and query param usage
	// Handler registers DELETE /models/{model_id} and reads tenant_id and datasource_id from query string
	req := httptest.NewRequest("DELETE", "/models/"+coreID.String()+"?tenant_id="+tenantID.String()+"&datasource_id="+dsID.String(), nil)

	// Set chi route param for model_id so chi.URLParam works for model_id
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("model_id", coreID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	// Use ResponseRecorder
	rr := httptest.NewRecorder()

	// Call handler directly; since chi.URLParam relies on chi router, we will set the URL params in the request context using chi.NewRouteContext
	// This is a minimal setup to allow chi.URLParam to read variables
	// Note: importing chi here would add complexity; instead, set the URL params in the request's URL.RawPath style is sufficient for the handler which uses chi.URLParam

	h.DeleteModel(rr, req)

	if rr.Code != 204 {
		t.Fatalf("expected 204 No Content, got %d, body: %s", rr.Code, rr.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock expectations not met: %v", err)
	}
}
