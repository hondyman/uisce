package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

// TaskQueues defines all Temporal task queues used by Semlayer
const (
	TaskQueueAnalytics   = "semlayer-analytics"
	TaskQueueCatalog     = "semlayer-catalog"
	TaskQueueCompliance  = "semlayer-compliance"
	TaskQueueMaintenance = "semlayer-maintenance"
	TaskQueueReporting   = "semlayer-reporting"
	TaskQueueScheduler   = "semlayer-scheduler" // Scheduler Intelligence Layer
)

// WorkerConfig holds configuration for a Temporal worker
type WorkerConfig struct {
	TaskQueue                 string
	MaxConcurrentActivities   int
	MaxConcurrentWorkflows    int
	WorkerActivitiesPerSecond float64
	EnableSessionWorker       bool
}

// DefaultWorkerConfigs returns the default worker configurations
func DefaultWorkerConfigs() map[string]WorkerConfig {
	return map[string]WorkerConfig{
		TaskQueueAnalytics: {
			TaskQueue:                 TaskQueueAnalytics,
			MaxConcurrentActivities:   10,
			MaxConcurrentWorkflows:    50,
			WorkerActivitiesPerSecond: 100,
			EnableSessionWorker:       false,
		},
		TaskQueueCatalog: {
			TaskQueue:                 TaskQueueCatalog,
			MaxConcurrentActivities:   20,
			MaxConcurrentWorkflows:    100,
			WorkerActivitiesPerSecond: 200,
			EnableSessionWorker:       false,
		},
		TaskQueueCompliance: {
			TaskQueue:                 TaskQueueCompliance,
			MaxConcurrentActivities:   5,
			MaxConcurrentWorkflows:    20,
			WorkerActivitiesPerSecond: 50,
			EnableSessionWorker:       false,
		},
		TaskQueueMaintenance: {
			TaskQueue:                 TaskQueueMaintenance,
			MaxConcurrentActivities:   5,
			MaxConcurrentWorkflows:    10,
			WorkerActivitiesPerSecond: 20,
			EnableSessionWorker:       false,
		},
		TaskQueueReporting: {
			TaskQueue:                 TaskQueueReporting,
			MaxConcurrentActivities:   10,
			MaxConcurrentWorkflows:    30,
			WorkerActivitiesPerSecond: 100,
			EnableSessionWorker:       false,
		},
		TaskQueueScheduler: {
			TaskQueue:                 TaskQueueScheduler,
			MaxConcurrentActivities:   20,
			MaxConcurrentWorkflows:    100,
			WorkerActivitiesPerSecond: 200,
			EnableSessionWorker:       false,
		},
	}
}

// WorkerManager manages multiple Temporal workers
type WorkerManager struct {
	client  client.Client
	workers map[string]worker.Worker
	logger  *slog.Logger
}

// NewWorkerManager creates a new worker manager
func NewWorkerManager(c client.Client, logger *slog.Logger) *WorkerManager {
	return &WorkerManager{
		client:  c,
		workers: make(map[string]worker.Worker),
		logger:  logger,
	}
}

// RegisterWorker creates and registers a worker for a task queue
func (wm *WorkerManager) RegisterWorker(config WorkerConfig, workflows []interface{}, activities []interface{}) error {
	opts := worker.Options{
		MaxConcurrentActivityExecutionSize:     config.MaxConcurrentActivities,
		MaxConcurrentWorkflowTaskExecutionSize: config.MaxConcurrentWorkflows,
		WorkerActivitiesPerSecond:              config.WorkerActivitiesPerSecond,
		EnableSessionWorker:                    config.EnableSessionWorker,
	}

	w := worker.New(wm.client, config.TaskQueue, opts)

	// Register workflows
	for _, wf := range workflows {
		w.RegisterWorkflow(wf)
	}

	// Register activities
	for _, act := range activities {
		w.RegisterActivity(act)
	}

	wm.workers[config.TaskQueue] = w
	wm.logger.Info("Registered worker", "taskQueue", config.TaskQueue)

	return nil
}

// Start starts all registered workers
func (wm *WorkerManager) Start() error {
	for taskQueue, w := range wm.workers {
		if err := w.Start(); err != nil {
			return fmt.Errorf("failed to start worker for %s: %w", taskQueue, err)
		}
		wm.logger.Info("Started worker", "taskQueue", taskQueue)
	}
	return nil
}

// Stop gracefully stops all workers
func (wm *WorkerManager) Stop() {
	for taskQueue, w := range wm.workers {
		w.Stop()
		wm.logger.Info("Stopped worker", "taskQueue", taskQueue)
	}
}

// ============================================================================
// Analytics Workflows
// ============================================================================

