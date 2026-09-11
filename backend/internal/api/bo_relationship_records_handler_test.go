package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

func newRelationshipTestRouter(handler *BOCRUDHandler) chi.Router {
	r := chi.NewRouter()
	r.Route("/api/v1/bo", func(r chi.Router) {
		r.Get("/{boKey}/records/{recordId}/relationships/{relKey}", handler.HandleListRelatedRecords)
		r.Post("/{boKey}/records/{recordId}/relationships/{relKey}", handler.HandleCreateRelatedRecord)
		r.Put("/{boKey}/records/{recordId}/relationships/{relKey}/{childId}", handler.HandleUpdateRelatedRecord)
		r.Delete("/{boKey}/records/{recordId}/relationships/{relKey}/{childId}", handler.HandleDeleteRelatedRecord)
	})
	return r
}

func TestHandleListRelatedRecords_ResolvesJoinAndFiltersByParentId(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	mock.MatchExpectationsInOrder(false)
	handler := NewBOCRUDHandler(sqlxDB, nil)
	r := newRelationshipTestRouter(handler)

	// resolveBusinessObjectID
	mock.ExpectQuery("SELECT id::text FROM business_objects").
		WithArgs(sqlmock.AnyArg(), "account").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("bo-account"))

	// resolveRelationship
	mock.ExpectQuery("FROM business_object_relationships").
		WithArgs(sqlmock.AnyArg(), "allocations", "bo-account").
		WillReturnRows(sqlmock.NewRows([]string{"id", "from_bo_id", "to_bo_id", "rel_key"}).
			AddRow("rel-1", "bo-tradeorder", "bo-account", "allocations"))

	// resolveBOKeyByID for the child (from_bo_id side, since root is the to_bo_id side here)
	mock.ExpectQuery("SELECT bo_key FROM business_objects").
		WithArgs("bo-tradeorder", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"bo_key"}).AddRow("trade_order"))

	// resolveBOMetadata for the child BO (trade_order)
	mock.ExpectQuery("SELECT COALESCE.*FROM public.business_objects").
		WithArgs("trade_order", sqlmock.AnyArg()).
		WillReturnError(sqlmock.ErrCancelled)
	mock.ExpectQuery("SELECT COALESCE.*FROM public.catalog_node").
		WithArgs("trade_order", sqlmock.AnyArg()).
		WillReturnError(sqlmock.ErrCancelled)
	mock.ExpectQuery("SELECT EXISTS.*information_schema.tables").
		WithArgs("oms", "trade_order").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	// resolveChildFKColumn: no relationship_bindings row, falls back to naming convention probe
	mock.ExpectQuery("SELECT rb.join_condition_sql").
		WithArgs(sqlmock.AnyArg(), "rel-1", "bo-tradeorder").
		WillReturnError(sqlmock.ErrCancelled)
	mock.ExpectQuery("SELECT EXISTS.*information_schema.columns").
		WithArgs("oms", "trade_order", "account_id").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	// The actual list query, scoped by the resolved FK column
	mock.ExpectQuery(`SELECT \* FROM oms.trade_order\s+WHERE tenant_id = \$1 AND account_id = \$2`).
		WithArgs(sqlmock.AnyArg(), "acc-123", 50, 0).
		WillReturnRows(sqlmock.NewRows([]string{"id", "account_id"}).AddRow("to-1", "acc-123"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/bo/account/records/acc-123/relationships/allocations", nil)
	req.Header.Set("X-Tenant-ID", "00000000-0000-0000-0000-000000000001")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var resp map[string]interface{}
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, float64(1), resp["count"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandleCreateRelatedRecord_ForcesParentFKOverridingClientPayload(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	mock.MatchExpectationsInOrder(false)
	handler := NewBOCRUDHandler(sqlxDB, nil)
	r := newRelationshipTestRouter(handler)

	mock.ExpectQuery("SELECT id::text FROM business_objects").
		WithArgs(sqlmock.AnyArg(), "account").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("bo-account"))

	mock.ExpectQuery("FROM business_object_relationships").
		WithArgs(sqlmock.AnyArg(), "allocations", "bo-account").
		WillReturnRows(sqlmock.NewRows([]string{"id", "from_bo_id", "to_bo_id", "rel_key"}).
			AddRow("rel-1", "bo-tradeorder", "bo-account", "allocations"))

	mock.ExpectQuery("SELECT bo_key FROM business_objects").
		WithArgs("bo-tradeorder", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"bo_key"}).AddRow("trade_order"))

	mock.ExpectQuery("SELECT COALESCE.*FROM public.business_objects").
		WithArgs("trade_order", sqlmock.AnyArg()).
		WillReturnError(sqlmock.ErrCancelled)
	mock.ExpectQuery("SELECT COALESCE.*FROM public.catalog_node").
		WithArgs("trade_order", sqlmock.AnyArg()).
		WillReturnError(sqlmock.ErrCancelled)
	mock.ExpectQuery("SELECT EXISTS.*information_schema.tables").
		WithArgs("oms", "trade_order").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	mock.ExpectQuery("SELECT rb.join_condition_sql").
		WithArgs(sqlmock.AnyArg(), "rel-1", "bo-tradeorder").
		WillReturnError(sqlmock.ErrCancelled)
	mock.ExpectQuery("SELECT EXISTS.*information_schema.columns").
		WithArgs("oms", "trade_order", "account_id").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	// The client tries to set account_id to a different (unrelated) account — this must be
	// overridden server-side to "acc-123" (the parent from the URL) regardless of payload content.
	// Column order in the generated INSERT depends on Go map iteration order, so match any args
	// here and assert the forced value via the returned row instead.
	mock.ExpectQuery("INSERT INTO oms.trade_order").
		WillReturnRows(sqlmock.NewRows([]string{"id", "account_id"}).AddRow("to-1", "acc-123"))

	payload := map[string]interface{}{"account_id": "someone-elses-account", "notional": 1000}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/bo/account/records/acc-123/relationships/allocations", bytes.NewBuffer(body))
	req.Header.Set("X-Tenant-ID", "00000000-0000-0000-0000-000000000001")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var resp map[string]interface{}
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "acc-123", resp["account_id"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestResolveRelationship_UnknownRelKey_Returns404(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	mock.MatchExpectationsInOrder(false)
	handler := NewBOCRUDHandler(sqlxDB, nil)
	r := newRelationshipTestRouter(handler)

	mock.ExpectQuery("SELECT id::text FROM business_objects").
		WithArgs(sqlmock.AnyArg(), "account").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("bo-account"))

	mock.ExpectQuery("FROM business_object_relationships").
		WithArgs(sqlmock.AnyArg(), "does_not_exist", "bo-account").
		WillReturnError(sql.ErrNoRows)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/bo/account/records/acc-123/relationships/does_not_exist", nil)
	req.Header.Set("X-Tenant-ID", "00000000-0000-0000-0000-000000000001")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestParseChildFKColumnFromJoinCondition_VariousFormats(t *testing.T) {
	cases := []struct {
		name     string
		joinSQL  string
		table    string
		expected string
	}{
		{"child on right", "account.id = trade_order.account_id", "oms.trade_order", "account_id"},
		{"child on left", "trade_order.account_id = account.id", "oms.trade_order", "account_id"},
		{"no schema prefix", "trade_order.account_id = account.id", "trade_order", "account_id"},
		{"malformed", "not a join condition", "oms.trade_order", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseChildFKColumnFromJoinCondition(tc.joinSQL, tc.table)
			assert.Equal(t, tc.expected, got)
		})
	}
}
