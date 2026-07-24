package business_process

import (

)

// ProcessTemplate defines the structure of a business process (Graph-based).
type ProcessTemplate struct {
	ProcessID  string       `json:"id"`
	Name       string       `json:"name"`
	ObjectType string       `json:"object_type"`
	Version    string       `json:"version"`
	Status     string       `json:"status"` // Draft, Active, Deprecated, Archived
	Steps      []Step       `json:"steps"`
	Transitions []Transition `json:"transitions"`
	Audit      AuditConfig  `json:"audit"`
}

// StepType defines the kind of step.
type StepType string

const (
	StepTypeActivity StepType = "activity"
	StepTypeApproval StepType = "approval"
	StepTypeEvent    StepType = "event"
	StepTypeDecision StepType = "decision"
)

// Step represents a node in the process graph.
type Step struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Type        StepType `json:"type"`
	ActivityRef string   `json:"activity_ref,omitempty"` // For StepTypeActivity
	Conditions  []string `json:"conditions,omitempty"`   // Pre-conditions
	Roles       []string `json:"roles,omitempty"`        // For StepTypeApproval
	SLA         string   `json:"sla,omitempty"`          // ISO 8601 Duration (e.g., "PT2H")
}

// Transition defines a directed edge between steps.
type Transition struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// AuditConfig defines the audit requirements for the process.
type AuditConfig struct {
	HashChain  bool     `json:"hash_chain"`
	PolicyRefs []string `json:"policy_refs"`
}

// BusinessObject is the interface that all domain objects must implement to flow through the process.
type BusinessObject interface {
	GetID() string
	GetType() string
	GetTenantID() string
	GetState() string
	SetState(state string)
	GetData() map[string]interface{}
}

// GenericBusinessObject is a flexible implementation of BusinessObject.
type GenericBusinessObject struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	TenantID string                 `json:"tenant_id"`
	State    string                 `json:"state"`
	Data     map[string]interface{} `json:"data"`
}

func (o *GenericBusinessObject) GetID() string {
	return o.ID
}

func (o *GenericBusinessObject) GetType() string {
	return o.Type
}

func (o *GenericBusinessObject) GetTenantID() string {
	return o.TenantID
}

func (o *GenericBusinessObject) GetState() string {
	return o.State
}

func (o *GenericBusinessObject) SetState(state string) {
	o.State = state
}

func (o *GenericBusinessObject) GetData() map[string]interface{} {
	return o.Data
}
