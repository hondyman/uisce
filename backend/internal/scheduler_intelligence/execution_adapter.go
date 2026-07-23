package scheduler_intelligence

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
)

// TaskQueueScheduler is the Temporal task queue for scheduler intelligence
const TaskQueueScheduler = "semlayer-scheduler"

// ExecutionAdapter bridges the Scheduler Intelligence Layer with Temporal
type ExecutionAdapter struct {
	temporalClient client.Client
	repo           *Repository
	logger         *slog.Logger
}

// NewExecutionAdapter creates a new execution adapter
func NewExecutionAdapter(tc client.Client, repo *Repository, logger *slog.Logger) *ExecutionAdapter {
	ea := &ExecutionAdapter{
		temporalClient: tc,
		repo:           repo,
		logger:         logger,
	}
	return ea
}

// ============================================================================
// Temporal Router
// ============================================================================

// RoutingConfig defines mapping for regional routing
type RoutingConfig struct {
	Region string
	NS     string
	TQ     string
}

// Route determines the target namespace and task queue for a job
func (ea *ExecutionAdapter) Route(job *Job) (string, string) {
	// 1. Check for residency rules in job
	// In production, lookup tenant region from a metadata service
	region := "global"
	if len(job.ResidencyRules) > 0 {
		var rules map[string]interface{}
		json.Unmarshal(job.ResidencyRules, &rules)
		if r, ok := rules["region"].(string); ok {
			region = r
		}
	}

	// 2. Map region to task queue
	switch strings.ToUpper(region) {
	case "EU":
		return "scheduler-eu", "semlayer-scheduler-eu"
	case "US":
		return "scheduler-us", "semlayer-scheduler-us"
	default:
		return "default", TaskQueueScheduler
	}
}

// ============================================================================
// Job Execution
// ============================================================================

// ScheduledJobWorkflowInput defines the input for scheduled job workflows
type ScheduledJobWorkflowInput struct {
	JobID        string                 `json:"job_id"`
	JobRunID     string                 `json:"job_run_id"`
	TenantID     string                 `json:"tenant_id"`
	Scope        string                 `json:"scope"`
	DatasourceID string                 `json:"datasource_id,omitempty"`
	JobType      string                 `json:"job_type"`
	Parameters   map[string]interface{} `json:"parameters"`
	Timeout      time.Duration          `json:"timeout"`
	Priority     int                    `json:"priority"`
}