// PreAggBuildWorkflowInput defines input for pre-aggregation building
type PreAggBuildWorkflowInput struct {
	TenantID     string   `json:"tenant_id"`
	DatasourceID string   `json:"datasource_id,omitempty"`
	CubeNames    []string `json:"cube_names,omitempty"`
	Force        bool     `json:"force"`
}

// PreAggBuildWorkflow builds pre-aggregations for Cube.js
func PreAggBuildWorkflow(ctx workflow.Context, input PreAggBuildWorkflowInput) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting pre-aggregation build", "tenantID", input.TenantID)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    5 * time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Build pre-aggregations
	var result PreAggBuildResult
	err := workflow.ExecuteActivity(ctx, BuildPreAggregations, input).Get(ctx, &result)
	if err != nil {
		return fmt.Errorf("failed to build pre-aggregations: %w", err)
	}

	logger.Info("Pre-aggregation build completed",
		"built", result.BuiltCount,
		"skipped", result.SkippedCount)

	return nil
}

type PreAggBuildResult struct {
	BuiltCount   int `json:"built_count"`
	SkippedCount int `json:"skipped_count"`
}

// CacheWarmingWorkflowInput defines input for cache warming
type CacheWarmingWorkflowInput struct {
	TenantID     string `json:"tenant_id"`
	DatasourceID string `json:"datasource_id,omitempty"`
	QueryType    string `json:"query_type,omitempty"` // "popular", "stale", "all"
}

// CacheWarmingWorkflow warms the analytics cache
func CacheWarmingWorkflow(ctx workflow.Context, input CacheWarmingWorkflowInput) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting cache warming", "tenantID", input.TenantID)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 15 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var result CacheWarmingResult
	err := workflow.ExecuteActivity(ctx, WarmCache, input).Get(ctx, &result)
	if err != nil {
		return fmt.Errorf("failed to warm cache: %w", err)
	}

	logger.Info("Cache warming completed", "warmed", result.QueriesWarmed)
	return nil
}

type CacheWarmingResult struct {
	QueriesWarmed int `json:"queries_warmed"`
}

// ============================================================================
// Catalog Workflows
// ============================================================================

// CatalogSyncWorkflowInput defines input for catalog synchronization
type CatalogSyncWorkflowInput struct {
	TenantID     string `json:"tenant_id"`
	DatasourceID string `json:"datasource_id"`
	FullSync     bool   `json:"full_sync"`
}

// CatalogSyncWorkflow synchronizes metadata catalog
func CatalogSyncWorkflow(ctx workflow.Context, input CatalogSyncWorkflowInput) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting catalog sync", "tenantID", input.TenantID, "fullSync", input.FullSync)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 60 * time.Minute,
		HeartbeatTimeout:    5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    5 * time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Step 1: Discover objects from source
	var discovery CatalogDiscoveryResult
	err := workflow.ExecuteActivity(ctx, DiscoverCatalogObjects, input).Get(ctx, &discovery)
	if err != nil {
		return fmt.Errorf("failed to discover catalog objects: %w", err)
	}

	// Step 2: Sync objects to local catalog
	var syncResult CatalogSyncResult
	syncInput := CatalogSyncActivityInput{
		TenantID:     input.TenantID,
		DatasourceID: input.DatasourceID,
		Objects:      discovery.Objects,
	}
	err = workflow.ExecuteActivity(ctx, SyncCatalogObjects, syncInput).Get(ctx, &syncResult)
	if err != nil {
		return fmt.Errorf("failed to sync catalog objects: %w", err)
	}

	// Step 3: Update search index
	err = workflow.ExecuteActivity(ctx, UpdateSearchIndex, input).Get(ctx, nil)
	if err != nil {
		logger.Warn("Failed to update search index, will retry on next sync", "error", err)
	}

	logger.Info("Catalog sync completed",
		"created", syncResult.Created,
		"updated", syncResult.Updated,
		"deleted", syncResult.Deleted)

	return nil
}

type CatalogDiscoveryResult struct {
	Objects []CatalogObject `json:"objects"`
}

type CatalogObject struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Name     string                 `json:"name"`
	Metadata map[string]interface{} `json:"metadata"`
}

type CatalogSyncActivityInput struct {
	TenantID     string          `json:"tenant_id"`
	DatasourceID string          `json:"datasource_id"`
	Objects      []CatalogObject `json:"objects"`
}

type CatalogSyncResult struct {
	Created int `json:"created"`
	Updated int `json:"updated"`
	Deleted int `json:"deleted"`
}

// ============================================================================
// Compliance Workflows
// ============================================================================

// ComplianceValidationWorkflowInput defines input for compliance validation
type ComplianceValidationWorkflowInput struct {
	TenantID       string   `json:"tenant_id"`
	FrameworkIDs   []string `json:"framework_ids,omitempty"`
	BundleIDs      []string `json:"bundle_ids,omitempty"`
	GenerateReport bool     `json:"generate_report"`
}

