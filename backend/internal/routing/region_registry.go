package routing

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"go.temporal.io/sdk/client"
)

// RegionContext holds configuration for a specific region
type RegionContext struct {
	RegionCode           string
	RegionName           string
	TemporalNamespace    string
	TemporalAddress      string
	TrinoCatalog         string
	TrinoEndpoint        string
	IcebergCatalog       string
	IcebergS3Bucket      string
	IcebergWarehousePath string
	APIEndpoint          string
	APIPort              int
	MTLSEnabled          bool
	MTLSCACert           string
	MTLSClientCert       string
	MTLSClientKey        string
}

// RegionRegistry manages region metadata and client connections
type RegionRegistry struct {
	mu               sync.RWMutex
	db               *sql.DB
	regionCache      map[string]*RegionContext
	clientCache      map[string]client.Client
	cacheExpiry      time.Duration
	lastCacheRefresh time.Time
}

// NewRegionRegistry creates a new region registry backed by PostgreSQL
func NewRegionRegistry(db *sql.DB) *RegionRegistry {
	return &RegionRegistry{
		db:          db,
		regionCache: make(map[string]*RegionContext),
		clientCache: make(map[string]client.Client),
		cacheExpiry: 5 * time.Minute,
	}
}

// GetRegion retrieves a region's context by code
func (r *RegionRegistry) GetRegion(ctx context.Context, regionCode string) (*RegionContext, error) {
	r.mu.RLock()
	if cached, exists := r.regionCache[regionCode]; exists {
		if time.Since(r.lastCacheRefresh) < r.cacheExpiry {
			r.mu.RUnlock()
			return cached, nil
		}
	}
	r.mu.RUnlock()

	var regCtx RegionContext
	var mtlsCACert, mtlsClientCert, mtlsClientKey sql.NullString

	row := r.db.QueryRowContext(ctx,
		`SELECT region_code, region_name, temporal_namespace, temporal_address,
                trino_catalog, trino_endpoint, iceberg_catalog, iceberg_s3_bucket,
                iceberg_warehouse_path, api_endpoint, api_port, mTLS_ca_cert,
                mTLS_client_cert, mTLS_client_key
         FROM region_registry WHERE region_code = $1 AND is_active = TRUE`,
		regionCode,
	)

	if err := row.Scan(&regCtx.RegionCode, &regCtx.RegionName, &regCtx.TemporalNamespace,
		&regCtx.TemporalAddress, &regCtx.TrinoCatalog, &regCtx.TrinoEndpoint,
		&regCtx.IcebergCatalog, &regCtx.IcebergS3Bucket, &regCtx.IcebergWarehousePath,
		&regCtx.APIEndpoint, &regCtx.APIPort, &mtlsCACert, &mtlsClientCert, &mtlsClientKey); err != nil {
		return nil, fmt.Errorf("failed to retrieve region %s: %w", regionCode, err)
	}

	if mtlsCACert.Valid {
		regCtx.MTLSEnabled = true
		regCtx.MTLSCACert = mtlsCACert.String
		regCtx.MTLSClientCert = mtlsClientCert.String
		regCtx.MTLSClientKey = mtlsClientKey.String
	}

	// Cache the result
	r.mu.Lock()
	r.regionCache[regionCode] = &regCtx
	r.lastCacheRefresh = time.Now()
	r.mu.Unlock()

	return &regCtx, nil
}

