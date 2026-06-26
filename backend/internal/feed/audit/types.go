package audit

import (
	"time"
)

// AuditRecord represents a single immutable audit entry
type AuditRecord struct {
	TraceID        string                 `json:"trace_id"`
	Timestamp      time.Time              `json:"timestamp"`
	EventType      string                 `json:"event_type"` // "context_loaded", "rule_evaluated", "workflow_started", "approval_created", "approval_decided", "trade_executed"
	Actor          string                 `json:"actor"`      // "system", "user:advisor_id", "workflow:workflow_id"
	Action         string                 `json:"action"`
	Target         string                 `json:"target"` // client_id, workflow_id, etc.
	LogicSnapshot  map[string]interface{} `json:"logic_snapshot"`
	PreviousHash   string                 `json:"previous_hash"`
	CurrentHash    string                 `json:"current_hash"`
	SequenceNumber int                    `json:"sequence_number"`
}

// EvidenceBundle groups all audit records for a single action
type EvidenceBundle struct {
	ActionID      string         `json:"action_id"`
	TraceID       string         `json:"trace_id"`
	ClientID      string         `json:"client_id"`
	ActionType    string         `json:"action_type"`
	StartTime     time.Time      `json:"start_time"`
	EndTime       time.Time      `json:"end_time"`
	Status        string         `json:"status"` // "pending", "approved", "rejected", "executed", "failed"
	AuditRecords  []AuditRecord  `json:"audit_records"`
	HashChainValid bool          `json:"hash_chain_valid"`
}

// AuditRecorder defines the interface for recording audit events
type AuditRecorder interface {
	Record(traceID, eventType, actor, action, target string, logicSnapshot map[string]interface{}) error
	GetTrail(traceID string) ([]AuditRecord, error)
	GetEvidenceBundle(actionID string) (*EvidenceBundle, error)
	VerifyHashChain(records []AuditRecord) bool
}
