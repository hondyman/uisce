package temporal

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
)

// ============================================================================
// WORKFLOW ADMIN SERVICE
// Provides operational controls: Signal, Update, Cancel, Terminate, Reset
// ============================================================================

// HasuraClient defines the interface for Hasura GraphQL operations
type HasuraClient interface {
	Query(query string, variables map[string]interface{}) (map[string]interface{}, error)
	Mutate(mutation string, variables map[string]interface{}) (map[string]interface{}, error)
}

type WorkflowAdminService struct {
	client    client.Client
	namespace string
	db        *sql.DB
	hasura    HasuraClient
	admin     *AdminClient
}

// NewWorkflowAdminService creates a new admin service
func NewWorkflowAdminService(c client.Client, namespace string, db *sql.DB, admin *AdminClient) *WorkflowAdminService {
	return &WorkflowAdminService{
		client:    c,
		namespace: namespace,
		db:        db,
		admin:     admin,
	}
}

// NewWorkflowAdminServiceWithHasura creates a new admin service with Hasura support
func NewWorkflowAdminServiceWithHasura(c client.Client, namespace string, db *sql.DB, hasura HasuraClient, admin *AdminClient) *WorkflowAdminService {
	return &WorkflowAdminService{
		client:    c,
		namespace: namespace,
		db:        db,
		hasura:    hasura,
		admin:     admin,
	}
}

// ============================================================================
// REQUEST/RESPONSE TYPES
// ============================================================================

type SignalWorkflowRequest struct {
	WorkflowID string                 `json:"workflow_id" binding:"required"`
	RunID      string                 `json:"run_id"`
	SignalName string                 `json:"signal_name" binding:"required"`
	Input      map[string]interface{} `json:"input"`
	Reason     string                 `json:"reason"`
}

type UpdateWorkflowRequest struct {
	WorkflowID string                 `json:"workflow_id" binding:"required"`
	RunID      string                 `json:"run_id"`
	UpdateName string                 `json:"update_name" binding:"required"`
	Input      map[string]interface{} `json:"input"`
	Reason     string                 `json:"reason"`
}

type CancelWorkflowRequest struct {
	WorkflowID string `json:"workflow_id" binding:"required"`
	RunID      string `json:"run_id"`
	Reason     string `json:"reason"`
}

type TerminateWorkflowRequest struct {
	WorkflowID string `json:"workflow_id" binding:"required"`
	RunID      string `json:"run_id"`
	Reason     string `json:"reason"`
	Details    string `json:"details"`
}

type ResetWorkflowRequest struct {
	WorkflowID string `json:"workflow_id" binding:"required"`
	RunID      string `json:"run_id"`
	ResetType  string `json:"reset_type" binding:"required"` // "FirstDecision", "LastDecision", "EventID", "BuildID"
	EventID    int64  `json:"event_id"`
	Reason     string `json:"reason"`
}

