package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	// services not required here - we use metadata.CatalogScanService in tests
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/hondyman/semlayer/backend/internal/db"
	"github.com/hondyman/semlayer/backend/internal/metadata"
	"github.com/hondyman/semlayer/backend/models"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

// fakeMetadataScanner satisfies services.MetadataScanner for testing
type fakeMetadataScanner struct{}

func (f *fakeMetadataScanner) ExtractMetadata() ([]*models.CatalogNode, []models.CatalogEdge, error) {
	return nil, nil, nil
}

// fakeScanService implements scanServiceIface for testing
type fakeScanService struct {
	results []metadata.ScanResult
	err     error
}

func (f *fakeScanService) ScanDatasources(ctx context.Context, tenantDatasourceID *uuid.UUID) ([]metadata.ScanResult, error) {
	return f.results, f.err
}

func (f *fakeScanService) ScanWithProgress(ctx context.Context, tenantDatasourceID *uuid.UUID, progress chan<- models.ScanProgress) ([]metadata.ScanResult, error) {
	return f.results, f.err
}

func TestHandleCatalogScan_AllSuccess(t *testing.T) {
	fake := &fakeScanService{
		results: []metadata.ScanResult{{DatasourceID: uuid.New(), Name: "ds1", Success: true}},
		err:     nil,
	}

	h := NewCatalogScanHandler(fake)

	// call handler
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/catalog/scan", nil)
	h.HandleCatalogScan(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	assert.NoError(t, err)
	assert.Equal(t, "All datasource scans completed successfully", body["message"])
	// results present
	r := body["results"].([]interface{})
	assert.Len(t, r, 1)
}

func TestHandleCatalogScan_PartialFailure(t *testing.T) {
	fake := &fakeScanService{
		results: []metadata.ScanResult{
			{DatasourceID: uuid.New(), Name: "ds1", Success: true},
			{DatasourceID: uuid.New(), Name: "ds2", Success: false, Error: "conn failed"},
		},
		err: nil,
	}

	h := NewCatalogScanHandler(fake)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/catalog/scan", nil)
	h.HandleCatalogScan(w, req)

	assert.Equal(t, http.StatusOK, w.Code) // Updated to expect 200 for Hasura compatibility
	var body map[string]interface{}
	e := json.Unmarshal(w.Body.Bytes(), &body)
	assert.NoError(t, e)
	assert.Equal(t, "partial", body["status"])
	// results present
	r := body["results"].([]interface{})
	assert.Len(t, r, 2)
}

func TestHandleCatalogScan_AllFailure(t *testing.T) {
	fake := &fakeScanService{
		results: []metadata.ScanResult{
			{DatasourceID: uuid.New(), Name: "ds1", Success: false, Error: "boom"},
		},
	}
	// handler should respond 200 (not 500) because service indicates all failed via returned error
	// But it returns 200 for Hasura compatibility even if all failed.
	fake2 := &fakeScanService{results: fake.results, err: assert.AnError}
	h2 := NewCatalogScanHandler(fake2)

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, "/api/catalog/scan", nil)
	h2.HandleCatalogScan(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code) // Updated to expect 200 for Hasura compatibility
	var body map[string]interface{}
	e := json.Unmarshal(w2.Body.Bytes(), &body)
	assert.NoError(t, e)
	// results included
	assert.NotNil(t, body["results"])
}

func TestHandleCatalogScan_InvokesUpsert_EndToEnd(t *testing.T) {
	// Initialize alphaDB with a sqlmock connection that won't panic on ExecContext
	mockDB, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer mockDB.Close()

	// Build a real service with hooks; we'll wrap it in a fake that calls through.
	svc := metadata.NewCatalogScanService(sqlx.NewDb(mockDB, "postgres"), nil)

	// Expect the status update in scanSingleDatasource
	mock.ExpectExec("UPDATE public.tenant_product_datasource").
		WithArgs("running", sqlmock.AnyArg(), "", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Expect the status update back to "complete" (or wherever it ends)
	mock.ExpectExec("UPDATE public.tenant_product_datasource").
		WithArgs("complete", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Inject target DB connector to avoid real DB connections
	svc.SetConnectTargetDBFunc(func(ctx context.Context, connectionDetails string) (*sql.DB, error) {
		return sql.Open("pgx", "postgres://u:p@localhost:5432/d?sslmode=disable")
	})

	// Inject storage no-op
	svc.SetStoreFunc(func(ctx context.Context, datasourceID uuid.UUID, nodes []*models.CatalogNode, edges []models.CatalogEdge, progress chan<- models.ScanProgress) (int64, int64, int64, error) {
		return 0, 0, 0, nil
	})

	// Provide one datasource via injected getDatasources
	tenantID := uuid.New()
	dsID := uuid.New()
	svc.SetGetDatasourcesFunc(func(tenantDatasourceID *uuid.UUID) ([]metadata.DatasourceConfig, error) {
		return []metadata.DatasourceConfig{{
			ID: dsID, TenantID: tenantID, Name: "http-ds", SourceSystem: "pg", ConnectionDetails: `{"auth":{"basic":{"username":"u","password":"p"}},"host":"h","port":5432,"database":"d"}`, IsGoldCopy: false,
		}}, nil
	})

	// Override chart builders to no-op
	restoreCharts := metadata.SetChartBuilderHooks(
		func(ctx context.Context, db *sql.DB, datasourceID string, isGoldCopy bool) error { return nil },
		func(ctx context.Context, db *sql.DB, datasourceID string, isGoldCopy bool) error { return nil },
		func(ctx context.Context, db *sql.DB, datasourceID string, isGoldCopy bool) error { return nil },
		func(ctx context.Context, db *sql.DB, datasourceID string, isGoldCopy bool) error { return nil },
	)
	defer restoreCharts()

	// Fake metadata scanner via global hook
	restoreScanner := metadata.SetNewMetadataScannerHook(func(targetDB *sql.DB, tenantID, datasourceID uuid.UUID, sourceSystem string, goldCopyNodes map[string]db.GoldCopyNodeInfo, isGoldCopy bool, schemaWhitelist []string) (metadata.MetadataScanner, error) {
		return &fakeMetadataScanner{}, nil
	})
	defer restoreScanner()

	// Capture upsert invocation via hook
	called := false
	restoreUpsert := metadata.SetUpsertBusinessTermsHook(func(ctx context.Context, db *sqlx.DB, tenant uuid.UUID, ds uuid.UUID, bt uuid.UUID) (int64, error) {
		called = true
		return 2, nil
	})
	defer restoreUpsert()

	// Ensure service uses its own scanSingleDatasource (on the injected instance) so our hooks are honored
	svc.SetScanFunc(func(ctx context.Context, ds metadata.DatasourceConfig, goldCopyNodes map[string]db.GoldCopyNodeInfo) (*metadata.ScanResult, error) {
		return svc.ScanSingleDatasourceForTest(ctx, ds, goldCopyNodes)
	})

	h := NewCatalogScanHandler(svc)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/catalog/scan", nil)
	h.HandleCatalogScan(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called, "expected Business Term upsert to be invoked via service during HTTP scan")
}
