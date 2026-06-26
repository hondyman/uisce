package approvals

import (
	"time"
)

// ApprovalRequest represents a pending approval for a high-risk action
type ApprovalRequest struct {
	ID              string
	TenantID        string
	ClientID        string
	ActionType      string // "tax_loss_harvest", "rebalance", etc.
	ActionDetails   map[string]interface{}
	RequesterID     string
	Status          string // "pending", "approved", "rejected"
	ApproverID      string
	ApprovalComment string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	WorkflowID      string // Temporal workflow ID
	RunID           string // Temporal run ID
}

// ApprovalPolicy defines which actions require approval
type ApprovalPolicy struct {
	ActionType       string
	RequiresApproval bool
	ApproverRoles    []string
	MaxAutoApprove   float64 // Max dollar value for auto-approval
}

// ApprovalDecision represents the outcome of an approval
type ApprovalDecision struct {
	Approved bool
	Comment  string
	ApproverID string
}
