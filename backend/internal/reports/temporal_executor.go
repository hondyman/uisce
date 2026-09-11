package reports

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/client"

	"github.com/hondyman/uisce/backend/internal/db"
	"github.com/hondyman/uisce/backend/internal/temporal/activities"
	"github.com/hondyman/uisce/backend/internal/temporal/workflows"
)

// DefaultTaskQueue is the analytics worker queue where ReportGenerationWorkflow is registered.
const DefaultTaskQueue = "analytics-worker"

// DispatchError represents a workflow dispatch failure and preserves the execution row ID.
type DispatchError struct {
	ExecutionID uuid.UUID
	Err         error
}

func (e *DispatchError) Error() string {
	return fmt.Sprintf("temporal dispatch error for execution %s: %v", e.ExecutionID, e.Err)
}

func (e *DispatchError) Unwrap() error {
	return e.Err
}

// TemporalReportExecutor implements ReportExecutor by dispatching ReportGenerationWorkflow
// on a live Temporal cluster.
//
// LIFECYCLE:
//   1. Inserts execution row with status='pending' (API-side) within WithTenantTransaction.
//   2. Dispatches workflows.ReportGenerationWorkflow via client.Client.
//   3. Updates execution row to status='running', populating workflow_id and run_id.
//   4. If dispatch fails, updates execution row to status='failed' with error_message and
//      returns a fail-loud error (causing HTTP 503).
//
// IDENTITY INVARIANT:
//   - The execution row is inserted under the template owner's tenant_id and requested_by.
//   - The triggering caller is tracked via triggered_by (enabling cross-tenant visibility
//     for runs of shared gold-copy templates without violating execution isolation).
type TemporalReportExecutor struct {
	db             *sql.DB
	temporalClient client.Client
	taskQueue      string
}

// NewTemporalReportExecutor creates a production-grade TemporalReportExecutor.
func NewTemporalReportExecutor(dbConn *sql.DB, temporalClient client.Client) *TemporalReportExecutor {
	return &TemporalReportExecutor{
		db:             dbConn,
		temporalClient: temporalClient,
		taskQueue:      DefaultTaskQueue,
	}
}

// NewTestTemporalReportExecutor creates an executor with a custom task queue for testing.
func NewTestTemporalReportExecutor(dbConn *sql.DB, temporalClient client.Client, taskQueue string) *TemporalReportExecutor {
	return &TemporalReportExecutor{
		db:             dbConn,
		temporalClient: temporalClient,
		taskQueue:      taskQueue,
	}
}

