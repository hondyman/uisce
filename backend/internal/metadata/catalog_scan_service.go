package metadata

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/db"
	"github.com/hondyman/semlayer/backend/internal/lineage"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/hondyman/semlayer/backend/internal/scanner"
	"github.com/hondyman/semlayer/backend/models"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

// overrideable function used to fetch gold copy node maps (helps testing)
var getCatalogNodeMapForGoldCopy = db.GetCatalogNodeMapForGoldCopy

// overrideable function used to upsert Business Terms from gold copy (helps testing)
var upsertBusinessTermsFromGold = db.UpsertBusinessTermsFromGold

// overrideable chart builder functions (helps testing)
var buildERDChart = db.BuildERDChart
var buildEnhancedERDChart = db.BuildEnhancedERDChart
var buildTechnicalLineageChart = db.BuildTechnicalLineageChart
var buildSemanticLineageChart = db.BuildSemanticLineageChart

// MetadataScanner is a minimal interface for scanners used to extract metadata.
// Exported for testing from other packages.
type MetadataScanner interface {
	ExtractMetadata() ([]*models.CatalogNode, []models.CatalogEdge, error)
}

// newMetadataScanner is an overrideable constructor for the metadata scanner (helps testing)
var newMetadataScanner = func(targetDB *sql.DB, tenantID, datasourceID uuid.UUID, sourceSystem string, goldCopyNodes map[string]db.GoldCopyNodeInfo, isGoldCopy bool, schemaWhitelist []string) (MetadataScanner, error) {
	return scanner.NewAnsiScanner(targetDB, tenantID, datasourceID, sourceSystem, goldCopyNodes, isGoldCopy, schemaWhitelist)
}

// CatalogScanService handles scanning datasources for metadata
type CatalogScanService struct {
	alphaDB *sqlx.DB // The alpha database connection
	// injectable hooks for testing
	getDatasourcesFunc func(tenantDatasourceID *uuid.UUID) ([]DatasourceConfig, error)
	scanFunc           func(ctx context.Context, ds DatasourceConfig, goldCopyNodes map[string]db.GoldCopyNodeInfo) (*ScanResult, error)
	// additional injectable hooks for deeper integration tests
	connectTargetDBFunc func(ctx context.Context, connectionDetails string) (*sql.DB, error)
	storeFunc           func(ctx context.Context, datasourceID uuid.UUID, nodes []*models.CatalogNode, edges []models.CatalogEdge, progress chan<- models.ScanProgress) (int64, int64, int64, error)
	lineageRepo         lineage.LineageRepository
}

// NewCatalogScanService creates a new catalog scan service
func NewCatalogScanService(alphaDB *sqlx.DB, lineageRepo lineage.LineageRepository) *CatalogScanService {
	return &CatalogScanService{
		alphaDB:     alphaDB,
		lineageRepo: lineageRepo,
		// default hooks
		// getDatasourcesFunc uses the receiver's method
		getDatasourcesFunc: func(tenantDatasourceID *uuid.UUID) ([]DatasourceConfig, error) {
			// We'll create a temporary service to call the method which uses alphaDB
			tmp := &CatalogScanService{alphaDB: alphaDB}
			return tmp.getDatasourcesToScan(tenantDatasourceID)
		},
		// default scanFunc delegates to the instance method
		scanFunc: func(ctx context.Context, ds DatasourceConfig, goldCopyNodes map[string]db.GoldCopyNodeInfo) (*ScanResult, error) {
			tmp := &CatalogScanService{alphaDB: alphaDB}
			return tmp.scanSingleDatasource(ctx, ds, goldCopyNodes, nil)
		},
		// defaults for deeper hooks
		connectTargetDBFunc: func(ctx context.Context, connectionDetails string) (*sql.DB, error) {
			// Use a temp receiver to call the method with the configured alphaDB
			tmp := &CatalogScanService{alphaDB: alphaDB}
			return tmp.connectToTargetDatabase(ctx, connectionDetails)
		},
		storeFunc: func(ctx context.Context, datasourceID uuid.UUID, nodes []*models.CatalogNode, edges []models.CatalogEdge, progress chan<- models.ScanProgress) (int64, int64, int64, error) {
			tmp := &CatalogScanService{alphaDB: alphaDB}
			return tmp.storeCatalogData(ctx, datasourceID, nodes, edges, progress)
		},
	}
}

// --- Test support: exported setters for injection ---

// SetConnectTargetDBFunc sets the connector used to open target DB connections.
func (s *CatalogScanService) SetConnectTargetDBFunc(fn func(ctx context.Context, connectionDetails string) (*sql.DB, error)) {
	s.connectTargetDBFunc = fn
}

