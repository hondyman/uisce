package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	httpapi "github.com/hondyman/semlayer/backend/internal/api"
	"github.com/hondyman/semlayer/backend/internal/reports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReportAPI(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := reports.NewReportService(db)
	handler := httpapi.NewReportHandler(service)
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	t.Run("Create Template", func(t *testing.T) {
		mock.ExpectExec("INSERT INTO report_templates").
			WillReturnResult(sqlmock.NewResult(1, 1))

		payload := map[string]interface{}{
			"template_name": "Monthly Performance",
			"description":   "Monthly portfolio performance report",
			"category":      "performance",
			"is_active":     true,
			"is_public":     false,
			"layout_config": map[string]interface{}{
				"sections": []interface{}{},
			},
			"parameter_schema": map[string]interface{}{},
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/api/v1/reports/", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("List Templates", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "tenant_id", "template_name", "description", "category", "is_active", "is_public", "created_at", "updated_at", "version"}).
			AddRow("00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000000", "Report 1", "Desc 1", "perf", true, false, time.Now(), time.Now(), 1)

		mock.ExpectQuery("SELECT id, tenant_id, template_name, description, category").
			WillReturnRows(rows)

		req := httptest.NewRequest("GET", "/api/v1/reports/", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Logf("Response Body: %s", w.Body.String())
		}
		assert.Equal(t, http.StatusOK, w.Code)
		var templates []map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&templates)
		require.NoError(t, err)
		require.Len(t, templates, 1)
		assert.Equal(t, "Report 1", templates[0]["template_name"])
	})
}
