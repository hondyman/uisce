package access

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"
)

type AccessService struct {
	db *sql.DB
}

func NewAccessService(db *sql.DB) *AccessService {
	return &AccessService{db: db}
}

type InitiatorPolicy struct {
	BPDefID              string
	InitiatorRole        string
	CanInitiateOnBehalf  bool
	MaxConcurrentPerUser int
	RateLimitPerHour     int
	RequiredFields       []string
	ReadonlyFields       []string
}

// CanInitiateWorkflow checks if a user can start a specific workflow
func (s *AccessService) CanInitiateWorkflow(
	ctx context.Context,
	userID, userRole, tenantID, bpDefID string,
) (bool, string, error) {
	// 1. Check if user has the required role
	var hasRole bool
	err := s.db.QueryRowContext(ctx, `
        SELECT EXISTS(
            SELECT 1 FROM user_roles
            WHERE user_id = $1 AND role = $2 AND tenant_id = $3 AND revoked_at IS NULL
        )
    `, userID, userRole, tenantID).Scan(&hasRole)
	if err != nil || !hasRole {
		return false, "User does not have required role", nil
	}

	// 2. Check if this role can initiate this workflow
	var policy InitiatorPolicy
	var reqFields, roFields []string
	err = s.db.QueryRowContext(ctx, `
        SELECT 
            bp_def_id, initiator_role, can_initiate_on_behalf_of,
            max_concurrent_per_user, rate_limit_per_hour,
            required_fields, readonly_fields_after_start
        FROM workflow_access
        WHERE bp_def_id = $1 AND initiator_role = $2 AND tenant_id = $3
    `, bpDefID, userRole, tenantID).Scan(
		&policy.BPDefID, &policy.InitiatorRole, &policy.CanInitiateOnBehalf,
		&policy.MaxConcurrentPerUser, &policy.RateLimitPerHour,
		(*pq.StringArray)(&reqFields), (*pq.StringArray)(&roFields),
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, "Role not authorized for workflow", nil
		}
		return false, fmt.Sprintf("Workflow access denied: %v", err), nil
	}
	policy.RequiredFields = reqFields
	policy.ReadonlyFields = roFields

	// 3. Check concurrent instance limit
	if policy.MaxConcurrentPerUser > 0 {
		var count int
		_ = s.db.QueryRowContext(ctx, `
            SELECT COUNT(*) FROM workflow_instance
            WHERE created_by = $1 AND bp_def_id = $2 AND status = 'running' AND tenant_id = $3
        `, userID, bpDefID, tenantID).Scan(&count)

		if count >= policy.MaxConcurrentPerUser {
			return false, fmt.Sprintf("Max concurrent instances (%d) exceeded", policy.MaxConcurrentPerUser), nil
		}
	}

	// 4. Check rate limit (per hour)
	if policy.RateLimitPerHour > 0 {
		var count int
		_ = s.db.QueryRowContext(ctx, `
            SELECT COUNT(*) FROM instance_creation_log
            WHERE user_id = $1 AND tenant_id = $2 AND bp_def_id = $3
            AND created_at > NOW() - INTERVAL '1 hour'
        `, userID, tenantID, bpDefID).Scan(&count)

		if count >= policy.RateLimitPerHour {
			return false, fmt.Sprintf("Rate limit (%d/hour) exceeded", policy.RateLimitPerHour), nil
		}
	}

	return true, "", nil
}

// ListInitiatableWorkflows returns all workflows a user can start
func (s *AccessService) ListInitiatableWorkflows(
	ctx context.Context,
	userID, tenantID string,
) ([]string, error) {
	// Get all roles for this user
	rows, err := s.db.QueryContext(ctx, `
        SELECT DISTINCT role FROM user_roles
        WHERE user_id = $1 AND tenant_id = $2 AND revoked_at IS NULL
    `, userID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var userRoles []string
	for rows.Next() {
		var role string
		rows.Scan(&role)
		userRoles = append(userRoles, role)
	}

	if len(userRoles) == 0 {
		return []string{}, nil
	}

	// Get all BPs accessible to these roles
	bpRows, err := s.db.QueryContext(ctx, `
        SELECT DISTINCT bp_def_id FROM workflow_access
        WHERE initiator_role = ANY($1) AND tenant_id = $2
    `, pq.Array(userRoles), tenantID)
	if err != nil {
		return nil, err
	}
	defer bpRows.Close()

	var bpDefIDs []string
	for bpRows.Next() {
		var bpDefID string
		bpRows.Scan(&bpDefID)
		bpDefIDs = append(bpDefIDs, bpDefID)
	}

	return bpDefIDs, nil
}

// LogInstanceCreation tracks for rate limiting
func (s *AccessService) LogInstanceCreation(
	ctx context.Context,
	userID, tenantID, bpDefID string,
) error {
	_, err := s.db.ExecContext(ctx, `
        INSERT INTO instance_creation_log (user_id, tenant_id, bp_def_id)
        VALUES ($1, $2, $3)
    `, userID, tenantID, bpDefID)
	return err
}
