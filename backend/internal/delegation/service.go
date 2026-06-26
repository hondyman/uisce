package delegation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type DelegationService struct {
	db *sql.DB
}

func NewDelegationService(db *sql.DB) *DelegationService {
	return &DelegationService{db: db}
}

type Delegation struct {
	ID         string
	TenantID   string
	FromUserID string
	ToUserID   string
	FromDate   time.Time
	ToDate     time.Time
	Reason     string
	Roles      []string
	Workflows  []string
	Status     string // active, expired, revoked, paused
	CreatedAt  time.Time
	RevokedAt  *time.Time
}

type CreateDelegationRequest struct {
	ToUserID  string // who to delegate to
	FromDate  time.Time
	ToDate    time.Time
	Reason    string
	Roles     []string // empty = all roles
	Workflows []string // empty = all workflows
}

// CreateDelegation creates a new delegation
func (s *DelegationService) CreateDelegation(
	ctx context.Context,
	tenantID, fromUserID, toUserID string,
	req CreateDelegationRequest,
) (string, error) {
	if fromUserID == toUserID {
		return "", fmt.Errorf("cannot delegate to yourself")
	}

	if req.ToDate.Before(req.FromDate) {
		return "", fmt.Errorf("end date must be after start date")
	}

	if req.ToDate.Before(time.Now()) {
		return "", fmt.Errorf("delegation end date must be in the future")
	}

	id := uuid.New().String()

	// Check overlapping
	var existingCount int
	s.db.QueryRowContext(ctx, `
        SELECT COUNT(*) FROM user_delegation
        WHERE from_user_id = $1
            AND to_user_id = $2
            AND tenant_id = $3
            AND status = 'active'
            AND from_date <= $5
            AND to_date >= $4
    `, fromUserID, toUserID, tenantID, req.FromDate, req.ToDate).Scan(&existingCount)

	if existingCount > 0 {
		return "", fmt.Errorf("active delegation to this user already exists for these dates")
	}

	_, err := s.db.ExecContext(ctx, `
        INSERT INTO user_delegation
        (id, tenant_id, from_user_id, to_user_id, from_date, to_date, reason, roles, workflows, status, created_at, created_by)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'active', NOW(), $3)
    `, id, tenantID, fromUserID, toUserID, req.FromDate, req.ToDate, req.Reason,
		pq.Array(req.Roles), pq.Array(req.Workflows))

	if err != nil {
		return "", err
	}

	s.logDelegationAction(ctx, id, "created", nil, fromUserID, map[string]interface{}{
		"to_user_id": toUserID,
		"reason":     req.Reason,
	})

	return id, nil
}

// GetActiveDelegationsForUser returns all delegations where user is the delegate (to_user_id)
func (s *DelegationService) GetActiveDelegationsForUser(
	ctx context.Context,
	tenantID, toUserID string,
) ([]Delegation, error) {
	rows, err := s.db.QueryContext(ctx, `
        SELECT 
            id, tenant_id, from_user_id, to_user_id, from_date, to_date,
            reason, roles, workflows, status, created_at, revoked_at
        FROM user_delegation
        WHERE tenant_id = $1
            AND to_user_id = $2
            AND status IN ('active')
            AND from_date <= CURRENT_DATE
            AND to_date >= CURRENT_DATE
        ORDER BY from_date ASC
    `, tenantID, toUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var delegations []Delegation
	for rows.Next() {
		var d Delegation
		var roles, workflows []string
		var revokedAt *time.Time

		err := rows.Scan(
			&d.ID, &d.TenantID, &d.FromUserID, &d.ToUserID, &d.FromDate, &d.ToDate,
			&d.Reason, (*pq.StringArray)(&roles), (*pq.StringArray)(&workflows), &d.Status, &d.CreatedAt, &revokedAt,
		)
		if err != nil {
			return nil, err
		}

		d.Roles = roles
		d.Workflows = workflows
		d.RevokedAt = revokedAt
		delegations = append(delegations, d)
	}

	return delegations, nil
}

// GetOutgoingDelegationsForUser returns delegations created by user (from_user_id)
func (s *DelegationService) GetOutgoingDelegationsForUser(
	ctx context.Context,
	tenantID, fromUserID string,
) ([]Delegation, error) {
	rows, err := s.db.QueryContext(ctx, `
        SELECT 
            id, tenant_id, from_user_id, to_user_id, from_date, to_date,
            reason, roles, workflows, status, created_at, revoked_at
        FROM user_delegation
        WHERE tenant_id = $1
            AND from_user_id = $2
        ORDER BY to_date DESC
    `, tenantID, fromUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var delegations []Delegation
	for rows.Next() {
		var d Delegation
		var roles, workflows []string
		var revokedAt *time.Time

		err := rows.Scan(
			&d.ID, &d.TenantID, &d.FromUserID, &d.ToUserID, &d.FromDate, &d.ToDate,
			&d.Reason, (*pq.StringArray)(&roles), (*pq.StringArray)(&workflows), &d.Status, &d.CreatedAt, &revokedAt,
		)
		if err != nil {
			return nil, err
		}

		d.Roles = roles
		d.Workflows = workflows
		d.RevokedAt = revokedAt
		delegations = append(delegations, d)
	}

	return delegations, nil
}

// RevokeDelegation revokes an active delegation
func (s *DelegationService) RevokeDelegation(
	ctx context.Context,
	delegationID, revokerUserID string,
) error {
	var currentStatus string
	err := s.db.QueryRowContext(ctx, `SELECT status FROM user_delegation WHERE id = $1`, delegationID).Scan(&currentStatus)
	if err != nil {
		return err
	}

	if currentStatus != "active" && currentStatus != "paused" {
		return fmt.Errorf("delegation is already %s", currentStatus)
	}

	_, err = s.db.ExecContext(ctx, `
        UPDATE user_delegation
        SET status = 'revoked', revoked_at = NOW()
        WHERE id = $1
    `, delegationID)

	s.logDelegationAction(ctx, delegationID, "revoked", nil, revokerUserID, nil)
	return err
}

func (s *DelegationService) PauseDelegation(ctx context.Context, delegationID string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE user_delegation SET status = 'paused' WHERE id = $1`, delegationID)
	return err
}

func (s *DelegationService) ResumeDelegation(ctx context.Context, delegationID string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE user_delegation SET status = 'active' WHERE id = $1`, delegationID)
	return err
}

// CheckIfTaskDelegated checks if a task is covered by an active delegation
func (s *DelegationService) CheckIfTaskDelegated(
	ctx context.Context,
	instanceID, tenantID, approverUserID, approverRole, bpDefID string,
) (*Delegation, error) {
	// Check if approver has active incoming delegations
	delegations, _ := s.GetActiveDelegationsForUser(ctx, tenantID, approverUserID)

	for _, d := range delegations {
		// Check if delegation covers this role
		if len(d.Roles) > 0 && !contains(d.Roles, approverRole) {
			continue
		}
		// Check if delegation covers this workflow
		if len(d.Workflows) > 0 && !contains(d.Workflows, bpDefID) {
			continue
		}
		return &d, nil
	}
	return nil, nil
}

func (s *DelegationService) logDelegationAction(
	ctx context.Context,
	delegationID string,
	action string,
	instanceID *string,
	actorUserID string,
	details map[string]interface{},
) error {
	detailsJSON, _ := json.Marshal(details)
	_, err := s.db.ExecContext(ctx, `
        INSERT INTO delegation_audit
        (id, delegation_id, action, instance_id, actor_user_id, details, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, NOW())
    `, uuid.New().String(), delegationID, action, instanceID, actorUserID, detailsJSON)
	return err
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
