package bp

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ============================================================================
// Types
// ============================================================================

type BusinessProcess struct {
	ID                 uuid.UUID  `db:"id" json:"id"`
	TenantID           uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	ProcessName        string     `db:"process_name" json:"processName"`
	Description        string     `db:"description" json:"description"`
	EntityType         string     `db:"entity_type" json:"entity"`
	Status             string     `db:"status" json:"status"`
	IsActive           bool       `db:"is_active" json:"isActive"`
	CreatedBy          string     `db:"created_by" json:"createdBy"`
	CreatedAt          time.Time  `db:"created_at" json:"createdAt"`
	UpdatedBy          *string    `db:"updated_by" json:"updatedBy"`
	UpdatedAt          *time.Time `db:"updated_at" json:"updatedAt"`
	TotalDurationHours *int       `db:"total_duration_hours" json:"totalDurationHours"`
	VersionNumber      int        `db:"version_number" json:"versionNumber"`
	Steps              []BPStep   `db:"-" json:"steps"`
}

type BPStep struct {
	ID            uuid.UUID       `db:"id" json:"id"`
	ProcessID     uuid.UUID       `db:"business_process_id" json:"processId"`
	StepOrder     int16           `db:"step_order" json:"stepOrder"`
	StepType      string          `db:"step_type" json:"stepType"`
	StepName      string          `db:"step_name" json:"stepName"`
	AssigneeRole  *string         `db:"assignee_role" json:"assigneeRole,omitempty"`
	Description   *string         `db:"description" json:"description"`
	DurationHours int16           `db:"duration_hours" json:"durationHours"`
	Status        string          `db:"status" json:"status"`
	Config        json.RawMessage `db:"config" json:"config"`
	CreatedAt     time.Time       `db:"created_at" json:"createdAt"`
	UpdatedAt     *time.Time      `db:"updated_at" json:"updatedAt"`
}

type BPExecution struct {
	ID                   uuid.UUID       `db:"id" json:"id"`
	TenantID             uuid.UUID       `db:"tenant_id" json:"tenantId"`
	BusinessProcessID    uuid.UUID       `db:"business_process_id" json:"businessProcessId"`
	WorkflowID           *string         `db:"workflow_id" json:"workflowId"`
	EntityID             uuid.UUID       `db:"entity_id" json:"entityId"`
	InitiatedBy          string          `db:"initiated_by" json:"initiatedBy"`
	InitiatedAt          time.Time       `db:"initiated_at" json:"initiatedAt"`
	CompletedAt          *time.Time      `db:"completed_at" json:"completedAt"`
	ExecutionStatus      string          `db:"execution_status" json:"executionStatus"`
	CurrentStepOrder     *int16          `db:"current_step_order" json:"currentStepOrder"`
	TotalDurationMinutes *int            `db:"total_duration_minutes" json:"totalDurationMinutes"`
	ErrorMessage         *string         `db:"error_message" json:"errorMessage"`
	Metadata             json.RawMessage `db:"metadata" json:"metadata"`
}

type AuditEntry struct {
	ID                uuid.UUID       `db:"id" json:"id"`
	TenantID          uuid.UUID       `db:"tenant_id" json:"tenantId"`
	BusinessProcessID *uuid.UUID      `db:"business_process_id" json:"businessProcessId"`
	ActionType        string          `db:"action_type" json:"actionType"`
	ActorEmail        string          `db:"actor_email" json:"actorEmail"`
	ActorRole         *string         `db:"actor_role" json:"actorRole"`
	ActionDetails     json.RawMessage `db:"action_details" json:"actionDetails"`
	Timestamp         time.Time       `db:"timestamp" json:"timestamp"`
	IPAddress         *string         `db:"ip_address" json:"ipAddress"`
}

// ============================================================================
// Service
// ============================================================================

type BPService struct {
	db *sqlx.DB
}

func NewBPService(db *sqlx.DB) *BPService {
	return &BPService{db: db}
}

// ============================================================================
// Create / Save
// ============================================================================

