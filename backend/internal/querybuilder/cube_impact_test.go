package querybuilder

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubjectPinFromQueryState(t *testing.T) {
	t.Parallel()

	assert.Nil(t, subjectPinFromQueryState(nil))
	assert.Nil(t, subjectPinFromQueryState([]byte(`{}`)))
	assert.Nil(t, subjectPinFromQueryState([]byte(`{"subject":{"kind":"business_object","boId":"x"}}`)))

	pin := subjectPinFromQueryState([]byte(`{"subject":{"kind":"cube","cubeId":"c1","contractVersion":3}}`))
	require.NotNil(t, pin)
	assert.EqualValues(t, float64(3), pin.ContractVersion)

	pinLatest := subjectPinFromQueryState([]byte(`{"subject":{"kind":"cube","cubeId":"c1","contractVersion":"latest"}}`))
	require.NotNil(t, pinLatest)
	assert.Equal(t, "latest", pinLatest.ContractVersion)
}

func TestCubeHandler_GetImpact_Inventory(t *testing.T) {
	_, router, mock, makeReq := setupCubeHandlerEnv(t)

	dims, _ := json.Marshal([]CubeDimension{{TermNodeID: "currency"}})
	metrics, _ := json.Marshal([]string{cubeTestMetric})
	grains, _ := json.Marshal([][]string{{"currency"}})
	mat, _ := json.Marshal(CubeMaterializationConfig{Strategy: "starrocks_mv"})
	fed, _ := json.Marshal(CubeFederation{})
	hash := "impact_hash_v1"

	sqID := "cccccccc-cccc-cccc-cccc-cccccccccccc"
	pageID := "dddddddd-dddd-dddd-dddd-dddddddddddd"
	schedID := "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"
	pipeID := "ffffffff-ffff-ffff-ffff-ffffffffffff"
	reportID := "99999999-9999-9999-9999-999999999999"

	mock.ExpectQuery(`FROM data_explorer.cube_definition`).
		WithArgs(cubeTestID, cubeTestTenant).
		WillReturnRows(sampleCubeRow(t, cubeTestID, "account_snapshot", hash, 2, dims, metrics, grains, mat, fed))
	expectMetricForCube(t, mock, cubeTestMetric)

	props, _ := json.Marshal(map[string]interface{}{
		"cube_id":               cubeTestID,
		"grain_hash":            "gh1",
		"grain":                 []string{"currency"},
		"lifecycle_status":      "active",
		"iceberg_table":         "lakekeeper_iceberg.cubes.account_snapshot",
		"attempt_id":            "att-1",
		"contract_version":      2,
		"dual_commit_watermark": "2026-10-05T12:00:00Z",
	})
	mock.ExpectQuery(`FROM catalog_node`).
		WithArgs(cubeTestID).
		WillReturnRows(sqlmock.NewRows([]string{"node_name", "properties"}).
			AddRow("pre_aggregation/cube_account_currency", props))

	qs, _ := json.Marshal(map[string]interface{}{
		"subject": map[string]interface{}{
			"kind":            "cube",
			"cubeId":          cubeTestID,
			"contractVersion": 2,
		},
	})
	mock.ExpectQuery(`FROM data_explorer.saved_query`).
		WithArgs(cubeTestTenant, cubeTestID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "status", "query_state", "archived_at"}).
			AddRow(sqID, "MV by currency", "active", qs, nil))

	mock.ExpectQuery(`FROM page_definitions`).
		WithArgs(cubeTestTenant, cubeTestID, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "slug", "name", "status"}).
			AddRow(pageID, "desk-overview", "Desk Overview", "published"))

	mock.ExpectQuery(`FROM schedules`).
		WithArgs(cubeTestTenant, cubeTestID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "enabled"}).
			AddRow(schedID, "Nightly cube refresh", true))

	mock.ExpectQuery(`FROM data_pipeline_definitions`).
		WithArgs(cubeTestTenant, cubeTestID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "is_active"}).
			AddRow(pipeID, "Materialize account cube", true))

	mock.ExpectQuery(`FROM report_definitions`).
		WithArgs(cubeTestTenant, cubeTestID, "account_snapshot").
		WillReturnRows(sqlmock.NewRows([]string{"id", "report_key", "display_name", "legacy_cube", "subject_cube_id"}).
			AddRow(reportID, "acct_mv", "Account MV", "", cubeTestID))

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, makeReq("GET", "/api/cubes/"+cubeTestID+"/impact", nil))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var report CubeImpactReport
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &report))
	assert.Equal(t, cubeTestID, report.CubeID)
	assert.Equal(t, "account_snapshot", report.Composition.Name)
	assert.Equal(t, 2, report.Composition.ContractVersion)
	assert.Equal(t, hash, report.Composition.ContentHash)
	require.Len(t, report.Composition.Metrics, 1)
	assert.Equal(t, "market_value", report.Composition.Metrics[0].Name)
	require.NotNil(t, report.Composition.Physical)
	require.Len(t, report.Composition.Physical.Grains, 1)
	assert.Equal(t, "active", report.Composition.Physical.Grains[0].LifecycleStatus)
	assert.Equal(t, 5, report.Summary.ConsumerCount)
	assert.Equal(t, 5, report.Summary.BlockingCount)
	assert.Equal(t, 1, report.Summary.PhysicalGrainCount)

	kinds := map[string]bool{}
	for _, c := range report.Consumers {
		kinds[c.Kind] = true
		assert.True(t, c.Blocking, "consumer %s/%s should block", c.Kind, c.ID)
	}
	assert.True(t, kinds["saved_query"])
	assert.True(t, kinds["page_tile"])
	assert.True(t, kinds["schedule"])
	assert.True(t, kinds["pipeline"])
	assert.True(t, kinds["report"])

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCubeHandler_GetImpact_SkipPhysical(t *testing.T) {
	_, router, mock, makeReq := setupCubeHandlerEnv(t)

	dims, _ := json.Marshal([]CubeDimension{{TermNodeID: "currency"}})
	metrics, _ := json.Marshal([]string{cubeTestMetric})
	grains, _ := json.Marshal([][]string{{"currency"}})
	mat, _ := json.Marshal(CubeMaterializationConfig{Strategy: "starrocks_mv"})
	fed, _ := json.Marshal(CubeFederation{})

	mock.ExpectQuery(`FROM data_explorer.cube_definition`).
		WithArgs(cubeTestID, cubeTestTenant).
		WillReturnRows(sampleCubeRow(t, cubeTestID, "account_snapshot", "h", 1, dims, metrics, grains, mat, fed))
	expectMetricForCube(t, mock, cubeTestMetric)

	mock.ExpectQuery(`FROM data_explorer.saved_query`).
		WithArgs(cubeTestTenant, cubeTestID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "status", "query_state", "archived_at"}))
	mock.ExpectQuery(`FROM page_definitions`).
		WithArgs(cubeTestTenant, cubeTestID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "slug", "name", "status"}))
	mock.ExpectQuery(`FROM schedules`).
		WithArgs(cubeTestTenant, cubeTestID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "enabled"}))
	mock.ExpectQuery(`FROM data_pipeline_definitions`).
		WithArgs(cubeTestTenant, cubeTestID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "is_active"}))
	mock.ExpectQuery(`FROM report_definitions`).
		WithArgs(cubeTestTenant, cubeTestID, "account_snapshot").
		WillReturnRows(sqlmock.NewRows([]string{"id", "report_key", "display_name", "legacy_cube", "subject_cube_id"}))

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, makeReq("GET", "/api/cubes/"+cubeTestID+"/impact?includePhysical=0", nil))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var report CubeImpactReport
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &report))
	assert.Nil(t, report.Composition.Physical)
	assert.Equal(t, 0, report.Summary.PhysicalGrainCount)
	assert.Equal(t, 0, report.Summary.ConsumerCount)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCubeHandler_GetImpact_404(t *testing.T) {
	_, router, mock, makeReq := setupCubeHandlerEnv(t)

	mock.ExpectQuery(`FROM data_explorer.cube_definition`).
		WithArgs(cubeTestID, cubeTestTenant).
		WillReturnRows(sqlmock.NewRows(cubeDefColumns())) // empty → sql.ErrNoRows via GetContext
	expectGoldNil(mock)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, makeReq("GET", "/api/cubes/"+cubeTestID+"/impact", nil))
	require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), "cube not found")

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCubeHandler_GetImpact_WarningConsumers(t *testing.T) {
	_, router, mock, makeReq := setupCubeHandlerEnv(t)

	dims, _ := json.Marshal([]CubeDimension{{TermNodeID: "currency"}})
	metrics, _ := json.Marshal([]string{})
	grains, _ := json.Marshal([][]string{{"currency"}})
	mat, _ := json.Marshal(CubeMaterializationConfig{Strategy: "starrocks_mv"})
	fed, _ := json.Marshal(CubeFederation{})

	// Empty metric_ids → loadMetrics short-circuits (no metric query).
	now := sampleCubeRow(t, cubeTestID, "account_snapshot", "h", 1, dims, metrics, grains, mat, fed)

	mock.ExpectQuery(`FROM data_explorer.cube_definition`).
		WithArgs(cubeTestID, cubeTestTenant).
		WillReturnRows(now)

	mock.ExpectQuery(`FROM catalog_node`).
		WithArgs(cubeTestID).
		WillReturnRows(sqlmock.NewRows([]string{"node_name", "properties"}))

	mock.ExpectQuery(`FROM data_explorer.saved_query`).
		WithArgs(cubeTestTenant, cubeTestID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "status", "query_state", "archived_at"}).
			AddRow("sq-draft", "Draft SQ", "draft", []byte(`{}`), nil))

	mock.ExpectQuery(`FROM page_definitions`).
		WithArgs(cubeTestTenant, cubeTestID, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "slug", "name", "status"}).
			AddRow("pg-draft", "draft-page", "Draft Page", "draft"))

	mock.ExpectQuery(`FROM schedules`).
		WithArgs(cubeTestTenant, cubeTestID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "enabled"}).
			AddRow("sch-off", "Paused refresh", false))

	mock.ExpectQuery(`FROM data_pipeline_definitions`).
		WithArgs(cubeTestTenant, cubeTestID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "is_active"}).
			AddRow("pipe-off", "Inactive pipe", false))

	mock.ExpectQuery(`FROM report_definitions`).
		WithArgs(cubeTestTenant, cubeTestID, "account_snapshot").
		WillReturnRows(sqlmock.NewRows([]string{"id", "report_key", "display_name", "legacy_cube", "subject_cube_id"}).
			AddRow("rep-leg", "legacy_r", "Legacy Report", "account_snapshot", ""))

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, makeReq("GET", "/api/cubes/"+cubeTestID+"/impact", nil))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var report CubeImpactReport
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &report))
	assert.Equal(t, 5, report.Summary.ConsumerCount)
	assert.Equal(t, 0, report.Summary.BlockingCount)
	assert.Equal(t, 5, report.Summary.WarningCount)
	for _, c := range report.Consumers {
		assert.False(t, c.Blocking)
		assert.Equal(t, "warning", c.Severity)
	}

	require.NoError(t, mock.ExpectationsWereMet())
}
