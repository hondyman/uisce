package bp

import (
	"time"

	"github.com/lib/pq"
)

// BPDefinition represents a versioned business process definition.
type BPDefinition struct {
	ID          string    `json:"id" db:"id"`
	TenantID    string    `json:"tenant_id" db:"tenant_id"`
	Key         string    `json:"key" db:"key"`
	Version     int       `json:"version" db:"version"`
	Name        string    `json:"name" db:"name"`
	Entity      string    `json:"entity" db:"entity"`
	Status      string    `json:"status" db:"status"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	CreatedBy   string    `json:"created_by" db:"created_by"`
}

// BPStep represents a single step in a business process.
type BPStep struct {
	ID                    string         `json:"id" db:"id"`
	BPDefID               string         `json:"bp_def_id" db:"bp_def_id"`
	Seq                   int            `json:"seq" db:"seq"`
	StepKey               string         `json:"step_key" db:"step_key"`
	Type                  string         `json:"type" db:"type"`
	ActivityName          string         `json:"activity_name" db:"activity_name"`
	SignalName            string         `json:"signal_name" db:"signal_name"`
	Description           string         `json:"description" db:"description"`
	PreValidationRuleIDs  pq.StringArray `json:"pre_validation_rule_ids" db:"pre_validation_rule_ids"`
	PostValidationRuleIDs pq.StringArray `json:"post_validation_rule_ids" db:"post_validation_rule_ids"`
	ConditionExpr         string         `json:"condition_expr" db:"condition_expr"`
	ConditionExprType     string         `json:"condition_expr_type" db:"condition_expr_type"` // "json"
	CreatedAt             time.Time      `json:"created_at" db:"created_at"`

	// Advanced Features
	ApprovalChain *ApprovalChain   `json:"approval_chain,omitempty" db:"approval_chain"`
	RoutingRules  *RoutingRules    `json:"routing_rules,omitempty" db:"routing_rules"`
	Escalations   []EscalationStep `json:"escalations,omitempty" db:"escalations"`

	DelayExprType string `json:"delay_expr_type" db:"delay_expr_type"` // "hours"
	DelayExpr     string `json:"delay_expr,omitempty" db:"delay_expr"`

	SLAExprType string `json:"sla_expr_type" db:"sla_expr_type"` // "hours"
	SLAExpr     string `json:"sla_expr,omitempty" db:"sla_expr"`

	IntegrationConfig *IntegrationConfig `json:"integration_config,omitempty" db:"integration_config"`

	// Participants are hydrated separately usually
	Participants []BPStepParticipant `json:"participants,omitempty"`
}

// BPStepParticipant defines who can act on a step.
type BPStepParticipant struct {
	ID               string    `json:"id" db:"id"`
	StepID           string    `json:"step_id" db:"step_id"`
	RoleKey          string    `json:"role_key,omitempty" db:"role_key"` // Optional
	RuleID           string    `json:"rule_id,omitempty" db:"rule_id"`   // Optional: Rule to resolve dynamic users
	IncludeCondition string    `json:"include_condition,omitempty" db:"include_condition"`
	ExcludeInitiator bool      `json:"exclude_initiator" db:"exclude_initiator"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}

// -- Advanced Structs --

type ApprovalChain struct {
	Levels []ApprovalLevel `json:"levels"`
}

type ApprovalLevel struct {
	Name           string `json:"name"`
	ActorRole      string `json:"actorRole"` // or Rule
	EntryCondition string `json:"entryCondition"`
	ExitCondition  string `json:"exitCondition"`
	SkipIf         string `json:"skipIf"`
	StopCriteria   string `json:"stopCriteria"`
}

type RoutingRules struct {
	Routes       []RoutingRule `json:"routes"`
	FallbackRole string        `json:"fallbackRole"`
}

type RoutingRule struct {
	Condition string `json:"condition"`
	ActorRole string `json:"actorRole"`
}

type IntegrationConfig struct {
	TargetSystem string                 `json:"targetSystem"`
	Endpoint     string                 `json:"endpoint"`
	Method       string                 `json:"method"`
	PayloadMap   map[string]interface{} `json:"payloadMap"`
}

// -- Execution Models --

type BPExecution struct {
	ID          string     `json:"id" db:"id"`
	TenantID    string     `json:"tenant_id" db:"tenant_id"`
	BPDefID     string     `json:"bp_def_id" db:"bp_def_id"`
	BPRunID     string     `json:"bp_run_id" db:"bp_run_id"`
	Entity      string     `json:"entity" db:"entity"`
	EntityID    string     `json:"entity_id" db:"entity_id"`
	Status      string     `json:"status" db:"status"`
	StartedAt   time.Time  `json:"started_at" db:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	InitiatedBy string     `json:"initiated_by" db:"initiated_by"`
}

type BPStepExecution struct {
	ID                string                 `json:"id" db:"id"`
	BPExecID          string                 `json:"bp_exec_id" db:"bp_exec_id"`
	StepID            string                 `json:"step_id" db:"step_id"`
	StepKey           string                 `json:"step_key" db:"step_key"`
	Status            string                 `json:"status" db:"status"`
	StartedAt         *time.Time             `json:"started_at,omitempty" db:"started_at"`
	CompletedAt       *time.Time             `json:"completed_at,omitempty" db:"completed_at"`
	Actor             string                 `json:"actor,omitempty" db:"actor"`
	RoutingInfo       map[string]interface{} `json:"routing_info,omitempty" db:"routing_info"`
	ValidationResults map[string]interface{} `json:"validation_results,omitempty" db:"validation_results"`
}

// BPTask represents a pending action for a user (Inbox Item).
type BPTask struct {
	ID           string                 `json:"id" db:"id"`
	TenantID     string                 `json:"tenant_id" db:"tenant_id"`
	BPRunID      string                 `json:"bp_run_id" db:"bp_run_id"`
	StepID       string                 `json:"step_id" db:"step_id"`
	Status       string                 `json:"status" db:"status"`
	AssigneeID   string                 `json:"assignee_id" db:"assignee_id"`
	AssigneeRole string                 `json:"assignee_role" db:"assignee_role"`
	DueDate      time.Time              `json:"due_date" db:"due_date"`
	DataPayload  map[string]interface{} `json:"data_payload" db:"data_payload"`
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty" db:"completed_at"`
}

// BPEvent represents an audit log entry.
type BPEvent struct {
	ID        string                 `json:"id" db:"id"`
	TenantID  string                 `json:"tenant_id" db:"tenant_id"`
	BPRunID   string                 `json:"bp_run_id" db:"bp_run_id"`
	StepKey   string                 `json:"step_key" db:"step_key"`
	EventType string                 `json:"event_type" db:"event_type"`
	Details   map[string]interface{} `json:"details" db:"details"` // JSONB
	CreatedAt time.Time              `json:"created_at" db:"created_at"`
}

// WorkflowContext carries runtime initialization data for the BP engine.
type WorkflowContext struct {
	TenantID  string
	BpKey     string
	BpVersion int
	Entity    string
	EntityID  string
	Initiator string

	// Data payload for initiation (optional)
	InputData map[string]interface{}
}
