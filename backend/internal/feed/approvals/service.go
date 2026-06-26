package approvals

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	feedaudit "github.com/hondyman/semlayer/backend/internal/feed/audit"
)

// Service manages approval requests (in-memory for Phase 3, will be DB in Phase 4)
type Service struct {
	mu          sync.RWMutex
	requests    map[string]*ApprovalRequest
	policies    map[string]*ApprovalPolicy
	auditRecorder feedaudit.AuditRecorder
}

func NewService() *Service {
	return &Service{
		requests: make(map[string]*ApprovalRequest),
		policies: getHardcodedPolicies(),
	}
}

// SetAuditRecorder sets the audit recorder for the service
func (s *Service) SetAuditRecorder(recorder feedaudit.AuditRecorder) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.auditRecorder = recorder
}

func (s *Service) CreateApprovalRequest(tenantID, clientID, actionType string, actionDetails map[string]interface{}, workflowID, runID string) (*ApprovalRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	req := &ApprovalRequest{
		ID:            uuid.New().String(),
		TenantID:      tenantID,
		ClientID:      clientID,
		ActionType:    actionType,
		ActionDetails: actionDetails,
		RequesterID:   "system", // In reality, would come from auth context
		Status:        "pending",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		WorkflowID:    workflowID,
		RunID:         runID,
	}

	s.requests[req.ID] = req
	return req, nil
}

func (s *Service) GetPendingApprovals(tenantID string) ([]*ApprovalRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var pending []*ApprovalRequest
	for _, req := range s.requests {
		if req.TenantID == tenantID && req.Status == "pending" {
			pending = append(pending, req)
		}
	}
	return pending, nil
}

func (s *Service) ApproveRequest(id, approverID, comment string) (*ApprovalRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	req, exists := s.requests[id]
	if !exists {
		return nil, fmt.Errorf("approval request not found: %s", id)
	}

	if req.Status != "pending" {
		return nil, fmt.Errorf("approval request already processed: %s", req.Status)
	}

	req.Status = "approved"
	req.ApproverID = approverID
	req.ApprovalComment = comment
	req.UpdatedAt = time.Now()

	// Record audit event
	if s.auditRecorder != nil {
		_ = s.auditRecorder.Record(
			req.WorkflowID,
			"approval_decided",
			"user:"+approverID,
			"approve_request",
			req.ClientID,
			map[string]interface{}{
				"approval_id": req.ID,
				"approved":    true,
				"comment":     comment,
				"approver_id": approverID,
			},
		)
	}

	return req, nil
}

func (s *Service) RejectRequest(id, approverID, comment string) (*ApprovalRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	req, exists := s.requests[id]
	if !exists {
		return nil, fmt.Errorf("approval request not found: %s", id)
	}

	if req.Status != "pending" {
		return nil, fmt.Errorf("approval request already processed: %s", req.Status)
	}

	req.Status = "rejected"
	req.ApproverID = approverID
	req.ApprovalComment = comment
	req.UpdatedAt = time.Now()

	// Record audit event
	if s.auditRecorder != nil {
		_ = s.auditRecorder.Record(
			req.WorkflowID,
			"approval_decided",
			"user:"+approverID,
			"reject_request",
			req.ClientID,
			map[string]interface{}{
				"approval_id": req.ID,
				"approved":    false,
				"comment":     comment,
				"approver_id": approverID,
			},
		)
	}

	return req, nil
}

func (s *Service) GetRequest(id string) (*ApprovalRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	req, exists := s.requests[id]
	if !exists {
		return nil, fmt.Errorf("approval request not found: %s", id)
	}
	return req, nil
}

func (s *Service) RequiresApproval(actionType string, value float64) bool {
	policy, exists := s.policies[actionType]
	if !exists {
		return true // Default to requiring approval if no policy
	}

	if !policy.RequiresApproval {
		return false
	}

	// Auto-approve if under threshold
	if value > 0 && value <= policy.MaxAutoApprove {
		return false
	}

	return true
}

func getHardcodedPolicies() map[string]*ApprovalPolicy {
	return map[string]*ApprovalPolicy{
		"tax_loss_harvest": {
			ActionType:       "tax_loss_harvest",
			RequiresApproval: true,
			ApproverRoles:    []string{"advisor", "portfolio_manager"},
			MaxAutoApprove:   10000, // Auto-approve trades under $10k
		},
		"rebalance": {
			ActionType:       "rebalance",
			RequiresApproval: true,
			ApproverRoles:    []string{"advisor", "portfolio_manager"},
			MaxAutoApprove:   25000, // Auto-approve rebalances under $25k
		},
	}
}