// SetStoreFunc sets the function used to persist extracted metadata.
func (s *CatalogScanService) SetStoreFunc(fn func(ctx context.Context, datasourceID uuid.UUID, nodes []*models.CatalogNode, edges []models.CatalogEdge, progress chan<- models.ScanProgress) (int64, int64, int64, error)) {
	s.storeFunc = fn
}

// SetGetDatasourcesFunc sets the function used to list datasources to scan.
func (s *CatalogScanService) SetGetDatasourcesFunc(fn func(tenantDatasourceID *uuid.UUID) ([]DatasourceConfig, error)) {
	s.getDatasourcesFunc = fn
}

// SetAlphaDB allows tests to supply an alpha DB handle.
func (s *CatalogScanService) SetAlphaDB(db *sqlx.DB) { s.alphaDB = db }

// SetUpsertBusinessTermsHook overrides the upsert function used in scans and returns a restore func.
func SetUpsertBusinessTermsHook(fn func(ctx context.Context, db *sqlx.DB, tenantID uuid.UUID, tenantDatasourceID uuid.UUID, businessTermTypeID uuid.UUID) (int64, error)) func() {
	prev := upsertBusinessTermsFromGold
	upsertBusinessTermsFromGold = fn
	return func() { upsertBusinessTermsFromGold = prev }
}

// SetScanFunc overrides the per-datasource scan function used by ScanDatasources.
func (s *CatalogScanService) SetScanFunc(fn func(ctx context.Context, ds DatasourceConfig, goldCopyNodes map[string]db.GoldCopyNodeInfo) (*ScanResult, error)) {
	s.scanFunc = fn
}

// SetNewMetadataScannerHook overrides the metadata scanner constructor and returns a restore func.
func SetNewMetadataScannerHook(fn func(targetDB *sql.DB, tenantID, datasourceID uuid.UUID, sourceSystem string, goldCopyNodes map[string]db.GoldCopyNodeInfo, isGoldCopy bool, schemaWhitelist []string) (MetadataScanner, error)) func() {
	prev := newMetadataScanner
	newMetadataScanner = fn
	return func() { newMetadataScanner = prev }
}

// SetChartBuilderHooks overrides chart builder functions and returns a restore func.
func SetChartBuilderHooks(
	erd func(ctx context.Context, db *sql.DB, datasourceID string, isGoldCopy bool) error,
	enhanced func(ctx context.Context, db *sql.DB, datasourceID string, isGoldCopy bool) error,
	technical func(ctx context.Context, db *sql.DB, datasourceID string, isGoldCopy bool) error,
	semantic func(ctx context.Context, db *sql.DB, datasourceID string, isGoldCopy bool) error,
) func() {
	prev1, prev2, prev3, prev4 := buildERDChart, buildEnhancedERDChart, buildTechnicalLineageChart, buildSemanticLineageChart
	if erd != nil {
		buildERDChart = erd
	}
	if enhanced != nil {
		buildEnhancedERDChart = enhanced
	}
	if technical != nil {
		buildTechnicalLineageChart = technical
	}
	if semantic != nil {
		buildSemanticLineageChart = semantic
	}
	return func() {
		buildERDChart, buildEnhancedERDChart, buildTechnicalLineageChart, buildSemanticLineageChart = prev1, prev2, prev3, prev4
	}
}

// DatasourceConfig represents a datasource configuration from the database
type DatasourceConfig struct {
	ID                uuid.UUID `db:"id"`
	TenantID          uuid.UUID `db:"tenant_id"`
	Name              string    `db:"name"`
	SourceSystem      string    `db:"source_system"`
	ConnectionDetails string    `db:"connection_details"` // JSON with connection info
	IsGoldCopy        bool      `db:"is_gold_copy"`
}

// ScanResult represents the result for a single datasource scan
type ScanResult struct {
	DatasourceID  uuid.UUID `json:"datasource_id"`
	Name          string    `json:"name"`
	Success       bool      `json:"success"`
	Error         string    `json:"error,omitempty"`
	Added         int64     `json:"added"`
	Updated       int64     `json:"updated"`
	Removed       int64     `json:"removed"`
	ChartsRebuilt bool      `json:"charts_rebuilt"`
}

// ScanProgress is now defined in models/core.go

