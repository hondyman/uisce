package metadata

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/db"
	"github.com/hondyman/semlayer/backend/models"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

func TestScanDatasources_AllSuccess(t *testing.T) {
	svc := &CatalogScanService{}
	// Provide a non-nil alphaDB to avoid nil deref when calling chart builders
	svc.alphaDB = sqlx.NewDb(&sql.DB{}, "pgx")

	// fake getDatasourcesFunc returns two datasources
	dsid1 := uuid.New()
	dsid2 := uuid.New()
	svc.getDatasourcesFunc = func(tenantDatasourceID *uuid.UUID) ([]DatasourceConfig, error) {
		return []DatasourceConfig{
			{ID: dsid1, Name: "ds1"},
			{ID: dsid2, Name: "ds2"},
		}, nil
	}

	// fake scanFunc always succeeds
	svc.scanFunc = func(ctx context.Context, ds DatasourceConfig, goldCopyNodes map[string]db.GoldCopyNodeInfo) (*ScanResult, error) {
		return &ScanResult{Success: true}, nil
	}

	results, err := svc.ScanDatasources(context.Background(), nil)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	for _, r := range results {
		assert.True(t, r.Success)
	}
}

func TestScanDatasources_PartialFailure(t *testing.T) {
	svc := &CatalogScanService{}
	dsid1 := uuid.New()
	dsid2 := uuid.New()
	svc.getDatasourcesFunc = func(tenantDatasourceID *uuid.UUID) ([]DatasourceConfig, error) {
		return []DatasourceConfig{
			{ID: dsid1, Name: "ds1"},
			{ID: dsid2, Name: "ds2"},
		}, nil
	}

	// fail second
	svc.scanFunc = func(ctx context.Context, ds DatasourceConfig, goldCopyNodes map[string]db.GoldCopyNodeInfo) (*ScanResult, error) {
		if ds.ID == dsid2 {
			return nil, assert.AnError
		}
		return &ScanResult{Success: true}, nil
	}

	results, err := svc.ScanDatasources(context.Background(), nil)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	var successes, failures int
	for _, r := range results {
		if r.Success {
			successes++
		} else {
			failures++
		}
	}
	assert.Equal(t, 1, successes)
	assert.Equal(t, 1, failures)
}

func TestScanDatasources_AllFailure(t *testing.T) {
	svc := &CatalogScanService{}
	dsid1 := uuid.New()
	dsid2 := uuid.New()
	svc.getDatasourcesFunc = func(tenantDatasourceID *uuid.UUID) ([]DatasourceConfig, error) {
		return []DatasourceConfig{
			{ID: dsid1, Name: "ds1"},
			{ID: dsid2, Name: "ds2"},
		}, nil
	}

	// always fail
	svc.scanFunc = func(ctx context.Context, ds DatasourceConfig, goldCopyNodes map[string]db.GoldCopyNodeInfo) (*ScanResult, error) {
		return nil, assert.AnError
	}

	results, err := svc.ScanDatasources(context.Background(), nil)
	assert.Error(t, err)
	assert.Len(t, results, 2)
	for _, r := range results {
		assert.False(t, r.Success)
	}
}

