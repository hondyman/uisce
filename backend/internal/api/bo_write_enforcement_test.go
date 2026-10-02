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
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"

	"github.com/hondyman/uisce/backend/internal/metadata"
)

// txEnforcer stands in for metadata.BusinessObjectService: it runs the
// handler's write in a real (mocked) transaction and rejects the row
// indices listed in reject, the way the rule engine would.
type txEnforcer struct {
	db      *sqlx.DB
	reject  map[int]bool
	missing map[int]bool // rows that leave a required field empty
	boKeys  []string
}

func (e *txEnforcer) EnforceWrite(ctx context.Context, _ string, boKey string, do func(*sqlx.Tx) (map[string]interface{}, error)) (map[string]interface{}, error) {
	e.boKeys = append(e.boKeys, boKey)
	tx, err := e.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	rec, err := do(tx)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if e.reject[0] {
		_ = tx.Rollback()
		return nil, &metadata.RuleRejectionError{Rules: []string{"notional within limit"}}
	}
	return rec, tx.Commit()
}

func (e *txEnforcer) EnforceWriteBatch(ctx context.Context, _ string, boKey string, n int, dryRun bool, do func(*sqlx.Tx, int) (map[string]interface{}, error)) ([]metadata.BatchRowResult, error) {
	e.boKeys = append(e.boKeys, boKey)
	tx, err := e.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	out := make([]metadata.BatchRowResult, n)
	for i := 0; i < n; i++ {
		out[i].Index = i
		rec, err := do(tx, i)
		switch {
		case err != nil:
			out[i].Err = err
		case e.missing[i]:
			out[i].Err = &metadata.RequiredFieldsError{Fields: []string{"Issuer (issuer_id)"}}
		case e.reject[i]:
			out[i].Err = &metadata.RuleRejectionError{Rules: []string{"notional within limit"}}
		default:
			out[i].Record = rec
		}
	}
	if dryRun {
		return out, tx.Rollback()
	}
	return out, tx.Commit()
}

func expectCreateRelatedChain(mock sqlmock.Sqlmock) {
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
}