// ScanWithProgress runs scan with progress updates sent to the provided channel
func (s *CatalogScanService) ScanWithProgress(ctx context.Context, tenantDatasourceID *uuid.UUID, progress chan<- models.ScanProgress) ([]ScanResult, error) {
	// Send initial progress
	progress <- models.ScanProgress{Phase: "starting", Percent: 0, Message: "Initializing scan..."}

	// Step 1: Get datasources to scan
	progress <- models.ScanProgress{Phase: "fetching", Percent: 5, Message: "Fetching datasource configuration..."}
	datasources, err := s.getDatasourcesFunc(tenantDatasourceID)
	if err != nil {
		progress <- models.ScanProgress{Phase: "error", Percent: 0, Message: fmt.Sprintf("Failed to get datasources: %v", err)}
		return nil, fmt.Errorf("failed to get datasources to scan: %w", err)
	}

	if len(datasources) == 0 {
		progress <- models.ScanProgress{Phase: "complete", Percent: 100, Message: "No datasources found to scan"}
		return []ScanResult{}, nil
	}

	// Step 2: Find gold copy
	progress <- models.ScanProgress{Phase: "preparing", Percent: 10, Message: "Loading gold copy mappings..."}
	var goldCopyNodes map[string]db.GoldCopyNodeInfo
	var goldCopyCount int
	var lastGoldCopyID uuid.UUID
	for _, ds := range datasources {
		if ds.IsGoldCopy {
			goldCopyCount++
			lastGoldCopyID = ds.ID
		}
	}

	if goldCopyCount > 1 {
		progress <- models.ScanProgress{Phase: "error", Percent: 0, Message: "Multiple gold copies found"}
		return nil, fmt.Errorf("misconfiguration: found %d datasources marked as gold copy", goldCopyCount)
	}

	if goldCopyCount == 1 {
		goldCopyNodes, err = getCatalogNodeMapForGoldCopy(s.alphaDB, lastGoldCopyID)
		if err != nil {
			logging.GetLogger().Sugar().Warnf("Warning: failed to get gold copy nodes: %v", err)
		}
	}

	// Step 3: Scan each datasource with progress
	var results []ScanResult
	for i, ds := range datasources {
		basePercent := 15 + float64(i)*80/float64(len(datasources))

		progress <- models.ScanProgress{
			Phase:       "connecting",
			Percent:     basePercent,
			CurrentItem: ds.Name,
			Total:       len(datasources),
			Completed:   i,
			Message:     fmt.Sprintf("Connecting to %s...", ds.Name),
		}

		res, scanErr := s.scanSingleDatasourceWithProgress(ctx, ds, goldCopyNodes, progress, basePercent, 80/float64(len(datasources)))
		if scanErr != nil {
			results = append(results, ScanResult{
				DatasourceID: ds.ID,
				Name:         ds.Name,
				Success:      false,
				Error:        scanErr.Error(),
			})
		} else {
			res.Name = ds.Name
			res.DatasourceID = ds.ID
			res.Success = true
			results = append(results, *res)
		}
	}

	progress <- models.ScanProgress{Phase: "complete", Percent: 100, Message: "Scan complete!", Total: len(datasources), Completed: len(datasources)}
	return results, nil
}

// ScanDatasources scans metadata for specified datasources and returns per-datasource results.
// It returns an error only when all requested scans fail.
func (s *CatalogScanService) ScanDatasources(ctx context.Context, tenantDatasourceID *uuid.UUID) ([]ScanResult, error) {
	// Step 1: Get datasources to scan
	// Use injectable hook to retrieve datasources (testable)
	datasources, err := s.getDatasourcesFunc(tenantDatasourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get datasources to scan: %w", err)
	}

	if len(datasources) == 0 {
		logging.GetLogger().Sugar().Info("No datasources found to scan")
		return []ScanResult{}, nil
	}

	// Step 2: Find gold copy for core ID mapping (done once)
	var goldCopyNodes map[string]db.GoldCopyNodeInfo
	var goldCopyCount int
	var lastGoldCopyID uuid.UUID
	for _, ds := range datasources {
		if ds.IsGoldCopy {
			goldCopyCount++
			lastGoldCopyID = ds.ID
		}
	}

	if goldCopyCount > 1 {
		return nil, fmt.Errorf("misconfiguration: found %d datasources marked as gold copy; only one is allowed", goldCopyCount)
	}

	if goldCopyCount == 1 {
		goldCopyNodes, err = getCatalogNodeMapForGoldCopy(s.alphaDB, lastGoldCopyID)
		if err != nil {
			logging.GetLogger().Sugar().Warnf("Warning: failed to get gold copy nodes: %v", err)
		}
	}

	// Log whether we loaded a gold copy node map and how many entries it contains
	if goldCopyNodes != nil {
		logging.GetLogger().Sugar().Infof("Loaded %d gold copy node mappings for core ID resolution", len(goldCopyNodes))
	} else {
		logging.GetLogger().Sugar().Info("No gold copy node mappings loaded; core ID resolution will be skipped")
	}

	// Step 3: Scan each datasource in parallel
	var wg sync.WaitGroup
	// Using a buffered channel to limit concurrency to 10 scans at a time.
	// Adjust this number based on system capacity.
	concurrencyLimit := 10
	sem := make(chan struct{}, concurrencyLimit)
	resultChan := make(chan ScanResult, len(datasources))

	for _, ds := range datasources {
		wg.Add(1)
		go func(ds DatasourceConfig) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire a token
			defer func() { <-sem }() // Release the token

			res, err := s.scanFunc(ctx, ds, goldCopyNodes)
			if err != nil {
				logging.GetLogger().Sugar().Warnf("Failed to scan datasource %s (%s): %v", ds.Name, ds.ID, err)
				resultChan <- ScanResult{
					DatasourceID: ds.ID,
					Name:         ds.Name,
					Success:      false,
					Error:        err.Error(),
				}
			} else {
				res.Name = ds.Name       // Ensure name is set
				res.DatasourceID = ds.ID // Ensure ID is set
				res.Success = true
				resultChan <- *res
			}
		}(ds)
	}

	wg.Wait()
	close(resultChan)

	var results []ScanResult
	var failures int
	for r := range resultChan {
		results = append(results, r)
		if !r.Success {
			failures++
		}
	}

	if failures == len(datasources) {
		// All failed
		var errMsgs []string
		for _, r := range results {
			errMsgs = append(errMsgs, fmt.Sprintf("%s: %s", r.Name, r.Error))
		}
		return results, fmt.Errorf("completed scan with %d errors: \n- %s", failures, strings.Join(errMsgs, "\n- "))
	}

	if failures > 0 {
		logging.GetLogger().Sugar().Warnf("Completed scan with %d failures; %d success(es)", failures, len(results)-failures)
	} else {
		logging.GetLogger().Sugar().Info("All datasource scans completed successfully.")
	}

	return results, nil
}