func TestScanDatasources_TimeoutAndConcurrency(t *testing.T) {
	svc := &CatalogScanService{}

	// create 20 fake datasources
	var datas []DatasourceConfig
	for i := 0; i < 20; i++ {
		datas = append(datas, DatasourceConfig{ID: uuid.New(), Name: fmt.Sprintf("ds-%d", i)})
	}

	svc.getDatasourcesFunc = func(tenantDatasourceID *uuid.UUID) ([]DatasourceConfig, error) {
		return datas, nil
	}

	// scanFunc sleeps 200ms per datasource but honors ctx cancellation
	svc.scanFunc = func(ctx context.Context, ds DatasourceConfig, goldCopyNodes map[string]db.GoldCopyNodeInfo) (*ScanResult, error) {
		select {
		case <-time.After(200 * time.Millisecond):
			return &ScanResult{Success: true}, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// timeout context shorter than total work so some will be cancelled
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	results, err := svc.ScanDatasources(ctx, nil)
	// Some scans should have been cancelled; overall should not return error as some may succeed
	if err != nil {
		t.Logf("ScanDatasources returned error: %v", err)
	}
	var successes, failures int
	for _, r := range results {
		if r.Success {
			successes++
		} else {
			failures++
		}
	}
	t.Logf("concurrency test: successes=%d failures=%d", successes, failures)
	assert.Greater(t, successes, 0)
	assert.Greater(t, failures, 0)
}

func TestScanDatasources_MultipleGoldCopiesError(t *testing.T) {
	svc := &CatalogScanService{}
	// two datasources both marked as gold copy
	dsid1 := uuid.New()
	dsid2 := uuid.New()
	svc.getDatasourcesFunc = func(tenantDatasourceID *uuid.UUID) ([]DatasourceConfig, error) {
		return []DatasourceConfig{
			{ID: dsid1, Name: "gold1", IsGoldCopy: true},
			{ID: dsid2, Name: "gold2", IsGoldCopy: true},
		}, nil
	}

	// scanFunc should not be called because service should error earlier
	called := false
	svc.scanFunc = func(ctx context.Context, ds DatasourceConfig, goldCopyNodes map[string]db.GoldCopyNodeInfo) (*ScanResult, error) {
		called = true
		return &ScanResult{Success: true}, nil
	}

	_, err := svc.ScanDatasources(context.Background(), nil)
	assert.Error(t, err)
	assert.False(t, called)
}

func TestScanDatasources_GoldCopyMapPassedToScans(t *testing.T) {
	svc := &CatalogScanService{}

	goldID := uuid.New()
	otherID := uuid.New()

	// Provide two datasources: one gold, one not
	svc.getDatasourcesFunc = func(tenantDatasourceID *uuid.UUID) ([]DatasourceConfig, error) {
		return []DatasourceConfig{
			{ID: goldID, Name: "gold-ds", IsGoldCopy: true},
			{ID: otherID, Name: "other-ds", IsGoldCopy: false},
		}, nil
	}

	// Override the gold map loader to return a known map
	originalLoader := getCatalogNodeMapForGoldCopy
	defer func() { getCatalogNodeMapForGoldCopy = originalLoader }()

	testKey := uuid.New()
	getCatalogNodeMapForGoldCopy = func(dbx *sqlx.DB, goldCopyDatasourceID uuid.UUID) (map[string]db.GoldCopyNodeInfo, error) {
		// ensure called with the goldID
		if goldCopyDatasourceID != goldID {
			return nil, fmt.Errorf("unexpected gold copy ID: %s", goldCopyDatasourceID)
		}
		return map[string]db.GoldCopyNodeInfo{"a:b": {ID: testKey}}, nil
	}

	// scanFunc will assert that the gold map is nil for the gold scan (we pass nil explicitly) and non-nil for the other scan
	calledGold := false
	calledOther := false
	svc.scanFunc = func(ctx context.Context, ds DatasourceConfig, goldCopyNodes map[string]db.GoldCopyNodeInfo) (*ScanResult, error) {
		if ds.ID == goldID {
			calledGold = true
			if goldCopyNodes != nil {
				return nil, fmt.Errorf("gold datasource should not receive a goldCopyNodes map")
			}
		}
		if ds.ID == otherID {
			calledOther = true
			if goldCopyNodes == nil {
				return nil, fmt.Errorf("non-gold datasource should receive a goldCopyNodes map")
			}
			if val, ok := goldCopyNodes["a:b"]; !ok || val.ID != testKey {
				return nil, fmt.Errorf("goldCopyNodes missing expected mapping")
			}
		}
		return &ScanResult{Success: true}, nil
	}

	results, err := svc.ScanDatasources(context.Background(), nil)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.True(t, calledGold)
	assert.True(t, calledOther)
}

// --- Integration-style test for scan path upserting Business Terms ---

// fakeScanner implements metadataScanner for testing
type fakeScanner struct {
	nodes []*models.CatalogNode
	edges []models.CatalogEdge
}

func (f *fakeScanner) ExtractMetadata() ([]*models.CatalogNode, []models.CatalogEdge, error) {
	return f.nodes, f.edges, nil
}

func TestScanSingleDatasource_InvokesBusinessTermUpsert_EndToEnd(t *testing.T) {
	// Save and restore global hooks
	originalNewScanner := newMetadataScanner
	originalUpsert := upsertBusinessTermsFromGold
	originalBuild1 := buildERDChart
	originalBuild2 := buildEnhancedERDChart
	originalBuild3 := buildTechnicalLineageChart
	originalBuild4 := buildSemanticLineageChart
	defer func() {
		newMetadataScanner = originalNewScanner
		upsertBusinessTermsFromGold = originalUpsert
		buildERDChart = originalBuild1
		buildEnhancedERDChart = originalBuild2
		buildTechnicalLineageChart = originalBuild3
		buildSemanticLineageChart = originalBuild4
	}()

	tenantID := uuid.New()
	dsID := uuid.New()

	// Initialize alphaDB with a sqlmock connection that won't panic on ExecContext
	mockDB, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer mockDB.Close()

	svc := &CatalogScanService{alphaDB: sqlx.NewDb(mockDB, "postgres")}

	// Expect the status update in scanSingleDatasource
	mock.ExpectExec("UPDATE public.tenant_product_datasource").
		WithArgs("running", sqlmock.AnyArg(), "", dsID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Expect the status update back to "complete" (or wherever it ends)
	mock.ExpectExec("UPDATE public.tenant_product_datasource").
		WithArgs("complete", sqlmock.AnyArg(), sqlmock.AnyArg(), dsID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Provide a fake target DB connector that returns a *sql.DB; we won't use it beyond Close.
	svc.connectTargetDBFunc = func(ctx context.Context, connectionDetails string) (*sql.DB, error) {
		// driver "pgx" is registered by the service package import; Open doesn't connect until used.
		return sql.Open("pgx", "postgres://u:p@localhost:5432/d?sslmode=disable")
	}

	// Stub storeFunc to accept the nodes and edges without touching a real DB
	svc.storeFunc = func(ctx context.Context, datasourceID uuid.UUID, nodes []*models.CatalogNode, edges []models.CatalogEdge, progress chan<- models.ScanProgress) (int64, int64, int64, error) {
		return 0, 0, 0, nil
	}

	// Provide a fake scanner that returns minimal data
	newMetadataScanner = func(targetDB *sql.DB, tenantID, datasourceID uuid.UUID, sourceSystem string, goldCopyNodes map[string]db.GoldCopyNodeInfo, isGoldCopy bool, schemaWhitelist []string) (MetadataScanner, error) {
		return &fakeScanner{nodes: []*models.CatalogNode{}, edges: []models.CatalogEdge{}}, nil
	}

	// Stub chart builders to no-op
	buildERDChart = func(ctx context.Context, db *sql.DB, datasourceID string, isGoldCopy bool) error { return nil }
	buildEnhancedERDChart = func(ctx context.Context, db *sql.DB, datasourceID string, isGoldCopy bool) error { return nil }
	buildTechnicalLineageChart = func(ctx context.Context, db *sql.DB, datasourceID string, isGoldCopy bool) error { return nil }
	buildSemanticLineageChart = func(ctx context.Context, db *sql.DB, datasourceID string, isGoldCopy bool) error { return nil }

	called := false
	var gotTenant, gotDatasource uuid.UUID
	upsertBusinessTermsFromGold = func(ctx context.Context, alphaDB *sqlx.DB, tenantID uuid.UUID, datasourceID uuid.UUID, businessTermTypeID uuid.UUID) (int64, error) {
		called = true
		gotTenant = tenantID
		gotDatasource = datasourceID
		// Return a dummy count
		return 3, nil
	}

	// Non-gold datasource should trigger upsert
	ds := DatasourceConfig{
		ID:                dsID,
		TenantID:          tenantID,
		Name:              "test-ds",
		SourceSystem:      "pg",
		ConnectionDetails: `{"auth":{"basic":{"username":"u","password":"p"}},"host":"h","port":5432,"database":"d"}`,
		IsGoldCopy:        false,
	}

	_, err = svc.scanSingleDatasource(context.Background(), ds, nil, nil)
	assert.NoError(t, err)
	assert.True(t, called, "expected business term upsert to be called")
	assert.Equal(t, tenantID, gotTenant)
	assert.Equal(t, dsID, gotDatasource)

	// When gold copy, upsert must not be called
	called = false
	ds.IsGoldCopy = true
	_, err = svc.scanSingleDatasource(context.Background(), ds, nil, nil)
	assert.NoError(t, err)
	assert.False(t, called, "business term upsert should not be called for gold copy scans")
}