// ListActiveRegions returns all active regions
func (r *RegionRegistry) ListActiveRegions(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT region_code FROM region_registry WHERE is_active = TRUE ORDER BY region_code`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var regions []string
	for rows.Next() {
		var region string
		if err := rows.Scan(&region); err != nil {
			return nil, err
		}
		regions = append(regions, region)
	}

	return regions, rows.Err()
}

// GetRegionClient returns a Temporal client for the specified region
func (r *RegionRegistry) GetRegionClient(ctx context.Context, regionCode string) (client.Client, error) {
	r.mu.RLock()
	if cached, exists := r.clientCache[regionCode]; exists {
		r.mu.RUnlock()
		return cached, nil
	}
	r.mu.RUnlock()

	regCtx, err := r.GetRegion(ctx, regionCode)
	if err != nil {
		return nil, err
	}

	clientOptions := client.Options{
		HostPort:  regCtx.TemporalAddress,
		Namespace: regCtx.TemporalNamespace,
	}

	// If mTLS is enabled, configure TLS
	if regCtx.MTLSEnabled {
		// TLS configuration would be set here
		// clientOptions.ConnectionOptions = ...
	}

	temporalClient, err := client.NewClient(clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to create Temporal client for region %s: %w", regionCode, err)
	}

	// Cache the client
	r.mu.Lock()
	r.clientCache[regionCode] = temporalClient
	r.mu.Unlock()

	return temporalClient, nil
}

// CloseCachedClients closes all cached Temporal clients
func (r *RegionRegistry) CloseCachedClients() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for regionCode, client := range r.clientCache {
		client.Close()
		delete(r.clientCache, regionCode)
	}
}

// InvalidateCache clears the region cache
func (r *RegionRegistry) InvalidateCache() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.regionCache = make(map[string]*RegionContext)
}

// ==================================================================================
// REGION-AWARE WORKFLOW EXECUTION
// ==================================================================================

// ExecuteRegionWorkflow executes a workflow in a specific region
func ExecuteRegionWorkflow(ctx context.Context, registry *RegionRegistry, regionCode string, workflowName string, input interface{}, options client.StartWorkflowOptions) (client.WorkflowRun, error) {
	regionClient, err := registry.GetRegionClient(ctx, regionCode)
	if err != nil {
		return nil, err
	}

	// Set region-specific task queue
	options.TaskQueue = workflowName + "_" + regionCode

	return regionClient.ExecuteWorkflow(ctx, options, workflowName, input)
}

// FanOutRegionWorkflows executes a workflow across multiple regions concurrently
func FanOutRegionWorkflows(ctx context.Context, registry *RegionRegistry, regions []string, workflowName string, inputFactory func(region string) interface{}, options client.StartWorkflowOptions) (map[string]client.WorkflowRun, error) {
	results := make(map[string]client.WorkflowRun)
	errChan := make(chan error, len(regions))
	resultChan := make(chan struct {
		region string
		run    client.WorkflowRun
	}, len(regions))

	var wg sync.WaitGroup
	for _, region := range regions {
		wg.Add(1)
		go func(regionCode string) {
			defer wg.Done()

			input := inputFactory(regionCode)
			run, err := ExecuteRegionWorkflow(ctx, registry, regionCode, workflowName, input, options)
			if err != nil {
				errChan <- fmt.Errorf("failed to execute workflow in region %s: %w", regionCode, err)
				return
			}

			resultChan <- struct {
				region string
				run    client.WorkflowRun
			}{regionCode, run}
		}(region)
	}

	wg.Wait()
	close(resultChan)
	close(errChan)

	// Collect results
	for run := range resultChan {
		results[run.region] = run.run
	}

	// Check for errors
	for err := range errChan {
		return results, err // Return partial results with first error
	}

	return results, nil
}

// WaitForRegionWorkflows waits for completion of multiple region workflows
func WaitForRegionWorkflows(ctx context.Context, runs map[string]client.WorkflowRun) (map[string]interface{}, error) {
	results := make(map[string]interface{})
	errMap := make(map[string]error)

	var mu sync.Mutex
	var wg sync.WaitGroup

	for region, run := range runs {
		wg.Add(1)
		go func(r string, w client.WorkflowRun) {
			defer wg.Done()

			var result interface{}
			if err := w.Get(ctx, &result); err != nil {
				mu.Lock()
				errMap[r] = err
				mu.Unlock()
				return
			}

			mu.Lock()
			results[r] = result
			mu.Unlock()
		}(region, run)
	}

	wg.Wait()

	// If there are partial failures, return results + error indicator
	if len(errMap) > 0 {
		return results, fmt.Errorf("partial failure: %d regions failed", len(errMap))
	}

	return results, nil
}

// ==================================================================================
// REGION-AWARE ACTIVITY EXECUTION OPTIONS
// ==================================================================================

// GetRegionActivityOptions returns activity options configured for a specific region
func GetRegionActivityOptions(regionCode string) map[string]interface{} {
	return map[string]interface{}{
		"TaskQueue":           "activity_" + regionCode,
		"StartToCloseTimeout": 30 * time.Minute,
	}
}

// ==================================================================================
// GLOBAL WORKFLOW EXECUTION TRACKING
// ==================================================================================

// RecordGlobalWorkflowExecution logs a global workflow execution to the database
func RecordGlobalWorkflowExecution(ctx context.Context, db *sql.DB, executionID string, workflowName string, featureID string, targetRegions []string, status string) error {
	regionStr := ""
	for i, r := range targetRegions {
		if i > 0 {
			regionStr += ","
		}
		regionStr += r
	}

	_, err := db.ExecContext(ctx,
		`INSERT INTO global_workflow_execution(execution_id, workflow_name, feature_id, target_regions, status, total_regions)
         VALUES($1, $2, $3, $4, $5, $6)`,
		executionID, workflowName, featureID, regionStr, status, len(targetRegions),
	)

	return err
}

// UpdateGlobalWorkflowExecution updates workflow execution status
func UpdateGlobalWorkflowExecution(ctx context.Context, db *sql.DB, executionID string, status string, successful int, failed int, errorMsg string) error {
	_, err := db.ExecContext(ctx,
		`UPDATE global_workflow_execution
         SET status = $1, completed_at = now(), successful_regions = $2, failed_regions = $3, error_message = $4
         WHERE execution_id = $5`,
		status, successful, failed, errorMsg, executionID,
	)

	return err
}

// RecordRegionWorkflowExecution logs a region-specific workflow execution
func RecordRegionWorkflowExecution(ctx context.Context, db *sql.DB, executionID string, regionCode string, status string, durationMs int64, errorMsg string) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO region_workflow_execution(execution_id, region_code, status, duration_ms, error_message)
         VALUES($1, $2, $3, $4, $5)
         ON CONFLICT(execution_id, region_code)
         DO UPDATE SET status = $3, duration_ms = $4, error_message = $5, completed_at = now()`,
		executionID, regionCode, status, durationMs, errorMsg,
	)

	return err
}