// getDatasourcesToScan retrieves the list of datasources to scan
func (s *CatalogScanService) getDatasourcesToScan(tenantDatasourceID *uuid.UUID) ([]DatasourceConfig, error) {
	var datasources []DatasourceConfig

	var query string
	var args []interface{}

	logging.GetLogger().Sugar().Infof("[DEBUG] getDatasourcesToScan called with tenantDatasourceID=%v", tenantDatasourceID)

	baseQuery := `
		SELECT
			tpd.id,
			ti.tenant_id,
			tpd.source_name AS name,
			ad.datasource_code AS source_system,
			CASE 
				WHEN c.id IS NOT NULL THEN 
					json_build_object(
						'host', c.host,
						'port', c.port,
						'database', c.database,
						'username', c.username,
						'password', c.password,
						'schema', c.schema
					)::jsonb
				ELSE tpd.config 
			END AS connection_details,
			t.gold_copy AS is_gold_copy
		FROM
			public.tenant_product_datasource tpd
		LEFT JOIN
			public.connections c ON tpd.connection_id = c.id
		JOIN
			public.alpha_datasource ad ON tpd.alpha_datasource_id = ad.id
		JOIN
			public.tenant_product tp ON tpd.tenant_product_id = tp.id
		JOIN
			public.tenant_instance ti ON ti.id = tp.datasource_id
		JOIN
			public.tenants t ON t.id = ti.tenant_id
	`

	if tenantDatasourceID != nil {
		// Scan specific datasource
		query = baseQuery + " WHERE tpd.id = $1"
		args = []interface{}{*tenantDatasourceID}
	} else {
		// Scan all datasources
		query = baseQuery + " ORDER BY tpd.source_name"
	}

	logging.GetLogger().Sugar().Infof("[DEBUG] Executing query: %s with args: %v", query, args)
	err := s.alphaDB.Select(&datasources, query, args...)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("[DEBUG] Query error: %v", err)
		return nil, fmt.Errorf("failed to query tenant_product_datasource: %w", err)
	}

	logging.GetLogger().Sugar().Infof("Found %d datasource(s) to scan", len(datasources))
	return datasources, nil
}

