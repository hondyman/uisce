package governance

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/ai/orchestration"
)

// Phase 3: Governance & Controls

// MetricLifecycleState represents the approval state of a metric or calculation
type MetricLifecycleState string

const (
	StateDraft      MetricLifecycleState = "draft"
	StateReview     MetricLifecycleState = "review"
	StateApproved   MetricLifecycleState = "approved"
	StateDeprecated MetricLifecycleState = "deprecated"
)

// Version represents a version of a catalog entity
type Version struct {
	ID            string                 `json:"id"`
	EntityID      string                 `json:"entity_id"`
	VersionNumber int                    `json:"version_number"`
	State         MetricLifecycleState   `json:"state"`
	EffectiveDate time.Time              `json:"effective_date"`
	ApprovedBy    string                 `json:"approved_by,omitempty"`
	ApprovedAt    *time.Time             `json:"approved_at,omitempty"`
	DeprecatedAt  *time.Time             `json:"deprecated_at,omitempty"`
	ChangeLog     string                 `json:"change_log,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// Provenance tracks the lineage and history of data
type Provenance struct {
	QualifiedPath  string                 `json:"qualified_path"`
	Version        int                    `json:"version"`
	EffectiveDate  time.Time              `json:"effective_date"`
	LastModified   time.Time              `json:"last_modified"`
	ModifiedBy     string                 `json:"modified_by"`
	ApprovalStatus MetricLifecycleState   `json:"approval_status"`
	Lineage        []string               `json:"lineage"` // Upstream dependencies
	DataQuality    map[string]interface{} `json:"data_quality,omitempty"`
	SLA            map[string]interface{} `json:"sla,omitempty"`
}

// PolicyScope defines access control boundaries
type PolicyScope struct {
	TenantID      string   `json:"tenant_id"`
	PortfolioIDs  []string `json:"portfolio_ids,omitempty"`
	StrategyIDs   []string `json:"strategy_ids,omitempty"`
	Regions       []string `json:"regions,omitempty"`
	AssetClasses  []string `json:"asset_classes,omitempty"`
	SecurityLevel string   `json:"security_level"` // public, internal, confidential, restricted
}

// AccessPolicy defines who can access what
type AccessPolicy struct {
	ID           string                 `json:"id"`
	Scope        PolicyScope            `json:"scope"`
	AllowedRoles []string               `json:"allowed_roles"`
	DeniedRoles  []string               `json:"denied_roles,omitempty"`
	Conditions   map[string]interface{} `json:"conditions,omitempty"`
}

// ChangeRequest represents a proposed change to a metric or calculation
type ChangeRequest struct {
	ID              string                 `json:"id"`
	EntityID        string                 `json:"entity_id"`
	CurrentVersion  int                    `json:"current_version"`
	ProposedChanges map[string]interface{} `json:"proposed_changes"`
	Justification   string                 `json:"justification"`
	RequestedBy     string                 `json:"requested_by"`
	RequestedAt     time.Time              `json:"requested_at"`
	ReviewedBy      []string               `json:"reviewed_by,omitempty"`
	ApprovedBy      string                 `json:"approved_by,omitempty"`
	State           MetricLifecycleState   `json:"state"`
	ImpactAnalysis  string                 `json:"impact_analysis,omitempty"`
}

// AuditEvent tracks all significant events for compliance
type AuditEvent struct {
	ID         string                 `json:"id"`
	Timestamp  time.Time              `json:"timestamp"`
	TenantID   string                 `json:"tenant_id"`
	UserID     string                 `json:"user_id"`
	Action     string                 `json:"action"` // query, modify, approve, etc.
	EntityType string                 `json:"entity_type"`
	EntityID   string                 `json:"entity_id"`
	Details    map[string]interface{} `json:"details,omitempty"`
	IPAddress  string                 `json:"ip_address,omitempty"`
	UserAgent  string                 `json:"user_agent,omitempty"`
}

// GovernanceService manages governance policies and controls
type GovernanceService struct {
	aiOrchestrator *orchestration.AIOrchestrator
}

func NewGovernanceService(ai *orchestration.AIOrchestrator) *GovernanceService {
	return &GovernanceService{
		aiOrchestrator: ai,
	}
}

// ValidateAccess checks if a user has access to an entity based on policy
func (s *GovernanceService) ValidateAccess(userID string, entityPath string, scope PolicyScope) (bool, error) {
	// TODO: Implement actual policy evaluation
	// 1. Fetch user roles
	// 2. Fetch applicable policies for entity
	// 3. Evaluate scope constraints
	// 4. Check allow/deny rules
	return true, nil
}

// GetProvenance returns the provenance information for an entity
func (s *GovernanceService) GetProvenance(entityPath string, tenantID string) (*Provenance, error) {
	// TODO: Implement actual provenance retrieval from catalog
	return &Provenance{
		QualifiedPath:  entityPath,
		Version:        1,
		EffectiveDate:  time.Now().AddDate(0, -1, 0),
		LastModified:   time.Now(),
		ApprovalStatus: StateApproved,
	}, nil
}

// ChangeSet represents a discrete change to the system
type ChangeSet struct {
	ID                 string          `json:"id"`
	Type               string          `json:"type"`
	Scope              string          `json:"scope"`   // e.g. "urn:tenant:123"
	Payload            json.RawMessage `json:"payload"` // The actual change content
	CreatedBy          string          `json:"created_by"`
	CreatedAt          time.Time       `json:"created_at"`
	Status             string          `json:"status"` // PENDING, APPROVED, REJECTED
	Title              string          `json:"title,omitempty"`
	Summary            string          `json:"summary,omitempty"`
	RiskScore          float64         `json:"risk_score,omitempty"`
	RiskLevel          string          `json:"risk_level,omitempty"`
	SuggestedReviewers []string        `json:"suggested_reviewers,omitempty"`
}

type ChangeSetInput struct {
	Type             string          `json:"type"`
	Scope            string          `json:"scope"`
	Payload          json.RawMessage `json:"payload"`
	Actor            string          `json:"actor"`
	ObjectType       string          `json:"object_type"`
	OldValue         json.RawMessage `json:"old_value"`
	NewValue         json.RawMessage `json:"new_value"`
	SemanticImpact   json.RawMessage `json:"semantic_impact"`
	ComplianceImpact json.RawMessage `json:"compliance_impact"`
}

// CreateChangeSet creates a new change set and triggers AI enrichment
func (s *GovernanceService) CreateChangeSet(ctx context.Context, input ChangeSetInput) (*ChangeSet, error) {
	cs := &ChangeSet{
		ID:        uuid.New().String(),
		Type:      input.Type,
		Scope:     input.Scope,
		Payload:   input.Payload,
		CreatedBy: input.Actor,
		CreatedAt: time.Now(),
		Status:    "PENDING",
	}

	// 1. Save to DB (mock for now)
	// s.repo.Save(cs)

	// 2. Trigger AI Orchestration
	if s.aiOrchestrator != nil {
		// Construct specific payload for the ChangeSetStrategy
		aiPayload := map[string]interface{}{
			"change_set_id":     cs.ID, // Add correlation ID
			"object_type":       input.ObjectType,
			"old_value":         input.OldValue,
			"new_value":         input.NewValue,
			"semantic_impact":   input.SemanticImpact,
			"compliance_impact": input.ComplianceImpact,
		}

		// Enqueue the request (fire and forget from user perspective, or track ID)
		_, err := s.aiOrchestrator.Enqueue(ctx, orchestration.TypeChangeSet, aiPayload)
		if err != nil {
			// Log error but don't fail the transaction?
			// For strict governance, maybe we SHOULD fail?
			// For now, let's log and proceed.
			// logger.Error("Failed to enqueue AI analysis", "error", err)
		}
	}

	return cs, nil
}
