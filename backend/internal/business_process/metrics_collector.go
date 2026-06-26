package business_process

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"go.temporal.io/api/history/v1"
	"go.temporal.io/sdk/client"
)

// ProcessMetricsCollector collects workflow execution metrics for analytics
type ProcessMetricsCollector struct {
	db             *sqlx.DB
	temporalClient client.Client
}

// NewProcessMetricsCollector creates a new metrics collector
func NewProcessMetricsCollector(db *sqlx.DB, temporalClient client.Client) *ProcessMetricsCollector {
	return &ProcessMetricsCollector{
		db:             db,
		temporalClient: temporalClient,
	}
}

// StepMetric represents a single step execution metric
type StepMetric struct {
	WorkflowID    string
	WorkflowType  string
	TenantID      string
	StepName      string
	StepType      string
	StartTime     time.Time
	EndTime       *time.Time
	Status        string
	ErrorMessage  *string
	ResourceUsage map[string]interface{}
	Metadata      map[string]interface{}
}

// RecordStepStart records when a workflow step begins execution
func (c *ProcessMetricsCollector) RecordStepStart(ctx context.Context, metric StepMetric) error {
	resourceUsageJSON, _ := json.Marshal(metric.ResourceUsage)
	metadataJSON, _ := json.Marshal(metric.Metadata)

	query := `
		INSERT INTO process_execution_metrics 
			(workflow_id, workflow_type, tenant_id, step_name, step_type, start_time, status, resource_usage, metadata, created_at, updated_at)
		VALUES 
			($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		RETURNING id
	`

	var id string
	err := c.db.QueryRowContext(ctx, query,
		metric.WorkflowID,
		metric.WorkflowType,
		metric.TenantID,
		metric.StepName,
		metric.StepType,
		metric.StartTime,
		"running",
		resourceUsageJSON,
		metadataJSON,
	).Scan(&id)

	return err
}

// RecordStepCompletion records when a workflow step completes (success or failure)
func (c *ProcessMetricsCollector) RecordStepCompletion(ctx context.Context, workflowID, stepName string, endTime time.Time, status string, errorMessage *string) error {
	query := `
		UPDATE process_execution_metrics 
		SET 
			end_time = $1,
			duration = $1 - start_time,
			status = $2,
			error_message = $3,
			updated_at = NOW()
		WHERE workflow_id = $4 AND step_name = $5 AND end_time IS NULL
	`

	_, err := c.db.ExecContext(ctx, query, endTime, status, errorMessage, workflowID, stepName)
	return err
}

// UpdateResourceUsage updates resource usage metrics during step execution
func (c *ProcessMetricsCollector) UpdateResourceUsage(ctx context.Context, workflowID, stepName string, resourceUsage map[string]interface{}) error {
	resourceUsageJSON, _ := json.Marshal(resourceUsage)

	query := `
		UPDATE process_execution_metrics 
		SET 
			resource_usage = $1,
			updated_at = NOW()
		WHERE workflow_id = $2 AND step_name = $3 AND end_time IS NULL
	`

	_, err := c.db.ExecContext(ctx, query, resourceUsageJSON, workflowID, stepName)
	return err
}

