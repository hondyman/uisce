package querybuilder

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCubeHandler_CascadeArchive_FailClosedClean(t *testing.T) {
	h, router, mock, makeReq := setupCubeHandlerEnv(t)
	h.impactConfirmSecret = []byte("unit-test-confirm")
	router.Post("/api/cubes/{id}/cascade", h.HandlePostCubeCascade)

	dims, _ := json.Marshal([]CubeDimension{{TermNodeID: "currency"}})
	metrics, _ := json.Marshal([]string{cubeTestMetric})
	grains, _ := json.Marshal([][]string{{"currency"}})
	mat, _ := json.Marshal(CubeMaterializationConfig{Strategy: "starrocks_mv"})
	fed, _ := json.Marshal(CubeFederation{})
	hash := "cascade_clean"

	// Mint confirm token via preview.
	expectImpactCubeLoad(t, mock, "account_snapshot", hash, 1, dims, metrics, grains, mat, fed)
	expectEmptyImpactConsumers(mock, cubeTestID, "account_snapshot")
	mock.ExpectQuery(`FROM data_explorer.cube_definition`).
		WithArgs(cubeTestID, cubeTestTenant).
		WillReturnRows(sampleCubeRow(t, cubeTestID, "account_snapshot", hash, 1, dims, metrics, grains, mat, fed))

	previewBody, _ := json.Marshal(map[string]interface{}{"action": "archive"})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, makeReq("POST", "/api/cubes/"+cubeTestID+"/impact/preview", previewBody))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var preview CubeImpactPreviewReport
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &preview))
	require.NotEmpty(t, preview.ConfirmToken)

	// Cascade: rebuild preview, authorize, archive.
	expectImpactCubeLoad(t, mock, "account_snapshot", hash, 1, dims, metrics, grains, mat, fed)
	expectEmptyImpactConsumers(mock, cubeTestID, "account_snapshot")
	mock.ExpectQuery(`FROM data_explorer.cube_definition`).
		WithArgs(cubeTestID, cubeTestTenant).
		WillReturnRows(sampleCubeRow(t, cubeTestID, "account_snapshot", hash, 1, dims, metrics, grains, mat, fed))
	mock.ExpectQuery(`FROM data_explorer.cube_definition`).
		WithArgs(cubeTestID, cubeTestTenant).
		WillReturnRows(sampleCubeRow(t, cubeTestID, "account_snapshot", hash, 1, dims, metrics, grains, mat, fed))
	mock.ExpectQuery(`UPDATE data_explorer.cube_definition SET`).
		WillReturnRows(sampleCubeRow(t, cubeTestID, "account_snapshot", hash, 1, dims, metrics, grains, mat, fed))

	body, _ := json.Marshal(map[string]interface{}{
		"action":       "archive",
		"mode":         "fail_closed",
		"confirmToken": preview.ConfirmToken,
	})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, makeReq("POST", "/api/cubes/"+cubeTestID+"/cascade", body))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var receipt CubeCascadeReceipt
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &receipt))
	assert.Equal(t, CubeImpactClassArchive, receipt.ChangeClass)
	assert.Equal(t, CubeImpactModeFailClosed, receipt.Mode)
	require.NotEmpty(t, receipt.ConsumersAffected)
	assert.Equal(t, "archived", receipt.ConsumersAffected[len(receipt.ConsumersAffected)-1].Action)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCubeHandler_CascadeArchive_FailClosedBlocked(t *testing.T) {
	h, router, mock, makeReq := setupCubeHandlerEnv(t)
	h.impactConfirmSecret = []byte("unit-test-confirm")
	router.Post("/api/cubes/{id}/cascade", h.HandlePostCubeCascade)

	dims, _ := json.Marshal([]CubeDimension{{TermNodeID: "currency"}})
	metrics, _ := json.Marshal([]string{cubeTestMetric})
	grains, _ := json.Marshal([][]string{{"currency"}})
	mat, _ := json.Marshal(CubeMaterializationConfig{Strategy: "starrocks_mv"})
	fed, _ := json.Marshal(CubeFederation{})
	hash := "cascade_block"
	schedID := "dddddddd-dddd-dddd-dddd-dddddddddddd"

	expectConsumersWithSchedule := func() {
		mock.ExpectQuery(`FROM catalog_node`).
			WithArgs(cubeTestID).
			WillReturnRows(sqlmock.NewRows([]string{"node_name", "properties"}))
		mock.ExpectQuery(`FROM data_explorer.saved_query`).
			WithArgs(cubeTestTenant, cubeTestID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "status", "query_state", "archived_at"}))
		mock.ExpectQuery(`FROM page_definitions`).
			WithArgs(cubeTestTenant, cubeTestID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "slug", "name", "status"}))
		mock.ExpectQuery(`FROM schedules`).
			WithArgs(cubeTestTenant, cubeTestID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "enabled"}).
				AddRow(schedID, "refresh-cube", true))
		mock.ExpectQuery(`FROM data_pipeline_definitions`).
			WithArgs(cubeTestTenant, cubeTestID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "is_active"}))
		mock.ExpectQuery(`FROM report_definitions`).
			WithArgs(cubeTestTenant, cubeTestID, "account_snapshot").
			WillReturnRows(sqlmock.NewRows([]string{"id", "report_key", "display_name", "legacy_cube", "subject_cube_id"}))
	}

	// Preview to mint token.
	expectImpactCubeLoad(t, mock, "account_snapshot", hash, 1, dims, metrics, grains, mat, fed)
	expectConsumersWithSchedule()
	mock.ExpectQuery(`FROM data_explorer.cube_definition`).
		WithArgs(cubeTestID, cubeTestTenant).
		WillReturnRows(sampleCubeRow(t, cubeTestID, "account_snapshot", hash, 1, dims, metrics, grains, mat, fed))

	previewBody, _ := json.Marshal(map[string]interface{}{"action": "archive"})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, makeReq("POST", "/api/cubes/"+cubeTestID+"/impact/preview", previewBody))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var preview CubeImpactPreviewReport
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &preview))
	require.Greater(t, preview.BlockingCount, 0)
	require.NotEmpty(t, preview.ConfirmToken)

	// Cascade fail_closed → 409
	expectImpactCubeLoad(t, mock, "account_snapshot", hash, 1, dims, metrics, grains, mat, fed)
	expectConsumersWithSchedule()
	mock.ExpectQuery(`FROM data_explorer.cube_definition`).
		WithArgs(cubeTestID, cubeTestTenant).
		WillReturnRows(sampleCubeRow(t, cubeTestID, "account_snapshot", hash, 1, dims, metrics, grains, mat, fed))

	body, _ := json.Marshal(map[string]interface{}{
		"action":       "archive",
		"mode":         "fail_closed",
		"confirmToken": preview.ConfirmToken,
	})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, makeReq("POST", "/api/cubes/"+cubeTestID+"/cascade", body))
	require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())

	var blocked map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &blocked))
	assert.Contains(t, string(blocked["error"]), "cascade blocked")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCubeHandler_CascadeArchive_DisableConsumers(t *testing.T) {
	h, router, mock, makeReq := setupCubeHandlerEnv(t)
	h.impactConfirmSecret = []byte("unit-test-confirm")
	router.Post("/api/cubes/{id}/cascade", h.HandlePostCubeCascade)

	dims, _ := json.Marshal([]CubeDimension{{TermNodeID: "currency"}})
	metrics, _ := json.Marshal([]string{cubeTestMetric})
	grains, _ := json.Marshal([][]string{{"currency"}})
	mat, _ := json.Marshal(CubeMaterializationConfig{Strategy: "starrocks_mv"})
	fed, _ := json.Marshal(CubeFederation{})
	hash := "cascade_disable"
	schedID := "dddddddd-dddd-dddd-dddd-dddddddddddd"
	pipeID := "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"
	sqID := "ffffffff-ffff-ffff-ffff-ffffffffffff"

	expectConsumers := func() {
		mock.ExpectQuery(`FROM catalog_node`).
			WithArgs(cubeTestID).
			WillReturnRows(sqlmock.NewRows([]string{"node_name", "properties"}))
		mock.ExpectQuery(`FROM data_explorer.saved_query`).
			WithArgs(cubeTestTenant, cubeTestID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "status", "query_state", "archived_at"}).
				AddRow(sqID, "sq-cube", "active", []byte(`{}`), nil))
		mock.ExpectQuery(`FROM page_definitions`).
			WithArgs(cubeTestTenant, cubeTestID, pq.Array([]string{"%" + sqID + "%"})).
			WillReturnRows(sqlmock.NewRows([]string{"id", "slug", "name", "status"}))
		mock.ExpectQuery(`FROM schedules`).
			WithArgs(cubeTestTenant, cubeTestID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "enabled"}).
				AddRow(schedID, "refresh-cube", true))
		mock.ExpectQuery(`FROM data_pipeline_definitions`).
			WithArgs(cubeTestTenant, cubeTestID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "is_active"}).
				AddRow(pipeID, "mat-pipe", true))
		mock.ExpectQuery(`FROM report_definitions`).
			WithArgs(cubeTestTenant, cubeTestID, "account_snapshot").
			WillReturnRows(sqlmock.NewRows([]string{"id", "report_key", "display_name", "legacy_cube", "subject_cube_id"}))
	}

	expectImpactCubeLoad(t, mock, "account_snapshot", hash, 1, dims, metrics, grains, mat, fed)
	expectConsumers()
	mock.ExpectQuery(`FROM data_explorer.cube_definition`).
		WithArgs(cubeTestID, cubeTestTenant).
		WillReturnRows(sampleCubeRow(t, cubeTestID, "account_snapshot", hash, 1, dims, metrics, grains, mat, fed))

	previewBody, _ := json.Marshal(map[string]interface{}{"action": "archive"})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, makeReq("POST", "/api/cubes/"+cubeTestID+"/impact/preview", previewBody))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var preview CubeImpactPreviewReport
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &preview))
	require.NotEmpty(t, preview.ConfirmToken)

	expectImpactCubeLoad(t, mock, "account_snapshot", hash, 1, dims, metrics, grains, mat, fed)
	expectConsumers()
	mock.ExpectQuery(`FROM data_explorer.cube_definition`).
		WithArgs(cubeTestID, cubeTestTenant).
		WillReturnRows(sampleCubeRow(t, cubeTestID, "account_snapshot", hash, 1, dims, metrics, grains, mat, fed))
	mock.ExpectQuery(`FROM data_explorer.cube_definition`).
		WithArgs(cubeTestID, cubeTestTenant).
		WillReturnRows(sampleCubeRow(t, cubeTestID, "account_snapshot", hash, 1, dims, metrics, grains, mat, fed))

	mock.ExpectExec(`UPDATE public.schedules`).
		WithArgs(schedID, cubeTestTenant, cubeTestUser).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE data_pipeline_definitions`).
		WithArgs(pipeID, cubeTestTenant).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`UPDATE data_explorer.cube_definition SET`).
		WillReturnRows(sampleCubeRow(t, cubeTestID, "account_snapshot", hash, 1, dims, metrics, grains, mat, fed))

	body, _ := json.Marshal(map[string]interface{}{
		"action":       "archive",
		"mode":         "disable_consumers",
		"confirmToken": preview.ConfirmToken,
	})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, makeReq("POST", "/api/cubes/"+cubeTestID+"/cascade", body))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var receipt CubeCascadeReceipt
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &receipt))
	assert.Equal(t, CubeImpactModeDisableConsumers, receipt.Mode)
	actions := map[string]int{}
	for _, a := range receipt.ConsumersAffected {
		actions[a.Action]++
	}
	assert.Equal(t, 1, actions["disabled"])
	assert.Equal(t, 1, actions["deactivated"])
	assert.Equal(t, 1, actions["left_pin"])
	assert.Equal(t, 1, actions["archived"])
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCubeHandler_CascadeArchive_PublishVersionRejected(t *testing.T) {
	h, router, _, makeReq := setupCubeHandlerEnv(t)
	h.impactConfirmSecret = []byte("unit-test-confirm")
	router.Post("/api/cubes/{id}/cascade", h.HandlePostCubeCascade)

	body, _ := json.Marshal(map[string]interface{}{
		"action":       "publish_version",
		"mode":         "fail_closed",
		"confirmToken": "x.y",
	})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, makeReq("POST", "/api/cubes/"+cubeTestID+"/cascade", body))
	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), "A5")
}