// scanSingleDatasource scans a single datasource for metadata
// scanSingleDatasource scans a single datasource for metadata
func (s *CatalogScanService) scanSingleDatasource(ctx context.Context, ds DatasourceConfig, goldCopyNodes map[string]db.GoldCopyNodeInfo, progress chan<- models.ScanProgress) (*ScanResult, error) {
	// UPDATE STATUS: Running
	_, _ = s.alphaDB.ExecContext(ctx, `
		UPDATE public.tenant_product_datasource
		SET last_scan_status = 'running', last_scan_at = NOW(), last_scan_message = ''
		WHERE id = $1
	`, ds.ID)

	logging.GetLogger().Sugar().Infof("Starting scan for datasource: %s (ID: %s)", ds.Name, ds.ID)

	result := &ScanResult{
		DatasourceID: ds.ID,
		Name:         ds.Name,
	}

	// Parse connection details (assuming it's a PostgreSQL connection string)
	// Use injectable connection hook to facilitate testing
	var targetDB *sql.DB
	var err error

	if progress != nil {
		progress <- models.ScanProgress{Phase: "connecting", Percent: 0, Message: fmt.Sprintf("Connecting to %s...", ds.Name)}
	}
	if s.connectTargetDBFunc != nil {
		targetDB, err = s.connectTargetDBFunc(ctx, ds.ConnectionDetails)
	} else {
		targetDB, err = s.connectToTargetDatabase(ctx, ds.ConnectionDetails)
	}
	if err != nil {
		// UPDATE STATUS: Failed
		_, _ = s.alphaDB.ExecContext(ctx, `
			UPDATE public.tenant_product_datasource
			SET last_scan_status = 'failure', last_scan_at = NOW(), last_scan_message = $2
			WHERE id = $1
		`, ds.ID, err.Error())
		return nil, fmt.Errorf("failed to connect to target database %s: %w", ds.Name, err)
	}
	defer targetDB.Close()

	// Parse connection details to extract schema info if available
	var schemaWhitelist []string
	var connConfig struct {
		Schema string `json:"schema"`
	}
	if err := json.Unmarshal([]byte(ds.ConnectionDetails), &connConfig); err == nil && connConfig.Schema != "" {
		schemaWhitelist = []string{connConfig.Schema}
		logging.GetLogger().Sugar().Infof("Configuring scanner with schema whitelist: %v", schemaWhitelist)
	}

	// Create scanner via overrideable constructor for testing
	ansiScanner, err := newMetadataScanner(targetDB, ds.TenantID, ds.ID, ds.SourceSystem, goldCopyNodes, ds.IsGoldCopy, schemaWhitelist)
	if err != nil {
		// UPDATE STATUS: Failed
		_, _ = s.alphaDB.ExecContext(ctx, `
			UPDATE public.tenant_product_datasource
			SET last_scan_status = 'failure', last_scan_at = NOW(), last_scan_message = $2
			WHERE id = $1
		`, ds.ID, err.Error())
		return nil, fmt.Errorf("failed to create scanner for %s: %w", ds.Name, err)
	}

	if progress != nil {
		progress <- models.ScanProgress{Phase: "scanning", Percent: 0, Message: "Extracting metadata (tables, columns, keys)..."}
	}
	// Extract metadata from the TARGET database
	nodes, edges, err := ansiScanner.ExtractMetadata()
	if err != nil {
		// UPDATE STATUS: Failed
		_, _ = s.alphaDB.ExecContext(ctx, `
			UPDATE public.tenant_product_datasource
			SET last_scan_status = 'failure', last_scan_at = NOW(), last_scan_message = $2
			WHERE id = $1
		`, ds.ID, err.Error())
		return nil, fmt.Errorf("failed to extract metadata from %s: %w", ds.Name, err)
	}

	logging.GetLogger().Sugar().Infof("Extracted %d nodes and %d edges from %s", len(nodes), len(edges), ds.Name)

	if progress != nil {
		progress <- models.ScanProgress{Phase: "storing", Percent: 0, Message: fmt.Sprintf("Storing %d nodes and %d edges...", len(nodes), len(edges))}
	}
	// Store the extracted metadata in the alpha database
	// Store via injectable hook
	var added, updated, removed int64
	if s.storeFunc != nil {
		if added, updated, removed, err = s.storeFunc(ctx, ds.ID, nodes, edges, progress); err != nil {
			// UPDATE STATUS: Failed
			_, _ = s.alphaDB.ExecContext(ctx, `
				UPDATE public.tenant_product_datasource
				SET last_scan_status = 'failure', last_scan_at = NOW(), last_scan_message = $2
				WHERE id = $1
			`, ds.ID, err.Error())
			return nil, fmt.Errorf("failed to store catalog data for %s: %w", ds.Name, err)
		}
	} else {
		if added, updated, removed, err = s.storeCatalogData(ctx, ds.ID, nodes, edges, progress); err != nil {
			// UPDATE STATUS: Failed
			_, _ = s.alphaDB.ExecContext(ctx, `
				UPDATE public.tenant_product_datasource
				SET last_scan_status = 'failure', last_scan_at = NOW(), last_scan_message = $2
				WHERE id = $1
			`, ds.ID, err.Error())
			return nil, fmt.Errorf("failed to store catalog data for %s: %w", ds.Name, err)
		}
	}

	result.Added = added
	result.Updated = updated
	result.Removed = removed

	// Special handling: upsert Business Terms from gold-copy into this non-gold tenant/datasource
	if !ds.IsGoldCopy {
		if progress != nil {
			progress <- models.ScanProgress{Phase: "enriching", Percent: 0, Message: "Upserting business terms..."}
		}
		// Business Term node type UUID can be overridden via env; default to known constant
		btEnv := os.Getenv("SEMLAYER_NODETYPE_BUSINESS_TERM")
		btID := uuid.MustParse("21645d21-de5f-4feb-af99-99273ea75626")
		if btEnv != "" {
			if parsed, err := uuid.Parse(btEnv); err == nil {
				btID = parsed
			}
		}
		if count, err := upsertBusinessTermsFromGold(ctx, s.alphaDB, ds.TenantID, ds.ID, btID); err != nil {
			logging.GetLogger().Sugar().Warnf("Warning: failed to upsert business terms for datasource %s: %v", ds.ID, err)
		} else {
			logging.GetLogger().Sugar().Infof("Upserted %d business terms into tenant %s for datasource %s", count, ds.TenantID, ds.ID)
		}
	}

	// Step 4: Build charts ONLY if there were schema changes (delta optimization)
	hasSchemaChanges := added > 0 || updated > 0 || removed > 0
	result.ChartsRebuilt = false

	logging.GetLogger().Sugar().Infof("[DEBUG] Chart build check: hasSchemaChanges=%v, alphaDB=%v, alphaDB.DB=%v", hasSchemaChanges, s.alphaDB != nil, s.alphaDB != nil && s.alphaDB.DB != nil)

	// Force chart rebuild to ensure data profile stats are included
	// if !hasSchemaChanges {
	// 	logging.GetLogger().Sugar().Infof("No schema changes for datasource %s (added=0, updated=0, removed=0); using cached charts", ds.Name)
	// } else if s.alphaDB == nil || s.alphaDB.DB == nil {

	if s.alphaDB == nil || s.alphaDB.DB == nil {
		logging.GetLogger().Sugar().Warnf("Skipping chart builds for datasource %s: alphaDB not available", ds.Name)
	} else {
		logging.GetLogger().Sugar().Infof("Schema changes detected for datasource %s (added=%d, updated=%d, removed=%d); rebuilding charts...", ds.Name, added, updated, removed)
		result.ChartsRebuilt = true

		if progress != nil {
			progress <- models.ScanProgress{Phase: "calibrating", Percent: 0, Message: "Rebuilding ERD charts..."}
		}
		logging.GetLogger().Sugar().Infof("[DEBUG] Starting buildERDChart for %s", ds.ID.String())
		if err := buildERDChart(ctx, s.alphaDB.DB, ds.ID.String(), ds.IsGoldCopy); err != nil {
			// Log the error but don't fail the whole scan, as other charts might succeed
			logging.GetLogger().Sugar().Warnf("Warning: failed to build ERD chart for %s: %v", ds.Name, err)
		}
		logging.GetLogger().Sugar().Infof("[DEBUG] Completed buildERDChart")

		if progress != nil {
			progress <- models.ScanProgress{Phase: "calibrating", Percent: 0, Message: "Building enhanced ERD..."}
		}
		logging.GetLogger().Sugar().Infof("[DEBUG] Starting buildEnhancedERDChart")
		if err := buildEnhancedERDChart(ctx, s.alphaDB.DB, ds.ID.String(), ds.IsGoldCopy); err != nil {
			logging.GetLogger().Sugar().Warnf("Warning: failed to build enhanced ERD chart for %s: %v", ds.Name, err)
		}
		logging.GetLogger().Sugar().Infof("[DEBUG] Completed buildEnhancedERDChart")

		if progress != nil {
			progress <- models.ScanProgress{Phase: "calibrating", Percent: 0, Message: "Mapping technical lineage..."}
		}
		logging.GetLogger().Sugar().Infof("[DEBUG] Starting buildTechnicalLineageChart")
		if err := buildTechnicalLineageChart(ctx, s.alphaDB.DB, ds.ID.String(), ds.IsGoldCopy); err != nil {
			logging.GetLogger().Sugar().Warnf("Warning: failed to build technical lineage chart for %s: %v", ds.Name, err)
		}
		logging.GetLogger().Sugar().Infof("[DEBUG] Completed buildTechnicalLineageChart")

		if progress != nil {
			progress <- models.ScanProgress{Phase: "calibrating", Percent: 0, Message: "Mapping semantic lineage..."}
		}
		logging.GetLogger().Sugar().Infof("[DEBUG] Starting buildSemanticLineageChart")
		if err := buildSemanticLineageChart(ctx, s.alphaDB.DB, ds.ID.String(), ds.IsGoldCopy); err != nil {
			logging.GetLogger().Sugar().Warnf("Warning: failed to build semantic lineage chart for %s: %v", ds.Name, err)
		}
		logging.GetLogger().Sugar().Infof("[DEBUG] Completed buildSemanticLineageChart")
	}

	// UPDATE STATUS: Success
	_, _ = s.alphaDB.ExecContext(ctx, `
		UPDATE public.tenant_product_datasource
		SET last_scan_status = 'success', last_scan_at = NOW(), last_scan_message = 'Scan completed successfully'
		WHERE id = $1
	`, ds.ID)

	logging.GetLogger().Sugar().Infof("Successfully completed scan for datasource: %s (charts_rebuilt=%v)", ds.Name, result.ChartsRebuilt)

	// Sync to lineage graph
	if s.lineageRepo != nil {
		if progress != nil {
			progress <- models.ScanProgress{Phase: "calibrating", Percent: 0, Message: "Synchronizing graph-native lineage..."}
		}
		if err := s.lineageRepo.SyncDatasource(ctx, ds.ID.String()); err != nil {
			logging.GetLogger().Sugar().Warnf("Warning: Failed to sync datasource to graph: %v", err)
		}
	}

	return result, nil
}