// CollectTemporalMetrics queries Temporal for workflow execution data and stores metrics
func (c *ProcessMetricsCollector) CollectTemporalMetrics(ctx context.Context, workflowID string, tenantID string) error {
	if c.temporalClient == nil {
		return fmt.Errorf("temporal client not configured")
	}

	// Get workflow description from Temporal
	describe, err := c.temporalClient.DescribeWorkflowExecution(ctx, workflowID, "")
	if err != nil {
		return fmt.Errorf("failed to describe workflow: %w", err)
	}

	workflowType := describe.WorkflowExecutionInfo.Type.Name
	// startTime := describe.WorkflowExecutionInfo.StartTime.AsTime() // unused

	// Get workflow history to extract step-level metrics
	iter := c.temporalClient.GetWorkflowHistory(ctx, workflowID, "", false, 0)

	stepMetrics := make(map[string]*StepMetric)

	for iter.HasNext() {
		event, err := iter.Next()
		if err != nil {
			return fmt.Errorf("failed to iterate history: %w", err)
		}

		eventType := event.GetEventType().String()

		switch eventType {
		case "ActivityTaskScheduled":
			attrs := event.GetActivityTaskScheduledEventAttributes()
			if attrs != nil {
				activityName := attrs.ActivityType.GetName()
				stepMetrics[activityName] = &StepMetric{
					WorkflowID:    workflowID,
					WorkflowType:  workflowType,
					TenantID:      tenantID,
					StepName:      activityName,
					StepType:      "activity",
					StartTime:     event.GetEventTime().AsTime(),
					Status:        "running",
					ResourceUsage: make(map[string]interface{}),
					Metadata: map[string]interface{}{
						"event_id": event.GetEventId(),
					},
				}
			}

		case "ActivityTaskCompleted":
			attrs := event.GetActivityTaskCompletedEventAttributes()
			if attrs != nil {
				// Find the corresponding scheduled event
				scheduledEventID := attrs.GetScheduledEventId()
				scheduledEvent, _ := findEventByID(ctx, c.temporalClient, workflowID, scheduledEventID)
				if scheduledEvent != nil {
					scheduledAttrs := scheduledEvent.GetActivityTaskScheduledEventAttributes()
					if scheduledAttrs != nil {
						activityName := scheduledAttrs.ActivityType.GetName()
						if metric, exists := stepMetrics[activityName]; exists {
							endTime := event.GetEventTime().AsTime()
							metric.EndTime = &endTime
							metric.Status = "completed"
						}
					}
				}
			}

		case "ActivityTaskFailed":
			attrs := event.GetActivityTaskFailedEventAttributes()
			if attrs != nil {
				scheduledEventID := attrs.GetScheduledEventId()
				scheduledEvent, _ := findEventByID(ctx, c.temporalClient, workflowID, scheduledEventID)
				if scheduledEvent != nil {
					scheduledAttrs := scheduledEvent.GetActivityTaskScheduledEventAttributes()
					if scheduledAttrs != nil {
						activityName := scheduledAttrs.ActivityType.GetName()
						if metric, exists := stepMetrics[activityName]; exists {
							endTime := event.GetEventTime().AsTime()
							metric.EndTime = &endTime
							metric.Status = "failed"
							errorMsg := attrs.GetFailure().GetMessage()
							metric.ErrorMessage = &errorMsg
						}
					}
				}
			}

		case "ActivityTaskTimedOut":
			attrs := event.GetActivityTaskTimedOutEventAttributes()
			if attrs != nil {
				scheduledEventID := attrs.GetScheduledEventId()
				scheduledEvent, _ := findEventByID(ctx, c.temporalClient, workflowID, scheduledEventID)
				if scheduledEvent != nil {
					scheduledAttrs := scheduledEvent.GetActivityTaskScheduledEventAttributes()
					if scheduledAttrs != nil {
						activityName := scheduledAttrs.ActivityType.GetName()
						if metric, exists := stepMetrics[activityName]; exists {
							endTime := event.GetEventTime().AsTime()
							metric.EndTime = &endTime
							metric.Status = "timeout"
							errorMsg := "Activity timed out"
							metric.ErrorMessage = &errorMsg
						}
					}
				}
			}
		}
	}

	// Store all collected metrics
	for _, metric := range stepMetrics {
		err := c.RecordStepStart(ctx, *metric)
		if err != nil {
			// Log error but continue with other metrics
			fmt.Printf("Failed to record step start for %s: %v\n", metric.StepName, err)
			continue
		}

		if metric.EndTime != nil {
			err = c.RecordStepCompletion(ctx, metric.WorkflowID, metric.StepName, *metric.EndTime, metric.Status, metric.ErrorMessage)
			if err != nil {
				fmt.Printf("Failed to record step completion for %s: %v\n", metric.StepName, err)
			}
		}
	}

	return nil
}

// findEventByID finds a specific event in workflow history by event ID
func findEventByID(ctx context.Context, temporalClient client.Client, workflowID string, eventID int64) (*history.HistoryEvent, error) {
	iter := temporalClient.GetWorkflowHistory(ctx, workflowID, "", false, 0)

	for iter.HasNext() {
		event, err := iter.Next()
		if err != nil {
			return nil, err
		}

		if event.GetEventId() == eventID {
			return event, nil
		}
	}

	return nil, fmt.Errorf("event %d not found", eventID)
}

