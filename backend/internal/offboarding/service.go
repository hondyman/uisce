package offboarding

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type OffboardingService struct {
	db *sql.DB
}

func NewOffboardingService(db *sql.DB) *OffboardingService {
	return &OffboardingService{db: db}
}

type Offboarding struct {
	ID               string
	TenantID         string
	UserID           string
	OffboardedBy     string
	OffboardDate     time.Time
	ReassignToUserID string
	Reason           string
	Status           string // active, completed, reversed
	PendingCount     int
	CreatedAt        time.Time
	CompletedAt      *time.Time
}

// InitiateOffboarding creates permanent delegations for offboarded user
func (s *OffboardingService) InitiateOffboarding(
	ctx context.Context,
	tenantID, userID, reassignToUserID, adminID, reason string,
) (string, error) {
	if userID == reassignToUserID {
		return "", fmt.Errorf("cannot reassign to the same user")
	}

	var existingCount int
	s.db.QueryRowContext(ctx, `
        SELECT COUNT(*) FROM user_offboarding
        WHERE user_id = $1 AND tenant_id = $2 AND status = 'active'
    `, userID, tenantID).Scan(&existingCount)

	if existingCount > 0 {
		return "", fmt.Errorf("user already has active offboarding")
	}

	offboardID := uuid.New().String()
	offboardDate := time.Now()

	_, err := s.db.ExecContext(ctx, `
        INSERT INTO user_offboarding
        (id, tenant_id, user_id, offboarded_by, offboard_date, reassign_to_user_id, reason, status)
        VALUES ($1, $2, $3, $4, $5, $6, $7, 'active')
    `, offboardID, tenantID, userID, adminID, offboardDate, reassignToUserID, reason)

	if err != nil {
		return "", err
	}

	// Create Delegation for all roles (until 2999)
	// Get all active roles for this user
	// Simplified: In real app we might query user_roles table.
	// Hack for MVP: Just create one delegation with NO roles specified (implies ALL)
	delegationID := uuid.New().String()
	_, err = s.db.ExecContext(ctx, `
        INSERT INTO user_delegation
        (id, tenant_id, from_user_id, to_user_id, from_date, to_date, reason, roles, status, offboarding_id, created_by)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'active', $9, $10)
    `, delegationID, tenantID, userID, reassignToUserID, offboardDate,
		time.Date(2999, 12, 31, 23, 59, 59, 0, time.UTC),
		fmt.Sprintf("Offboarding: %s", reason),
		pq.Array([]string{}), // Empty = ALL roles
		offboardID, adminID)

	if err != nil {
		log.Printf("Failed to create offboarding delegation: %v", err)
	}

	// Revoke sessions would go here (omitted for MVP)

	return offboardID, nil
}

func (s *OffboardingService) ListAllOffboardings(
	ctx context.Context,
	tenantID string,
	limit, offset int,
) ([]Offboarding, int64, error) {
	var totalCount int64
	s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM user_offboarding WHERE tenant_id = $1`, tenantID).Scan(&totalCount)

	rows, err := s.db.QueryContext(ctx, `
        SELECT id, tenant_id, user_id, offboarded_by, offboard_date, 
               reassign_to_user_id, reason, status, pending_count, created_at, completed_at
        FROM user_offboarding
        WHERE tenant_id = $1
        ORDER BY offboard_date DESC
        LIMIT $2 OFFSET $3
    `, tenantID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var offboardings []Offboarding
	for rows.Next() {
		var ob Offboarding
		var completedAt *time.Time
		rows.Scan(&ob.ID, &ob.TenantID, &ob.UserID, &ob.OffboardedBy, &ob.OffboardDate, &ob.ReassignToUserID, &ob.Reason, &ob.Status, &ob.PendingCount, &ob.CreatedAt, &completedAt)
		ob.CompletedAt = completedAt
		offboardings = append(offboardings, ob)
	}
	return offboardings, totalCount, nil
}

func (s *OffboardingService) ReverseOffboarding(ctx context.Context, offboardingID, adminID string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE user_offboarding SET status = 'reversed', completed_at = NOW() WHERE id = $1`, offboardingID)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `UPDATE user_delegation SET status = 'revoked', revoked_at = NOW() WHERE offboarding_id = $1`, offboardingID)
	return err
}
