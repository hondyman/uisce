package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// Note: TaskQueueScheduler is defined in temporal_workers.go

// ============================================================================
// Scheduled Job Workflow
// ============================================================================

// ScheduledJobWorkflowInput defines the input for scheduled job workflows
type ScheduledJobWorkflowInput struct {
	JobID         string                 `json:"job_id"`
	JobRunID      string                 `json:"job_run_id"`
	TenantID      string                 `json:"tenant_id"`
	DatasourceID  string                 `json:"datasource_id,omitempty"`
	JobType       string                 `json:"job_type"`
	Parameters    map[string]interface{} `json:"parameters"`
	Timeout       time.Duration          `json:"timeout"`
	Priority      int                    `json:"priority"`
	ExecutionMode string                 `json:"execution_mode,omitempty"` // NORMAL, DRY_RUN, CANARY
	Compliance    *ComplianceInfo        `json:"compliance,omitempty"`
}

// ComplianceInfo carries compliance metadata to the workflow
type ComplianceInfo struct {
	PII         bool   `json:"pii"`
	Residency   string `json:"residency"`
	Sensitivity string `json:"sensitivity"`
}

// ScheduledJobWorkflowResult defines the result of a job execution
type ScheduledJobWorkflowResult struct {
	Success      bool                   `json:"success"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	Result       map[string]interface{} `json:"result,omitempty"`
	DurationMS   int64                  `json:"duration_ms"`
}

// ScheduledJobWorkflow executes a scheduled job
func ScheduledJobWorkflow(ctx workflow.Context, input ScheduledJobWorkflowInput) (*ScheduledJobWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting scheduled job workflow",
		"job_id", input.JobID,
		"job_run_id", input.JobRunID,
		"job_type", input.JobType,
		"mode", input.ExecutionMode,
		"compliance", input.Compliance,
	)

	if input.ExecutionMode == "DRY_RUN" {
		logger.Info("DRY_RUN: Skipping side-effecting activity", "job_id", input.JobID)
		return &ScheduledJobWorkflowResult{
			Success:    true,
			Result:     map[string]interface{}{"dry_run": true, "msg": "Job execution skipped in dry-run mode"},
			DurationMS: 0,
		}, nil
	}

	startTime := workflow.Now(ctx)

	// Set up activity options
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: input.Timeout,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute * 5,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	// Signal channels for pause/resume/cancel
	pauseChan := workflow.GetSignalChannel(ctx, "pause")
	resumeChan := workflow.GetSignalChannel(ctx, "resume")
	cancelChan := workflow.GetSignalChannel(ctx, "cancel")

	// Execution state
	paused := false
	cancelled := false

	// Activity result
	var activityResult map[string]interface{}
	var activityErr error

	// Run the job activity with signal handling
	activityFuture := workflow.ExecuteActivity(ctx, "ExecuteScheduledJobActivity", input)

	// Wait for completion or signals
	for {
		selector := workflow.NewSelector(ctx)

		// Activity completion
		selector.AddFuture(activityFuture, func(f workflow.Future) {
			activityErr = f.Get(ctx, &activityResult)
		})

		// Pause signal
		selector.AddReceive(pauseChan, func(c workflow.ReceiveChannel, more bool) {
			var signal interface{}
			c.Receive(ctx, &signal)
			paused = true
			logger.Info("Job paused", "job_id", input.JobID)
		})

		// Resume signal
		selector.AddReceive(resumeChan, func(c workflow.ReceiveChannel, more bool) {
			var signal interface{}
			c.Receive(ctx, &signal)
			paused = false
			logger.Info("Job resumed", "job_id", input.JobID)
		})

		// Cancel signal
		selector.AddReceive(cancelChan, func(c workflow.ReceiveChannel, more bool) {
			var signal interface{}
			c.Receive(ctx, &signal)
			cancelled = true
			logger.Info("Job cancelled", "job_id", input.JobID)
		})

		selector.Select(ctx)

		// Check if cancelled
		if cancelled {
			return &ScheduledJobWorkflowResult{
				Success:      false,
				ErrorMessage: "Job was cancelled",
				DurationMS:   workflow.Now(ctx).Sub(startTime).Milliseconds(),
			}, nil
		}

		// If paused, wait for resume
		if paused {
			for paused && !cancelled {
				selector := workflow.NewSelector(ctx)
				selector.AddReceive(resumeChan, func(c workflow.ReceiveChannel, more bool) {
					var signal interface{}
					c.Receive(ctx, &signal)
					paused = false
				})
				selector.AddReceive(cancelChan, func(c workflow.ReceiveChannel, more bool) {
					var signal interface{}
					c.Receive(ctx, &signal)
					cancelled = true
				})
				selector.Select(ctx)
			}
			if cancelled {
				return &ScheduledJobWorkflowResult{
					Success:      false,
					ErrorMessage: "Job was cancelled while paused",
					DurationMS:   workflow.Now(ctx).Sub(startTime).Milliseconds(),
				}, nil
			}
			continue
		}

		// Activity completed
		if activityFuture.IsReady() {
			break
		}
	}

	endTime := workflow.Now(ctx)
	durationMS := endTime.Sub(startTime).Milliseconds()

	if activityErr != nil {
		logger.Error("Job failed", "job_id", input.JobID, "error", activityErr)
		return &ScheduledJobWorkflowResult{
			Success:      false,
			ErrorMessage: activityErr.Error(),
			DurationMS:   durationMS,
		}, nil
	}

	logger.Info("Job completed successfully",
		"job_id", input.JobID,
		"duration_ms", durationMS,
	)

	return &ScheduledJobWorkflowResult{
		Success:    true,
		Result:     activityResult,
		DurationMS: durationMS,
	}, nil
}

// ============================================================================
// DAG Execution Workflow
// ============================================================================

// DAGNode represents a node in the DAG
type DAGNode struct {
	ID         string                 `json:"id"`
	JobID      string                 `json:"job_id"`
	Conditions map[string]interface{} `json:"conditions,omitempty"`
}

// DAGEdge represents an edge in the DAG
type DAGEdge struct {
	FromNodeID string                 `json:"from_node_id"`
	ToNodeID   string                 `json:"to_node_id"`
	Type       string                 `json:"type,omitempty"` // success, completion, any
	Conditions map[string]interface{} `json:"conditions,omitempty"`
}

// DAGExecutionWorkflowInput defines the input for DAG execution workflows
type DAGExecutionWorkflowInput struct {
	DAGID           string        `json:"dag_id"`
	DAGRunID        string        `json:"dag_run_id"`
	TenantID        string        `json:"tenant_id"`
	Nodes           []DAGNode     `json:"nodes"`
	Edges           []DAGEdge     `json:"edges"`
	MaxParallelJobs int           `json:"max_parallel_jobs"`
	FailFast        bool          `json:"fail_fast"`
	Timeout         time.Duration `json:"timeout"`
	ExecutionMode   string        `json:"execution_mode,omitempty"` // NORMAL, DRY_RUN, CANARY
}

// DAGExecutionWorkflowResult defines the result of DAG execution
type DAGExecutionWorkflowResult struct {
	Success       bool              `json:"success"`
	CompletedJobs int               `json:"completed_jobs"`
	FailedJobs    int               `json:"failed_jobs"`
	SkippedJobs   int               `json:"skipped_jobs"`
	NodeResults   map[string]string `json:"node_results"` // nodeID -> status
	ErrorMessage  string            `json:"error_message,omitempty"`
	DurationMS    int64             `json:"duration_ms"`
}

// DAGExecutionWorkflow executes a DAG of jobs
func DAGExecutionWorkflow(ctx workflow.Context, input DAGExecutionWorkflowInput) (*DAGExecutionWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting DAG execution workflow",
		"dag_id", input.DAGID,
		"dag_run_id", input.DAGRunID,
		"nodes", len(input.Nodes),
		"mode", input.ExecutionMode,
	)

	if input.ExecutionMode == "DRY_RUN" {
		logger.Info("DRY_RUN: Skipping DAG execution", "dag_id", input.DAGID)
		nodeResults := make(map[string]string)
		for _, n := range input.Nodes {
			nodeResults[n.ID] = "dry_run"
		}
		return &DAGExecutionWorkflowResult{
			Success:       true,
			CompletedJobs: 0,
			NodeResults:   nodeResults,
			ErrorMessage:  "DAG execution skipped in dry-run mode",
			DurationMS:    0,
		}, nil
	}

	startTime := workflow.Now(ctx)

	// Build dependency graph
	inDegree := make(map[string]int)
	dependents := make(map[string][]string)
	nodeMap := make(map[string]DAGNode)

	for _, node := range input.Nodes {
		inDegree[node.ID] = 0
		nodeMap[node.ID] = node
	}

	for _, edge := range input.Edges {
		inDegree[edge.ToNodeID]++
		dependents[edge.FromNodeID] = append(dependents[edge.FromNodeID], edge.ToNodeID)
	}

	// Track node states
	nodeResults := make(map[string]string)
	completedJobs := 0
	failedJobs := 0
	skippedJobs := 0

	// Activity options
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: input.Timeout,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute * 5,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	// Find ready nodes (in-degree = 0)
	var readyNodes []string
	for nodeID, degree := range inDegree {
		if degree == 0 {
			readyNodes = append(readyNodes, nodeID)
		}
	}

	// Execute nodes in topological order
	for len(readyNodes) > 0 {
		// Execute ready nodes (up to max parallel)
		batchSize := input.MaxParallelJobs
		if batchSize > len(readyNodes) {
			batchSize = len(readyNodes)
		}

		batch := readyNodes[:batchSize]
		readyNodes = readyNodes[batchSize:]

		// Start activities for batch
		futures := make(map[string]workflow.Future)
		for _, nodeID := range batch {
			node := nodeMap[nodeID]
			futures[nodeID] = workflow.ExecuteActivity(ctx, "ExecuteDAGNodeActivity", node)
		}

		// Wait for batch to complete
		for nodeID, future := range futures {
			var result map[string]interface{}
			err := future.Get(ctx, &result)

			if err != nil {
				failedJobs++
				nodeResults[nodeID] = "failed"
				logger.Error("Node failed", "node_id", nodeID, "error", err)

				if input.FailFast {
					// Skip remaining nodes
					for id := range inDegree {
						if nodeResults[id] == "" {
							nodeResults[id] = "skipped"
							skippedJobs++
						}
					}

					return &DAGExecutionWorkflowResult{
						Success:       false,
						CompletedJobs: completedJobs,
						FailedJobs:    failedJobs,
						SkippedJobs:   skippedJobs,
						NodeResults:   nodeResults,
						ErrorMessage:  fmt.Sprintf("DAG failed at node %s: %v", nodeID, err),
						DurationMS:    workflow.Now(ctx).Sub(startTime).Milliseconds(),
					}, nil
				}
			} else {
				completedJobs++
				nodeResults[nodeID] = "completed"
				logger.Info("Node completed", "node_id", nodeID)
			}

			// Update dependents
			for _, dependentID := range dependents[nodeID] {
				inDegree[dependentID]--
				if inDegree[dependentID] == 0 {
					// Check if all dependencies succeeded (for success-type edges)
					canRun := true
					for _, edge := range input.Edges {
						if edge.ToNodeID == dependentID && edge.Type == "success" {
							if nodeResults[edge.FromNodeID] != "completed" {
								canRun = false
								break
							}
						}
					}

					if canRun {
						readyNodes = append(readyNodes, dependentID)
					} else {
						nodeResults[dependentID] = "skipped"
						skippedJobs++
					}
				}
			}
		}
	}

	endTime := workflow.Now(ctx)
	durationMS := endTime.Sub(startTime).Milliseconds()

	success := failedJobs == 0

	logger.Info("DAG execution completed",
		"dag_id", input.DAGID,
		"success", success,
		"completed", completedJobs,
		"failed", failedJobs,
		"skipped", skippedJobs,
		"duration_ms", durationMS,
	)

	return &DAGExecutionWorkflowResult{
		Success:       success,
		CompletedJobs: completedJobs,
		FailedJobs:    failedJobs,
		SkippedJobs:   skippedJobs,
		NodeResults:   nodeResults,
		DurationMS:    durationMS,
	}, nil
}

// ============================================================================
// Scheduler Trigger Workflow
// ============================================================================

// SchedulerTriggerInput defines input for the scheduler trigger workflow
type SchedulerTriggerInput struct {
	TenantID string `json:"tenant_id,omitempty"`
}

// SchedulerTriggerWorkflow runs on a cron schedule to trigger due jobs
func SchedulerTriggerWorkflow(ctx workflow.Context, input SchedulerTriggerInput) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Running scheduler trigger check", "tenant_id", input.TenantID)

	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 5,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	// Get jobs due for execution
	var dueJobs []string
	err := workflow.ExecuteActivity(ctx, "GetDueJobsActivity", input.TenantID).Get(ctx, &dueJobs)
	if err != nil {
		logger.Error("Failed to get due jobs", "error", err)
		return err
	}

	logger.Info("Found due jobs", "count", len(dueJobs))

	// Trigger each due job
	for _, jobID := range dueJobs {
		err := workflow.ExecuteActivity(ctx, "TriggerJobActivity", jobID).Get(ctx, nil)
		if err != nil {
			logger.Error("Failed to trigger job", "job_id", jobID, "error", err)
			// Continue with other jobs
		}
	}

	return nil
}

// ============================================================================
// Activities
// ============================================================================

// SchedulerActivities contains scheduler-related Temporal activities
type SchedulerActivities struct {
	// Dependencies would be injected here (db, services, etc.)
}

// NewSchedulerActivities creates new scheduler activities
func NewSchedulerActivities() *SchedulerActivities {
	return &SchedulerActivities{}
}

// ExecuteScheduledJobActivity executes a scheduled job
func (a *SchedulerActivities) ExecuteScheduledJobActivity(ctx context.Context, input ScheduledJobWorkflowInput) (map[string]interface{}, error) {
	// This is a dispatcher that routes to specific job type handlers
	switch input.JobType {
	case "report":
		return a.executeReportJob(ctx, input)
	case "preagg":
		return a.executePreAggJob(ctx, input)
	case "data_quality":
		return a.executeDataQualityJob(ctx, input)
	case "integration":
		return a.executeIntegrationJob(ctx, input)
	case "compliance":
		return a.executeComplianceJob(ctx, input)
	case "ai":
		return a.executeAIJob(ctx, input)
	default:
		return a.executeGenericJob(ctx, input)
	}
}

// Job type handlers (stubs for now, would be implemented based on job type)
func (a *SchedulerActivities) executeReportJob(ctx context.Context, input ScheduledJobWorkflowInput) (map[string]interface{}, error) {
	// TODO: Implement report generation
	return map[string]interface{}{"type": "report", "status": "completed"}, nil
}

func (a *SchedulerActivities) executePreAggJob(ctx context.Context, input ScheduledJobWorkflowInput) (map[string]interface{}, error) {
	// TODO: Call existing PreAggBuildWorkflow or activity
	return map[string]interface{}{"type": "preagg", "status": "completed"}, nil
}

func (a *SchedulerActivities) executeDataQualityJob(ctx context.Context, input ScheduledJobWorkflowInput) (map[string]interface{}, error) {
	// TODO: Run data quality checks
	return map[string]interface{}{"type": "data_quality", "status": "completed"}, nil
}

func (a *SchedulerActivities) executeIntegrationJob(ctx context.Context, input ScheduledJobWorkflowInput) (map[string]interface{}, error) {
	// TODO: Run integration sync
	return map[string]interface{}{"type": "integration", "status": "completed"}, nil
}

func (a *SchedulerActivities) executeComplianceJob(ctx context.Context, input ScheduledJobWorkflowInput) (map[string]interface{}, error) {
	// TODO: Run compliance checks
	return map[string]interface{}{"type": "compliance", "status": "completed"}, nil
}

func (a *SchedulerActivities) executeAIJob(ctx context.Context, input ScheduledJobWorkflowInput) (map[string]interface{}, error) {
	// TODO: Run AI jobs (model training, inference, etc.)
	return map[string]interface{}{"type": "ai", "status": "completed"}, nil
}

func (a *SchedulerActivities) executeGenericJob(ctx context.Context, input ScheduledJobWorkflowInput) (map[string]interface{}, error) {
	// Generic job execution
	paramsJSON, _ := json.Marshal(input.Parameters)
	return map[string]interface{}{
		"type":       input.JobType,
		"status":     "completed",
		"parameters": string(paramsJSON),
	}, nil
}

// ExecuteDAGNodeActivity executes a single node in a DAG
func (a *SchedulerActivities) ExecuteDAGNodeActivity(ctx context.Context, node DAGNode) (map[string]interface{}, error) {
	// Would fetch the job by ID and execute it
	// For now, return success
	return map[string]interface{}{
		"node_id": node.ID,
		"job_id":  node.JobID,
		"status":  "completed",
	}, nil
}

// GetDueJobsActivity retrieves jobs that are due for execution
func (a *SchedulerActivities) GetDueJobsActivity(ctx context.Context, tenantID string) ([]string, error) {
	// Would query the database for jobs where next_run_at <= now
	// For now, return empty slice
	return []string{}, nil
}

// TriggerJobActivity triggers a job run
func (a *SchedulerActivities) TriggerJobActivity(ctx context.Context, jobID string) error {
	// Would call the scheduler service to trigger the job
	return nil
}
