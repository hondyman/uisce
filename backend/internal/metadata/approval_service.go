package metadata

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ApprovalService manages maker-checker approval workflows for upgrade deployments
type ApprovalService struct {
	db *sqlx.DB
}

// NewApprovalService creates a new approval service
func NewApprovalService(db *sqlx.DB) *ApprovalService {
	return &ApprovalService{db: db}
}

// ApprovalRequest represents a request for deployment approval
type ApprovalRequest struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	BundleID     uuid.UUID  `json:"bundle_id" db:"bundle_id"`
	RequestedBy  string     `json:"requested_by" db:"requested_by"`
	RequestedAt  time.Time  `json:"requested_at" db:"requested_at"`
	RequiredRole string     `json:"required_role" db:"required_role"`
	Status       ApprovalStatus `json:"status" db:"status"`
	ApproverID   *string    `json:"approver_id,omitempty" db:"approver_id"`
	Decision     *string    `json:"decision,omitempty" db:"decision"`
	Justification *string   `json:"justification,omitempty" db:"justification"`
	DecidedAt    *time.Time `json:"decided_at,omitempty" db:"decided_at"`
}

// ApprovalStatus represents the state of an approval request
type ApprovalStatus string

const (
	ApprovalStatusPending  ApprovalStatus = "pending"
	ApprovalStatusApproved ApprovalStatus = "approved"
	ApprovalStatusRejected ApprovalStatus = "rejected"
	ApprovalStatusExpired  ApprovalStatus = "expired"
)

// ApprovalDecision represents a maker-checker decision
type ApprovalDecision struct {
	RequestID     uuid.UUID `json:"request_id" db:"request_id"`
	ApproverID    string    `json:"approver_id" db:"approver_id"`
	Decision      string    `json:"decision" db:"decision"` // "approved" or "rejected"
	Justification string    `json:"justification" db:"justification"`
	DecidedAt     time.Time `json:"decided_at" db:"decided_at"`
}

// RequestApproval creates a new approval request for an evidence bundle
func (s *ApprovalService) RequestApproval(ctx context.Context, bundleID uuid.UUID, requesterID string, requiredRole string) (*ApprovalRequest, error) {
	req := &ApprovalRequest{
		ID:           uuid.New(),
		BundleID:     bundleID,
		RequestedBy:  requesterID,
		RequestedAt:  time.Now(),
		RequiredRole: requiredRole,
		Status:       ApprovalStatusPending,
	}

	query := `
		INSERT INTO metadata.approval_requests (id, bundle_id, requested_by, requested_at, required_role, status)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := s.db.ExecContext(ctx, query, req.ID, req.BundleID, req.RequestedBy, req.RequestedAt, req.RequiredRole, req.Status)
	if err != nil {
		return nil, fmt.Errorf("failed to create approval request: %w", err)
	}

	return req, nil
}

// RecordDecision logs an approval or rejection decision
func (s *ApprovalService) RecordDecision(ctx context.Context, requestID uuid.UUID, approverID string, decision string, justification string) error {
	if decision != "approved" && decision != "rejected" {
		return fmt.Errorf("invalid decision: must be 'approved' or 'rejected'")
	}

	decidedAt := time.Now()
	status := ApprovalStatusApproved
	if decision == "rejected" {
		status = ApprovalStatusRejected
	}

	query := `
		UPDATE metadata.approval_requests
		SET approver_id = $1, decision = $2, justification = $3, decided_at = $4, status = $5
		WHERE id = $6
	`
	result, err := s.db.ExecContext(ctx, query, approverID, decision, justification, decidedAt, status, requestID)
	if err != nil {
		return fmt.Errorf("failed to record approval decision: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("approval request not found: %s", requestID)
	}

	return nil
}

// GetApprovalChain retrieves the complete approval history for a bundle
func (s *ApprovalService) GetApprovalChain(ctx context.Context, bundleID uuid.UUID) ([]ApprovalDecision, error) {
	var decisions []ApprovalDecision
	query := `
		SELECT id as request_id, approver_id, decision, justification, decided_at
		FROM metadata.approval_requests
		WHERE bundle_id = $1 AND decision IS NOT NULL
		ORDER BY decided_at ASC
	`
	err := s.db.SelectContext(ctx, &decisions, query, bundleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get approval chain: %w", err)
	}

	return decisions, nil
}

// GetPendingApprovals retrieves all pending approval requests for a specific role
func (s *ApprovalService) GetPendingApprovals(ctx context.Context, requiredRole string) ([]ApprovalRequest, error) {
	var requests []ApprovalRequest
	query := `
		SELECT id, bundle_id, requested_by, requested_at, required_role, status
		FROM metadata.approval_requests
		WHERE status = $1 AND required_role = $2
		ORDER BY requested_at ASC
	`
	err := s.db.SelectContext(ctx, &requests, query, ApprovalStatusPending, requiredRole)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending approvals: %w", err)
	}

	return requests, nil
}

// IsApproved checks if a bundle has been approved
func (s *ApprovalService) IsApproved(ctx context.Context, bundleID uuid.UUID) (bool, error) {
	var count int
	query := `
		SELECT COUNT(*) FROM metadata.approval_requests
		WHERE bundle_id = $1 AND status = $2
	`
	err := s.db.GetContext(ctx, &count, query, bundleID, ApprovalStatusApproved)
	if err != nil {
		return false, fmt.Errorf("failed to check approval status: %w", err)
	}

	return count > 0, nil
}
