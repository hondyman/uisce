package querybuilder

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func expectEmptyImpactConsumers(mock sqlmock.Sqlmock, cubeID, name string) {
	mock.ExpectQuery(`FROM catalog_node`).
		WithArgs(cubeID).
		WillReturnRows(sqlmock.NewRows([]string{"node_name", "properties"}))
	mock.ExpectQuery(`FROM data_explorer.saved_query`).
		WithArgs(cubeTestTenant, cubeID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "status", "query_state", "archived_at"}))
	mock.ExpectQuery(`FROM page_definitions`).
		WithArgs(cubeTestTenant, cubeID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "slug", "name", "status"}))
	mock.ExpectQuery(`FROM schedules`).
		WithArgs(cubeTestTenant, cubeID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "enabled"}))
	mock.ExpectQuery(`FROM data_pipeline_definitions`).
		WithArgs(cubeTestTenant, cubeID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "is_active"}))
	mock.ExpectQuery(`FROM report_definitions`).
		WithArgs(cubeTestTenant, cubeID, name).
		WillReturnRows(sqlmock.NewRows([]string{"id", "report_key", "display_name", "legacy_cube", "subject_cube_id"}))
}

func expectImpactCubeLoad(t *testing.T, mock sqlmock.Sqlmock, name, hash string, version int, dims, metrics, grains, mat, fed []byte) {
	t.Helper()
	mock.ExpectQuery(`FROM data_explorer.cube_definition`).
		WithArgs(cubeTestID, cubeTestTenant).
		WillReturnRows(sampleCubeRow(t, cubeTestID, name, hash, version, dims, metrics, grains, mat, fed))
	expectMetricForCube(t, mock, cubeTestMetric)
}

func TestMintVerifyCubeImpactConfirmToken(t *testing.T) {
	t.Parallel()
	key := []byte("test-impact-confirm-secret")
	claims := cubeImpactConfirmClaims{
		CubeID:      cubeTestID,
		TenantID:    cubeTestTenant,
		Action:      CubeImpactActionArchive,
		ContentHash: "abc",
		PatchHash:   "def",
		ChangeClass: CubeImpactClassArchive,
		Exp:         time.Now().UTC().Add(5 * time.Minute).Unix(),
	}
	tok, err := mintCubeImpactConfirmToken(key, claims)
	require.NoError(t, err)
	got, err := VerifyCubeImpactConfirmToken(key, tok, cubeTestTenant, cubeTestID, CubeImpactActionArchive, "abc", "def")
	require.NoError(t, err)
	assert.Equal(t, CubeImpactClassArchive, got.ChangeClass)

	_, err = VerifyCubeImpactConfirmToken(key, tok, cubeTestTenant, cubeTestID, CubeImpactActionPatch, "abc", "def")
	assert.Error(t, err)

	_, err = VerifyCubeImpactConfirmToken([]byte("other"), tok, cubeTestTenant, cubeTestID, CubeImpactActionArchive, "abc", "def")
	assert.Error(t, err)

	expired := claims
	expired.Exp = time.Now().UTC().Add(-time.Minute).Unix()
	bad, err := mintCubeImpactConfirmToken(key, expired)
	require.NoError(t, err)
	_, err = VerifyCubeImpactConfirmToken(key, bad, cubeTestTenant, cubeTestID, CubeImpactActionArchive, "abc", "def")
	assert.Error(t, err)
}

func TestReclassifyConsumersForPreview(t *testing.T) {
	t.Parallel()
	en := true
	consumers := []CubeImpactConsumer{
		{Kind: "saved_query", ID: "1", Severity: "blocking", Blocking: true, Pin: &struct {
			ContractVersion interface{} `json:"contractVersion,omitempty"`
		}{ContractVersion: float64(1)}},
		{Kind: "schedule", ID: "2", Severity: "blocking", Blocking: true, Enabled: &en},
		{Kind: "pipeline", ID: "3", Severity: "blocking", Blocking: true, Enabled: &en},
		{Kind: "page_tile", ID: "4", Severity: "blocking", Blocking: true, Detail: "page references cube"},
		{Kind: "report", ID: "5", Severity: "warning", Blocking: false, Detail: "legacy dataBindings.primary.cube name=x"},
	}

	out := reclassifyConsumersForPreview(CubeImpactActionPublishVersion, CubeImpactClassBreakingContract, consumers, 2)
	assert.True(t, out[0].Blocking)
	assert.False(t, out[1].Blocking)
	assert.Equal(t, "warning", out[1].Severity)
	assert.False(t, out[2].Blocking)
	assert.True(t, out[3].Blocking)
	assert.False(t, out[4].Blocking)
	assert.Equal(t, "info", out[4].Severity)

	info := reclassifyConsumersForPreview(CubeImpactActionPatch, CubeImpactClassNonBreakingPatch, consumers, 1)
	for _, c := range info {
		assert.False(t, c.Blocking)
		assert.Equal(t, "info", c.Severity)
	}
}

