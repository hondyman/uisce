package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// routedEnforcer is a txEnforcer whose BO records live in another database.
type routedEnforcer struct {
	*txEnforcer
	err error
}

func (r *routedEnforcer) RecordsDB(context.Context, string, string) (*sqlx.DB, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.txEnforcer.db, nil
}

// A BO bound to another datasource is written and read there; metadata
// lookups stay in the metadata database.
func TestBORecords_UseTheBOsOwnDatabase(t *testing.T) {
	metaDB, meta, _ := sqlmock.New()
	defer metaDB.Close()
	recDB, rec, _ := sqlmock.New()
	defer recDB.Close()
	meta.MatchExpectationsInOrder(false)
	rec.MatchExpectationsInOrder(false)
	records := sqlx.NewDb(recDB, "sqlmock")
	h := NewBOCRUDHandler(sqlx.NewDb(metaDB, "sqlmock"), nil, &routedEnforcer{txEnforcer: &txEnforcer{db: records}})
	r := chi.NewRouter()
	r.Route("/api/v1", h.RegisterRoutes)

	meta.ExpectQuery("SELECT COALESCE.*FROM public.business_objects").
		WillReturnRows(sqlmock.NewRows([]string{"driving_table", "key_column"}).AddRow("orm.security", "id"))
	rec.ExpectQuery("SELECT column_name FROM information_schema.columns").
		WillReturnRows(sqlmock.NewRows([]string{"column_name"}).AddRow("id").AddRow("sec_name").AddRow("isin"))
	rec.ExpectQuery("SELECT EXISTS").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	rec.ExpectBegin()
	rec.ExpectQuery(`INSERT INTO orm.security`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(7))
	rec.ExpectCommit()

	body, _ := json.Marshal(map[string]any{"records": []map[string]any{{"sec_name": "Alpha", "isin": "IE0019722233"}}})
	req := withTestAuth(httptest.NewRequest(http.MethodPost, "/api/v1/bo/security/records/bulk", bytes.NewBuffer(body)), pipeTenant)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var resp boBulkResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 1, resp.Written)
	assert.NoError(t, meta.ExpectationsWereMet(), "only metadata in the metadata DB")
	assert.NoError(t, rec.ExpectationsWereMet(), "schema checks and the insert in the BO's own DB")
}

// A BO whose datasource cannot be resolved is refused - never written to
// the metadata database instead.
func TestBORecords_UnresolvableDatasourceIsRefused(t *testing.T) {
	metaDB, meta, _ := sqlmock.New()
	defer metaDB.Close()
	meta.ExpectQuery("SELECT COALESCE.*FROM public.business_objects").
		WillReturnRows(sqlmock.NewRows([]string{"driving_table", "key_column"}).AddRow("orm.security", "id"))
	enf := &routedEnforcer{txEnforcer: &txEnforcer{db: sqlx.NewDb(metaDB, "sqlmock")}, err: errors.New("datasource has no connection configuration")}
	h := NewBOCRUDHandler(sqlx.NewDb(metaDB, "sqlmock"), nil, enf)
	r := chi.NewRouter()
	r.Route("/api/v1", h.RegisterRoutes)

	for _, c := range []struct{ method, path, body string }{
		{http.MethodPost, "/api/v1/bo/security/records/bulk", `{"records":[{"sec_name":"x"}]}`},
		{http.MethodGet, "/api/v1/bo/security/records", ""},
	} {
		meta.ExpectQuery("SELECT COALESCE.*FROM public.business_objects").
			WillReturnRows(sqlmock.NewRows([]string{"driving_table", "key_column"}).AddRow("orm.security", "id"))
		req := withTestAuth(httptest.NewRequest(c.method, c.path, bytes.NewBufferString(c.body)), pipeTenant)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.GreaterOrEqual(t, w.Code, 400, "%s %s must be refused", c.method, c.path)
		assert.Contains(t, w.Body.String(), "no connection configuration")
	}
	// No INSERT/SELECT against the record table was attempted in the metadata DB:
	// sqlmock fails any unexpected query, and none was expected.
}
