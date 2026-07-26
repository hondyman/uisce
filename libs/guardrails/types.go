package guardrails

import (
	"context"
	"time"
)

type Auditor interface {
	LogEvent(ctx context.Context, event AuditEvent) error
}

type AuditEvent struct {
	AuditID       string
	EventType     string
	Version       string
	TenantID      string
	ActorID       string
	Timestamp     time.Time
	ObjectType    string
	ObjectID      string
	PayloadDigest string
	Narrative     string
	Status        string
	Action        string
	Metadata      map[string]interface{}
}

type GuardrailResult struct {
	Approved           bool              `json:"approved"`
	RedactedContent    string            `json:"redacted_content"`
	ViolationsDetected []PolicyViolation `json:"violations_detected"`
	ExecutionTimeMs    int64             `json:"execution_time_ms"`
	PolicyVersionsUsed []string          `json:"policy_versions_used"`
	AuditEventID       string            `json:"audit_event_id"`
}

type PolicyViolation struct {
	PolicyID    string `json:"policy_id"`
	PolicyName  string `json:"policy_name"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Location    string `json:"location"`
	Remediation string `json:"remediation"`
}

type AuditEventResult struct {
	EventID string
}

type PolicyRegistry struct {
	policies map[string]Policy
}

type Policy struct {
	ID          string
	Name        string
	Description string
	Version     string
	Active      bool
}

func NewPolicyRegistry() *PolicyRegistry {
	return &PolicyRegistry{
		policies: make(map[string]Policy),
	}
}