// ==================================================================================
// REGION HEALTH & STATUS MONITORING
// ==================================================================================

// UpdateFeatureRegionStatus updates feature health status for a region
func UpdateFeatureRegionStatus(ctx context.Context, db *sql.DB, featureID string, regionCode string, status string, driftScore *float64, importanceScore *float64) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO global_feature_status(feature_id, region_code, status, last_drift_score, last_importance_score)
         VALUES($1, $2, $3, $4, $5)
         ON CONFLICT(feature_id, region_code)
         DO UPDATE SET status = $3, last_drift_score = COALESCE($4, last_drift_score),
                       last_importance_score = COALESCE($5, last_importance_score)`,
		featureID, regionCode, status, driftScore, importanceScore,
	)

	return err
}

// GetRegionHealth retrieves health status for a region
func GetRegionHealth(ctx context.Context, db *sql.DB, regionCode string) (map[string]interface{}, error) {
	row := db.QueryRowContext(ctx,
		`SELECT
                COUNT(DISTINCT feature_id) as total_features,
                SUM(CASE WHEN status = 'healthy' THEN 1 ELSE 0 END) as healthy_features,
                SUM(CASE WHEN status = 'degraded' THEN 1 ELSE 0 END) as degraded_features,
                SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed_features
         FROM global_feature_status
         WHERE region_code = $1`,
		regionCode,
	)

	var totalFeatures, healthyFeatures, degradedFeatures, failedFeatures int
	if err := row.Scan(&totalFeatures, &healthyFeatures, &degradedFeatures, &failedFeatures); err != nil {
		return nil, err
	}

	health := make(map[string]interface{})
	health["region"] = regionCode
	health["total_features"] = totalFeatures
	health["healthy_features"] = healthyFeatures
	health["degraded_features"] = degradedFeatures
	health["failed_features"] = failedFeatures

	if totalFeatures > 0 {
		health["health_percentage"] = float64(healthyFeatures) / float64(totalFeatures) * 100.0
	}

	return health, nil
}