// ScheduledJobWorkflowResult defines the result of a job execution
type ScheduledJobWorkflowResult struct {
	Success      bool                   `json:"success"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	Result       map[string]interface{} `json:"result,omitempty"`
	DurationMS   int64                  `json:"duration_ms"`
}

// ExecuteJob starts a Temporal workflow for a job
func (ea *ExecutionAdapter) ExecuteJob(ctx context.Context, job *Job, run *JobRun) error {
	// Parse parameters
	var params map[string]interface{}
	if len(job.Parameters) > 0 {
		json.Unmarshal(job.Parameters, &params)
	}

	tenantIDStr := ""
	if job.TenantID != nil {
		tenantIDStr = job.TenantID.String()
	}
	// Snapshot semantic bindings from job
	run.SemanticBindings = job.SemanticBindings

	// Build workflow input
	input := ScheduledJobWorkflowInput{
		JobID:      job.ID.String(),
		JobRunID:   run.ID.String(),
		TenantID:   tenantIDStr,
		Scope:      string(job.Scope),
		JobType:    job.JobType,
		Parameters: params,
		Timeout:    time.Duration(job.TimeoutSeconds) * time.Second,
		Priority:   job.Priority,
	}

	if job.DatasourceID != nil {
		input.DatasourceID = job.DatasourceID.String()
	}

	workflowID := fmt.Sprintf("scheduled-job-%s-%s", job.ID.String(), run.ID.String())

	_, taskQueue := ea.Route(job)

	workflowOptions := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: taskQueue,
	}

	we, err := ea.temporalClient.ExecuteWorkflow(ctx, workflowOptions, "ScheduledJobWorkflow", input)
	if err != nil {
		ea.logger.Error("Failed to start job workflow",
			"job_id", job.ID,
			"run_id", run.ID,
			"error", err,
		)
		return fmt.Errorf("failed to start workflow: %w", err)
	}

	// Update job run with Temporal IDs
	run.TemporalWorkflowID = we.GetID()
	run.TemporalRunID = we.GetRunID()
	run.TaskQueue = taskQueue

	ea.logger.Info("Started job workflow",
		"job_id", job.ID,
		"run_id", run.ID,
		"workflow_id", we.GetID(),
		"temporal_run_id", we.GetRunID(),
	)

	return nil
}

// ============================================================================
// DAG Execution
// ============================================================================

// DAGExecutionWorkflowInput defines the input for DAG execution workflows
type DAGExecutionWorkflowInput struct {
	DAGID           string        `json:"dag_id"`
	DAGRunID        string        `json:"dag_run_id"`
	TenantID        string        `json:"tenant_id"`
	Scope           string        `json:"scope"`
	Nodes           []DAGNode     `json:"nodes"`
	Edges           []DAGEdge     `json:"edges"`
	MaxParallelJobs int           `json:"max_parallel_jobs"`
	FailFast        bool          `json:"fail_fast"`
	Timeout         time.Duration `json:"timeout"`
}

// DAGExecutionWorkflowResult defines the result of DAG execution
type DAGExecutionWorkflowResult struct {
	Success       bool   `json:"success"`
	CompletedJobs int    `json:"completed_jobs"`
	FailedJobs    int    `json:"failed_jobs"`
	SkippedJobs   int    `json:"skipped_jobs"`
	ErrorMessage  string `json:"error_message,omitempty"`
	DurationMS    int64  `json:"duration_ms"`
}

// ExecuteDAG starts a Temporal workflow for a DAG
func (ea *ExecutionAdapter) ExecuteDAG(ctx context.Context, dag *DAG, run *DAGRun) error {
	// Parse nodes and edges
	var nodes []DAGNode
	var edges []DAGEdge
	json.Unmarshal(dag.Nodes, &nodes)
	json.Unmarshal(dag.Edges, &edges)

	tenantIDStr := ""
	if dag.TenantID != nil {
		tenantIDStr = dag.TenantID.String()
	}

	input := DAGExecutionWorkflowInput{
		DAGID:           dag.ID.String(),
		DAGRunID:        run.ID.String(),
		TenantID:        tenantIDStr,
		Scope:           string(dag.Scope),
		Nodes:           nodes,
		Edges:           edges,
		MaxParallelJobs: dag.MaxParallelJobs,
		FailFast:        dag.FailFast,
		Timeout:         time.Duration(dag.TimeoutSeconds) * time.Second,
	}

	workflowID := fmt.Sprintf("dag-execution-%s-%s", dag.ID.String(), run.ID.String())

	workflowOptions := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: TaskQueueScheduler,
	}

	we, err := ea.temporalClient.ExecuteWorkflow(ctx, workflowOptions, "DAGExecutionWorkflow", input)
	if err != nil {
		ea.logger.Error("Failed to start DAG workflow",
			"dag_id", dag.ID,
			"run_id", run.ID,
			"error", err,
		)
		return fmt.Errorf("failed to start DAG workflow: %w", err)
	}

	// Update DAG run with Temporal IDs
	run.TemporalWorkflowID = we.GetID()
	run.TemporalRunID = we.GetRunID()

	ea.logger.Info("Started DAG workflow",
		"dag_id", dag.ID,
		"run_id", run.ID,
		"workflow_id", we.GetID(),
		"temporal_run_id", we.GetRunID(),
	)

	return nil
}

// ============================================================================
// Workflow Control
// ============================================================================

// Signal types for workflow control
const (
	SignalPause  = "pause"
	SignalResume = "resume"
	SignalCancel = "cancel"
)

// PauseJobRun sends a pause signal to a running job workflow
func (ea *ExecutionAdapter) PauseJobRun(ctx context.Context, run *JobRun) error {
	if run.TemporalWorkflowID == "" {
		return fmt.Errorf("job run has no associated workflow")
	}

	err := ea.temporalClient.SignalWorkflow(ctx, run.TemporalWorkflowID, run.TemporalRunID, SignalPause, nil)
	if err != nil {
		return fmt.Errorf("failed to signal pause: %w", err)
	}

	ea.logger.Info("Sent pause signal to job workflow",
		"run_id", run.ID,
		"workflow_id", run.TemporalWorkflowID,
	)

	return nil
}

// ResumeJobRun sends a resume signal to a paused job workflow
func (ea *ExecutionAdapter) ResumeJobRun(ctx context.Context, run *JobRun) error {
	if run.TemporalWorkflowID == "" {
		return fmt.Errorf("job run has no associated workflow")
	}

	err := ea.temporalClient.SignalWorkflow(ctx, run.TemporalWorkflowID, run.TemporalRunID, SignalResume, nil)
	if err != nil {
		return fmt.Errorf("failed to signal resume: %w", err)
	}

	ea.logger.Info("Sent resume signal to job workflow",
		"run_id", run.ID,
		"workflow_id", run.TemporalWorkflowID,
	)

	return nil
}

// CancelJobRun cancels a running job workflow
func (ea *ExecutionAdapter) CancelJobRun(ctx context.Context, run *JobRun) error {
	if run.TemporalWorkflowID == "" {
		return fmt.Errorf("job run has no associated workflow")
	}

	err := ea.temporalClient.CancelWorkflow(ctx, run.TemporalWorkflowID, run.TemporalRunID)
	if err != nil {
		return fmt.Errorf("failed to cancel workflow: %w", err)
	}

	ea.logger.Info("Cancelled job workflow",
		"run_id", run.ID,
		"workflow_id", run.TemporalWorkflowID,
	)

	return nil
}

// TerminateJobRun forcefully terminates a running job workflow
func (ea *ExecutionAdapter) TerminateJobRun(ctx context.Context, run *JobRun, reason string) error {
	if run.TemporalWorkflowID == "" {
		return fmt.Errorf("job run has no associated workflow")
	}

	err := ea.temporalClient.TerminateWorkflow(ctx, run.TemporalWorkflowID, run.TemporalRunID, reason)
	if err != nil {
		return fmt.Errorf("failed to terminate workflow: %w", err)
	}

	ea.logger.Info("Terminated job workflow",
		"run_id", run.ID,
		"workflow_id", run.TemporalWorkflowID,
		"reason", reason,
	)

	return nil
}

// ============================================================================
// Workflow Status Queries
// ============================================================================

// GetWorkflowStatus queries the current status of a workflow
func (ea *ExecutionAdapter) GetWorkflowStatus(ctx context.Context, workflowID, runID string) (string, error) {
	resp, err := ea.temporalClient.DescribeWorkflowExecution(ctx, workflowID, runID)
	if err != nil {
		return "", fmt.Errorf("failed to describe workflow: %w", err)
	}

	return resp.WorkflowExecutionInfo.Status.String(), nil
}

// SyncJobRunStatus synchronizes the job run status with the Temporal workflow status
func (ea *ExecutionAdapter) SyncJobRunStatus(ctx context.Context, run *JobRun) error {
	if run.TemporalWorkflowID == "" {
		return nil
	}

	resp, err := ea.temporalClient.DescribeWorkflowExecution(ctx, run.TemporalWorkflowID, run.TemporalRunID)
	if err != nil {
		return fmt.Errorf("failed to describe workflow: %w", err)
	}

	// Map Temporal status to our status
	var newStatus RunStatus
	switch resp.WorkflowExecutionInfo.Status.String() {
	case "Running":
		newStatus = RunStatusRunning
	case "Completed":
		newStatus = RunStatusCompleted
	case "Failed":
		newStatus = RunStatusFailed
	case "Canceled", "Cancelled":
		newStatus = RunStatusCancelled
	case "Terminated":
		newStatus = RunStatusFailed
	case "TimedOut":
		newStatus = RunStatusFailed
	default:
		newStatus = RunStatusPending
	}

	if string(newStatus) != run.Status {
		run.Status = string(newStatus)
		// Update in repository would be called by the service layer
	}

	return nil
}

// ============================================================================
// Scheduled Trigger
// ============================================================================

// SchedulerTriggerInput defines input for the scheduler trigger workflow
type SchedulerTriggerInput struct {
	TenantID string `json:"tenant_id,omitempty"` // Empty means all tenants
}

// StartSchedulerTrigger starts the recurring scheduler trigger workflow
// This workflow runs on a schedule and triggers jobs that are due
func (ea *ExecutionAdapter) StartSchedulerTrigger(ctx context.Context, input SchedulerTriggerInput) error {
	workflowID := "scheduler-trigger"
	if input.TenantID != "" {
		workflowID = fmt.Sprintf("scheduler-trigger-%s", input.TenantID)
	}

	workflowOptions := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: TaskQueueScheduler,
		// Runs every minute to check for due jobs
		CronSchedule: "* * * * *",
	}

	_, err := ea.temporalClient.ExecuteWorkflow(ctx, workflowOptions, "SchedulerTriggerWorkflow", input)
	if err != nil {
		return fmt.Errorf("failed to start scheduler trigger: %w", err)
	}

	ea.logger.Info("Started scheduler trigger workflow",
		"workflow_id", workflowID,
		"tenant_id", input.TenantID,
	)

	return nil
}

// ============================================================================
// Retry/Rerun Support
// ============================================================================

// RetryJobRun creates a new run for a failed job
func (ea *ExecutionAdapter) RetryJobRun(ctx context.Context, failedRun *JobRun) (*JobRun, error) {
	newRun := &JobRun{
		ID:              uuid.New(),
		JobID:           failedRun.JobID,
		TenantID:        failedRun.TenantID,
		Status:          string(RunStatusPending),
		AttemptNumber:   failedRun.AttemptNumber + 1,
		TriggerType:     string(TriggerTypeAPI),
		InputParameters: failedRun.InputParameters,
	}

	now := time.Now()
	newRun.ScheduledAt = &now

	if err := ea.repo.CreateJobRun(ctx, newRun); err != nil {
		return nil, fmt.Errorf("failed to create retry run: %w", err)
	}

	ea.logger.Info("Created retry job run",
		"original_run_id", failedRun.ID,
		"new_run_id", newRun.ID,
		"attempt", newRun.AttemptNumber,
	)

	return newRun, nil
}
