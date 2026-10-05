package querybuilder

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/handlers"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	cubeTestTenant = "11111111-1111-1111-1111-111111111111"
	cubeTestUser   = "22222222-2222-2222-2222-222222222222"
	cubeTestMetric = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	cubeTestID     = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
)

func setupCubeHandlerEnv(t *testing.T) (*CubeHandler, *chi.Mux, sqlmock.Sqlmock, func(method, path string, body []byte) *http.Request) {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = mockDB.Close() })
	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")

	deps := handlers.SecurityContextDeps{Resolver: &mockDSResolver{defaultTenantID: cubeTestTenant}}
	h := NewCubeHandler(sqlxDB, deps)

	r := chi.NewRouter()
	r.Get("/api/cubes", h.HandleListCubes)
	r.Post("/api/cubes", h.HandleCreateCube)
	r.Get("/api/cubes/{id}", h.HandleGetCube)
	r.Get("/api/cubes/{id}/impact", h.HandleGetCubeImpact)
	r.Post("/api/cubes/{id}/impact/preview", h.HandlePostCubeImpactPreview)
	r.Post("/api/cubes/{id}/cascade", h.HandlePostCubeCascade)
	r.Patch("/api/cubes/{id}", h.HandlePatchCube)
	r.Post("/api/cubes/{id}/validate", h.HandleValidateCube)
	r.Post("/api/cubes/{id}/versions", h.HandlePublishCubeVersion)

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
		req.Header.Set("X-Region", "us-east-1")
		auth := security.AuthInfo{
			TenantIDs: []string{cubeTestTenant},
			UserID:    cubeTestUser,
			Roles:     []string{"tenant_admin"},
		}
		return req.WithContext(security.WithAuthInfo(req.Context(), auth))
	}
	return h, r, mock, makeReq
}

func cubeDefColumns() []string {
	return []string{
		"id", "tenant_id", "name", "description", "bo_id", "dimensions",
		"time_dimension", "metric_ids", "grains", "materialization",
		"federation", "contract_version", "content_hash", "is_core", "status",
		"archived_at", "created_by", "created_at", "updated_at",
	}
}

func sampleCubeRow(t *testing.T, id, name, hash string, version int, dims, metrics, grains, mat, fed []byte) *sqlmock.Rows {
	t.Helper()
	now := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	return sqlmock.NewRows(cubeDefColumns()).AddRow(
		id, cubeTestTenant, name, "desc", "oms.account",
		dims, nil, metrics, grains, mat, fed, version, hash,
		false, "active", nil, cubeTestUser, now, now,
	)
}

