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
	"github.com/google/uuid"
	httpapi "github.com/hondyman/semlayer/backend/internal/api"
	"github.com/hondyman/semlayer/backend/internal/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricAPI(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := metrics.NewMetricService(db)
	handler := httpapi.NewMetricHandler(service)
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	t.Run("Create Metric", func(t *testing.T) {
		metric := metrics.MetricDefinition{
			Name:                "test_metric",
			DisplayName:         "Test Metric",
			Domain:              "Finance",
			Granularity:         "day",
			AggregationFunction: "SUM",
			Owner:               "user@example.com",
		}
		body, _ := json.Marshal(metric)

		mock.ExpectExec("INSERT INTO metric_definitions").
			WithArgs(sqlmock.AnyArg(), metric.Name, metric.DisplayName, metric.Description, metric.Domain, metric.Granularity, metric.AggregationFunction, metric.BaseQuery, sqlmock.AnyArg(), sqlmock.AnyArg(), metric.Owner).
			WillReturnResult(sqlmock.NewResult(1, 1))

		req, _ := http.NewRequest("POST", "/metrics/definitions", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("List Metrics", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "name", "display_name", "description", "domain", "granularity", "aggregation_function", "base_query", "dimensions", "sla_config", "owner", "created_at", "updated_at"}).
			AddRow(uuid.New(), "test_metric", "Test Metric", "", "Finance", "day", "SUM", "", []byte("[]"), []byte("{}"), "user@example.com", time.Now(), time.Now())

		mock.ExpectQuery("SELECT id, name, display_name").
			WillReturnRows(rows)

		req, _ := http.NewRequest("GET", "/metrics/definitions", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var res []metrics.MetricDefinition
		json.Unmarshal(w.Body.Bytes(), &res)
		assert.Len(t, res, 1)
		assert.Equal(t, "test_metric", res[0].Name)
	})
}