// ExecuteReport implements the ReportExecutor interface.
func (e *TemporalReportExecutor) ExecuteReport(ctx context.Context, tmpl *ReportTemplate, params map[string]interface{}) (*ScheduleExecutionResult, error) {
	if tmpl == nil {
		return nil, errors.New("template cannot be nil")
	}

	execID := uuid.New()
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		paramsJSON = []byte("{}")
	}

	// 1. Resolve execution identity (template owner identity)
	var reqBy *string
	if tmpl.CreatedByID != nil && *tmpl.CreatedByID != "" {
		reqBy = tmpl.CreatedByID
	} else if tmpl.CreatedBy != "" {
		reqBy = &tmpl.CreatedBy
	}

	// 2. Resolve caller identity (triggering user)
	var triggeredBy *string
	if val, ok := params["triggered_by"].(string); ok && val != "" {
		triggeredBy = &val
	}

	metaJSON, _ := json.Marshal(map[string]interface{}{
		"engine": "temporal_workflow",
	})

	// 3. Insert 'pending' row via WithTenantTransaction (enforces FORCE RLS)
	// Execution row belongs to tmpl.TenantID (template owner tenant).
	// Event insert is in the SAME transaction — audit trail completeness invariant:
	// no execution row without its CREATED event, no CREATED event without execution row.
	actorID := "system:scheduler" // default; overridden if triggeredBy is present
	if triggeredBy != nil && *triggeredBy != "" {
		actorID = *triggeredBy
	}
	// Extract schedule_id before the closure so the closure can capture it
	scheduleIDStr := ""
	if sidVal, ok := params["schedule_id"].(string); ok && sidVal != "" {
		scheduleIDStr = sidVal
	}
	err = db.WithTenantTransaction(ctx, e.db, tmpl.TenantID.String(), func(tx *sql.Tx) error {
		insertQuery := `
			INSERT INTO public.report_executions (
				id, tenant_id, template_id, schedule_id, report_key, status, parameters,
				output_format, requested_by, triggered_by, metadata, created_at
			) VALUES (
				$1, $2, $3, NULLIF($4, '')::uuid, $5, 'pending', $6,
				'pdf', $7, $8, $9, NOW()
			)
		`
		_, txErr := tx.ExecContext(ctx, insertQuery,
			execID, tmpl.TenantID, tmpl.ID, scheduleIDStr, tmpl.TemplateName, paramsJSON,
			reqBy, triggeredBy, metaJSON,
		)
		if txErr != nil {
			return txErr
		}
		// detail: include schedule_id when present (self-contained audit record)
		var detailJSON []byte
		if scheduleIDStr != "" {
			detailJSON, _ = json.Marshal(map[string]interface{}{"schedule_id": scheduleIDStr})
		} else {
			detailJSON = []byte(`{}`)
		}
		_, txErr = tx.ExecContext(ctx, `
			INSERT INTO public.report_execution_events (
				id, execution_id, tenant_id, event, from_status, to_status, actor_id, detail
			) VALUES (
				gen_random_uuid(), $1, $2, 'CREATED', NULL, 'pending', $3, $4
			)
		`, execID, tmpl.TenantID, actorID, detailJSON)
		return txErr
	})
	if err != nil {
		return nil, fmt.Errorf("failed to insert pending execution record: %w", err)
	}

	// Convert SemanticViewIDs from []uuid.UUID to []string
	viewIDs := make([]string, len(tmpl.SemanticViewIDs))
	for i, vid := range tmpl.SemanticViewIDs {
		viewIDs[i] = vid.String()
	}

	// 4. Hydrate template snapshot for TOCTOU safety
	hydratedTmpl := activities.HydratedReportTemplate{
		ID:              tmpl.ID.String(),
		TenantID:        tmpl.TenantID.String(),
		TemplateName:    tmpl.TemplateName,
		Description:     tmpl.Description,
		Category:        tmpl.Category,
		SemanticViewIDs: viewIDs,
		LayoutConfig:    tmpl.LayoutConfig,
		ParameterSchema: tmpl.ParameterSchema,
		IsPublic:        tmpl.IsPublic,
		IsPersonal:      tmpl.IsPersonal,
	}
	if reqBy != nil {
		hydratedTmpl.CreatedByID = *reqBy
	}
	if tmpl.CreatedBy != "" {
		hydratedTmpl.CreatedBy = tmpl.CreatedBy
	}

	scheduleID := ""
	if sidVal, ok := params["schedule_id"].(string); ok {
		scheduleID = sidVal
	}

	workflowParams := workflows.ReportGenerationWorkflowParams{
		ExecutionID:   execID.String(),
		Template:      hydratedTmpl,
		ScheduleID:    scheduleID,
		TriggerParams: params,
	}

	workflowID := fmt.Sprintf("report-exec-%s", execID)
	workflowOptions := client.StartWorkflowOptions{
		ID:                       workflowID,
		TaskQueue:                e.taskQueue,
		WorkflowExecutionTimeout: 10 * time.Minute,
	}

	// 5. Dispatch workflow to Temporal
	if e.temporalClient == nil {
		// Fail loud when Temporal is unconfigured or unavailable
		e.markExecutionFailed(ctx, tmpl.TenantID.String(), execID, "Temporal client is nil / unavailable")
		return nil, &DispatchError{ExecutionID: execID, Err: errors.New("temporal service unavailable")}
	}

	run, err := e.temporalClient.ExecuteWorkflow(ctx, workflowOptions, workflows.ReportGenerationWorkflow, workflowParams)
	if err != nil {
		// Temporal dispatch failed: mark row as failed loud and return error
		errMsg := fmt.Sprintf("temporal workflow dispatch failed: %v", err)
		log.Printf("[ERROR] %s (execution_id=%s)", errMsg, execID)
		e.markExecutionFailed(ctx, tmpl.TenantID.String(), execID, errMsg)
		return nil, &DispatchError{ExecutionID: execID, Err: fmt.Errorf("workflow dispatch error: %w", err)}
	}

	runID := run.GetRunID()

	// 6. Transition execution row to 'running'
	err = db.WithTenantTransaction(ctx, e.db, tmpl.TenantID.String(), func(tx *sql.Tx) error {
		updateQuery := `
			UPDATE public.report_executions
			SET status = 'running',
			    workflow_id = $1,
			    run_id = $2
			WHERE id = $3
		`
		_, txErr := tx.ExecContext(ctx, updateQuery, workflowID, runID, execID)
		return txErr
	})
	if err != nil {
		log.Printf("[WARN] failed to update execution %s to running: %v", execID, err)
	}

	res := &ScheduleExecutionResult{
		ExecutionID: execID,
		TenantID:    tmpl.TenantID,
		TemplateID:  tmpl.ID,
		Status:      "pending", // Truthful response — worker hasn't finished yet
		OutputURL:   "",
		RequestedBy: "",
	}
	if reqBy != nil {
		res.RequestedBy = *reqBy
	}

	return res, nil
}

func (e *TemporalReportExecutor) markExecutionFailed(ctx context.Context, tenantID string, execID uuid.UUID, errMsg string) {
	_ = db.WithTenantTransaction(ctx, e.db, tenantID, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
			UPDATE public.report_executions
			SET status = 'failed',
			    error_message = $1,
			    completed_at = NOW()
			WHERE id = $2
		`, errMsg, execID)
		return err
	})
}