// SaveBusinessProcess creates a new BP or updates existing (with version control)
func (s *BPService) SaveBusinessProcess(ctx context.Context, tenantID uuid.UUID, bp *BusinessProcess, createdBy string) (*BusinessProcess, error) {
	if bp == nil {
		return nil, errors.New("business process cannot be nil")
	}

	bp.TenantID = tenantID
	bp.CreatedBy = createdBy
	bp.CreatedAt = time.Now()

	// Calculate total duration
	totalDuration := int16(0)
	for _, step := range bp.Steps {
		totalDuration += step.DurationHours
	}
	totalDurationInt := int(totalDuration)
	bp.TotalDurationHours = &totalDurationInt

	// Generate ID if new
	if bp.ID == uuid.Nil {
		bp.ID = uuid.New()
	}

	// Start transaction
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert or update business_processes
	// TODO: Refactor to Hasura GraphQL
	// mutation {
	//   insert_business_processes_one(
	//     object: {
	//       id: "uuid", tenant_id: "tenant-uuid", process_name: "Onboarding"
	//       description: "New client onboarding", entity_type: "client", status: "active"
	//       is_active: true, created_by: "user@example.com", total_duration_hours: 48
	//       version_number: 1
	//       bp_steps: {data: [{step_order: 1, step_type: "data_entry", step_name: "Basic Info", ...}]}
	//     }
	//     on_conflict: {constraint: business_processes_pkey, update_columns: [process_name, description, entity_type, status, is_active, updated_by, updated_at, total_duration_hours]}
	//   ) { id version_number }
	// }
	// Note: Include nested bp_steps insert, use _inc for version_number
	insertBPSQL := `
		INSERT INTO business_processes (
			id, tenant_id, process_name, description, entity_type, status,
			is_active, created_by, created_at, total_duration_hours, version_number
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id) DO UPDATE SET
			process_name = $3,
			description = $4,
			entity_type = $5,
			status = $6,
			is_active = $7,
			updated_by = $8,
			updated_at = CURRENT_TIMESTAMP,
			total_duration_hours = $10,
			version_number = version_number + 1
	`

	_, err = tx.ExecContext(ctx, insertBPSQL,
		bp.ID, bp.TenantID, bp.ProcessName, bp.Description, bp.EntityType,
		bp.Status, bp.IsActive, bp.CreatedBy, bp.CreatedAt,
		bp.TotalDurationHours, bp.VersionNumber,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert business process: %w", err)
	}

	// Delete existing steps for this BP
	_, err = tx.ExecContext(ctx, "DELETE FROM bp_steps WHERE business_process_id = $1", bp.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete existing steps: %w", err)
	}

	// Insert new steps
	for _, step := range bp.Steps {
		step.ProcessID = bp.ID
		step.CreatedAt = time.Now()

		insertStepSQL := `
			INSERT INTO bp_steps (
				id, business_process_id, step_order, step_type, step_name,
				assignee_role, description, duration_hours, status, config, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`

		configJSON, err := json.Marshal(step.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal step config: %w", err)
		}

		_, err = tx.ExecContext(ctx, insertStepSQL,
			uuid.New(), step.ProcessID, step.StepOrder, step.StepType, step.StepName,
			step.AssigneeRole, step.Description, step.DurationHours, "pending", configJSON, step.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to insert step: %w", err)
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Log to audit trail
	_ = s.LogAuditEntry(ctx, tenantID, &bp.ID, createdBy, "created", map[string]interface{}{
		"processName": bp.ProcessName,
		"stepsCount":  len(bp.Steps),
	})

	return bp, nil
}

// ============================================================================
// Read
// ============================================================================

// GetBusinessProcess retrieves a BP with all its steps
func (s *BPService) GetBusinessProcess(ctx context.Context, tenantID uuid.UUID, processID uuid.UUID) (*BusinessProcess, error) {
	var bp BusinessProcess

	// Query business_processes
	// TODO: Refactor to Hasura GraphQL
	// query {
	//   business_processes(
	//     where: {id: {_eq: "process-uuid"}, tenant_id: {_eq: "tenant-uuid"}}
	//   ) {
	//     id tenant_id process_name description entity_type status is_active
	//     created_by created_at updated_by updated_at total_duration_hours version_number
	//     bp_steps(order_by: {step_order: asc}) {
	//       id business_process_id step_order step_type step_name assignee_role
	//       description duration_hours status config created_at updated_at
	//     }
	//   }
	// }
	err := s.db.GetContext(ctx, &bp, `
		SELECT id, tenant_id, process_name, description, entity_type, status,
		       is_active, created_by, created_at, updated_by, updated_at,
		       total_duration_hours, version_number
		FROM business_processes
		WHERE id = $1 AND tenant_id = $2
	`, processID, tenantID)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("business process not found")
		}
		return nil, fmt.Errorf("failed to get business process: %w", err)
	}

	// Query steps
	var steps []BPStep
	err = s.db.SelectContext(ctx, &steps, `
		SELECT id, business_process_id, step_order, step_type, step_name,
		       assignee_role, description, duration_hours, status, config, created_at, updated_at
		FROM bp_steps
		WHERE business_process_id = $1
		ORDER BY step_order ASC
	`, processID)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to get steps: %w", err)
	}

	bp.Steps = steps
	return &bp, nil
}

