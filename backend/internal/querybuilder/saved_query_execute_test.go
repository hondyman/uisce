package querybuilder

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/hondyman/uisce/backend/internal/boresolver"
	"github.com/hondyman/uisce/backend/internal/handlers"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupExecuteTestEnv(t *testing.T, tenantID, userID string) (*SavedQueryHandler, *chi.Mux, sqlmock.Sqlmock, func(method, path string, body []byte) *http.Request) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")

	deps := handlers.SecurityContextDeps{Resolver: &mockDSResolver{defaultTenantID: tenantID}}
	h := NewSavedQueryHandler(sqlxDB, nil, nil, deps)

	r := chi.NewRouter()
	r.Post("/api/explorer/saved-queries/{id}/execute", h.HandleExecuteSavedQuery)

	makeReq := func(method, path string, body []byte) *http.Request {
		var bodyReader *bytes.Reader
		if body != nil {
			bodyReader = bytes.NewReader(body)
		} else {
			bodyReader = bytes.NewReader([]byte{})
		}
		req := httptest.NewRequest(method, path, bodyReader)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Datasource-Id", "ds-1")
		auth := security.AuthInfo{
			TenantIDs: []string{tenantID},
			UserID:    userID,
			Roles:     []string{"tenant_admin"},
		}
		return req.WithContext(security.WithAuthInfo(req.Context(), auth))
	}

	return h, r, mock, makeReq
}

func TestValidateRuntimeFilters(t *testing.T) {
	sq := &SavedQuery{
		ID:   "sq-100",
		BOID: "bo-account",
		State: SavedQueryState{
			Dimensions: []SavedQueryDimension{{TermNodeID: "dim-region", Alias: "region"}},
			Measures:   []SavedQueryMeasure{{TermNodeID: "meas-rev", Alias: "revenue"}},
			Filters:    []SavedQueryFilter{{TermNodeID: "filter-status", Operator: "eq", Value: "active"}},
		},
	}

	// 1. Valid runtime filters
	valid := []RuntimeFilter{
		{TermNodeID: "dim-region", Operator: "eq", Value: "EMEA"},
		{TermNodeID: "filter-status", Operator: "in", Value: []string{"active", "pending"}},
	}
	assert.NoError(t, validateRuntimeFilters(valid, sq))

	// 2. Unknown term returns error
	unknown := []RuntimeFilter{
		{TermNodeID: "completely_random_term_xyz", Operator: "eq", Value: "foo"},
	}
	err := validateRuntimeFilters(unknown, sq)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown termNodeId")

	// 3. Invalid operator returns error
	badOp := []RuntimeFilter{
		{TermNodeID: "dim-region", Operator: "INVALID_SQL_OP", Value: "foo"},
	}
	errBadOp := validateRuntimeFilters(badOp, sq)
	require.Error(t, errBadOp)
	assert.Contains(t, errBadOp.Error(), "invalid runtime filter operator")
}

