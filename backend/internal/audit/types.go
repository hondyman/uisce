package audit

import (
	"context"
	"time"
)

// UnifiedAuditRecord (UAR) represents a single, immutable audit entry
// that spans business operations, approvals, accounting, compliance, and AI outputs.
type UnifiedAuditRecord struct {
	// Event Identity
	AuditID   string    `json:"audit_id"`
	EventType string    `json:"event_type"`
	Version   string    `json:"version"`
	TenantID  string    `json:"tenant_id"`
	ActorID   string    `json:"actor_id"`
	Roles     []string  `json:"roles"`
	Timestamp time.Time `json:"timestamp"`

	// Causality
	ObjectType       string `json:"object_type"`
	ObjectID         string `json:"object_id"`
	PrevEventID      string `json:"prev_event_id"`      // Link to previous event for this object
	CorrelationID    string `json:"correlation_id"`     // Workflow or Transaction ID
	ParentWorkflowID string `json:"parent_workflow_id"` // If nested

	// Content
	PayloadDigest  string            `json:"payload_digest"`  // SHA-256 of the payload
	PayloadPointer string            `json:"payload_pointer"` // URL/Path to full payload in object store
	Narrative      string            `json:"narrative"`       // Human-readable summary
	PolicyRefs     []string          `json:"policy_refs"`     // IDs of policies checked/enforced
	DataQuality    map[string]string `json:"data_quality"`    // Metadata about data freshness/accuracy

	// Cryptography
	PrevHash  string `json:"prev_hash"` // Hash of the previous record in the chain
	Hash      string `json:"hash"`      // Hash of this record (including PrevHash)
	Signature string `json:"signature"` // Digital signature of Hash
	KeyID     string `json:"key_id"`    // ID of the key used to sign

	// Outcome
	Status        string                 `json:"status"` // Success, Failed, Escalated
	ErrorCode     string                 `json:"error_code,omitempty"`
	RemediationID string                 `json:"remediation_id,omitempty"`
	Action        string                 `json:"action,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// Auditor defines the interface for logging audit events.
type Auditor interface {
	LogEvent(ctx context.Context, event UnifiedAuditRecord) error
	VerifyChain(ctx context.Context, objectID string) (bool, error)
}