// ScanSingleDatasourceForTest exposes scanSingleDatasource to tests in other packages.
func (s *CatalogScanService) ScanSingleDatasourceForTest(ctx context.Context, ds DatasourceConfig, goldCopyNodes map[string]db.GoldCopyNodeInfo) (*ScanResult, error) {
	return s.scanSingleDatasource(ctx, ds, goldCopyNodes, nil)
}

// scanSingleDatasourceWithProgress runs scan with progress updates
func (s *CatalogScanService) scanSingleDatasourceWithProgress(ctx context.Context, ds DatasourceConfig, goldCopyNodes map[string]db.GoldCopyNodeInfo, progress chan<- models.ScanProgress, basePercent, weight float64) (*ScanResult, error) {
	// UPDATE STATUS: Running phase progress
	progress <- models.ScanProgress{
		Phase:       "scanning",
		Percent:     basePercent + weight*0.2,
		CurrentItem: ds.Name,
		Message:     fmt.Sprintf("Extracting metadata from %s...", ds.Name),
	}

	// Delegate to the modified scan method that supports granular progress
	result, err := s.scanSingleDatasource(ctx, ds, goldCopyNodes, progress)

	if err != nil {
		progress <- models.ScanProgress{
			Phase:       "error",
			CurrentItem: ds.Name,
			Message:     fmt.Sprintf("Failed to scan %s: %v", ds.Name, err),
		}
		return nil, err
	}
	// Emit storing phase progress
	progress <- models.ScanProgress{
		Phase:       "storing",
		Percent:     basePercent + weight*0.2,
		CurrentItem: ds.Name,
		Message:     fmt.Sprintf("Stored %d tables, %d updated for %s", result.Added, result.Updated, ds.Name),
	}

	return result, nil
}