func TestHandleExecuteSavedQuery_Endpoint(t *testing.T) {
	tenantID := "11111111-1111-1111-1111-111111111111"
	userID := "22222222-2222-2222-2222-222222222222"
	queryID := "33333333-3333-3333-3333-333333333333"

	t.Run("execute with runtimeFilters and bound params returns 200", func(t *testing.T) {
		_, router, mock, makeReq := setupExecuteTestEnv(t, tenantID, userID)

		mock.ExpectQuery(`SELECT .* FROM data_explorer\.saved_query WHERE id = \$1 AND tenant_id = \$2`).
			WithArgs(queryID, tenantID, userID).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "tenant_id", "user_id", "name", "description", "source_id", "binding_id", "related_bo_ids",
				"chart_type", "query_state", "tags", "folder_id", "is_favorite", "visibility", "is_core",
				"status", "archived_at", "created_by", "created_at", "updated_at",
			}).AddRow(
				queryID, tenantID, userID, "Regional Sales", "Desc", "bo-account", nil, nil,
				"bar", []byte(`{"dimensions": [{"termNodeId": "region", "alias": "region"}], "filters": [{"termNodeId": "region", "operator": "eq", "value": "EMEA"}]}`),
				nil, nil, false, "shared", false,
				"active", nil, userID, time.Now(), time.Now(),
			))

		body := []byte(`{
			"params": { "year": "2026" },
			"runtimeFilters": [
				{ "termNodeId": "region", "operator": "eq", "value": "EMEA" }
			],
			"limit": 50,
			"format": "json"
		}`)

		req := makeReq("POST", "/api/explorer/saved-queries/"+queryID+"/execute", body)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "Regional Sales", resp["name"])
	})

	t.Run("execute with unknown term in runtimeFilters returns 422 Unprocessable Entity", func(t *testing.T) {
		_, router, mock, makeReq := setupExecuteTestEnv(t, tenantID, userID)

		mock.ExpectQuery(`SELECT .* FROM data_explorer\.saved_query WHERE id = \$1 AND tenant_id = \$2`).
			WithArgs(queryID, tenantID, userID).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "tenant_id", "user_id", "name", "description", "source_id", "binding_id", "related_bo_ids",
				"chart_type", "query_state", "tags", "folder_id", "is_favorite", "visibility", "is_core",
				"status", "archived_at", "created_by", "created_at", "updated_at",
			}).AddRow(
				queryID, tenantID, userID, "Regional Sales", "Desc", "bo-account", nil, nil,
				"bar", []byte(`{"dimensions": [{"termNodeId": "region", "alias": "region"}]}`),
				nil, nil, false, "shared", false,
				"active", nil, userID, time.Now(), time.Now(),
			))

		body := []byte(`{
			"runtimeFilters": [
				{ "termNodeId": "unmapped_foreign_term_id", "operator": "eq", "value": "x" }
			]
		}`)

		req := makeReq("POST", "/api/explorer/saved-queries/"+queryID+"/execute", body)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "unknown termNodeId")
	})

	t.Run("execute across tenant boundary returns 404", func(t *testing.T) {
		_, router, mock, makeReq := setupExecuteTestEnv(t, tenantID, userID)

		mock.ExpectQuery(`SELECT .* FROM data_explorer\.saved_query WHERE id = \$1 AND tenant_id = \$2`).
			WithArgs(queryID, tenantID, userID).
			WillReturnError(sqlmock.ErrCancelled)

		// Gold tenant check returns non-matching
		mock.ExpectQuery(`SELECT id FROM \(SELECT public\.uisce_gold_copy_tenant_id\(\) AS id\) g WHERE id IS NOT NULL`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}))

		req := makeReq("POST", "/api/explorer/saved-queries/"+queryID+"/execute", []byte(`{}`))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("archived query resolves and executes successfully for page continuity", func(t *testing.T) {
		_, router, mock, makeReq := setupExecuteTestEnv(t, tenantID, userID)

		archivedTime := time.Now().Add(-24 * time.Hour)
		mock.ExpectQuery(`SELECT .* FROM data_explorer\.saved_query WHERE id = \$1 AND tenant_id = \$2`).
			WithArgs(queryID, tenantID, userID).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "tenant_id", "user_id", "name", "description", "source_id", "binding_id", "related_bo_ids",
				"chart_type", "query_state", "tags", "folder_id", "is_favorite", "visibility", "is_core",
				"status", "archived_at", "created_by", "created_at", "updated_at",
			}).AddRow(
				queryID, tenantID, userID, "Archived Regional Sales", "Desc", "bo-account", nil, nil,
				"bar", []byte(`{"dimensions": [{"termNodeId": "region", "alias": "region"}]}`),
				nil, nil, false, "shared", false,
				"archived", archivedTime, userID, time.Now(), time.Now(),
			))

		req := makeReq("POST", "/api/explorer/saved-queries/"+queryID+"/execute", []byte(`{}`))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "Archived Regional Sales", resp["name"])
	})
}

