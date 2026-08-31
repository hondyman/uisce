package temporal

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.uber.org/zap"

	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/hondyman/uisce/backend/internal/temporal/activities"
	"github.com/hondyman/uisce/backend/internal/temporal/workflows"
	preaggactivities "github.com/hondyman/uisce/backend/temporal/activities"
	preaggworkflows "github.com/hondyman/uisce/backend/temporal/workflows"
)

// WorkerConfig wraps configuration for starting a Temporal worker
type WorkerConfig struct {
	TemporalServerAddress string
	Namespace             string
	TaskQueue             string
	DataConverter         interface{}
	DB                    *sql.DB
	ControlDB             *sql.DB
	Logger                *zap.SugaredLogger
	// StarRocksDSN, if set, enables RefreshPreAggWorkflow's StarRocks
	// materialized-view refresh activity. DSN format matches
	// go-sql-driver/mysql, e.g. "user:pass@tcp(host:9030)/". Left empty,
	// the workflow still registers and runs, but its StarRocks refresh
	// step is a no-op (see RefreshStarRocksMVActivity's nil-conn guard).
	StarRocksDSN string
}

// StartWorker creates and starts a Temporal worker with all workflows and activities registered
func StartWorker(cfg WorkerConfig) (worker.Worker, error) {
	// Default values for missing config
	if cfg.TemporalServerAddress == "" {
		cfg.TemporalServerAddress = "localhost:7233"
	}
	if cfg.Namespace == "" {
		cfg.Namespace = "default"
	}
	if cfg.TaskQueue == "" {
		cfg.TaskQueue = "analytics-worker"
	}

	// Create Temporal client
	c, err := client.NewClient(client.Options{
		HostPort:  cfg.TemporalServerAddress,
		Namespace: cfg.Namespace,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to create Temporal client: %w", err)
	}

	// Create worker
	w := worker.New(c, cfg.TaskQueue, worker.Options{})

	// Register all workflows
	registerWorkflows(w)

	// Register all activities with dependencies
	registerActivities(w, cfg.DB, cfg.ControlDB, cfg.Logger)
	registerPreAggRefresh(w, cfg.DB, cfg.StarRocksDSN)

	log.Printf("Temporal worker initialized: TaskQueue=%s, Namespace=%s", cfg.TaskQueue, cfg.Namespace)

	return w, nil
}

// registerWorkflows registers all workflow definitions
func registerWorkflows(w worker.Worker) {
	w.RegisterWorkflow(workflows.HourlyRollupWorkflow)
	w.RegisterWorkflow(workflows.RegionHourlyRollupWorkflow)
	w.RegisterWorkflow(workflows.DailySLAWorkflow)
	w.RegisterWorkflow(workflows.MLTrainingWorkflow)
	w.RegisterWorkflow(TenantOnboardingWorkflow)
	w.RegisterWorkflow(LakehouseMaintenanceWorkflow)
	w.RegisterWorkflow(workflows.CustomizationIntelligenceWorkflow)
	w.RegisterWorkflow(workflows.TenantInstanceProvisioningWorkflowFn)

	log.Println("Workflows registered: HourlyRollupWorkflow, RegionHourlyRollupWorkflow, DailySLAWorkflow, MLTrainingWorkflow, TenantOnboardingWorkflow, LakehouseMaintenanceWorkflow, CustomizationIntelligenceWorkflow, TenantInstanceProvisioningWorkflowFn")
}

// registerActivities registers all activity definitions
func registerActivities(w worker.Worker, db *sql.DB, controlDB *sql.DB, logger *zap.SugaredLogger) {
	// Register activity functions directly
	w.RegisterActivity(activities.RunSparkJobActivity)
	w.RegisterActivity(activities.RunPythonScriptActivity)
	w.RegisterActivity(activities.RunCustomizationIntelligenceETL)
	w.RegisterActivity(activities.PublishEventActivity)

	// Register tenant activities struct methods
	act := &TenantActivities{}
	w.RegisterActivity(act.CreatePostgresTenant)
	w.RegisterActivity(act.RollbackPostgresTenant)
	w.RegisterActivity(act.InitializeMinIOPrefix)
	w.RegisterActivity(act.RollbackMinIOPrefix)
	w.RegisterActivity(act.ProvisionPolarisCatalog)
	w.RegisterActivity(act.DeprovisionPolarisCatalog)
	w.RegisterActivity(act.ExpireIcebergSnapshots)
	w.RegisterActivity(act.RemoveOrphanFiles)
	w.RegisterActivity(act.CompactManifests)

	// Register tenant provisioning activities
	if db != nil && controlDB != nil && logger != nil {
		provisioningActs := activities.NewTenantProvisioningActivities(db, controlDB, logger)
		w.RegisterActivity(provisioningActs.RegisterTenant)
		w.RegisterActivity(provisioningActs.RollbackRegisterTenant)
		w.RegisterActivity(provisioningActs.RegisterInstance)
		w.RegisterActivity(provisioningActs.RollbackRegisterInstance)
		w.RegisterActivity(provisioningActs.CreateTenantDatabase)
		w.RegisterActivity(provisioningActs.RollbackCreateTenantDatabase)
		w.RegisterActivity(provisioningActs.CloneSchemaFromGoldCopy)
		w.RegisterActivity(provisioningActs.CreateLakekeeperNamespace)
		w.RegisterActivity(provisioningActs.RollbackCreateLakekeeperNamespace)
		w.RegisterActivity(provisioningActs.CloneGoldCopyProducts)
		w.RegisterActivity(provisioningActs.RollbackCloneGoldCopyProducts)
		w.RegisterActivity(provisioningActs.EmitProvisioningEvent)
		w.RegisterActivity(provisioningActs.UpdateTenantStatus)
		w.RegisterActivity(provisioningActs.UpdateInstanceStatus)
		w.RegisterActivity(provisioningActs.GetGoldCopyInfo)
		w.RegisterActivity(provisioningActs.HealthCheck)
		log.Println("Tenant provisioning activities registered")
	}

	log.Println("Activities registered: RunDataFusionQueryActivity, RunSparkJobActivity, RunPythonScriptActivity, PublishEventActivity, TenantActivities, TenantProvisioningActivities")
}

// registerPreAggRefresh registers RefreshPreAggWorkflow and its activities.
// This workflow previously existed but was never registered with any
// Temporal worker, so nothing could ever execute it — see
// PLAN_STUDIO_EVENTS_AUDIT.md item 8. db is required (activities need it
// for lifecycle state transitions); starrocksDSN is optional and, left
// empty, makes the StarRocks refresh step a documented no-op rather than
// an error.
func registerPreAggRefresh(w worker.Worker, db *sql.DB, starrocksDSN string) {
	if db == nil {
		log.Println("registerPreAggRefresh: no DB configured, skipping RefreshPreAggWorkflow registration")
		return
	}
	sqlxDB := sqlx.NewDb(db, "postgres")

	var starrocksConn *sqlx.DB
	if starrocksDSN != "" {
		conn, err := sqlx.Open("mysql", starrocksDSN)
		if err != nil {
			log.Printf("registerPreAggRefresh: failed to open StarRocks connection, refresh step will no-op: %v", err)
		} else {
			starrocksConn = conn
		}
	}

	lifecycleSvc := analytics.NewPreAggLifecycleService(sqlxDB)
	// preAggSvc and trinoConn are unused by the activities RefreshPreAggWorkflow
	// actually calls (Iceberg/Trino refresh is disabled) — nil is safe here.
	preaggActs := preaggactivities.NewPreAggRefreshActivities(sqlxDB, nil, lifecycleSvc, nil, starrocksConn)

	w.RegisterWorkflow(preaggworkflows.RefreshPreAggWorkflow)
	w.RegisterActivity(preaggActs.MarkPreAggRefreshingActivity)
	w.RegisterActivity(preaggActs.RefreshStarRocksMVActivity)
	w.RegisterActivity(preaggActs.MarkPreAggFailedActivity)
	w.RegisterActivity(preaggActs.MarkPreAggActiveActivity)
	w.RegisterActivity(preaggActs.FetchPreAggStatsActivity)
	w.RegisterActivity(preaggActs.ScheduleNextRefreshActivity)

	log.Printf("RefreshPreAggWorkflow registered (StarRocks refresh %s)", map[bool]string{true: "enabled", false: "disabled: no STARROCKS_DSN configured"}[starrocksConn != nil])
}
