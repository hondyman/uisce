package api_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	httpapi "github.com/hondyman/uisce/backend/internal/api"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newValidationRouter(db *sql.DB) http.Handler {
	r := chi.NewRouter()
	httpapi.RegisterValidationRulesRoutes(r, db, nil, &mockResolver{})
	return r
}

func TestValidationRulesAPI_ListRulesMissingTenant(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	defer func() { require.NoError(t, mock.ExpectationsWereMet()) }()

	router := newValidationRouter(db)
	req := httptest.NewRequest(http.MethodGet, "/validation-rules?datasource_id=ds-1", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusUnauthorized, resp.Code)
	var errResp httpapi.ErrorResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &errResp))
	assert.Equal(t, "auth_error", errResp.ErrorCode)
}

func TestValidationRulesAPI_ListRulesSuccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	router := newValidationRouter(db)
	tenantID := "ten"
	datasourceID := "datasource-456"
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) as count FROM catalog_validation_rules WHERE tenant_id = $1 AND datasource_id = $2")).
		WithArgs(tenantID, datasourceID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	rows := sqlmock.NewRows([]string{"id", "tenant_id", "datasource_id", "rule_name", "rule_type", "description", "target_entity", "target_entity_id", "target_entities", "target_entity_ids", "condition_json", "severity", "is_active", "is_core", "created_by", "created_at", "updated_at", "script_content"}).
		AddRow(
			"rule-1",
			tenantID,
			datasourceID,
			"Max Position",
			"business_logic",
			"Ensures max position",
			"account",
			sql.NullString{},
			pq.StringArray{"account"},
			pq.StringArray{},
			[]byte("{\"maxPercentage\":25}"),
			"error",
			true,
			false,
			sql.NullString{},
			now,
			now,
			sql.NullString{},
		)

	query := `(?s)SELECT id, tenant_id, datasource_id, rule_name, rule_type, description, target_entity,\s+target_entity_id, target_entities, target_entity_ids, condition_json,\s+severity, (?:COALESCE\(is_active, true\)|is_active), (?:COALESCE\(is_core, false\)|is_core), created_by, created_at, updated_at, script_content\s+FROM catalog_validation_rules\s+WHERE tenant_id = \$1 AND datasource_id = \$2\s+ORDER BY rule_name\s+LIMIT \$(?:3|4) OFFSET \$(?:4|5)`
	mock.ExpectQuery(query).
		WithArgs(tenantID, datasourceID, 20, 0).
		WillReturnRows(rows)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT DISTINCT rule_type, COUNT(*) as count FROM catalog_validation_rules WHERE tenant_id = $1 AND datasource_id = $2 GROUP BY rule_type ORDER BY count DESC")).
		WithArgs(tenantID, datasourceID).
		WillReturnRows(sqlmock.NewRows([]string{"rule_type", "count"}).AddRow("business_logic", 1))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT DISTINCT severity, COUNT(*) as count FROM catalog_validation_rules WHERE tenant_id = $1 AND datasource_id = $2 GROUP BY severity ORDER BY count DESC")).
		WithArgs(tenantID, datasourceID).
		WillReturnRows(sqlmock.NewRows([]string{"severity", "count"}).AddRow("error", 1))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT DISTINCT target_entity, COUNT(*) as count FROM catalog_validation_rules WHERE tenant_id = $1 AND datasource_id = $2 GROUP BY target_entity ORDER BY count DESC")).
		WithArgs(tenantID, datasourceID).
		WillReturnRows(sqlmock.NewRows([]string{"target_entity", "count"}).AddRow("account", 1))

	req := httptest.NewRequest(http.MethodGet, "/validation-rules?tenant_id="+tenantID+"&datasource_id="+datasourceID, nil)
	req = withValidHeaders(req, tenantID, datasourceID)
	req = withAuth(req, tenantID)

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
	assert.Equal(t, float64(1), payload["total"])
	assert.NotEmpty(t, payload["rules"])
	require.NoError(t, mock.ExpectationsWereMet())
}

// Rule creation through catalog_validation_rules is retired (handleCreateValidationRule): every
// POST, authenticated or not, gets 410 endpoint_retired and never touches the database.
func TestValidationRulesAPI_CreateRuleIsRetired(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	defer func() { require.NoError(t, mock.ExpectationsWereMet()) }()

	router := newValidationRouter(db)
	for name, authed := range map[string]bool{"authenticated": true, "unauthenticated": false} {
		t.Run(name, func(t *testing.T) {
			body := bytes.NewBufferString(`{"rule_name":"Exposure Limit","rule_type":"business_logic","target_entity":"portfolio"}`)
			req := httptest.NewRequest(http.MethodPost, "/validation-rules", body)
			req.Header.Set("Content-Type", "application/json")
			if authed {
				req = withValidHeaders(req, "ten", "datasource-def")
				req = withAuth(req, "ten")
			}
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			require.Equal(t, http.StatusGone, resp.Code)
			var errResp httpapi.ErrorResponse
			require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &errResp))
			assert.Equal(t, "endpoint_retired", errResp.ErrorCode)
		})
	}
}

func TestValidationRulesAPI_ListRulesCountError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	defer func() { require.NoError(t, mock.ExpectationsWereMet()) }()

	tenantID := "ten"
	datasourceID := "ds"
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) as count FROM catalog_validation_rules WHERE tenant_id = $1 AND datasource_id = $2")).
		WithArgs(tenantID, datasourceID).
		WillReturnError(fmt.Errorf("boom"))

	router := newValidationRouter(db)
	req := httptest.NewRequest(http.MethodGet, "/validation-rules?tenant_id="+tenantID+"&datasource_id="+datasourceID, nil)
	req = withValidHeaders(req, tenantID, datasourceID)
	req = withAuth(req, tenantID)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusInternalServerError, resp.Code)
	var errResp httpapi.ErrorResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &errResp))
	assert.Equal(t, "query_error", errResp.ErrorCode)
}

// The legacy catalog_validation_rules execution surface ran rules through a
// CUE engine; it is retired so internal/rules/vm is the only rule engine.
// None of these may touch the database or evaluate anything.
func TestValidationRulesAPI_LegacyEvaluationEndpointsAreGone(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	router := newValidationRouter(db)

	for _, c := range []struct{ method, path string }{
		{http.MethodGet, "/validation-rules/schema?tenant_id=t&bo_id=b"},
		{http.MethodPost, "/validation-rules/r1/execute"},
		{http.MethodPost, "/validation-rules/r1/simulate-with-instance"},
	} {
		req := httptest.NewRequest(c.method, c.path, bytes.NewBufferString(`{}`))
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		require.Equal(t, http.StatusGone, resp.Code, c.path)
		var errResp httpapi.ErrorResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &errResp), c.path)
		assert.Equal(t, "endpoint_retired", errResp.ErrorCode, c.path)
	}
	require.NoError(t, mock.ExpectationsWereMet()) // no queries ran
}