func TestSavedQueryExecute_NarrowingOnlyContract(t *testing.T) {
	// Base query definition representing a core query with a mandatory base filter
	sq := SavedQuery{
		ID:   "core-sq-emea",
		BOID: "bo_account",
		State: SavedQueryState{
			Dimensions: []SavedQueryDimension{{TermNodeID: "region", Alias: "region", BOID: "bo_account"}},
			Measures:   []SavedQueryMeasure{{TermNodeID: "revenue", Alias: "revenue", Aggregation: "SUM", BOID: "bo_account"}},
			Filters: []SavedQueryFilter{
				{TermNodeID: "region", Operator: "eq", Value: "EMEA"},
			},
		},
	}

	// 1. Resolve base filters
	baseFilters, err := resolveParams(sq.State, nil)
	require.NoError(t, err)
	require.Len(t, baseFilters, 1)
	assert.Equal(t, "region", baseFilters[0].TermNodeID)
	assert.Equal(t, "eq", baseFilters[0].Operator)
	assert.Equal(t, "EMEA", baseFilters[0].Value)

	// 2. Client runtime filter injection: add sector = 'Financials'
	runtimeFilters := []RuntimeFilter{
		{TermNodeID: "sector", Operator: "eq", Value: "Financials"},
	}
	allFilters := make([]boresolver.FilterDef, 0, len(baseFilters)+len(runtimeFilters))
	allFilters = append(allFilters, baseFilters...)
	for _, rf := range runtimeFilters {
		allFilters = append(allFilters, boresolver.FilterDef{
			TermNodeID: rf.TermNodeID,
			Operator:   rf.Operator,
			Value:      rf.Value,
			BOID:       sq.BOID,
		})
	}

	qd := savedQueryDef(sq, "tenant-1", allFilters, 100)
	require.Len(t, qd.Query.Filters, 2)
	assert.Equal(t, "region", qd.Query.Filters[0].TermNodeID)
	assert.Equal(t, "EMEA", qd.Query.Filters[0].Value)
	assert.Equal(t, "sector", qd.Query.Filters[1].TermNodeID)
	assert.Equal(t, "Financials", qd.Query.Filters[1].Value)

	// 3. Compile SQL and verify both filters exist as AND-composed predicates
	primary := &boresolver.BODefinition{
		ID:           "bo_account",
		DrivingTable: "account",
		Fields: []boresolver.BOField{
			{ID: "f1", Name: "region", Type: "string", PhysicalColumn: "account.region"},
			{ID: "f2", Name: "revenue", Type: "number", PhysicalColumn: "account.revenue"},
			{ID: "f3", Name: "sector", Type: "string", PhysicalColumn: "account.sector"},
		},
	}
	gen, _ := boresolver.NewBOSQLGenerator(nil, "postgres")
	compiledSQL, args, _, err := buildMultiBOSQL(gen, primary, nil, qd, "tenant-1")
	require.NoError(t, err)

	// Verify parameterized placeholder binding and AND composition
	assert.Contains(t, compiledSQL, `t0.region = $`)
	assert.Contains(t, compiledSQL, `t0.sector = $`)
	assert.Contains(t, compiledSQL, `AND`)
	assert.Equal(t, []interface{}{"EMEA", "Financials", "tenant-1"}, args)

	// 4. Adversarial narrowing test: client attempts to widen by injecting conflicting filter
	adversarialFilters := []RuntimeFilter{
		{TermNodeID: "region", Operator: "neq", Value: "EMEA"},
	}
	advAll := append(baseFilters, boresolver.FilterDef{
		TermNodeID: adversarialFilters[0].TermNodeID,
		Operator:   adversarialFilters[0].Operator,
		Value:      adversarialFilters[0].Value,
		BOID:       sq.BOID,
	})
	advQD := savedQueryDef(sq, "tenant-1", advAll, 100)
	advSQL, advArgs, _, err := buildMultiBOSQL(gen, primary, nil, advQD, "tenant-1")
	require.NoError(t, err)

	// Both the core base filter AND the adversarial runtime filter are AND-composed:
	// (t0.region = $1) AND (t0.region != $2) -> returns 0 rows (narrowed to empty), never widened
	assert.Contains(t, advSQL, `t0.region = $1`)
	assert.Contains(t, advSQL, `t0.region != $2`)
	assert.Equal(t, []interface{}{"EMEA", "EMEA", "tenant-1"}, advArgs)
}

