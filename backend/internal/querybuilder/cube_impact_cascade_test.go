package querybuilder

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestRewireCubeSubjectsInJSON(t *testing.T) {
	raw := []byte(`{"subject":{"kind":"cube","cubeId":"` + cubeTestID + `","contractVersion":1},"nested":{"props":{"subject":{"kind":"cube","cubeId":"` + cubeTestID + `","contractVersion":"latest"}}}}`)
	out, changed, err := rewireCubeSubjectsInJSON(raw, cubeTestID, 2)
	require.NoError(t, err)
	assert.True(t, changed)
	assert.Contains(t, string(out), `"contractVersion":2`)
	assert.NotContains(t, string(out), `"latest"`)

	other := []byte(`{"subject":{"kind":"cube","cubeId":"other","contractVersion":1}}`)
	_, changed, err = rewireCubeSubjectsInJSON(other, cubeTestID, 2)
	require.NoError(t, err)
	assert.False(t, changed)
}

func TestCubeHandler_CascadePublish_FailClosedClean(t *testing.T) {
	h, router, mock, makeReq := setupCubeHandlerEnv(t)
	h.impactConfirmSecret = []byte("unit-test-confirm")
	router.Post("/api/cubes/{id}/cascade", h.HandlePostCubeCascade)

	dims, _ := json.Marshal([]CubeDimension{{TermNodeID: "currency"}})
	metrics, _ := json.Marshal([]string{cubeTestMetric})
	grains, _ := json.Marshal([][]string{{"currency"}})
	newGrains, _ := json.Marshal([][]string{{"currency", "region"}})
	mat, _ := json.Marshal(CubeMaterializationConfig{Strategy: "starrocks_mv"})
	fed, _ := json.Marshal(CubeFederation{})
	hash := "pub_clean"
	patch := map[string]interface{}{"grains": [][]string{{"currency", "region"}}}

	expectImpactCubeLoad(t, mock, "account_snapshot", hash, 1, dims, metrics, grains, mat, fed)
	expectEmptyImpactConsumers(mock, cubeTestID, "account_snapshot")
	mock.ExpectQuery(`FROM data_explorer.cube_definition`).
		WithArgs(cubeTestID, cubeTestTenant).
		WillReturnRows(sampleCubeRow(t, cubeTestID, "account_snapshot", hash, 1, dims, metrics, grains, mat, fed))

	previewBody, _ := json.Marshal(map[string]interface{}{"action": "publish_version", "patch": patch})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, makeReq("POST", "/api/cubes/"+cubeTestID+"/impact/preview", previewBody))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var preview CubeImpactPreviewReport
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &preview))
	require.Equal(t, CubeImpactClassBreakingContract, preview.ChangeClass)

	expectImpactCubeLoad(t, mock, "account_snapshot", hash, 1, dims, metrics, grains, mat, fed)
	expectEmptyImpactConsumers(mock, cubeTestID, "account_snapshot")
	mock.ExpectQuery(`FROM data_explorer.cube_definition`).
		WithArgs(cubeTestID, cubeTestTenant).
		WillReturnRows(sampleCubeRow(t, cubeTestID, "account_snapshot", hash, 1, dims, metrics, grains, mat, fed))
	mock.ExpectQuery(`FROM data_explorer.cube_definition`).
		WithArgs(cubeTestID, cubeTestTenant).
		WillReturnRows(sampleCubeRow(t, cubeTestID, "account_snapshot", hash, 1, dims, metrics, grains, mat, fed))
	expectMetricForCube(t, mock, cubeTestMetric)
	mock.ExpectQuery(`UPDATE data_explorer.cube_definition SET`).
		WillReturnRows(sampleCubeRow(t, cubeTestID, "account_snapshot", hash, 2, dims, metrics, newGrains, mat, fed))

	body, _ := json.Marshal(map[string]interface{}{
		"action":       "publish_version",
		"mode":         "fail_closed",
		"confirmToken": preview.ConfirmToken,
		"patch":        patch,
	})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, makeReq("POST", "/api/cubes/"+cubeTestID+"/cascade", body))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var receipt CubeCascadeReceipt
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &receipt))
	assert.Equal(t, CubeImpactClassBreakingContract, receipt.ChangeClass)
	assert.Equal(t, CubeImpactModeFailClosed, receipt.Mode)
	assert.Equal(t, 1, receipt.PreviousVersion)
	assert.Equal(t, 2, receipt.Cube.ContractVersion)
	assert.Equal(t, "published", receipt.ConsumersAffected[0].Action)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCubeHandler_CascadePublish_FailClosedBlocked(t *testing.T) {
	h, router, mock, makeReq := setupCubeHandlerEnv(t)
	h.impactConfirmSecret = []byte("unit-test-confirm")
	router.Post("/api/cubes/{id}/cascade", h.HandlePostCubeCascade)

	dims, _ := json.Marshal([]CubeDimension{{TermNodeID: "currency"}})
	metrics, _ := json.Marshal([]string{cubeTestMetric})
	grains, _ := json.Marshal([][]string{{"currency"}})
	mat, _ := json.Marshal(CubeMaterializationConfig{Strategy: "starrocks_mv"})
	fed, _ := json.Marshal(CubeFederation{})
	hash := "pub_block"
	sqID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	qs, _ := json.Marshal(map[string]interface{}{
		"subject": map[string]interface{}{"kind": "cube", "cubeId": cubeTestID, "contractVersion": 1},
	})
	patch := map[string]interface{}{"grains": [][]string{{"currency", "region"}}}

	expectConsumersWithSQ := func() {
		mock.ExpectQuery(`FROM catalog_node`).
			WithArgs(cubeTestID).
			WillReturnRows(sqlmock.NewRows([]string{"node_name", "properties"}))
		mock.ExpectQuery(`FROM data_explorer.saved_query`).
			WithArgs(cubeTestTenant, cubeTestID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "status", "query_state", "archived_at"}).
				AddRow(sqID, "Pinned SQ", "active", qs, nil))
		mock.ExpectQuery(`FROM page_definitions`).
			WithArgs(cubeTestTenant, cubeTestID, sqlmock.AnyArg()).
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
	}

	expectImpactCubeLoad(t, mock, "account_snapshot", hash, 1, dims, metrics, grains, mat, fed)
	expectConsumersWithSQ()
	mock.ExpectQuery(`FROM data_explorer.cube_definition`).
		WithArgs(cubeTestID, cubeTestTenant).
		WillReturnRows(sampleCubeRow(t, cubeTestID, "account_snapshot", hash, 1, dims, metrics, grains, mat, fed))

	previewBody, _ := json.Marshal(map[string]interface{}{"action": "publish_version", "patch": patch})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, makeReq("POST", "/api/cubes/"+cubeTestID+"/impact/preview", previewBody))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var preview CubeImpactPreviewReport
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &preview))
	require.GreaterOrEqual(t, preview.BlockingCount, 1)

	expectImpactCubeLoad(t, mock, "account_snapshot", hash, 1, dims, metrics, grains, mat, fed)
	expectConsumersWithSQ()
	mock.ExpectQuery(`FROM data_explorer.cube_definition`).
		WithArgs(cubeTestID, cubeTestTenant).
		WillReturnRows(sampleCubeRow(t, cubeTestID, "account_snapshot", hash, 1, dims, metrics, grains, mat, fed))

	body, _ := json.Marshal(map[string]interface{}{
		"action":       "publish_version",
		"mode":         "fail_closed",
		"confirmToken": preview.ConfirmToken,
		"patch":        patch,
	})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, makeReq("POST", "/api/cubes/"+cubeTestID+"/cascade", body))
	require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), "cascade blocked")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCubeHandler_CascadePublish_RewireLatest(t *testing.T) {
	h, router, mock, makeReq := setupCubeHandlerEnv(t)
	h.impactConfirmSecret = []byte("unit-test-confirm")
	router.Post("/api/cubes/{id}/cascade", h.HandlePostCubeCascade)

	var started []CubeMaterializeRequest
	h.SetCascadeSideEffects(CubeCascadeSideEffects{
		StartMaterialize: func(_ context.Context, req CubeMaterializeRequest) (string, *CubeMaterializePlan, error) {
			started = append(started, req)
			return "wf-" + strings.Join(req.Grain, "+"), &CubeMaterializePlan{GrainHash: "gh", AttemptID: "att"}, nil
		},
	})

	dims, _ := json.Marshal([]CubeDimension{{TermNodeID: "currency"}})
	metrics, _ := json.Marshal([]string{cubeTestMetric})
	grains, _ := json.Marshal([][]string{{"currency"}})
	newGrains, _ := json.Marshal([][]string{{"currency", "region"}})
	mat, _ := json.Marshal(CubeMaterializationConfig{Strategy: "starrocks_mv"})
	fed, _ := json.Marshal(CubeFederation{})
	hash := "pub_rewire"
	sqID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	qs, _ := json.Marshal(map[string]interface{}{
		"subject": map[string]interface{}{"kind": "cube", "cubeId": cubeTestID, "contractVersion": 1},
	})
	patch := map[string]interface{}{"grains": [][]string{{"currency", "region"}}}

	expectConsumersWithSQ := func() {
		mock.ExpectQuery(`FROM catalog_node`).
			WithArgs(cubeTestID).
			WillReturnRows(sqlmock.NewRows([]string{"node_name", "properties"}))
		mock.ExpectQuery(`FROM data_explorer.saved_query`).
			WithArgs(cubeTestTenant, cubeTestID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "status", "query_state", "archived_at"}).
				AddRow(sqID, "Pinned SQ", "active", qs, nil))
		mock.ExpectQuery(`FROM page_definitions`).
			WithArgs(cubeTestTenant, cubeTestID, sqlmock.AnyArg()).
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
	}

	expectImpactCubeLoad(t, mock, "account_snapshot", hash, 1, dims, metrics, grains, mat, fed)
	expectConsumersWithSQ()
	mock.ExpectQuery(`FROM data_explorer.cube_definition`).
		WithArgs(cubeTestID, cubeTestTenant).
		WillReturnRows(sampleCubeRow(t, cubeTestID, "account_snapshot", hash, 1, dims, metrics, grains, mat, fed))

	previewBody, _ := json.Marshal(map[string]interface{}{"action": "publish_version", "patch": patch})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, makeReq("POST", "/api/cubes/"+cubeTestID+"/impact/preview", previewBody))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var preview CubeImpactPreviewReport
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &preview))

	expectImpactCubeLoad(t, mock, "account_snapshot", hash, 1, dims, metrics, grains, mat, fed)
	expectConsumersWithSQ()
	mock.ExpectQuery(`FROM data_explorer.cube_definition`).
		WithArgs(cubeTestID, cubeTestTenant).
		WillReturnRows(sampleCubeRow(t, cubeTestID, "account_snapshot", hash, 1, dims, metrics, grains, mat, fed))
	mock.ExpectQuery(`FROM data_explorer.cube_definition`).
		WithArgs(cubeTestID, cubeTestTenant).
		WillReturnRows(sampleCubeRow(t, cubeTestID, "account_snapshot", hash, 1, dims, metrics, grains, mat, fed))
	expectMetricForCube(t, mock, cubeTestMetric)
	mock.ExpectQuery(`UPDATE data_explorer.cube_definition SET`).
		WillReturnRows(sampleCubeRow(t, cubeTestID, "account_snapshot", hash, 2, dims, metrics, newGrains, mat, fed))
	mock.ExpectQuery(`SELECT query_state FROM data_explorer.saved_query`).
		WithArgs(sqID, cubeTestTenant).
		WillReturnRows(sqlmock.NewRows([]string{"query_state"}).AddRow(qs))
	mock.ExpectExec(`UPDATE data_explorer.saved_query`).
		WithArgs(sqID, cubeTestTenant, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	body, _ := json.Marshal(map[string]interface{}{
		"action":       "publish_version",
		"mode":         "rewire_latest",
		"confirmToken": preview.ConfirmToken,
		"patch":        patch,
	})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, makeReq("POST", "/api/cubes/"+cubeTestID+"/cascade", body))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var receipt CubeCascadeReceipt
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &receipt))
	assert.Equal(t, CubeImpactModeRewireLatest, receipt.Mode)
	assert.Equal(t, 2, receipt.Cube.ContractVersion)
	require.Len(t, started, 1)
	assert.Equal(t, []string{"currency", "region"}, started[0].Grain)
	require.NotEmpty(t, receipt.MaterializeStarts)
	assert.Equal(t, "wf-currency+region", receipt.MaterializeStarts[0].WorkflowID)
	actions := map[string]int{}
	for _, a := range receipt.ConsumersAffected {
		actions[a.Action]++
	}
	assert.Equal(t, 1, actions["published"])
	assert.Equal(t, 1, actions["rewired"])
	assert.Equal(t, 1, actions["materialize_started"])
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCubeHandler_CascadePublish_BumpAndRefreshLeavesPins(t *testing.T) {
	h, router, mock, makeReq := setupCubeHandlerEnv(t)
	h.impactConfirmSecret = []byte("unit-test-confirm")
	router.Post("/api/cubes/{id}/cascade", h.HandlePostCubeCascade)
	h.SetCascadeSideEffects(CubeCascadeSideEffects{
		StartMaterialize: func(_ context.Context, req CubeMaterializeRequest) (string, *CubeMaterializePlan, error) {
			return "wf-bump", &CubeMaterializePlan{GrainHash: "gh"}, nil
		},
	})

	dims, _ := json.Marshal([]CubeDimension{{TermNodeID: "currency"}})
	metrics, _ := json.Marshal([]string{cubeTestMetric})
	grains, _ := json.Marshal([][]string{{"currency"}})
	newGrains, _ := json.Marshal([][]string{{"currency", "region"}})
	mat, _ := json.Marshal(CubeMaterializationConfig{Strategy: "starrocks_mv"})
	fed, _ := json.Marshal(CubeFederation{})
	hash := "pub_bump"
	sqID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	qs, _ := json.Marshal(map[string]interface{}{
		"subject": map[string]interface{}{"kind": "cube", "cubeId": cubeTestID, "contractVersion": 1},
	})
	patch := map[string]interface{}{"grains": [][]string{{"currency", "region"}}}

	expectConsumersWithSQ := func() {
		mock.ExpectQuery(`FROM catalog_node`).
			WithArgs(cubeTestID).
			WillReturnRows(sqlmock.NewRows([]string{"node_name", "properties"}))
		mock.ExpectQuery(`FROM data_explorer.saved_query`).
			WithArgs(cubeTestTenant, cubeTestID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "status", "query_state", "archived_at"}).
				AddRow(sqID, "Pinned SQ", "active", qs, nil))
		mock.ExpectQuery(`FROM page_definitions`).
			WithArgs(cubeTestTenant, cubeTestID, sqlmock.AnyArg()).
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
	}

	expectImpactCubeLoad(t, mock, "account_snapshot", hash, 1, dims, metrics, grains, mat, fed)
	expectConsumersWithSQ()
	mock.ExpectQuery(`FROM data_explorer.cube_definition`).
		WithArgs(cubeTestID, cubeTestTenant).
		WillReturnRows(sampleCubeRow(t, cubeTestID, "account_snapshot", hash, 1, dims, metrics, grains, mat, fed))

	previewBody, _ := json.Marshal(map[string]interface{}{"action": "publish_version", "patch": patch})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, makeReq("POST", "/api/cubes/"+cubeTestID+"/impact/preview", previewBody))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var preview CubeImpactPreviewReport
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &preview))

	expectImpactCubeLoad(t, mock, "account_snapshot", hash, 1, dims, metrics, grains, mat, fed)
	expectConsumersWithSQ()
	mock.ExpectQuery(`FROM data_explorer.cube_definition`).
		WithArgs(cubeTestID, cubeTestTenant).
		WillReturnRows(sampleCubeRow(t, cubeTestID, "account_snapshot", hash, 1, dims, metrics, grains, mat, fed))
	mock.ExpectQuery(`FROM data_explorer.cube_definition`).
		WithArgs(cubeTestID, cubeTestTenant).
		WillReturnRows(sampleCubeRow(t, cubeTestID, "account_snapshot", hash, 1, dims, metrics, grains, mat, fed))
	expectMetricForCube(t, mock, cubeTestMetric)
	mock.ExpectQuery(`UPDATE data_explorer.cube_definition SET`).
		WillReturnRows(sampleCubeRow(t, cubeTestID, "account_snapshot", hash, 2, dims, metrics, newGrains, mat, fed))

	body, _ := json.Marshal(map[string]interface{}{
		"action":       "publish_version",
		"mode":         "bump_and_refresh",
		"confirmToken": preview.ConfirmToken,
		"patch":        patch,
	})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, makeReq("POST", "/api/cubes/"+cubeTestID+"/cascade", body))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var receipt CubeCascadeReceipt
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &receipt))
	assert.Equal(t, CubeImpactModeBumpAndRefresh, receipt.Mode)
	actions := map[string]int{}
	for _, a := range receipt.ConsumersAffected {
		actions[a.Action]++
	}
	assert.Equal(t, 1, actions["published"])
	assert.Equal(t, 1, actions["left_pin"])
	assert.Equal(t, 1, actions["materialize_started"])
	require.NoError(t, mock.ExpectationsWereMet())
}