// SaveFormData saves form data for an entity
func (s *BPService) SaveFormData(ctx context.Context, entityID string, formData map[string]interface{}, status string) error {
	// Convert form data to JSON
	dataJSON, err := json.Marshal(formData)
	if err != nil {
		return fmt.Errorf("failed to marshal form data: %w", err)
	}

	// Update or insert form data
	// TODO: Refactor to Hasura GraphQL
	// mutation {
	//   insert_business_process_form_data_one(
	//     object: {entity_id: "entity-uuid", form_data: {field1: "value1"}, status: "draft", updated_at: "now()"}
	//     on_conflict: {constraint: business_process_form_data_pkey, update_columns: [form_data, status, updated_at]}
	//   ) { entity_id }
	// }
	query := `
		INSERT INTO business_process_form_data (entity_id, form_data, status, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (entity_id) 
		DO UPDATE SET 
			form_data = EXCLUDED.form_data,
			status = EXCLUDED.status,
			updated_at = NOW()
	`

	_, err = s.db.ExecContext(ctx, query, entityID, dataJSON, status)
	if err != nil {
		return fmt.Errorf("failed to save form data: %w", err)
	}

	return nil
}

// ListBusinessProcesses returns all BPs for a tenant with pagination
func (s *BPService) ListBusinessProcesses(ctx context.Context, tenantID uuid.UUID, offset int, limit int) ([]BusinessProcess, int64, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var bps []BusinessProcess

	// Get processes
	// TODO: Refactor to Hasura GraphQL
	// query {
	//   business_processes(
	//     where: {tenant_id: {_eq: "tenant-uuid"}}
	//     order_by: {created_at: desc}, offset: 0, limit: 20
	//   ) {
	//     id tenant_id process_name description entity_type status is_active
	//     created_by created_at updated_by updated_at total_duration_hours version_number
	//   }
	//   business_processes_aggregate(where: {tenant_id: {_eq: "tenant-uuid"}}) {
	//     aggregate { count }
	//   }
	// }
	err := s.db.SelectContext(ctx, &bps, `
		SELECT id, tenant_id, process_name, description, entity_type, status,
		       is_active, created_by, created_at, updated_by, updated_at,
		       total_duration_hours, version_number
		FROM business_processes
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, tenantID, limit, offset)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, 0, fmt.Errorf("failed to list business processes: %w", err)
	}

	// Get total count
	var total int64
	err = s.db.GetContext(ctx, &total, `
		SELECT COUNT(*) FROM business_processes WHERE tenant_id = $1
	`, tenantID)

	if err != nil {
		return nil, 0, fmt.Errorf("failed to get business process count: %w", err)
	}

	// Load steps for each BP
	for i := range bps {
		var steps []BPStep
		err = s.db.SelectContext(ctx, &steps, `
			SELECT id, business_process_id, step_order, step_type, step_name,
			       assignee_role, description, duration_hours, status, config, created_at, updated_at
			FROM bp_steps
			WHERE business_process_id = $1
			ORDER BY step_order ASC
		`, bps[i].ID)

		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, 0, fmt.Errorf("failed to load steps for BP %s: %w", bps[i].ID, err)
		}

		bps[i].Steps = steps
	}

	return bps, total, nil
}

// ============================================================================
// Execution
// ============================================================================

// StartExecution creates a BP execution record
func (s *BPService) StartExecution(ctx context.Context, tenantID uuid.UUID, processID uuid.UUID, entityID uuid.UUID, initiatedBy string) (*BPExecution, error) {
	exec := &BPExecution{
		ID:                uuid.New(),
		TenantID:          tenantID,
		BusinessProcessID: processID,
		EntityID:          entityID,
		InitiatedBy:       initiatedBy,
		InitiatedAt:       time.Now(),
		ExecutionStatus:   "running",
	}

	// TODO: Refactor to Hasura GraphQL
	// mutation {
	//   insert_bp_executions_one(object: {
	//     id: "exec-uuid", tenant_id: "tenant-uuid", business_process_id: "process-uuid"
	//     entity_id: "entity-uuid", initiated_by: "user@example.com"
	//     initiated_at: "2024-01-15T10:00:00Z", execution_status: "running"
	//   }) { id }
	// }
	insertSQL := `
		INSERT INTO bp_executions (
			id, tenant_id, business_process_id, entity_id, initiated_by,
			initiated_at, execution_status
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := s.db.ExecContext(ctx, insertSQL,
		exec.ID, exec.TenantID, exec.BusinessProcessID, exec.EntityID,
		exec.InitiatedBy, exec.InitiatedAt, exec.ExecutionStatus,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create execution: %w", err)
	}

	return exec, nil
}

// UpdateExecutionStatus updates execution status
func (s *BPService) UpdateExecutionStatus(ctx context.Context, executionID uuid.UUID, status string, workflowID *string) error {
	// TODO: Refactor to Hasura GraphQL
	// mutation {
	//   update_bp_executions(
	//     where: {id: {_eq: "exec-uuid"}}
	//     _set: {execution_status: "completed", workflow_id: "workflow-123", updated_at: "now()"}
	//   ) { affected_rows }
	// }
	updateSQL := `
		UPDATE bp_executions
		SET execution_status = $1, workflow_id = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
	`

	result, err := s.db.ExecContext(ctx, updateSQL, status, workflowID, executionID)
	if err != nil {
		return fmt.Errorf("failed to update execution status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("execution not found")
	}

	return nil
}

// ============================================================================
// Audit Trail
// ============================================================================

// LogAuditEntry logs an action to the audit trail
func (s *BPService) LogAuditEntry(ctx context.Context, tenantID uuid.UUID, processID *uuid.UUID, actor string, actionType string, details interface{}) error {
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("failed to marshal audit details: %w", err)
	}

	// TODO: Refactor to Hasura GraphQL
	// mutation {
	//   insert_bp_audit_trail_one(object: {
	//     id: "audit-uuid", tenant_id: "tenant-uuid", business_process_id: "process-uuid"
	//     action_type: "created", actor_email: "user@example.com"
	//     action_details: {processName: "Onboarding", stepsCount: 5}
	//     timestamp: "now()"
	//   }) { id }
	// }
	insertSQL := `
		INSERT INTO bp_audit_trail (
			id, tenant_id, business_process_id, action_type, actor_email,
			action_details, timestamp
		) VALUES ($1, $2, $3, $4, $5, $6, CURRENT_TIMESTAMP)
	`

	_, err = s.db.ExecContext(ctx, insertSQL,
		uuid.New(), tenantID, processID, actionType, actor, detailsJSON,
	)

	return err
}

// GetAuditTrail retrieves audit entries for a BP
func (s *BPService) GetAuditTrail(ctx context.Context, tenantID uuid.UUID, processID uuid.UUID, limit int) ([]AuditEntry, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var entries []AuditEntry

	// TODO: Refactor to Hasura GraphQL
	// query {
	//   bp_audit_trail(
	//     where: {tenant_id: {_eq: "tenant-uuid"}, business_process_id: {_eq: "process-uuid"}}
	//     order_by: {timestamp: desc}, limit: 50
	//   ) {
	//     id tenant_id business_process_id action_type actor_email actor_role
	//     action_details timestamp ip_address
	//   }
	// }
	err := s.db.SelectContext(ctx, &entries, `
		SELECT id, tenant_id, business_process_id, action_type, actor_email,
		       actor_role, action_details, timestamp, ip_address
		FROM bp_audit_trail
		WHERE tenant_id = $1 AND business_process_id = $2
		ORDER BY timestamp DESC
		LIMIT $3
	`, tenantID, processID, limit)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to get audit trail: %w", err)
	}

	return entries, nil
}

// ============================================================================
// Validation
// ============================================================================

// ValidateBusinessProcess checks BP structure and rules
func (s *BPService) ValidateBusinessProcess(bp *BusinessProcess) []string {
	var errors []string

	if bp.ProcessName == "" {
		errors = append(errors, "process name is required")
	}

	if bp.EntityType == "" {
		errors = append(errors, "entity type is required")
	}

	if len(bp.Steps) == 0 {
		errors = append(errors, "at least one step is required")
	}

	// Validate steps
	for i, step := range bp.Steps {
		if step.StepName == "" {
			errors = append(errors, fmt.Sprintf("step %d: name is required", i+1))
		}

		if step.StepType == "" {
			errors = append(errors, fmt.Sprintf("step %d: type is required", i+1))
		}

		validStepTypes := []string{"data_entry", "validate", "approve", "notify", "integrate", "condition"}
		validType := false
		for _, t := range validStepTypes {
			if step.StepType == t {
				validType = true
				break
			}
		}

		if !validType {
			errors = append(errors, fmt.Sprintf("step %d: invalid step type '%s'", i+1, step.StepType))
		}

		// Step-specific validation
		if step.StepType == "validate" && step.Config == nil {
			errors = append(errors, fmt.Sprintf("step %d: validation rules are required", i+1))
		}

		if step.StepType == "approve" && step.Config == nil {
			errors = append(errors, fmt.Sprintf("step %d: approver assignment required", i+1))
		}
	}

	return errors
}

// ============================================================================
// Utilities
// ============================================================================

// DeleteBusinessProcess soft-deletes a BP (archive)
func (s *BPService) DeleteBusinessProcess(ctx context.Context, tenantID uuid.UUID, processID uuid.UUID) error {
	// TODO: Refactor to Hasura GraphQL
	// mutation {
	//   update_business_processes(
	//     where: {id: {_eq: "process-uuid"}, tenant_id: {_eq: "tenant-uuid"}}
	//     _set: {status: "archived", is_active: false, updated_at: "now()"}
	//   ) { affected_rows }
	// }
	updateSQL := `
		UPDATE business_processes
		SET status = 'archived', is_active = false, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND tenant_id = $2
	`

	result, err := s.db.ExecContext(ctx, updateSQL, processID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to delete business process: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("business process not found")
	}

	return nil
}

// GetExecutionHistory retrieves past executions
func (s *BPService) GetExecutionHistory(ctx context.Context, tenantID uuid.UUID, processID uuid.UUID, limit int) ([]BPExecution, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var execs []BPExecution

	// TODO: Refactor to Hasura GraphQL
	// query {
	//   bp_executions(
	//     where: {tenant_id: {_eq: "tenant-uuid"}, business_process_id: {_eq: "process-uuid"}}
	//     order_by: {initiated_at: desc}, limit: 20
	//   ) {
	//     id tenant_id business_process_id workflow_id entity_id initiated_by
	//     initiated_at completed_at execution_status current_step_order
	//     total_duration_minutes error_message metadata
	//   }
	// }
	err := s.db.SelectContext(ctx, &execs, `
		SELECT id, tenant_id, business_process_id, workflow_id, entity_id,
		       initiated_by, initiated_at, completed_at, execution_status,
		       current_step_order, total_duration_minutes, error_message, metadata
		FROM bp_executions
		WHERE tenant_id = $1 AND business_process_id = $2
		ORDER BY initiated_at DESC
		LIMIT $3
	`, tenantID, processID, limit)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to get execution history: %w", err)
	}

	return execs, nil
}