func TestCubeHandler_ImpactPreview_Archive(t *testing.T) {
	h, router, mock, makeReq := setupCubeHandlerEnv(t)
	h.impactConfirmSecret = []byte("unit-test-confirm")

	dims, _ := json.Marshal([]CubeDimension{{TermNodeID: "currency"}})
	metrics, _ := json.Marshal([]string{cubeTestMetric})
	grains, _ := json.Marshal([][]string{{"currency"}})
	mat, _ := json.Marshal(CubeMaterializationConfig{Strategy: "starrocks_mv"})
	fed, _ := json.Marshal(CubeFederation{})
	hash := "preview_hash"

	expectImpactCubeLoad(t, mock, "account_snapshot", hash, 1, dims, metrics, grains, mat, fed)
	expectEmptyImpactConsumers(mock, cubeTestID, "account_snapshot")
	// BuildCubeImpactPreview reloads cube for draft base.
	mock.ExpectQuery(`FROM data_explorer.cube_definition`).
		WithArgs(cubeTestID, cubeTestTenant).
		WillReturnRows(sampleCubeRow(t, cubeTestID, "account_snapshot", hash, 1, dims, metrics, grains, mat, fed))

	body, _ := json.Marshal(map[string]interface{}{"action": "archive"})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, makeReq("POST", "/api/cubes/"+cubeTestID+"/impact/preview", body))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var report CubeImpactPreviewReport
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &report))
	assert.Equal(t, CubeImpactClassArchive, report.ChangeClass)
	assert.Empty(t, report.BreakReasons)
	assert.Equal(t, 0, report.BlockingCount)
	assert.Equal(t, []string{CubeImpactModeFailClosed, CubeImpactModeDisableConsumers}, report.AllowedModes)
	assert.Equal(t, CubeImpactModeFailClosed, report.RecommendedMode)
	require.NotEmpty(t, report.ConfirmToken)

	_, err := VerifyCubeImpactConfirmToken(h.impactConfirmSecret, report.ConfirmToken, cubeTestTenant, cubeTestID, CubeImpactActionArchive, hash, report.PatchHash)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCubeHandler_ImpactPreview_BreakingPublish(t *testing.T) {
	h, router, mock, makeReq := setupCubeHandlerEnv(t)
	h.impactConfirmSecret = []byte("unit-test-confirm")

	dims, _ := json.Marshal([]CubeDimension{{TermNodeID: "currency"}})
	metrics, _ := json.Marshal([]string{cubeTestMetric})
	grains, _ := json.Marshal([][]string{{"currency"}})
	mat, _ := json.Marshal(CubeMaterializationConfig{Strategy: "starrocks_mv"})
	fed, _ := json.Marshal(CubeFederation{})
	hash := "preview_break"

	sqID := "cccccccc-cccc-cccc-cccc-cccccccccccc"
	qs, _ := json.Marshal(map[string]interface{}{
		"subject": map[string]interface{}{"kind": "cube", "cubeId": cubeTestID, "contractVersion": 1},
	})

	expectImpactCubeLoad(t, mock, "account_snapshot", hash, 1, dims, metrics, grains, mat, fed)
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
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "enabled"}).
			AddRow("sch-1", "Refresh", true))
	mock.ExpectQuery(`FROM data_pipeline_definitions`).
		WithArgs(cubeTestTenant, cubeTestID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "is_active"}))
	mock.ExpectQuery(`FROM report_definitions`).
		WithArgs(cubeTestTenant, cubeTestID, "account_snapshot").
		WillReturnRows(sqlmock.NewRows([]string{"id", "report_key", "display_name", "legacy_cube", "subject_cube_id"}))

	mock.ExpectQuery(`FROM data_explorer.cube_definition`).
		WithArgs(cubeTestID, cubeTestTenant).
		WillReturnRows(sampleCubeRow(t, cubeTestID, "account_snapshot", hash, 1, dims, metrics, grains, mat, fed))

	body, _ := json.Marshal(map[string]interface{}{
		"action": "publish_version",
		"patch": map[string]interface{}{
			"grains": [][]string{{"currency", "region"}},
		},
	})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, makeReq("POST", "/api/cubes/"+cubeTestID+"/impact/preview", body))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var report CubeImpactPreviewReport
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &report))
	assert.Equal(t, CubeImpactClassBreakingContract, report.ChangeClass)
	assert.Contains(t, report.BreakReasons, string(CubeBreakGrainChange))
	assert.Equal(t, 2, report.NextVersion)
	assert.Equal(t, []string{CubeImpactModeFailClosed, CubeImpactModeRewireLatest, CubeImpactModeBumpAndRefresh}, report.AllowedModes)
	assert.GreaterOrEqual(t, report.BlockingCount, 1)

	var sq *CubeImpactConsumer
	var sch *CubeImpactConsumer
	for i := range report.Consumers {
		c := &report.Consumers[i]
		if c.Kind == "saved_query" {
			sq = c
		}
		if c.Kind == "schedule" {
			sch = c
		}
	}
	require.NotNil(t, sq)
	assert.True(t, sq.Blocking)
	require.NotNil(t, sch)
	assert.False(t, sch.Blocking)
	assert.Equal(t, "warning", sch.Severity)

	_, err := VerifyCubeImpactConfirmToken(h.impactConfirmSecret, report.ConfirmToken, cubeTestTenant, cubeTestID, CubeImpactActionPublishVersion, hash, report.PatchHash)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCubeHandler_ImpactPreview_NonBreakingPatch(t *testing.T) {
	h, router, mock, makeReq := setupCubeHandlerEnv(t)
	h.impactConfirmSecret = []byte("unit-test-confirm")

	dims, _ := json.Marshal([]CubeDimension{{TermNodeID: "currency"}})
	metrics, _ := json.Marshal([]string{cubeTestMetric})
	grains, _ := json.Marshal([][]string{{"currency"}})
	mat, _ := json.Marshal(CubeMaterializationConfig{Strategy: "starrocks_mv"})
	fed, _ := json.Marshal(CubeFederation{})
	hash := "preview_patch"

	expectImpactCubeLoad(t, mock, "account_snapshot", hash, 1, dims, metrics, grains, mat, fed)
	expectEmptyImpactConsumers(mock, cubeTestID, "account_snapshot")
	mock.ExpectQuery(`FROM data_explorer.cube_definition`).
		WithArgs(cubeTestID, cubeTestTenant).
		WillReturnRows(sampleCubeRow(t, cubeTestID, "account_snapshot", hash, 1, dims, metrics, grains, mat, fed))

	body, _ := json.Marshal(map[string]interface{}{
		"action": "patch",
		"patch":  map[string]interface{}{"description": "docs only"},
	})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, makeReq("POST", "/api/cubes/"+cubeTestID+"/impact/preview", body))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var report CubeImpactPreviewReport
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &report))
	assert.Equal(t, CubeImpactClassNonBreakingPatch, report.ChangeClass)
	assert.Empty(t, report.BreakReasons)
	assert.Equal(t, []string{CubeImpactModeFailClosed}, report.AllowedModes)
	assert.Equal(t, 0, report.NextVersion)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCubeHandler_ImpactPreview_PublishRequiresBreak(t *testing.T) {
	h, router, mock, makeReq := setupCubeHandlerEnv(t)
	h.impactConfirmSecret = []byte("unit-test-confirm")

	dims, _ := json.Marshal([]CubeDimension{{TermNodeID: "currency"}})
	metrics, _ := json.Marshal([]string{cubeTestMetric})
	grains, _ := json.Marshal([][]string{{"currency"}})
	mat, _ := json.Marshal(CubeMaterializationConfig{Strategy: "starrocks_mv"})
	fed, _ := json.Marshal(CubeFederation{})
	hash := "preview_nobreak"

	expectImpactCubeLoad(t, mock, "account_snapshot", hash, 1, dims, metrics, grains, mat, fed)
	expectEmptyImpactConsumers(mock, cubeTestID, "account_snapshot")
	mock.ExpectQuery(`FROM data_explorer.cube_definition`).
		WithArgs(cubeTestID, cubeTestTenant).
		WillReturnRows(sampleCubeRow(t, cubeTestID, "account_snapshot", hash, 1, dims, metrics, grains, mat, fed))

	body, _ := json.Marshal(map[string]interface{}{
		"action": "publish_version",
		"patch":  map[string]interface{}{"description": "no break"},
	})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, makeReq("POST", "/api/cubes/"+cubeTestID+"/impact/preview", body))
	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), "not a breaking change")
	require.NoError(t, mock.ExpectationsWereMet())
}
