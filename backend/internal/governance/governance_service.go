package governance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type GovernanceStatus string

const (
	StatusDraft           GovernanceStatus = "DRAFT"
	StatusPendingApproval GovernanceStatus = "PENDING_APPROVAL"
	StatusActive          GovernanceStatus = "ACTIVE"
	StatusRejected        GovernanceStatus = "REJECTED"
)

// CatalogChangeRequest represents a proposed catalog metadata change requiring Four-Eyes approval
type CatalogChangeRequest struct {
	ID             string           `json:"requestId"`
	ProposalID     string           `json:"proposalId"`
	TenantID       string           `json:"tenantId"`
	BOID           string           `json:"boId,omitempty"`
	BranchID       string           `json:"branchId,omitempty"`
	Title          string           `json:"title,omitempty"`
	Description    string           `json:"description,omitempty"`
	MakerUserID    string           `json:"makerUserId"`
	CheckerUserID  string           `json:"checkerUserId,omitempty"`
	Status         GovernanceStatus `json:"status"`
	ProposedDiff   string           `json:"proposedDiff,omitempty"`
	DiffPayload    json.RawMessage  `json:"diffPayload,omitempty"`
	Justification  string           `json:"justification,omitempty"`
	ASTDiffSummary string           `json:"astDiffSummary,omitempty"`
	CreatedAt      time.Time        `json:"createdAt"`
	UpdatedAt      time.Time        `json:"updatedAt"`
}

// MakerCheckerService handles approval workflows for catalog metadata changes enforcing Four-Eyes governance
type MakerCheckerService struct {
	requests map[string]*CatalogChangeRequest
}

// NewMakerCheckerService creates a new governance service
func NewMakerCheckerService() *MakerCheckerService {
	return &MakerCheckerService{
		requests: make(map[string]*CatalogChangeRequest),
	}
}

// ProposeChange creates a new catalog change request in PENDING_APPROVAL status (Maker role)
func (s *MakerCheckerService) ProposeChange(ctx context.Context, tenantID, makerUserID, title, desc, diff string) (*CatalogChangeRequest, error) {
	if tenantID == "" || makerUserID == "" {
		return nil, fmt.Errorf("tenantID and makerUserID are required")
	}

	req := &CatalogChangeRequest{
		ID:           uuid.New().String(),
		ProposalID:   uuid.New().String(),
		TenantID:     tenantID,
		BranchID:     uuid.New().String(),
		Title:        title,
		Description:  desc,
		MakerUserID:  makerUserID,
		Status:       StatusPendingApproval,
		ProposedDiff: diff,
		DiffPayload:  json.RawMessage(diff),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	s.requests[req.ID] = req
	s.requests[req.ProposalID] = req
	return req, nil
}

// SubmitProposal registers a new semantic modification proposal extracting tenant_id and user_id from context (Rule 7)
func (s *MakerCheckerService) SubmitProposal(ctx context.Context, boID string, diffPayload json.RawMessage, justification string) (*CatalogChangeRequest, error) {
	tenantID, ok := ctx.Value("tenant_id").(string)
	if !ok || tenantID == "" {
		return nil, errors.New("FATAL: missing tenant_id in context (Rule 7 Violation)")
	}

	makerID, ok := ctx.Value("user_id").(string)
	if !ok || makerID == "" {
		return nil, errors.New("FATAL: missing user_id in context")
	}

	proposalID := uuid.New().String()
	req := &CatalogChangeRequest{
		ID:            proposalID,
		ProposalID:    proposalID,
		TenantID:      tenantID,
		BOID:          boID,
		MakerUserID:   makerID,
		Status:        StatusPendingApproval,
		DiffPayload:   diffPayload,
		Justification: justification,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	s.requests[proposalID] = req
	return req, nil
}

// ReviewProposal approves or rejects a pending proposal enforcing the Four-Eyes principle (Maker != Checker)
func (s *MakerCheckerService) ReviewProposal(ctx context.Context, proposalID string, approve bool) error {
	tenantID, _ := ctx.Value("tenant_id").(string)
	checkerID, _ := ctx.Value("user_id").(string)

	if checkerID == "" {
		return errors.New("FATAL: missing checker user_id in context")
	}

	req, ok := s.requests[proposalID]
	if !ok || (tenantID != "" && req.TenantID != tenantID) {
		return errors.New("proposal not found or tenant mismatch")
	}

	if req.Status != StatusPendingApproval {
		return fmt.Errorf("proposal is not in PENDING_APPROVAL state, current state: %s", req.Status)
	}

	// 🛡️ The Core Four-Eyes Constraint
	if req.MakerUserID == checkerID {
		return errors.New("GOVERNANCE VIOLATION: The Maker and Checker cannot be the same user")
	}

	newStatus := StatusRejected
	if approve {
		newStatus = StatusActive
	}

	req.Status = newStatus
	req.CheckerUserID = checkerID
	req.UpdatedAt = time.Now()
	return nil
}

// ApproveChange approves and promotes a proposal to ACTIVE status enforcing Four-Eyes Principle (Checker role)
func (s *MakerCheckerService) ApproveChange(ctx context.Context, requestID, checkerUserID string) (*CatalogChangeRequest, error) {
	req, ok := s.requests[requestID]
	if !ok {
		return nil, fmt.Errorf("change request '%s' not found", requestID)
	}

	// Four-Eyes Principle: Maker cannot approve their own change request
	if req.MakerUserID == checkerUserID {
		return nil, fmt.Errorf("FOUR-EYES VIOLATION: maker user '%s' cannot approve their own change request", checkerUserID)
	}

	req.Status = StatusActive
	req.CheckerUserID = checkerUserID
	req.UpdatedAt = time.Now()
	return req, nil
}

// RejectChange rejects a proposal (Checker role)
func (s *MakerCheckerService) RejectChange(ctx context.Context, requestID, checkerUserID, reason string) (*CatalogChangeRequest, error) {
	req, ok := s.requests[requestID]
	if !ok {
		return nil, fmt.Errorf("change request '%s' not found", requestID)
	}

	req.Status = StatusRejected
	req.CheckerUserID = checkerUserID
	req.UpdatedAt = time.Now()
	return req, nil
}
