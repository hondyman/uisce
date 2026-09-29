package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLakehouseStreamingHandler(t *testing.T) {
	dbMock, _, err := sqlmock.New()
	require.NoError(t, err)
	defer dbMock.Close()

	sqlxDB := sqlx.NewDb(dbMock, "sqlmock")
	handler := NewLakehouseStreamingHandler(sqlxDB)

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	t.Run("GET /lakehouse-streaming/overview", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/lakehouse-streaming/overview", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp LakehouseOverviewResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.NotEmpty(t, resp.Loaders)
		assert.Equal(t, 5, len(resp.Loaders))
		assert.Equal(t, "orm_oms.orm.order", resp.Loaders[0].Topic)
	})

	t.Run("GET /lakehouse-streaming/tables", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/lakehouse-streaming/tables", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var tables []IcebergTableDefinition
		err := json.Unmarshal(rec.Body.Bytes(), &tables)
		require.NoError(t, err)
		assert.NotEmpty(t, tables)
		assert.Equal(t, "raw_market_data", tables[0].Namespace)
		assert.Equal(t, "source_attribute_value", tables[0].TableName)
		assert.Contains(t, tables[0].PartitionFields, "tenant_id")
	})

	t.Run("POST /lakehouse-streaming/provision", func(t *testing.T) {
		payload := `{"namespace": "analytics", "table_name": "daily_exposure", "partition_fields": ["as_of_date"]}`
		req := httptest.NewRequest(http.MethodPost, "/lakehouse-streaming/provision", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "PROVISIONED", resp["status"])
		assert.Equal(t, "analytics", resp["namespace"])
		assert.Equal(t, "daily_exposure", resp["table_name"])
	})

	t.Run("POST /lakehouse-streaming/dlq/replay", func(t *testing.T) {
		payload := `{"violation_ids": ["v-1", "v-2"], "action": "replay"}`
		req := httptest.NewRequest(http.MethodPost, "/lakehouse-streaming/dlq/replay", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "PROCESSED", resp["status"])
		assert.Equal(t, float64(2), resp["replayed"])
	})
}