func TestHandleGetSavedQuerySchema(t *testing.T) {
	tenantID := "11111111-1111-1111-1111-111111111111"
	userID := "22222222-2222-2222-2222-222222222222"
	queryID := "33333333-3333-3333-3333-333333333333"

	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")

	deps := handlers.SecurityContextDeps{Resolver: &mockDSResolver{defaultTenantID: tenantID}}
	h := NewSavedQueryHandler(sqlxDB, nil, nil, deps)

	r := chi.NewRouter()
	r.Get("/api/explorer/saved-queries/{id}/schema", h.HandleGetSavedQuerySchema)

	stateJSON, _ := json.Marshal(SavedQueryState{
		Dimensions: []SavedQueryDimension{{TermNodeID: "dim-region", Alias: "Region", BOID: "bo-acc"}},
		Measures:   []SavedQueryMeasure{{TermNodeID: "meas-rev", Alias: "TotalRevenue", Aggregation: "SUM", BOID: "bo-acc"}},
		Parameters: []SavedQueryParameter{{Name: "minAmount", Type: "number", Default: 1000, Required: true, Label: "Min Amount"}},
	})

	mock.ExpectQuery(`SELECT .* FROM data_explorer\.saved_query WHERE id = \$1 AND tenant_id = \$2`).
		WithArgs(queryID, tenantID, userID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "tenant_id", "user_id", "name", "description", "source_id", "binding_id", "related_bo_ids",
			"chart_type", "query_state", "tags", "folder_id", "is_favorite", "visibility", "is_core",
			"status", "archived_at", "created_by", "created_at", "updated_at",
		}).AddRow(
			queryID, tenantID, userID, "Regional Revenue", "Schema test", "bo-acc",
			nil, "{}", "bar", stateJSON, "{}", nil, false, "shared", false,
			"active", nil, nil, time.Now(), time.Now(),
		))

	req := httptest.NewRequest(http.MethodGet, "/api/explorer/saved-queries/"+queryID+"/schema", nil)
	auth := security.AuthInfo{TenantIDs: []string{tenantID}, UserID: userID}
	req = req.WithContext(security.WithAuthInfo(req.Context(), auth))
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var schemaResp QuerySchemaResponse
	err = json.Unmarshal(rec.Body.Bytes(), &schemaResp)
	require.NoError(t, err)

	assert.Equal(t, queryID, schemaResp.ID)
	assert.Equal(t, "Regional Revenue", schemaResp.Name)
	assert.Equal(t, "bo-acc", schemaResp.BOID)
	assert.Len(t, schemaResp.Parameters, 1)
	assert.Equal(t, "minAmount", schemaResp.Parameters[0].Name)
	assert.Len(t, schemaResp.Columns, 2)
	assert.Equal(t, "Region", schemaResp.Columns[0].Alias)
	assert.Equal(t, "dimension", schemaResp.Columns[0].Type)
	assert.Equal(t, "TotalRevenue", schemaResp.Columns[1].Alias)
	assert.Equal(t, "measure", schemaResp.Columns[1].Type)
	assert.Equal(t, "SUM", schemaResp.Columns[1].Aggregation)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandleExecuteSavedQuery_CSVFormat(t *testing.T) {
	tenantID := "11111111-1111-1111-1111-111111111111"
	userID := "22222222-2222-2222-2222-222222222222"
	queryID := "33333333-3333-3333-3333-333333333333"

	_, router, mock, makeReq := setupExecuteTestEnv(t, tenantID, userID)

	mock.ExpectQuery(`SELECT .* FROM data_explorer\.saved_query WHERE id = \$1 AND tenant_id = \$2`).
		WithArgs(queryID, tenantID, userID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "tenant_id", "user_id", "name", "description", "source_id", "binding_id", "related_bo_ids",
			"chart_type", "query_state", "tags", "folder_id", "is_favorite", "visibility", "is_core",
			"status", "archived_at", "created_by", "created_at", "updated_at",
		}).AddRow(
			queryID, tenantID, userID, "Export Deals", "", "bo-acc",
			nil, "{}", "table", []byte(`{"dimensions":[{"termNodeId":"d1"}],"measures":[]}`), "{}", nil, false, "shared", false,
			"active", nil, nil, time.Now(), time.Now(),
		))

	req := makeReq(http.MethodPost, "/api/explorer/saved-queries/"+queryID+"/execute?format=csv", nil)
	req.Header.Set("Accept", "text/csv")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "text/csv")
	assert.Contains(t, rec.Header().Get("Content-Disposition"), "attachment; filename=\"Export Deals.csv\"")
	assert.Contains(t, rec.Body.String(), "col1,col2")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandleExecuteSavedQuery_RateLimiting(t *testing.T) {
	tenantID := "rate-tenant-1"
	userID := "rate-user-1"
	queryID := "rate-q-1"

	// Configure rate limit to 2 requests per 10 seconds
	SetExecuteRateLimit(2, 10*time.Second)
	defer SetExecuteRateLimit(300, time.Minute)

	_, router, mock, makeReq := setupExecuteTestEnv(t, tenantID, userID)

	// Mock 2 successful executions
	for i := 0; i < 2; i++ {
		mock.ExpectQuery(`SELECT .* FROM data_explorer\.saved_query WHERE id = \$1 AND tenant_id = \$2`).
			WithArgs(queryID, tenantID, userID).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "tenant_id", "user_id", "name", "description", "source_id", "binding_id", "related_bo_ids",
				"chart_type", "query_state", "tags", "folder_id", "is_favorite", "visibility", "is_core",
				"status", "archived_at", "created_by", "created_at", "updated_at",
			}).AddRow(
				queryID, tenantID, userID, "Rate Test", "", "bo-acc",
				nil, "{}", "table", []byte(`{}`), "{}", nil, false, "shared", false,
				"active", nil, nil, time.Now(), time.Now(),
			))
	}

	// 1st request -> 200 OK
	req1 := makeReq(http.MethodPost, "/api/explorer/saved-queries/"+queryID+"/execute", nil)
	rec1 := httptest.NewRecorder()
	router.ServeHTTP(rec1, req1)
	assert.Equal(t, http.StatusOK, rec1.Code)

	// 2nd request -> 200 OK
	req2 := makeReq(http.MethodPost, "/api/explorer/saved-queries/"+queryID+"/execute", nil)
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusOK, rec2.Code)

	// 3rd request -> Must hit Rate Limit and return 429 Too Many Requests
	mock.ExpectQuery(`SELECT .* FROM data_explorer\.saved_query WHERE id = \$1 AND tenant_id = \$2`).
		WithArgs(queryID, tenantID, userID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "tenant_id", "user_id", "name", "description", "source_id", "binding_id", "related_bo_ids",
			"chart_type", "query_state", "tags", "folder_id", "is_favorite", "visibility", "is_core",
			"status", "archived_at", "created_by", "created_at", "updated_at",
		}).AddRow(
			queryID, tenantID, userID, "Rate Test", "", "bo-acc",
			nil, "{}", "table", []byte(`{}`), "{}", nil, false, "shared", false,
			"active", nil, nil, time.Now(), time.Now(),
		))

	req3 := makeReq(http.MethodPost, "/api/explorer/saved-queries/"+queryID+"/execute", nil)
	rec3 := httptest.NewRecorder()
	router.ServeHTTP(rec3, req3)

	assert.Equal(t, http.StatusTooManyRequests, rec3.Code)
	assert.Equal(t, "60", rec3.Header().Get("Retry-After"))
	assert.Contains(t, rec3.Body.String(), "Rate limit exceeded")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestComputeParamsHash_Deterministic(t *testing.T) {
	params1 := map[string]interface{}{"region": "EMEA", "minAmount": 500}
	rf1 := []RuntimeFilter{{TermNodeID: "stage", Operator: "eq", Value: "closed"}}

	h1 := computeParamsHash(params1, rf1)
	h2 := computeParamsHash(params1, rf1)

	assert.NotEmpty(t, h1)
	assert.Equal(t, h1, h2, "params hash must be deterministic")

	params2 := map[string]interface{}{"region": "APAC", "minAmount": 500}
	h3 := computeParamsHash(params2, rf1)
	assert.NotEqual(t, h1, h3, "different params must yield different hash")
}