// TestConnection attempts to connect to a target database to verify connection details.
func (s *CatalogScanService) TestConnection(ctx context.Context, connectionDetails string) error {
	var targetDB *sql.DB
	var err error
	if s.connectTargetDBFunc != nil {
		targetDB, err = s.connectTargetDBFunc(ctx, connectionDetails)
	} else {
		targetDB, err = s.connectToTargetDatabase(ctx, connectionDetails)
	}

	if err != nil {
		return err // connectToTargetDatabase returns a wrapped error
	}
	defer targetDB.Close()
	return nil
}

// TestConnectionByID fetches connection details for a datasource ID and tests the connection
func (s *CatalogScanService) TestConnectionByID(ctx context.Context, datasourceID uuid.UUID) error {
	// Re-use the existing query logic from getDatasourcesToScan but strictly for one ID
	datasources, err := s.getDatasourcesFunc(&datasourceID)
	if err != nil {
		return fmt.Errorf("failed to fetch datasource details: %w", err)
	}
	if len(datasources) == 0 {
		return fmt.Errorf("datasource not found: %s", datasourceID)
	}

	ds := datasources[0]
	// If connection details are empty, we can't test
	if ds.ConnectionDetails == "" {
		return fmt.Errorf("no connection details found for datasource: %s", ds.Name)
	}

	// Use existing TestConnection logic
	return s.TestConnection(ctx, ds.ConnectionDetails)
}

