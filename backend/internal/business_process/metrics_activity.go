package business_process

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jmoiron/sqlx"
)

// RecordStepMetricsActivity records workflow step execution metrics to the database
type RecordStepMetricsActivity struct {
	db *sqlx.DB
}

// NewRecordStepMetricsActivity creates a new metrics recording activity
func NewRecordStepMetricsActivity(db *sqlx.DB) *RecordStepMetricsActivity {
	return &RecordStepMetricsActivity{db: db}
}

// Execute records step execution metrics
func (a *RecordStepMetricsActivity) Execute(ctx context.Context, params map[string]interface{}) error {
	// Extract parameters
	workflowID, _ := params["workflow_id"].(string)
	workflowType, _ := params["workflow_type"].(string)
	tenantID, _ := params["tenant_id"].(string)
	stepName, _ := params["step_name"].(string)
	stepType, _ := params["step_type"].(string)
	status, _ := params["status"].(string)

	// Parse times
	startTime, _ := params["start_time"].(time.Time)
	endTime, _ := params["end_time"].(time.Time)

	// Error message (optional)
	var errorMsg *string
	if errMsgVal, ok := params["error_message"]; ok && errMsgVal != nil {
		if errStr, ok := errMsgVal.(*string); ok {
			errorMsg = errStr
		}
	}

	// Metadata
	metadata := params["metadata"]
	if metadata == nil {
		metadata = map[string]interface{}{}
	}
	metadataJSON, _ := json.Marshal(metadata)

	// Resource usage (empty for now, can be populated with actual metrics)
	resourceUsage := map[string]interface{}{}
	resourceUsageJSON, _ := json.Marshal(resourceUsage)

	// Calculate duration
	var durationInterval string
	if !endTime.IsZero() {
		duration := endTime.Sub(startTime)
		// Convert to PostgreSQL interval format
		durationInterval = duration.String()
	}

	// Insert metrics record
	query := `
		INSERT INTO process_execution_metrics 
			(workflow_id, workflow_type, tenant_id, step_name, step_type, 
			 start_time, end_time, duration, status, error_message, 
			 resource_usage, metadata, created_at, updated_at)
		VALUES 
			($1, $2, $3, $4, $5, $6, $7, $8::interval, $9, $10, $11, $12, NOW(), NOW())
		ON CONFLICT (workflow_id, step_name) 
		DO UPDATE SET
			end_time = EXCLUDED.end_time,
			duration = EXCLUDED.duration,
			status = EXCLUDED.status,
			error_message = EXCLUDED.error_message,
			updated_at = NOW()
	`

	_, err := a.db.ExecContext(ctx, query,
		workflowID,
		workflowType,
		tenantID,
		stepName,
		stepType,
		startTime,
		endTime,
		durationInterval,
		status,
		errorMsg,
		resourceUsageJSON,
		metadataJSON,
	)

	if err != nil {
		// Log error but don't fail the workflow
		// In production, you might want to use a proper logging framework
		// log.Printf("Failed to record step metrics: %v", err)
		return nil // Don't propagate error to avoid failing the workflow
	}

	return nil
}

// RecordStepMetrics is the activity function that Temporal will call
func RecordStepMetrics(ctx context.Context, params map[string]interface{}) error {
	// This should be called through an activity registered with Temporal
	// The actual implementation will be provided by the activity instance
	return nil
}