func TestHandleExecuteSavedQuery_TokenAuth(t *testing.T) {
	tenantID := "11111111-1111-1111-1111-111111111111"
	userID := "machine-user-1"
	queryID := "33333333-3333-3333-3333-333333333333"
	rawToken := "uk_live_abcdef1234567890"

	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")

	deps := handlers.SecurityContextDeps{Resolver: &mockDSResolver{defaultTenantID: tenantID}}
	h := NewSavedQueryHandler(sqlxDB, nil, nil, deps)

	r := chi.NewRouter()
	r.Post("/api/explorer/saved-queries/{id}/execute", h.HandleExecuteSavedQuery)

	// 1. Mock API Key lookup
	mock.ExpectQuery(`SELECT user_id, tenant_id, roles, tenant_ids, is_active, expires_at FROM public\.api_keys WHERE key_hash = \$1`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "tenant_id", "roles", "tenant_ids", "is_active", "expires_at"}).
			AddRow(userID, tenantID, "{api_client}", "{"+tenantID+"}", true, nil))

	// 2. Mock query lookup
	mock.ExpectQuery(`SELECT .* FROM data_explorer\.saved_query WHERE id = \$1 AND tenant_id = \$2`).
		WithArgs(queryID, tenantID, userID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "tenant_id", "user_id", "name", "description", "source_id", "binding_id", "related_bo_ids",
			"chart_type", "query_state", "tags", "folder_id", "is_favorite", "visibility", "is_core",
			"status", "archived_at", "created_by", "created_at", "updated_at",
		}).AddRow(
			queryID, tenantID, userID, "Machine API Test", "", "bo-acc",
			nil, "{}", "table", []byte(`{}`), "{}", nil, false, "shared", false,
			"active", nil, nil, time.Now(), time.Now(),
		))

	// 3. Mock synchronous audit log insertion
	mock.ExpectExec(`INSERT INTO data_explorer\.saved_query_audit_log`).
		WithArgs(queryID, tenantID, userID, sqlmock.AnyArg(), 0, sqlmock.AnyArg(), http.StatusOK).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Request with Authorization: Bearer <token>
	req := httptest.NewRequest(http.MethodPost, "/api/explorer/saved-queries/"+queryID+"/execute", nil)
	req.Header.Set("Authorization", "Bearer "+rawToken)
	req.Header.Set("X-Datasource-Id", "ds-1")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSavedQueryExecute_InOperator_ParameterizedAndNarrowing(t *testing.T) {
	sq := &SavedQuery{
		ID:   "sq-200",
		BOID: "bo-trade",
		State: SavedQueryState{
			Dimensions: []SavedQueryDimension{{TermNodeID: "term-desk", Alias: "desk"}},
			Measures:   []SavedQueryMeasure{{TermNodeID: "term-notional", Alias: "notional"}},
			Filters: []SavedQueryFilter{
				{TermNodeID: "term-status", Operator: "eq", Value: "SETTLED"},
			},
		},
	}

	// 1. Validator accepts "in" operator with slice of strings or interfaces
	runtimeFilters := []RuntimeFilter{
		{
			TermNodeID: "term-desk",
			Operator:   "in",
			Value:      []string{"NY", "LON", "HK"},
			BOID:       "bo-trade",
		},
	}
	err := validateRuntimeFilters(runtimeFilters, sq)
	require.NoError(t, err, "validator must accept 'in' operator with string slice")

	// 2. Composing narrowing-only filter list: base filters (1) + runtime filter (1) = 2 filters AND-composed
	baseFilters, err := resolveParams(sq.State, nil)
	require.NoError(t, err)
	require.Len(t, baseFilters, 1)

	allFilters := make([]boresolver.FilterDef, 0, len(baseFilters)+len(runtimeFilters))
	allFilters = append(allFilters, baseFilters...)
	for _, rf := range runtimeFilters {
		allFilters = append(allFilters, boresolver.FilterDef{
			TermNodeID: rf.TermNodeID,
			Operator:   rf.Operator,
			Value:      rf.Value,
			BOID:       rf.BOID,
		})
	}
	require.Len(t, allFilters, 2)
	assert.Equal(t, "eq", allFilters[0].Operator)
	assert.Equal(t, "in", allFilters[1].Operator)

	// 3. Compile SQL predicate using boresolver and verify parameterization as flattened placeholders ($1, $2, $3)
	gen := &boresolver.BOSQLGenerator{Dialect: boresolver.PostgresDialect{}}
	genCtx := &boresolver.GenerationContext{}

	clause := boresolver.FilterClause{
		FieldID:  "term-desk",
		Operator: "in",
		Value:    []interface{}{"NY", "LON", "HK"},
	}
	sqlFrag, err := boresolver.CompileFilterPredicate(gen, genCtx, "t0.desk", clause)
	require.NoError(t, err)

	// Assert flattened placeholders ($1, $2, $3) — NEVER concatenated string literals
	assert.Equal(t, "t0.desk IN ($1, $2, $3)", sqlFrag)
	require.Len(t, genCtx.Args, 3)
	assert.Equal(t, "NY", genCtx.Args[0])
	assert.Equal(t, "LON", genCtx.Args[1])
	assert.Equal(t, "HK", genCtx.Args[2])

	// 4. Adversarial check: attempt SQL injection via array item in IN operator
	advGenCtx := &boresolver.GenerationContext{}
	injectionPayload := "') OR '1'='1' --"
	advClause := boresolver.FilterClause{
		FieldID:  "term-desk",
		Operator: "in",
		Value:    []interface{}{"NY", injectionPayload},
	}
	advSQLFrag, err := boresolver.CompileFilterPredicate(gen, advGenCtx, "t0.desk", advClause)
	require.NoError(t, err)

	// SQL fragment must remain strictly parameterized as ($1, $2)
	assert.Equal(t, "t0.desk IN ($1, $2)", advSQLFrag)
	require.Len(t, advGenCtx.Args, 2)
	assert.Equal(t, injectionPayload, advGenCtx.Args[1], "payload must remain isolated in bound parameter args, never injected into SQL text")
}