// connectToTargetDatabase establishes connection to a target database
func (s *CatalogScanService) connectToTargetDatabase(ctx context.Context, connectionDetails string) (*sql.DB, error) {
	// This struct is updated to match the nested JSON structure from the database
	// AND the flat structure from the frontend ConnectionForm.
	type ConnectionConfig struct {
		Auth struct {
			Basic struct {
				Password string `json:"password"`
				Username string `json:"username,omitempty"`
			} `json:"basic"`
		} `json:"auth"`
		Database string `json:"database"`
		Host     string `json:"host"`
		Password string `json:"password,omitempty"`
		Port     int    `json:"port"`
		Schema   string `json:"schema,omitempty"`
		SSL      bool   `json:"ssl,omitempty"`
		SSLMode  string `json:"sslMode,omitempty"`
		Username string `json:"username,omitempty"`
		Type     string `json:"type"`
		DSN      string `json:"dsn,omitempty"` // Support direct DSN
	}

	var config ConnectionConfig
	if err := json.Unmarshal([]byte(connectionDetails), &config); err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to unmarshal connection details: %v", err)
		return nil, fmt.Errorf("failed to parse connection details: %w", err)
	}

	// Determine effective sslmode
	// If DSN is provided, we use it directly (later logic).
	// If constructed from parts, we check SSL flags.

	// Normalize Username/Password (prefer flat if Auth struct is empty, or valid vice versa)
	if config.Auth.Basic.Username != "" {
		config.Username = config.Auth.Basic.Username
	}
	if config.Auth.Basic.Password != "" {
		config.Password = config.Auth.Basic.Password
	}

	// Default sslmode logic
	if config.SSLMode == "" {
		if config.SSL {
			config.SSLMode = "require"
		} else {
			config.SSLMode = "disable"
		}
	}

	// Construct the DSN (Data Source Name) for the postgres driver.
	dsn := config.DSN
	if dsn == "" {
		// Validate required fields to avoid constructing invalid DSNs (e.g., port==0)
		if config.Host == "" {
			return nil, fmt.Errorf("invalid connection details: host is empty")
		}
		if config.Port <= 0 || config.Port > 65535 {
			return nil, fmt.Errorf("invalid connection details: port must be between 1 and 65535 (got %d)", config.Port)
		}
		if config.Username == "" {
			return nil, fmt.Errorf("invalid connection details: missing username")
		}
		if config.Database == "" {
			return nil, fmt.Errorf("invalid connection details: missing database name")
		}

		u := url.URL{
			Scheme: "postgres",
			Host:   fmt.Sprintf("%s:%d", config.Host, config.Port),
			Path:   config.Database,
			User:   url.UserPassword(config.Username, config.Password),
		}

		q := u.Query()
		q.Set("sslmode", config.SSLMode)
		if config.Schema != "" {
			q.Set("search_path", config.Schema)
		}
		u.RawQuery = q.Encode()

		dsn = u.String()
	}

	targetDB, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open target database connection: %w", err)
	}

	// Use PingContext with timeout for fast failure if host unreachable
	fields := strings.Split(config.Host, ":")
	hostForLog := fields[0]
	logging.GetLogger().Sugar().Infof("Connecting to %s on %s (sslmode=%s)...", config.Database, hostForLog, config.SSLMode)

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := targetDB.PingContext(pingCtx); err != nil {
		targetDB.Close() // Close the connection if ping fails
		return nil, fmt.Errorf("failed to ping target database (timeout 10s): %w", err)
	}

	logging.GetLogger().Sugar().Infof("Successfully connected to target database: %s on host %s", config.Database, config.Host)
	return targetDB, nil
}

// storeCatalogData stores extracted metadata in the alpha database
func (s *CatalogScanService) storeCatalogData(ctx context.Context, datasourceID uuid.UUID, nodes []*models.CatalogNode, edges []models.CatalogEdge, progress chan<- models.ScanProgress) (int64, int64, int64, error) {
	// Start transaction on alpha database
	tx, err := s.alphaDB.BeginTxx(ctx, nil)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Hash check optimization removed to ensure property updates (like is_core) are always persisted.
	// Previously, this skipped updates if the list of qualified paths was identical, ignoring JSON property changes.

	// Clean up temp tables
	if err := db.CleanupTempTables(tx, datasourceID); err != nil {
		return 0, 0, 0, fmt.Errorf("failed to cleanup temp tables: %w", err)
	}

	// Insert into temp tables
	if progress != nil {
		progress <- models.ScanProgress{Phase: "storing", Percent: 10, Message: fmt.Sprintf("Storing %d nodes (batching)...", len(nodes))}
	}
	if err := db.InsertTempCatalogNodes(ctx, tx, nodes, progress); err != nil {
		return 0, 0, 0, fmt.Errorf("failed to insert temp nodes: %w", err)
	}

	if progress != nil {
		progress <- models.ScanProgress{Phase: "storing", Percent: 50, Message: fmt.Sprintf("Storing %d edges...", len(edges))}
	}
	if err := db.InsertTempCatalogEdges(ctx, tx, edges); err != nil {
		return 0, 0, 0, fmt.Errorf("failed to insert temp edges: %w", err)
	}

	if progress != nil {
		progress <- models.ScanProgress{Phase: "storing", Percent: 70, Message: "Merging changes..."}
	}

	// Merge from temp to main tables
	var added, updated, removed int64
	if added, updated, removed, err = db.MergeCatalogData(tx, datasourceID); err != nil {
		return 0, 0, 0, fmt.Errorf("failed to merge catalog data: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return 0, 0, 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	logging.GetLogger().Sugar().Infof("Successfully stored catalog data for datasource %s", datasourceID)
	return added, updated, removed, nil
}
