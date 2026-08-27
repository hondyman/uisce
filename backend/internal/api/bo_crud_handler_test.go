package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

func TestHandleUpdateBORecord(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	handler := NewBOCRUDHandler(sqlxDB)

	r := chi.NewRouter()
	r.Put("/api/v1/bo/{boKey}/records/{recordId}", handler.HandleUpdateBORecord)

	// Mock metadata resolution
	metaRows := sqlmock.NewRows([]string{"driving_table", "key_column"}).
		AddRow("oms.account", "id")
	mock.ExpectQuery("SELECT COALESCE.*FROM public.business_objects").
		WithArgs("account", sqlmock.AnyArg()).
		WillReturnRows(metaRows)

	// Mock update returning row
	updateRows := sqlmock.NewRows([]string{"id", "account_name", "status"}).
		AddRow("acc-123", "Acme Prime Institutional", "ACTIVE")
	mock.ExpectQuery("UPDATE oms.account").
		WithArgs(sqlmock.AnyArg(), "acc-123", "Acme Prime Institutional").
		WillReturnRows(updateRows)

	payload := map[string]interface{}{
		"account_name": "Acme Prime Institutional",
		"tenant_id":    "forbidden-tenant-override",
		"id":           "forbidden-id-override",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/bo/account/records/acc-123", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "00000000-0000-0000-0000-000000000001")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "acc-123", resp["id"])
	assert.Equal(t, "Acme Prime Institutional", resp["account_name"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPageLayoutHandler_GetAndSave(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	handler := NewPageDesignerLayoutHandler(sqlxDB)

	r := chi.NewRouter()
	r.Get("/page-designer/pages/{pageKey}", handler.GetPage)
	r.Post("/page-designer/pages", handler.SavePage)

	now := time.Now()
	// 1. Get Page with Gold Copy union fallback
	pageRows := sqlmock.NewRows([]string{
		"id", "tenant_id", "page_key", "title", "description", "layout_spec", "is_gold_copy", "is_active", "created_at", "updated_at",
	}).AddRow(
		"page-1", "00000000-0000-0000-0000-000000000001", "account_hub", "Account Hub", nil, []byte(`{"declaredParameters":[]}`), true, true, now, now,
	)

	mock.ExpectQuery("FROM public.page_registry").
		WithArgs("account_hub", "00000000-0000-0000-0000-000000000001").
		WillReturnRows(pageRows)

	req := httptest.NewRequest(http.MethodGet, "/page-designer/pages/account_hub", nil)
	req.Header.Set("X-Tenant-ID", "00000000-0000-0000-0000-000000000001")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code, "Expected 200, got: "+rec.Body.String())
	var getResp PageRegistryEntry
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &getResp))
	assert.Equal(t, "account_hub", getResp.PageKey)
	assert.True(t, getResp.IsGoldCopy)

	// 2. Save Page
	savePayload := map[string]interface{}{
		"page_key":    "custom_hub",
		"title":       "Custom Hub",
		"layout_spec": map[string]interface{}{"sections": []interface{}{}},
	}
	saveBody, _ := json.Marshal(savePayload)

	savedRows := sqlmock.NewRows([]string{
		"id", "tenant_id", "page_key", "title", "description", "layout_spec", "is_gold_copy", "is_active", "created_at", "updated_at",
	}).AddRow(
		"page-2", "00000000-0000-0000-0000-000000000001", "custom_hub", "Custom Hub", nil, []byte(`{"sections":[]}`), false, true, now, now,
	)

	mock.ExpectQuery("INSERT INTO public.page_registry").
		WithArgs(sqlmock.AnyArg(), "00000000-0000-0000-0000-000000000001", "custom_hub", "Custom Hub", nil, sqlmock.AnyArg(), false).
		WillReturnRows(savedRows)

	saveReq := httptest.NewRequest(http.MethodPost, "/page-designer/pages", bytes.NewBuffer(saveBody))
	saveReq.Header.Set("Content-Type", "application/json")
	saveReq.Header.Set("X-Tenant-ID", "00000000-0000-0000-0000-000000000001")
	saveRec := httptest.NewRecorder()
	r.ServeHTTP(saveRec, saveReq)

	assert.Equal(t, http.StatusOK, saveRec.Code, "Expected 200, got: "+saveRec.Body.String())
	assert.NoError(t, mock.ExpectationsWereMet())
}