// ComplianceValidationWorkflow runs compliance validation checks
func ComplianceValidationWorkflow(ctx workflow.Context, input ComplianceValidationWorkflowInput) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting compliance validation", "tenantID", input.TenantID)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    5 * time.Minute,
			MaximumAttempts:    2,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Run validation
	var validationResult ComplianceValidationResult
	err := workflow.ExecuteActivity(ctx, ValidateCompliance, input).Get(ctx, &validationResult)
	if err != nil {
		return fmt.Errorf("failed to validate compliance: %w", err)
	}

	// Generate report if requested
	if input.GenerateReport {
		reportInput := ComplianceReportInput{
			TenantID:         input.TenantID,
			ValidationResult: validationResult,
		}
		err = workflow.ExecuteActivity(ctx, GenerateComplianceReport, reportInput).Get(ctx, nil)
		if err != nil {
			logger.Warn("Failed to generate compliance report", "error", err)
		}
	}

	// Send notifications if violations found
	if validationResult.ViolationCount > 0 {
		notifyInput := ComplianceNotificationInput{
			TenantID:   input.TenantID,
			Violations: validationResult.Violations,
		}
		err = workflow.ExecuteActivity(ctx, SendComplianceNotifications, notifyInput).Get(ctx, nil)
		if err != nil {
			logger.Warn("Failed to send compliance notifications", "error", err)
		}
	}

	logger.Info("Compliance validation completed",
		"passed", validationResult.PassedCount,
		"violations", validationResult.ViolationCount)

	return nil
}

type ComplianceValidationResult struct {
	PassedCount    int                   `json:"passed_count"`
	ViolationCount int                   `json:"violation_count"`
	Violations     []ComplianceViolation `json:"violations"`
}