// StartMetricsCollectionWorker starts a background worker that periodically collects metrics from Temporal
func (c *ProcessMetricsCollector) StartMetricsCollectionWorker(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.collectActiveWorkflowMetrics(ctx)
		}
	}
}

// collectActiveWorkflowMetrics collects metrics for all active workflows
func (c *ProcessMetricsCollector) collectActiveWorkflowMetrics(ctx context.Context) {
	// Query database for active workflows (status = 'running')
	var activeWorkflows []struct {
		WorkflowID string `db:"workflow_id"`
		TenantID   string `db:"tenant_id"`
	}

	query := `
		SELECT DISTINCT workflow_id, tenant_id 
		FROM process_execution_metrics 
		WHERE status = 'running' AND created_at > NOW() - INTERVAL '24 hours'
	`

	err := c.db.SelectContext(ctx, &activeWorkflows, query)
	if err != nil {
		fmt.Printf("Failed to query active workflows: %v\n", err)
		return
	}

	// Collect metrics for each active workflow
	for _, wf := range activeWorkflows {
		err := c.CollectTemporalMetrics(ctx, wf.WorkflowID, wf.TenantID)
		if err != nil {
			fmt.Printf("Failed to collect metrics for workflow %s: %v\n", wf.WorkflowID, err)
		}
	}
}

// GetWorkflowMetricsSummary retrieves aggregated metrics for a specific workflow
func (c *ProcessMetricsCollector) GetWorkflowMetricsSummary(ctx context.Context, workflowID string) (map[string]interface{}, error) {
	var totalSteps int
	var completedSteps int
	var failedSteps int
	var avgDuration sql.NullFloat64

	err := c.db.QueryRowContext(ctx, `
		SELECT 
			COUNT(*) as total_steps,
			COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed_steps,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed_steps,
			AVG(EXTRACT(EPOCH FROM duration) / 60) as avg_duration_minutes
		FROM process_execution_metrics
		WHERE workflow_id = $1
	`, workflowID).Scan(&totalSteps, &completedSteps, &failedSteps, &avgDuration)

	if err != nil {
		return nil, err
	}

	summary := map[string]interface{}{
		"total_steps":      totalSteps,
		"completed_steps":  completedSteps,
		"failed_steps":     failedSteps,
		"avg_duration_min": 0.0,
		"status":           "unknown",
	}

	if avgDuration.Valid {
		summary["avg_duration_min"] = avgDuration.Float64
	}

	if failedSteps > 0 {
		summary["status"] = "failed"
	} else if completedSteps == totalSteps && totalSteps > 0 {
		summary["status"] = "completed"
	} else {
		summary["status"] = "in_progress"
	}

	return summary, nil
}

// CalculateWorkflowHealth calculates a health score (0-100) for a workflow type
func (c *ProcessMetricsCollector) CalculateWorkflowHealth(ctx context.Context, tenantID, workflowType string) (float64, error) {
	var successRate sql.NullFloat64
	var avgBottleneckSeverity sql.NullFloat64

	// Get success rate
	err := c.db.QueryRowContext(ctx, `
		SELECT AVG(CASE WHEN status = 'completed' THEN 1.0 ELSE 0.0 END) as success_rate
		FROM process_execution_metrics
		WHERE tenant_id = $1 AND workflow_type = $2 AND created_at > NOW() - INTERVAL '7 days'
	`, tenantID, workflowType).Scan(&successRate)

	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}

	// Get average bottleneck severity
	err = c.db.QueryRowContext(ctx, `
		SELECT AVG(severity) as avg_severity
		FROM process_bottleneck_analysis
		WHERE tenant_id = $1 AND workflow_type = $2
	`, tenantID, workflowType).Scan(&avgBottleneckSeverity)

	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}

	// Calculate health score (0-100)
	// Formula: (success_rate * 70) + ((1 - bottleneck_severity) * 30)
	health := 0.0

	if successRate.Valid {
		health += successRate.Float64 * 70
	}

	if avgBottleneckSeverity.Valid {
		health += (1.0 - avgBottleneckSeverity.Float64) * 30
	} else {
		health += 30 // No bottlenecks = full 30 points
	}

	return health, nil
}