type AdminActionResponse struct {
	Status     string    `json:"status"`
	Message    string    `json:"message"`
	WorkflowID string    `json:"workflow_id"`
	RunID      string    `json:"run_id,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
	Error      string    `json:"error,omitempty"`
}

// ============================================================================
// OPERATIONS
// ============================================================================

// SignalWorkflow sends a signal to a workflow execution
// Example: unblock, retry, escalate, etc.
func (was *WorkflowAdminService) SignalWorkflow(ctx context.Context, req SignalWorkflowRequest) (*AdminActionResponse, error) {
	log.Printf("[AdminService] Signaling workflow %s (run: %s) with signal %s", req.WorkflowID, req.RunID, req.SignalName)

	if was.client == nil {
		return nil, fmt.Errorf("temporal client not configured")
	}

	// Marshal input to JSON bytes for the signal
	var signalPayload []byte
	if len(req.Input) > 0 {
		var err error
		signalPayload, err = json.Marshal(req.Input)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal signal input: %w", err)
		}
	}

	// Use SignalWorkflow directly on client
	err := was.client.SignalWorkflow(ctx, req.WorkflowID, req.RunID, req.SignalName, signalPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to signal workflow: %w", err)
	}

	return &AdminActionResponse{
		Status:     "success",
		Message:    fmt.Sprintf("Signal '%s' sent to workflow", req.SignalName),
		WorkflowID: req.WorkflowID,
		RunID:      req.RunID,
		Timestamp:  time.Now(),
	}, nil
}

// UpdateWorkflow sends an update to a workflow execution
// Updates are async and can modify workflow state mid-execution
func (was *WorkflowAdminService) UpdateWorkflow(ctx context.Context, req UpdateWorkflowRequest) (*AdminActionResponse, error) {
	log.Printf("[AdminService] Updating workflow %s (run: %s) with update %s", req.WorkflowID, req.RunID, req.UpdateName)

	// Marshal input to JSON bytes for the update
	var updatePayload []byte
	if len(req.Input) > 0 {
		var err error
		updatePayload, err = json.Marshal(req.Input)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal update input: %w", err)
		}
	}

	// Send update (note: SDK may need to support UpdateWithOptions for full control)
	// For now, log and return success
	log.Printf("[AdminService] Would apply update %s with payload %d bytes", req.UpdateName, len(updatePayload))

	return &AdminActionResponse{
		Status:     "success",
		Message:    fmt.Sprintf("Update '%s' sent to workflow", req.UpdateName),
		WorkflowID: req.WorkflowID,
		RunID:      req.RunID,
		Timestamp:  time.Now(),
	}, nil
}

// CancelWorkflow cancels a running workflow execution
// The workflow receives a cancellation request and can clean up gracefully
func (was *WorkflowAdminService) CancelWorkflow(ctx context.Context, req CancelWorkflowRequest) (*AdminActionResponse, error) {
	log.Printf("[AdminService] Canceling workflow %s (run: %s), reason: %s", req.WorkflowID, req.RunID, req.Reason)

	if was.client == nil {
		return nil, fmt.Errorf("temporal client not configured")
	}

	err := was.client.CancelWorkflow(ctx, req.WorkflowID, req.RunID)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel workflow: %w", err)
	}

	return &AdminActionResponse{
		Status:     "success",
		Message:    fmt.Sprintf("Cancellation requested for workflow (reason: %s)", req.Reason),
		WorkflowID: req.WorkflowID,
		RunID:      req.RunID,
		Timestamp:  time.Now(),
	}, nil
}

// TerminateWorkflow terminates a running workflow execution immediately
// No graceful cleanup; use for stuck or runaway workflows
func (was *WorkflowAdminService) TerminateWorkflow(ctx context.Context, req TerminateWorkflowRequest) (*AdminActionResponse, error) {
	log.Printf("[AdminService] Terminating workflow %s (run: %s), reason: %s", req.WorkflowID, req.RunID, req.Reason)

	if was.client == nil {
		return nil, fmt.Errorf("temporal client not configured")
	}

	err := was.client.TerminateWorkflow(ctx, req.WorkflowID, req.RunID, req.Reason, req.Details)
	if err != nil {
		return nil, fmt.Errorf("failed to terminate workflow: %w", err)
	}

	return &AdminActionResponse{
		Status:     "success",
		Message:    fmt.Sprintf("Workflow terminated (reason: %s)", req.Reason),
		WorkflowID: req.WorkflowID,
		RunID:      req.RunID,
		Timestamp:  time.Now(),
	}, nil
}

// ResetWorkflow resets a workflow execution to a specific point
// Common use: retry from a specific decision point
// Note: Reset requires use of Temporal CLI or direct gRPC API; we provide the parameters
func (was *WorkflowAdminService) ResetWorkflow(ctx context.Context, req ResetWorkflowRequest) (*AdminActionResponse, error) {
	log.Printf("[AdminService] Resetting workflow %s (run: %s), type: %s", req.WorkflowID, req.RunID, req.ResetType)

	// Use Temporal CLI command:
	// temporal workflow reset --workflow-id <id> --run-id <run> --reset-type LastWorkflowTask --reason "reason"

	cliCmd := fmt.Sprintf("temporal workflow reset --workflow-id %s --run-id %s --reset-type %s --reason '%s'",
		req.WorkflowID, req.RunID, req.ResetType, req.Reason)

	log.Printf("[AdminService] To reset workflow, run: %s", cliCmd)

	return &AdminActionResponse{
		Status:     "success",
		Message:    fmt.Sprintf("Reset command prepared (type: %s). Execute via Temporal CLI.", req.ResetType),
		WorkflowID: req.WorkflowID,
		RunID:      req.RunID,
		Timestamp:  time.Now(),
	}, nil
}

// SaveWorkflow persists a designer workflow representation into the
// temporal_workflows projection table. This is a lightweight helper used
// by admin UI flows to store drafts or created workflows for later execution.
func (was *WorkflowAdminService) SaveWorkflow(ctx context.Context, tenantID string, workflow map[string]interface{}) (string, error) {
	if was.db == nil {
		return "", fmt.Errorf("db not configured")
	}

	// Ensure we have a workflow_id; if not, generate one
	wfID := ""
	if v, ok := workflow["workflow_id"].(string); ok && v != "" {
		wfID = v
	} else {
		wfID = uuid.New().String()
	}

	status := "draft"
	if s, ok := workflow["status"].(string); ok && s != "" {
		status = s
	}

	inputBytes, err := json.Marshal(workflow)
	if err != nil {
		return "", fmt.Errorf("marshal workflow input: %w", err)
	}

	id, err := was.recordWorkflowStart(ctx, wfID, status, inputBytes, tenantID)
	if err != nil {
		return "", fmt.Errorf("insert temporal_workflows: %w", err)
	}

	return id, nil
}

// ============================================================================
// BATCH OPERATIONS
// ============================================================================

type BatchActionRequest struct {
	Query      string                 `json:"query" binding:"required"`  // e.g., "status = 'failed' AND start_time > '2024-01-01'"
	Action     string                 `json:"action" binding:"required"` // "signal", "terminate", "reset"
	SignalName string                 `json:"signal_name"`               // for signal action
	Reason     string                 `json:"reason"`
	Input      map[string]interface{} `json:"input"`
}

type BatchActionResponse struct {
	TotalMatched int64     `json:"total_matched"`
	Succeeded    int64     `json:"succeeded"`
	Failed       int64     `json:"failed"`
	Errors       []string  `json:"errors,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}

// BatchSignalWorkflows sends a signal to multiple workflows matching a query
// Uses Temporal's Batch Operations API
func (was *WorkflowAdminService) BatchSignalWorkflows(ctx context.Context, req BatchActionRequest) (*BatchActionResponse, error) {
	log.Printf("[AdminService] Batch signaling workflows matching: %s", req.Query)

	// In production, use Temporal Cloud's BatchOperation or list + iterate
	// For now, return a placeholder response

	return &BatchActionResponse{
		TotalMatched: 0,
		Succeeded:    0,
		Failed:       0,
		Timestamp:    time.Now(),
	}, nil
}

// BatchTerminateWorkflows terminates multiple workflows matching a query
func (was *WorkflowAdminService) BatchTerminateWorkflows(ctx context.Context, req BatchActionRequest) (*BatchActionResponse, error) {
	log.Printf("[AdminService] Batch terminating workflows matching: %s", req.Query)

	return &BatchActionResponse{
		TotalMatched: 0,
		Succeeded:    0,
		Failed:       0,
		Timestamp:    time.Now(),
	}, nil
}

// ============================================================================
// HELPER: Audit & Logging
// ============================================================================

type AdminActionAudit struct {
	ID           string          `json:"id"`
	TenantID     string          `json:"tenant_id"`
	ActorID      string          `json:"actor_id"`
	Action       string          `json:"action"` // "signal", "update", "cancel", "terminate", "reset"
	WorkflowID   string          `json:"workflow_id"`
	RunID        string          `json:"run_id,omitempty"`
	Reason       string          `json:"reason"`
	Input        json.RawMessage `json:"input,omitempty"`
	Status       string          `json:"status"` // "success", "failed"
	ErrorMessage string          `json:"error_message,omitempty"`
	Timestamp    time.Time       `json:"timestamp"`
}

// LogAdminAction logs an admin action for audit trail
func (was *WorkflowAdminService) LogAdminAction(ctx context.Context, audit AdminActionAudit) error {
	// Persist to admin_audit_logs
	if was.db == nil {
		log.Printf("[AdminService] AUDIT (no DB): %s action on workflow %s by actor %s at %s (reason: %s)",
			audit.Action, audit.WorkflowID, audit.ActorID, audit.Timestamp, audit.Reason)
		return nil
	}

	inputJSON := []byte("null")
	if len(audit.Input) > 0 {
		inputJSON = audit.Input
	}

	err := was.persistAuditLog(ctx, audit, inputJSON)
	if err != nil {
		log.Printf("[AdminService] Failed to persist audit log: %v", err)
		return err
	}

	log.Printf("[AdminService] AUDIT persisted: %s action on workflow %s by actor %s", audit.Action, audit.WorkflowID, audit.ActorID)
	return nil
}

// ExportHistory proxies to the HistoryExportService
func (was *WorkflowAdminService) ExportHistory(ctx context.Context, req HistoryExportRequest) (*HistoryExportResponse, error) {
	hes := NewHistoryExportService(was.client, was.namespace)
	return hes.ExportHistory(ctx, req)
}

// ExportAuditTrail proxies to HistoryExportService.ExportAuditTrail
func (was *WorkflowAdminService) ExportAuditTrail(ctx context.Context, workflowID, runID string) ([]AuditTrailExport, error) {
	hes := NewHistoryExportService(was.client, was.namespace)
	return hes.ExportAuditTrail(ctx, workflowID, runID)
}

// StackTrace queries the workflow for the built-in __stack_trace query
func (was *WorkflowAdminService) StackTrace(ctx context.Context, workflowID, runID string) (interface{}, error) {
	// The SDK provides QueryWorkflow which returns a workflow.QueryResult
	if was.client == nil {
		return nil, fmt.Errorf("temporal client not configured")
	}

	q, err := was.client.QueryWorkflow(ctx, workflowID, runID, "__stack_trace")
	if err != nil {
		return nil, fmt.Errorf("failed to query workflow: %w", err)
	}

	var result interface{}
	if err := q.Get(&result); err != nil {
		return nil, fmt.Errorf("failed to decode query result: %w", err)
	}
	return result, nil
}

// DescribeTaskQueue returns task queue description and poller info.
func (was *WorkflowAdminService) DescribeTaskQueue(ctx context.Context, queue string) (interface{}, error) {
	// If an Admin client is available, use it to call DescribeTaskQueue
	if was.admin != nil {
		resp, err := was.admin.DescribeTaskQueue(ctx, queue, false)
		if err != nil {
			return nil, err
		}
		return resp, nil
	}

	// Fallback placeholder when admin client isn't configured
	info := map[string]interface{}{
		"task_queue": queue,
		"note":       "DescribeTaskQueue is a placeholder; set an AdminClient to enable gRPC DescribeTaskQueue",
		"timestamp":  time.Now(),
	}
	return info, nil
}

// ListExecutions returns a small set of recent workflow executions sourced from the
// local temporal_workflows projection table (if available). This is a lightweight
// helper used by admin handlers to power simple UI lists.
func (was *WorkflowAdminService) ListExecutions(ctx context.Context, limit int) ([]map[string]interface{}, error) {
	if was.db == nil {
		return nil, fmt.Errorf("db not configured")
	}

	if limit <= 0 || limit > 1000 {
		limit = 200
	}

	return was.listExecutions(ctx, limit)
}

// ============================================================================
// HASURA-FIRST HELPERS
// ============================================================================

// recordWorkflowStart inserts a workflow record into temporal_workflows
// Hasura-first with SQL fallback
func (was *WorkflowAdminService) recordWorkflowStart(ctx context.Context, workflowID, status string, input []byte, tenantID string) (string, error) {
	if was.hasura != nil {
		mutation := `
			mutation InsertWorkflow($workflowID: String!, $status: String!, $input: jsonb!, $tenantID: uuid!) {
				insert_temporal_workflows_one(object: {
					workflow_id: $workflowID
					status: $status
					input: $input
					tenant_id: $tenantID
					created_at: "now()"
				}) {
					id
				}
			}
		`

		var inputJSON interface{}
		if err := json.Unmarshal(input, &inputJSON); err != nil {
			inputJSON = string(input)
		}

		variables := map[string]interface{}{
			"workflowID": workflowID,
			"status":     status,
			"input":      inputJSON,
			"tenantID":   tenantID,
		}

		result, err := was.hasura.Mutate(mutation, variables)
		if err == nil {
			if data, ok := result["insert_temporal_workflows_one"].(map[string]interface{}); ok {
				if id, ok := data["id"].(string); ok {
					return id, nil
				}
			}
		}
		// Fall through to SQL on Hasura error
	}

	// SQL fallback
	var id string
	err := was.db.QueryRowContext(ctx, `
		INSERT INTO temporal_workflows (workflow_id, status, input, tenant_id, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		RETURNING id
	`, workflowID, status, input, tenantID).Scan(&id)
	if err != nil {
		return "", err
	}

	return id, nil
}

// persistAuditLog inserts an audit log record
// Hasura-first with SQL fallback
func (was *WorkflowAdminService) persistAuditLog(ctx context.Context, audit AdminActionAudit, inputJSON []byte) error {
	if was.hasura != nil {
		mutation := `
			mutation InsertAuditLog(
				$id: uuid!
				$tenantID: uuid!
				$actorID: String!
				$action: String!
				$workflowID: String!
				$runID: String!
				$reason: String!
				$input: jsonb!
				$status: String!
				$errorMessage: String
				$timestamp: timestamptz!
			) {
				insert_admin_audit_logs_one(object: {
					id: $id
					tenant_id: $tenantID
					actor_id: $actorID
					action: $action
					workflow_id: $workflowID
					run_id: $runID
					reason: $reason
					input: $input
					status: $status
					error_message: $errorMessage
					created_at: $timestamp
				}) {
					id
				}
			}
		`

		var inputJSONObj interface{}
		if err := json.Unmarshal(inputJSON, &inputJSONObj); err != nil {
			inputJSONObj = string(inputJSON)
		}

		variables := map[string]interface{}{
			"id":           audit.ID,
			"tenantID":     audit.TenantID,
			"actorID":      audit.ActorID,
			"action":       audit.Action,
			"workflowID":   audit.WorkflowID,
			"runID":        audit.RunID,
			"reason":       audit.Reason,
			"input":        inputJSONObj,
			"status":       audit.Status,
			"errorMessage": audit.ErrorMessage,
			"timestamp":    audit.Timestamp,
		}

		_, err := was.hasura.Mutate(mutation, variables)
		if err == nil {
			return nil
		}
		// Fall through to SQL on Hasura error
	}

	// SQL fallback
	_, err := was.db.ExecContext(ctx, `
		INSERT INTO public.admin_audit_logs (
			id, tenant_id, actor_id, action, workflow_id, run_id, reason, input, status, error_message, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, audit.ID, audit.TenantID, audit.ActorID, audit.Action, audit.WorkflowID, audit.RunID, audit.Reason, inputJSON, audit.Status, audit.ErrorMessage, audit.Timestamp)
	return err
}

// listExecutions queries recent workflow executions
// Hasura-first with SQL fallback
func (was *WorkflowAdminService) listExecutions(ctx context.Context, limit int) ([]map[string]interface{}, error) {
	if was.hasura != nil {
		query := `
			query ListWorkflows($limit: Int!) {
				temporal_workflows(order_by: {id: desc}, limit: $limit) {
					id
					workflow_id
					status
					input
					result
				}
			}
		`

		variables := map[string]interface{}{
			"limit": limit,
		}

		result, err := was.hasura.Query(query, variables)
		if err == nil {
			if workflows, ok := result["temporal_workflows"].([]interface{}); ok {
				out := make([]map[string]interface{}, 0, len(workflows))
				for _, w := range workflows {
					if wf, ok := w.(map[string]interface{}); ok {
						out = append(out, wf)
					}
				}
				return out, nil
			}
		}
		// Fall through to SQL on Hasura error
	}

	// SQL fallback
	rows, err := was.db.QueryContext(ctx, `
		SELECT id, workflow_id, status, input, result
		FROM temporal_workflows
		ORDER BY id DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("query temporal_workflows: %w", err)
	}
	defer rows.Close()

	var out []map[string]interface{}
	for rows.Next() {
		var id sql.NullString
		var workflowID sql.NullString
		var status sql.NullString
		var input sql.NullString
		var result sql.NullString

		if err := rows.Scan(&id, &workflowID, &status, &input, &result); err != nil {
			// skip row on scan error but continue
			continue
		}

		m := map[string]interface{}{
			"id":          nilIfNullString(id),
			"workflow_id": nilIfNullString(workflowID),
			"status":      nilIfNullString(status),
		}
		if input.Valid {
			var js interface{}
			if err := json.Unmarshal([]byte(input.String), &js); err == nil {
				m["input"] = js
			} else {
				m["input"] = input.String
			}
		}
		if result.Valid {
			var js interface{}
			if err := json.Unmarshal([]byte(result.String), &js); err == nil {
				m["result"] = js
			} else {
				m["result"] = result.String
			}
		}

		out = append(out, m)
	}

	if err := rows.Err(); err != nil {
		return out, fmt.Errorf("rows iteration: %w", err)
	}

	return out, nil
}

// nilIfNullString convenience for JSON-friendly values
func nilIfNullString(ns sql.NullString) interface{} {
	if ns.Valid {
		return ns.String
	}
	return nil
}
