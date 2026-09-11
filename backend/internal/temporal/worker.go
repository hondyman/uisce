package temporal

import (
	"database/sql"
	"fmt"
	"log"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.uber.org/zap"

	"github.com/hondyman/uisce/backend/internal/temporal/activities"
	"github.com/hondyman/uisce/backend/internal/temporal/workflows"
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

	// Report generation workflows registered on the analytics-worker task queue.
	// These were previously defined but unregistered (would have failed with
	// "unknown workflow type" on dispatch). Joining the existing worker avoids
	// separate process/supervision overhead at current scale.
	w.RegisterWorkflow(workflows.ReportGenerationWorkflow)
	w.RegisterWorkflow(workflows.ClientBurstReportWorkflow)

	log.Println("Workflows registered: HourlyRollupWorkflow, RegionHourlyRollupWorkflow, DailySLAWorkflow, MLTrainingWorkflow, TenantOnboardingWorkflow, LakehouseMaintenanceWorkflow, CustomizationIntelligenceWorkflow, TenantInstanceProvisioningWorkflowFn, ReportGenerationWorkflow, ClientBurstReportWorkflow")
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

	// Register report generation activities. db may be nil in test environments
	// that use the in-process test executor — registration is safe with nil db;
	// StoreExecutionResultActivity will return an error if called with nil db.
	reportActs := activities.NewReportActivities(db)
	w.RegisterActivity(reportActs.QuerySemanticViewsActivity)
	w.RegisterActivity(reportActs.GenerateArtifactActivity)
	w.RegisterActivity(reportActs.StoreExecutionResultActivity)
	// Burst workflow activities (used by ClientBurstReportWorkflow)
	w.RegisterActivity(reportActs.EvaluateReportCalendarActivity)
	w.RegisterActivity(reportActs.ResolveClientSlicesActivity)
	w.RegisterActivity(reportActs.InitBurstBatchActivity)
	w.RegisterActivity(reportActs.RenderAndStoreClientArtifactActivity)
	w.RegisterActivity(reportActs.FinalizeBurstBatchActivity)
	w.RegisterActivity(reportActs.DispatchClientDistributionsActivity)
	log.Println("Report activities registered: QuerySemanticViewsActivity, GenerateArtifactActivity, StoreExecutionResultActivity, burst activities")

	log.Println("Activities registered: RunDataFusionQueryActivity, RunSparkJobActivity, RunPythonScriptActivity, PublishEventActivity, TenantActivities, TenantProvisioningActivities, ReportActivities")
}
