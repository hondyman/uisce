package temporal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	sdktemporal "go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.uber.org/zap"

	"github.com/hondyman/uisce/backend/internal/temporal/activities"
	"github.com/hondyman/uisce/backend/internal/temporal/workflows"
	"github.com/jmoiron/sqlx"
)

// MDMReconciliationScheduleID is the schedule identifier for periodic MDM sweeps.
const MDMReconciliationScheduleID = "mdm-reconciliation-sweep"

// EnsureMDMReconciliationSchedule ensures that the 5-minute recurring reconciliation schedule
// is created in Temporal to sweep unlinked pending records and expire stale proposals.
// Idempotent across multiple concurrent worker instances (handles AlreadyExists safely).
func EnsureMDMReconciliationSchedule(ctx context.Context, c client.Client) error {
	if c == nil {
		return nil
	}
	spec := client.ScheduleSpec{
		CronExpressions: []string{"*/5 * * * *"},
	}
	action := &client.ScheduleWorkflowAction{
		ID:        "mdm-reconciliation-sweep-run",
		Workflow:  workflows.MDMReconciliationWorkflow,
		Args:      []any{workflows.MDMReconciliationInput{MaxPendingAge: 5 * time.Minute, ExpiryAge: 7 * 24 * time.Hour}},
		TaskQueue: workflows.MDMGovernanceTaskQueue,
	}
	_, err := c.ScheduleClient().Create(ctx, client.ScheduleOptions{
		ID:      MDMReconciliationScheduleID,
		Spec:    spec,
		Action:  action,
		Overlap: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
		Note:    "Periodic reconciliation sweep for MDM approval workflows",
	})
	if err == nil {
		log.Printf("[MDMGovernance] Created Temporal schedule: %s", MDMReconciliationScheduleID)
		return nil
	}
	var exists *serviceerror.AlreadyExists
	if errors.As(err, &exists) || errors.Is(err, sdktemporal.ErrScheduleAlreadyRunning) || (err != nil && strings.Contains(strings.ToLower(err.Error()), "already exists")) {
		log.Printf("[MDMGovernance] Temporal schedule %s already exists (concurrent worker startup safe)", MDMReconciliationScheduleID)
		return nil
	}
	return err
}

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

	// Ensure system schedules are established
	_ = EnsureMDMReconciliationSchedule(context.Background(), c)

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
	w.RegisterWorkflow(workflows.RuleReviewWorkflow)
	w.RegisterWorkflow(workflows.RuleHealthCheckWorkflow)
	w.RegisterWorkflow(workflows.ViolationAuditAnchorWorkflow)
	w.RegisterWorkflow(workflows.MDMOverrideApprovalWorkflow)
	w.RegisterWorkflow(workflows.MDMMergeApprovalWorkflow)
	w.RegisterWorkflow(workflows.MDMConfigChangeApprovalWorkflow)
	w.RegisterWorkflow(workflows.MDMReconciliationWorkflow)

	log.Println("Workflows registered: HourlyRollupWorkflow, RegionHourlyRollupWorkflow, DailySLAWorkflow, MLTrainingWorkflow, TenantOnboardingWorkflow, LakehouseMaintenanceWorkflow, CustomizationIntelligenceWorkflow, TenantInstanceProvisioningWorkflowFn, ReportGenerationWorkflow, ClientBurstReportWorkflow, RuleReviewWorkflow, RuleHealthCheckWorkflow, ViolationAuditAnchorWorkflow, MDMOverrideApprovalWorkflow, MDMMergeApprovalWorkflow, MDMConfigChangeApprovalWorkflow, MDMReconciliationWorkflow")
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

	// Register rule governance & health activities
	if db != nil {
		sqlxDB := sqlx.NewDb(db, "postgres")
		govActs := activities.NewRuleGovernanceActivities(sqlxDB, logger)
		w.RegisterActivity(govActs.RecordApprovalActivity)
		w.RegisterActivity(govActs.RecordRejectionActivity)
		w.RegisterActivity(govActs.PublishVersionActivity)
		w.RegisterActivity(govActs.EscalateReviewActivity)

		healthActs := activities.NewRuleHealthActivities(sqlxDB, logger)
		w.RegisterActivity(healthActs.RunRuleHealthCheckActivity)

		anchorActs := activities.NewAuditAnchorActivities(sqlxDB, logger)
		w.RegisterActivity(anchorActs.RunAuditAnchorActivity)

		mdmActs := activities.NewMDMApprovalActivities(sqlxDB, logger)
		w.RegisterActivity(mdmActs.CheckProposalStatusActivity)
		w.RegisterActivity(mdmActs.RecordOverrideVoteActivity)
		w.RegisterActivity(mdmActs.ApplyOverrideActivity)
		w.RegisterActivity(mdmActs.RejectOverrideActivity)
		w.RegisterActivity(mdmActs.WithdrawOverrideActivity)
		w.RegisterActivity(mdmActs.RecordMergeVoteActivity)
		w.RegisterActivity(mdmActs.ApplyMergeActivity)
		w.RegisterActivity(mdmActs.RejectMergeActivity)
		w.RegisterActivity(mdmActs.ApplyConfigChangeActivity)
		w.RegisterActivity(mdmActs.RejectConfigChangeActivity)
		w.RegisterActivity(mdmActs.WithdrawConfigChangeActivity)
		w.RegisterActivity(mdmActs.EscalateMDMReviewActivity)
		w.RegisterActivity(mdmActs.ReconcilePendingMDMWorkflowsActivity)
	}

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
