package approvals

import "time"

type ApprovalStatus string

const (
	StatusPending   ApprovalStatus = "Pending"
	StatusApproved  ApprovalStatus = "Approved"
	StatusRejected  ApprovalStatus = "Rejected"
	StatusDelegated ApprovalStatus = "Delegated"
)

// ApprovalRequest represents a request for a human to approve a process change or step.
type ApprovalRequest struct {
	ID            string         `json:"id"`
	ProcessID     string         `json:"process_id"`
	Version       string         `json:"version"`
	Stage         string         `json:"stage"` // e.g., "Compliance", "Risk"
	Status        ApprovalStatus `json:"status"`
	RequesterID   string         `json:"requester_id"`
	AssignedRoles []string       `json:"assigned_roles"`
	RiskLevel     string         `json:"risk_level"` // Low, Medium, High
	DueAt         time.Time      `json:"due_at"`
	CreatedAt     time.Time      `json:"created_at"`
}

// ApprovalDecision captures the action taken by an approver.
type ApprovalDecision struct {
	RequestID  string         `json:"request_id"`
	ActorID    string         `json:"actor_id"`
	Decision   ApprovalStatus `json:"decision"`
	Rationale  string         `json:"rationale"`
	PolicyRefs []string       `json:"policy_refs"`
	Timestamp  time.Time      `json:"timestamp"`
}
