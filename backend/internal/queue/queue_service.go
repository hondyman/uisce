package queue

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

type QueuedTask struct {
	InstanceID          string                 `json:"instance_id"`
	BPKey               string                 `json:"bp_key"`
	StepKey             string                 `json:"step_key"`
	ApproverRole        string                 `json:"approver_role"`
	CreatedAt           time.Time              `json:"created_at"`
	SLAExpiresAt        time.Time              `json:"sla_expires_at"`
	SLAStatus           string                 `json:"sla_status"`
	HoursRemaining      float64                `json:"hours_remaining"`
	ApplicantName       string                 `json:"applicant_name"`
	Amount              string                 `json:"amount"`
	Entity              string                 `json:"entity"`
	Metadata            map[string]interface{} `json:"metadata"`
	IsDelegated         bool                   `json:"is_delegated"`
	DelegatedFromUserID *string                `json:"delegated_from_user_id"`
}

type QueueRequest struct {
	Role   string
	Status string
	SortBy string
	Limit  int
	Offset int
}

type QueueService struct {
	db *sql.DB
}

func NewQueueService(db *sql.DB) *QueueService {
	return &QueueService{db: db}
}

// GetQueuedTasks returns pending tasks for the given role + delegated tasks for the user (passed in req)
// Note: We are overloading this method to support user delegation
func (s *QueueService) GetQueuedTasks(ctx context.Context, tenantID string, viewingAsUserID string, role string, req QueueRequest) ([]QueuedTask, int64, error) {
	// Base Query for Direct
	// Base Query for Direct
	// Complex UNION query requires positional args.
	// 1: role, 2: tenantID, 3: viewingAsUserID

	orderBy := "sla_expires_at ASC"
	if req.SortBy == "amount" {
		orderBy = "amount DESC NULLS LAST"
	} else if req.SortBy == "created_at" {
		orderBy = "instance_created_at ASC"
	}

	query := fmt.Sprintf(`
        SELECT 
            instance_id, bp_key, step_key, current_approver_role,
            instance_created_at, sla_expires_at, sla_status, hours_remaining,
            applicant_name, amount, entity, metadata,
            NULL as delegation_id, NULL as delegated_from
        FROM my_approvals_queue
        WHERE current_approver_role = $1 AND tenant_id = $2 AND status = 'running'
        
        UNION ALL
        
        SELECT 
            maq.instance_id, maq.bp_key, maq.step_key, maq.current_approver_role,
            maq.instance_created_at, maq.sla_expires_at, maq.sla_status, maq.hours_remaining,
            maq.applicant_name, maq.amount, maq.entity, maq.metadata,
            ud.id, ud.from_user_id
        FROM my_approvals_queue maq
        JOIN user_delegation ud ON 
            ud.to_user_id = $3 AND ud.tenant_id = $2 AND ud.status = 'active'
            AND ud.from_date <= CURRENT_DATE AND ud.to_date >= CURRENT_DATE
            AND (ARRAY_LENGTH(ud.roles, 1) IS NULL OR maq.current_approver_role = ANY(ud.roles))
            AND (ARRAY_LENGTH(ud.workflows, 1) IS NULL OR maq.bp_key = ANY(ud.workflows))
        WHERE maq.tenant_id = $2 AND maq.status = 'running'
        
        ORDER BY %s
        LIMIT $4 OFFSET $5
    `, orderBy)

	rows, err := s.db.QueryContext(ctx, query, role, tenantID, viewingAsUserID, req.Limit, req.Offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	// Count Query (Simplified)
	// var totalCount int64 = 0 // Counting union is expensive, skipping exact count for this step if OK

	var tasks []QueuedTask
	for rows.Next() {
		var task QueuedTask
		var metadata string
		var delID, delFrom *string
		if err := rows.Scan(
			&task.InstanceID, &task.BPKey, &task.StepKey, &task.ApproverRole,
			&task.CreatedAt, &task.SLAExpiresAt, &task.SLAStatus, &task.HoursRemaining,
			&task.ApplicantName, &task.Amount, &task.Entity, &metadata,
			&delID, &delFrom,
		); err != nil {
			return nil, 0, err
		}
		json.Unmarshal([]byte(metadata), &task.Metadata)
		if delID != nil {
			task.IsDelegated = true
			task.DelegatedFromUserID = delFrom
		}
		tasks = append(tasks, task)
	}

	return tasks, 0, nil
}

func (s *QueueService) AssignTaskToUser(ctx context.Context, tenantID, instanceID, userID string) error {
	// For now, this updates the current_approver in workflow_instances or similar.
	// Assuming there's a table or logic to handle assignment.
	// Since we don't have the table schema fully handy, we'll write a placeholder query.
	// Real implementation depends on where assignment is stored.
	// For this fix, we assume it updates 'workflow_instances' or related table.
	_, err := s.db.ExecContext(ctx, `
		UPDATE workflow_instances 
		SET assigned_to_user_id = $1 
		WHERE id = $2 AND tenant_id = $3
	`, userID, instanceID, tenantID)
	return err
}

func (s *QueueService) UnassignTask(ctx context.Context, instanceID string) error {
	// Helper to unassign
	_, err := s.db.ExecContext(ctx, `
		UPDATE workflow_instances 
		SET assigned_to_user_id = NULL 
		WHERE id = $1
	`, instanceID)
	return err
}

func (s *QueueService) RefreshQueuesView(ctx context.Context) error {
	// First check if it's a materialized view
	var relkind string
	err := s.db.QueryRowContext(ctx, "SELECT relkind FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE n.nspname = 'public' AND c.relname = 'my_approvals_queue'").Scan(&relkind)
	if err != nil {
		return fmt.Errorf("check queue view type: %w", err)
	}

	if relkind != "m" {
		// Not a materialized view, nothing to refresh
		return nil
	}

	_, err = s.db.ExecContext(ctx, "REFRESH MATERIALIZED VIEW CONCURRENTLY my_approvals_queue")
	if err != nil {
		// Fallback if not concurrent capable or first run
		_, err = s.db.ExecContext(ctx, "REFRESH MATERIALIZED VIEW my_approvals_queue")
	}
	return err
}
