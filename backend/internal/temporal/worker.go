package temporal

import (
	"fmt"
	"log"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"github.com/hondyman/uisce/backend/internal/temporal/activities"
	"github.com/hondyman/uisce/backend/internal/temporal/workflows"
)

// WorkerConfig wraps configuration for starting a Temporal worker
type WorkerConfig struct {
	TemporalServerAddress string
	Namespace             string
	TaskQueue             string
	DataConverter         interface{}
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

	// Register all activities
	registerActivities(w)

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

	log.Println("Workflows registered: HourlyRollupWorkflow, RegionHourlyRollupWorkflow, DailySLAWorkflow, MLTrainingWorkflow, TenantOnboardingWorkflow, LakehouseMaintenanceWorkflow")
}

// registerActivities registers all activity definitions
func registerActivities(w worker.Worker) {
	// Register activity functions directly
	w.RegisterActivity(activities.RunTrinoQueryActivity)
	w.RegisterActivity(activities.RunSparkJobActivity)
	w.RegisterActivity(activities.RunPythonScriptActivity)
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

	log.Println("Activities registered: RunTrinoQueryActivity, RunSparkJobActivity, RunPythonScriptActivity, PublishEventActivity, TenantActivities")
}