func expectGoldNil(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM (SELECT public.uisce_gold_copy_tenant_id() AS id) g WHERE id IS NOT NULL`)).
		WillReturnError(sql.ErrNoRows)
}

func expectMetricForCube(t *testing.T, mock sqlmock.Sqlmock, metricID string) {
	t.Helper()
	expr, _ := json.Marshal(MetricExpression{Kind: "aggregation", Fn: "sum", TermNodeID: "market_value"})
	rows := sqlmock.NewRows([]string{
		"id", "tenant_id", "name", "description", "bo_id", "catalog_term_id",
		"expression", "grain_allowlist", "format_config", "variables",
		"materialization_config", "decomposable", "content_hash", "tags",
		"is_core", "status", "archived_at", "created_by", "created_at", "updated_at",
	}).AddRow(
		metricID, cubeTestTenant, "market_value", nil, "oms.account",
		nil, expr, []byte(`[]`), []byte(`{}`), []byte(`[]`),
		[]byte(`{}`), true, "hash_metric_v1", pq.Array([]string{}), false,
		"active", nil, nil, time.Now().UTC(), time.Now().UTC(),
	)
	mock.ExpectQuery(`FROM data_explorer.metric_definition`).
		WithArgs(cubeTestTenant, pq.Array([]string{metricID})).
		WillReturnRows(rows)
}

func TestCubeHandler_CreateGetPatchValidate(t *testing.T) {
	_, router, mock, makeReq := setupCubeHandlerEnv(t)

	dims, _ := json.Marshal([]CubeDimension{{TermNodeID: "currency"}})
	metrics, _ := json.Marshal([]string{cubeTestMetric})
	grains, _ := json.Marshal([][]string{{"currency"}})
	mat, _ := json.Marshal(CubeMaterializationConfig{
		Strategy: "starrocks_mv", HotEngine: "starrocks", ColdEngine: "iceberg", StalePolicy: "serve_with_flag",
	})
	fed, _ := json.Marshal(CubeFederation{})

	createBody := map[string]interface{}{
		"name":      "account_snapshot",
		"boId":      "oms.account",
		"metricIds": []string{cubeTestMetric},
		"dimensions": []map[string]string{
			{"termNodeId": "currency"},
		},
		"grains": [][]string{{"currency"}},
	}
	bodyBytes, err := json.Marshal(createBody)
	require.NoError(t, err)

	// create: gold check for is_core=false path still may not call gold; create only calls isGoldCopy when IsCore
	expectMetricForCube(t, mock, cubeTestMetric)

	hash := ComputeCubeContentHash(CubeDefinition{
		Name: "account_snapshot", BOID: "oms.account", ContractVersion: 1,
		MetricIDs:  []string{cubeTestMetric},
		Dimensions: []CubeDimension{{TermNodeID: "currency"}},
		Grains:     [][]string{{"currency"}},
		Materialization: CubeMaterializationConfig{
			Strategy: "starrocks_mv", HotEngine: "starrocks", ColdEngine: "iceberg", StalePolicy: "serve_with_flag",
		},
	}, []string{"hash_metric_v1"})

	mock.ExpectQuery(`INSERT INTO data_explorer.cube_definition`).
		WillReturnRows(sampleCubeRow(t, cubeTestID, "account_snapshot", hash, 1, dims, metrics, grains, mat, fed))

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, makeReq("POST", "/api/cubes", bodyBytes))
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var created map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	var cube CubeDefinition
	require.NoError(t, json.Unmarshal(created["cube"], &cube))
	assert.Equal(t, cubeTestID, cube.ID)
	assert.Equal(t, 1, cube.ContractVersion)
	assert.Equal(t, hash, cube.ContentHash)

	// GET
	mock.ExpectQuery(`FROM data_explorer.cube_definition`).
		WithArgs(cubeTestID, cubeTestTenant).
		WillReturnRows(sampleCubeRow(t, cubeTestID, "account_snapshot", hash, 1, dims, metrics, grains, mat, fed))

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, makeReq("GET", "/api/cubes/"+cubeTestID, nil))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// PATCH no-op (same surface → same hash)
	mock.ExpectQuery(`FROM data_explorer.cube_definition`).
		WithArgs(cubeTestID, cubeTestTenant).
		WillReturnRows(sampleCubeRow(t, cubeTestID, "account_snapshot", hash, 1, dims, metrics, grains, mat, fed))
	expectMetricForCube(t, mock, cubeTestMetric)

	patchBody, _ := json.Marshal(map[string]interface{}{
		"description": "same surface",
		"metricIds":   []string{cubeTestMetric},
		"dimensions":  []map[string]string{{"termNodeId": "currency"}},
		"grains":      [][]string{{"currency"}},
	})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, makeReq("PATCH", "/api/cubes/"+cubeTestID, patchBody))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var patchResp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &patchResp))
	assert.Equal(t, true, patchResp["noop"])

	// VALIDATE 200
	mock.ExpectQuery(`FROM data_explorer.cube_definition`).
		WithArgs(cubeTestID, cubeTestTenant).
		WillReturnRows(sampleCubeRow(t, cubeTestID, "account_snapshot", hash, 1, dims, metrics, grains, mat, fed))
	expectMetricForCube(t, mock, cubeTestMetric)

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, makeReq("POST", "/api/cubes/"+cubeTestID+"/validate", nil))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var valResp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &valResp))
	assert.Equal(t, true, valResp["ok"])

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCubeHandler_ListScopes(t *testing.T) {
	_, router, mock, makeReq := setupCubeHandlerEnv(t)

	dims, _ := json.Marshal([]CubeDimension{{TermNodeID: "currency"}})
	metrics, _ := json.Marshal([]string{cubeTestMetric})
	grains, _ := json.Marshal([][]string{{"currency"}})
	mat, _ := json.Marshal(CubeMaterializationConfig{Strategy: "starrocks_mv"})
	fed, _ := json.Marshal(CubeFederation{})

	expectGoldNil(mock)
	mock.ExpectQuery(`FROM data_explorer.cube_definition`).
		WithArgs(cubeTestTenant, 51).
		WillReturnRows(sampleCubeRow(t, cubeTestID, "account_snapshot", "abc", 1, dims, metrics, grains, mat, fed))

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, makeReq("GET", "/api/cubes?scope=custom&limit=50", nil))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var listResp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &listResp))
	assert.Equal(t, "custom", listResp["scope"])
	cubes, ok := listResp["cubes"].([]interface{})
	require.True(t, ok)
	assert.Len(t, cubes, 1)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCubeHandler_PatchRejectsBreaking(t *testing.T) {
	_, router, mock, makeReq := setupCubeHandlerEnv(t)

	dims, _ := json.Marshal([]CubeDimension{{TermNodeID: "currency"}})
	metrics, _ := json.Marshal([]string{cubeTestMetric})
	grains, _ := json.Marshal([][]string{{"currency"}})
	mat, _ := json.Marshal(CubeMaterializationConfig{Strategy: "starrocks_mv"})
	fed, _ := json.Marshal(CubeFederation{})

	mock.ExpectQuery(`FROM data_explorer.cube_definition`).
		WithArgs(cubeTestID, cubeTestTenant).
		WillReturnRows(sampleCubeRow(t, cubeTestID, "account_snapshot", "abc", 1, dims, metrics, grains, mat, fed))

	body, _ := json.Marshal(map[string]interface{}{
		"grains": [][]string{{"currency", "region"}},
	})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, makeReq("PATCH", "/api/cubes/"+cubeTestID, body))
	require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), "breakReasons")
	assert.Contains(t, rec.Body.String(), "versions")

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCubeHandler_PublishVersion(t *testing.T) {
	_, router, mock, makeReq := setupCubeHandlerEnv(t)

	dims, _ := json.Marshal([]CubeDimension{{TermNodeID: "currency"}})
	metrics, _ := json.Marshal([]string{cubeTestMetric})
	grains, _ := json.Marshal([][]string{{"currency"}})
	mat, _ := json.Marshal(CubeMaterializationConfig{
		Strategy: "starrocks_mv", HotEngine: "starrocks", ColdEngine: "iceberg", StalePolicy: "serve_with_flag",
	})
	fed, _ := json.Marshal(CubeFederation{})

	mock.ExpectQuery(`FROM data_explorer.cube_definition`).
		WithArgs(cubeTestID, cubeTestTenant).
		WillReturnRows(sampleCubeRow(t, cubeTestID, "account_snapshot", "oldhash", 1, dims, metrics, grains, mat, fed))

	expectMetricForCube(t, mock, cubeTestMetric)

	newGrains, _ := json.Marshal([][]string{{"currency", "region"}})
	newDims, _ := json.Marshal([]CubeDimension{{TermNodeID: "currency"}, {TermNodeID: "region"}})
	mock.ExpectQuery(`UPDATE data_explorer.cube_definition SET`).
		WillReturnRows(sampleCubeRow(t, cubeTestID, "account_snapshot", "newhash", 2, newDims, metrics, newGrains, mat, fed))

	body, _ := json.Marshal(map[string]interface{}{
		"dimensions": []map[string]string{{"termNodeId": "currency"}, {"termNodeId": "region"}},
		"grains":     [][]string{{"currency", "region"}},
		"metricIds":  []string{cubeTestMetric},
	})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, makeReq("POST", "/api/cubes/"+cubeTestID+"/versions", body))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, float64(1), resp["previousVersion"])
	cubeRaw, _ := json.Marshal(resp["cube"])
	var cube CubeDefinition
	require.NoError(t, json.Unmarshal(cubeRaw, &cube))
	assert.Equal(t, 2, cube.ContractVersion)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCubeHandler_CursorRoundTrip(t *testing.T) {
	c := cubeCursor{
		UpdatedAt: time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC),
		ID:        uuid.MustParse(cubeTestID).String(),
	}
	enc := encodeCubeCursor(c)
	require.NotEmpty(t, enc)
	decoded, err := decodeCubeCursor(enc)
	require.NoError(t, err)
	assert.Equal(t, c.ID, decoded.ID)
	assert.True(t, c.UpdatedAt.Equal(decoded.UpdatedAt))
}