type ComplianceViolation struct {
	RuleID      string `json:"rule_id"`
	ObjectID    string `json:"object_id"`
	ObjectType  string `json:"object_type"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
}

type ComplianceReportInput struct {
	TenantID         string                     `json:"tenant_id"`
	ValidationResult ComplianceValidationResult `json:"validation_result"`
}

type ComplianceNotificationInput struct {
	TenantID   string                `json:"tenant_id"`
	Violations []ComplianceViolation `json:"violations"`
}

// ============================================================================
// Maintenance Workflows
// ============================================================================

// CleanupWorkflowInput defines input for cleanup workflows
type CleanupWorkflowInput struct {
	TenantID      string `json:"tenant_id,omitempty"` // Empty = all tenants
	CleanupType   string `json:"cleanup_type"`        // "sessions", "audit", "soft-deleted"
	RetentionDays int    `json:"retention_days"`
	DryRun        bool   `json:"dry_run"`
}

// CleanupWorkflow runs maintenance cleanup tasks
func CleanupWorkflow(ctx workflow.Context, input CleanupWorkflowInput) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting cleanup", "type", input.CleanupType)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 60 * time.Minute,
		HeartbeatTimeout:    10 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    5 * time.Minute,
			MaximumAttempts:    2,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var result CleanupResult
	err := workflow.ExecuteActivity(ctx, PerformCleanup, input).Get(ctx, &result)
	if err != nil {
		return fmt.Errorf("failed to perform cleanup: %w", err)
	}

	logger.Info("Cleanup completed",
		"type", input.CleanupType,
		"deleted", result.DeletedCount,
		"dryRun", input.DryRun)

	return nil
}

type CleanupResult struct {
	DeletedCount int `json:"deleted_count"`
}

// HealthCheckWorkflowInput defines input for health checks
type HealthCheckWorkflowInput struct {
	CheckDatabase bool `json:"check_database"`
	CheckCache    bool `json:"check_cache"`
	CheckServices bool `json:"check_services"`
}

// HealthCheckWorkflow runs system health checks
func HealthCheckWorkflow(ctx workflow.Context, input HealthCheckWorkflowInput) error {
	logger := workflow.GetLogger(ctx)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    30 * time.Second,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var result HealthCheckResult
	err := workflow.ExecuteActivity(ctx, PerformHealthCheck, input).Get(ctx, &result)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	// Record metrics
	err = workflow.ExecuteActivity(ctx, RecordHealthMetrics, result).Get(ctx, nil)
	if err != nil {
		logger.Warn("Failed to record health metrics", "error", err)
	}

	// Alert if unhealthy
	if !result.Healthy {
		err = workflow.ExecuteActivity(ctx, SendHealthAlert, result).Get(ctx, nil)
		if err != nil {
			logger.Error("Failed to send health alert", "error", err)
		}
	}

	return nil
}

type HealthCheckResult struct {
	Healthy    bool             `json:"healthy"`
	Components map[string]bool  `json:"components"`
	Latencies  map[string]int64 `json:"latencies_ms"`
	Errors     []string         `json:"errors,omitempty"`
}

// ============================================================================
// Reporting Workflows
// ============================================================================

// ReportGenerationWorkflowInput defines input for report generation
type ReportGenerationWorkflowInput struct {
	TenantID   string   `json:"tenant_id"`
	ReportType string   `json:"report_type"` // "usage", "billing", "slo", "audit"
	StartDate  string   `json:"start_date"`
	EndDate    string   `json:"end_date"`
	Recipients []string `json:"recipients,omitempty"`
}

// ReportGenerationWorkflow generates and distributes reports
func ReportGenerationWorkflow(ctx workflow.Context, input ReportGenerationWorkflowInput) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting report generation", "type", input.ReportType, "tenant", input.TenantID)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    5 * time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Generate report
	var reportResult ReportResult
	err := workflow.ExecuteActivity(ctx, GenerateReport, input).Get(ctx, &reportResult)
	if err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	// Store report
	err = workflow.ExecuteActivity(ctx, StoreReport, reportResult).Get(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to store report: %w", err)
	}

	// Distribute to recipients
	if len(input.Recipients) > 0 {
		distributeInput := ReportDistributionInput{
			ReportID:   reportResult.ReportID,
			Recipients: input.Recipients,
		}
		err = workflow.ExecuteActivity(ctx, DistributeReport, distributeInput).Get(ctx, nil)
		if err != nil {
			logger.Warn("Failed to distribute report", "error", err)
		}
	}

	logger.Info("Report generation completed", "reportID", reportResult.ReportID)
	return nil
}

type ReportResult struct {
	ReportID string `json:"report_id"`
	URL      string `json:"url"`
	Size     int64  `json:"size_bytes"`
}

type ReportDistributionInput struct {
	ReportID   string   `json:"report_id"`
	Recipients []string `json:"recipients"`
}

// ============================================================================
// Activity Stubs (implement these in separate files)
// ============================================================================

// Analytics activities
func BuildPreAggregations(ctx context.Context, input PreAggBuildWorkflowInput) (PreAggBuildResult, error) {
	// TODO: Implement
	return PreAggBuildResult{}, nil
}

func WarmCache(ctx context.Context, input CacheWarmingWorkflowInput) (CacheWarmingResult, error) {
	// TODO: Implement
	return CacheWarmingResult{}, nil
}

// Catalog activities
func DiscoverCatalogObjects(ctx context.Context, input CatalogSyncWorkflowInput) (CatalogDiscoveryResult, error) {
	// TODO: Implement
	return CatalogDiscoveryResult{}, nil
}

func SyncCatalogObjects(ctx context.Context, input CatalogSyncActivityInput) (CatalogSyncResult, error) {
	// TODO: Implement
	return CatalogSyncResult{}, nil
}

func UpdateSearchIndex(ctx context.Context, input CatalogSyncWorkflowInput) error {
	// TODO: Implement
	return nil
}

// Compliance activities
func ValidateCompliance(ctx context.Context, input ComplianceValidationWorkflowInput) (ComplianceValidationResult, error) {
	// TODO: Implement
	return ComplianceValidationResult{}, nil
}

func GenerateComplianceReport(ctx context.Context, input ComplianceReportInput) error {
	// TODO: Implement
	return nil
}

func SendComplianceNotifications(ctx context.Context, input ComplianceNotificationInput) error {
	// TODO: Implement
	return nil
}

// Maintenance activities
func PerformCleanup(ctx context.Context, input CleanupWorkflowInput) (CleanupResult, error) {
	// TODO: Implement
	return CleanupResult{}, nil
}

func PerformHealthCheck(ctx context.Context, input HealthCheckWorkflowInput) (HealthCheckResult, error) {
	// TODO: Implement
	return HealthCheckResult{Healthy: true, Components: make(map[string]bool), Latencies: make(map[string]int64)}, nil
}

func RecordHealthMetrics(ctx context.Context, result HealthCheckResult) error {
	// TODO: Implement
	return nil
}

func SendHealthAlert(ctx context.Context, result HealthCheckResult) error {
	// TODO: Implement
	return nil
}

// Reporting activities
func GenerateReport(ctx context.Context, input ReportGenerationWorkflowInput) (ReportResult, error) {
	// TODO: Implement
	return ReportResult{}, nil
}

func StoreReport(ctx context.Context, result ReportResult) error {
	// TODO: Implement
	return nil
}

func DistributeReport(ctx context.Context, input ReportDistributionInput) error {
	// TODO: Implement
	return nil
}