func postRelated(r http.Handler) *httptest.ResponseRecorder {
	body, _ := json.Marshal(map[string]interface{}{"notional": 5000000})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/bo/account/records/acc-123/relationships/allocations", bytes.NewBuffer(body))
	req.Header.Set("X-Tenant-ID", "00000000-0000-0000-0000-000000000001")
	req = withTestAuth(req, "00000000-0000-0000-0000-000000000001")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestBOWrite_RuleRejectionIs422AndRollsBack(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	mock.MatchExpectationsInOrder(false)
	r := newRelationshipTestRouter(NewBOCRUDHandler(sqlxDB, nil, &txEnforcer{db: sqlxDB, reject: map[int]bool{0: true}}))

	expectCreateRelatedChain(mock)
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO oms.trade_order").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("to-1"))
	mock.ExpectRollback() // never committed

	rec := postRelated(r)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
	var resp map[string]interface{}
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "9000-5", resp["error_code"])
	assert.Equal(t, map[string]interface{}{"rules": []interface{}{"notional within limit"}}, resp["details"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBOWrite_NoEnforcerRefusesInsteadOfBypassing(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	mock.MatchExpectationsInOrder(false)
	r := newRelationshipTestRouter(NewBOCRUDHandler(sqlxDB, nil, nil))

	expectCreateRelatedChain(mock)
	// No BEGIN and no INSERT expected: sqlmock fails the test if either runs.

	rec := postRelated(r)
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code, rec.Body.String())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBulkBORecords_EachRowJudgedFailuresAttributed(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	mock.MatchExpectationsInOrder(false)
	enf := &txEnforcer{db: sqlxDB, reject: map[int]bool{1: true}}
	h := NewBOCRUDHandler(sqlxDB, nil, enf)
	r := chi.NewRouter()
	r.Route("/api/v1", h.RegisterRoutes)

	mock.ExpectQuery("SELECT COALESCE.*FROM public.business_objects").
		WithArgs("fund", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"driving_table", "key_column"}).AddRow("mdm.fund", "id"))
	mock.ExpectQuery("SELECT column_name FROM information_schema.columns").
		WillReturnRows(sqlmock.NewRows([]string{"column_name"}).AddRow("id").AddRow("name").AddRow("aum").AddRow("tenant_id"))
	mock.ExpectQuery("SELECT EXISTS").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectBegin()
	// tenant_id is forced from the request as $1, whatever the payload says.
	for i := 0; i < 2; i++ {
		mock.ExpectQuery(`INSERT INTO mdm.fund \(tenant_id, aum, name\)`).
			WithArgs("00000000-0000-0000-0000-000000000001", sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(i))
	}
	mock.ExpectCommit()

	body, _ := json.Marshal(map[string]interface{}{"records": []map[string]interface{}{
		{"name": "Fund A", "aum": 10},
		{"name": "Fund B", "aum": 99, "tenant_id": "someone-else"},
		{"name": "Fund C", "bogus_col": 1},
	}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/bo/fund/records/bulk", bytes.NewBuffer(body))
	req.Header.Set("X-Tenant-ID", "00000000-0000-0000-0000-000000000001")
	req = withTestAuth(req, "00000000-0000-0000-0000-000000000001")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var resp boBulkResponse
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, 1, resp.Written)
	assert.Len(t, resp.Failed, 2)
	assert.Equal(t, 1, resp.Failed[0].Index)
	assert.Equal(t, []string{"notional within limit"}, resp.Failed[0].Rules)
	assert.Equal(t, 2, resp.Failed[1].Index)
	assert.Equal(t, "9000-3", resp.Failed[1].Code, "unknown field is a catalog message")
	assert.Equal(t, "9000-5", resp.Failed[0].Code)
	assert.Equal(t, []string{"fund"}, enf.boKeys)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBuildBOUpsertUpdate(t *testing.T) {
	w := map[string]bool{"isin": true, "name": true, "aum": true}
	q, args, err := buildBOUpsertUpdate("mdm.fund", "t", true, w, []string{"isin"}, map[string]interface{}{"isin": "X", "name": "N", "aum": 1})
	assert.NoError(t, err)
	assert.Contains(t, q, "SET aum = $3, name = $4, updated_at = NOW()")
	assert.Contains(t, q, "WHERE tenant_id = $1 AND isin = $2")
	assert.Equal(t, []interface{}{"t", "X", 1, "N"}, args)

	_, _, err = buildBOUpsertUpdate("mdm.fund", "t", true, w, []string{"isin"}, map[string]interface{}{"name": "N"})
	assert.ErrorContains(t, err, "9000-15 [isin]")
	_, _, err = buildBOUpsertUpdate("mdm.fund", "t", true, w, []string{"isin"}, map[string]interface{}{"isin": "X", "evil; DROP": 1})
	assert.ErrorContains(t, err, "9000-3")
}

// A write that leaves required fields empty is a 422 naming them, like a
// rule rejection.
func TestWriteBOWriteError_RequiredFieldsIs422(t *testing.T) {
	rec := httptest.NewRecorder()
	writeBOWriteError(rec, &metadata.RequiredFieldsError{Fields: []string{"Issuer (issuer_id)"}}, "not found", "failed", http.StatusInternalServerError)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	var resp map[string]interface{}
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, []interface{}{"Issuer (issuer_id)"}, resp["missing"])
}

func TestBulkBORecords_MissingRequiredFieldIsAttributed(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	mock.MatchExpectationsInOrder(false)
	h := NewBOCRUDHandler(sqlxDB, nil, &txEnforcer{db: sqlxDB, missing: map[int]bool{0: true}})
	r := chi.NewRouter()
	r.Route("/api/v1", h.RegisterRoutes)

	mock.ExpectQuery("SELECT COALESCE.*FROM public.business_objects").
		WillReturnRows(sqlmock.NewRows([]string{"driving_table", "key_column"}).AddRow("orm.security", "id"))
	mock.ExpectQuery("SELECT column_name FROM information_schema.columns").
		WillReturnRows(sqlmock.NewRows([]string{"column_name"}).AddRow("id").AddRow("sec_name"))
	mock.ExpectQuery("SELECT EXISTS").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectBegin()
	for i := 0; i < 2; i++ {
		mock.ExpectQuery(`INSERT INTO orm.security`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(i))
	}
	mock.ExpectCommit()

	body, _ := json.Marshal(map[string]interface{}{"records": []map[string]interface{}{{"sec_name": "A"}, {"sec_name": "B"}}})
	req := withTestAuth(httptest.NewRequest(http.MethodPost, "/api/v1/bo/security/records/bulk", bytes.NewBuffer(body)), "00000000-0000-0000-0000-000000000001")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var resp boBulkResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 1, resp.Written)
	if assert.Len(t, resp.Failed, 1) {
		assert.Equal(t, 0, resp.Failed[0].Index)
		assert.Equal(t, []string{"Issuer (issuer_id)"}, resp.Failed[0].Missing)
	}
}
