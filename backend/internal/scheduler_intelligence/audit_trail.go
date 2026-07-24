package scheduler_intelligence

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// AuditEventType defines types of audit events
type AuditEventType string

const (
	AuditEventJobCreated       AuditEventType = "job.created"
	AuditEventJobUpdated       AuditEventType = "job.updated"
	AuditEventJobDeleted       AuditEventType = "job.deleted"
	AuditEventJobTriggered     AuditEventType = "job.triggered"
	AuditEventJobPaused        AuditEventType = "job.paused"
	AuditEventJobResumed       AuditEventType = "job.resumed"
	AuditEventDAGCreated       AuditEventType = "dag.created"
	AuditEventDAGUpdated       AuditEventType = "dag.updated"
	AuditEventDAGDeleted       AuditEventType = "dag.deleted"
	AuditEventDAGTriggered     AuditEventType = "dag.triggered"
	AuditEventRunStarted       AuditEventType = "run.started"
	AuditEventRunCompleted     AuditEventType = "run.completed"
	AuditEventRunFailed        AuditEventType = "run.failed"
	AuditEventPolicyCreated    AuditEventType = "policy.created"
	AuditEventPolicyUpdated    AuditEventType = "policy.updated"
	AuditEventChangeApproved   AuditEventType = "change.approved"
	AuditEventChangeRejected   AuditEventType = "change.rejected"
	AuditEventChangeApplied    AuditEventType = "change.applied"
	AuditEventChangeRolledBack AuditEventType = "change.rolled_back"
)

// AuditRecord represents a scheduler audit log entry
type AuditRecord struct {
	ID            uuid.UUID       `json:"id" db:"id"`
	TenantID      uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	EventType     AuditEventType  `json:"event_type" db:"event_type"`
	TargetType    string          `json:"target_type" db:"target_type"` // job, dag, run, policy
	TargetID      uuid.UUID       `json:"target_id" db:"target_id"`
	TargetName    string          `json:"target_name" db:"target_name"`
	ActorID       string          `json:"actor_id" db:"actor_id"`
	ActorType     string          `json:"actor_type" db:"actor_type"` // user, system, ai
	Action        string          `json:"action" db:"action"`
	Details       json.RawMessage `json:"details,omitempty" db:"details"`
	PreviousState json.RawMessage `json:"previous_state,omitempty" db:"previous_state"`
	NewState      json.RawMessage `json:"new_state,omitempty" db:"new_state"`
	ChangeSetID   *uuid.UUID      `json:"changeset_id,omitempty" db:"changeset_id"`
	IPAddress     string          `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent     string          `json:"user_agent,omitempty" db:"user_agent"`
	Timestamp     time.Time       `json:"timestamp" db:"timestamp"`

	// Bitemporal fields
	ValidFrom  time.Time  `json:"valid_from" db:"valid_from"`
	ValidTo    *time.Time `json:"valid_to,omitempty" db:"valid_to"`
	SystemFrom time.Time  `json:"system_from" db:"system_from"`
	SystemTo   *time.Time `json:"system_to,omitempty" db:"system_to"`
}

// AuditTrailService manages audit logging
type AuditTrailService struct {
	repo *Repository
}

// NewAuditTrailService creates a new audit trail service
func NewAuditTrailService(repo *Repository) *AuditTrailService {
	return &AuditTrailService{
		repo: repo,
	}
}

// LogEvent records an audit event
func (a *AuditTrailService) LogEvent(ctx context.Context, record *AuditRecord) error {
	record.ID = uuid.New()
	record.Timestamp = time.Now()
	record.ValidFrom = time.Now()
	record.SystemFrom = time.Now()

	// Would insert into audit table
	return nil
}

// LogJobEvent logs a job-related event
func (a *AuditTrailService) LogJobEvent(
	ctx context.Context,
	eventType AuditEventType,
	tenantID, jobID uuid.UUID,
	jobName, actorID, actorType string,
	previousState, newState interface{},
	changeSetID *uuid.UUID,
) error {
	var prevJSON, newJSON json.RawMessage
	var err error

	if previousState != nil {
		prevJSON, err = json.Marshal(previousState)
		if err != nil {
			return fmt.Errorf("marshal previous state: %w", err)
		}
	}

	if newState != nil {
		newJSON, err = json.Marshal(newState)
		if err != nil {
			return fmt.Errorf("marshal new state: %w", err)
		}
	}

	record := &AuditRecord{
		TenantID:      tenantID,
		EventType:     eventType,
		TargetType:    "job",
		TargetID:      jobID,
		TargetName:    jobName,
		ActorID:       actorID,
		ActorType:     actorType,
		Action:        string(eventType),
		PreviousState: prevJSON,
		NewState:      newJSON,
		ChangeSetID:   changeSetID,
	}

	return a.LogEvent(ctx, record)
}

// LogDAGEvent logs a DAG-related event
func (a *AuditTrailService) LogDAGEvent(
	ctx context.Context,
	eventType AuditEventType,
	tenantID, dagID uuid.UUID,
	dagName, actorID, actorType string,
	previousState, newState interface{},
	changeSetID *uuid.UUID,
) error {
	var prevJSON, newJSON json.RawMessage
	var err error

	if previousState != nil {
		prevJSON, err = json.Marshal(previousState)
		if err != nil {
			return fmt.Errorf("marshal previous state: %w", err)
		}
	}

	if newState != nil {
		newJSON, err = json.Marshal(newState)
		if err != nil {
			return fmt.Errorf("marshal new state: %w", err)
		}
	}

	record := &AuditRecord{
		TenantID:      tenantID,
		EventType:     eventType,
		TargetType:    "dag",
		TargetID:      dagID,
		TargetName:    dagName,
		ActorID:       actorID,
		ActorType:     actorType,
		Action:        string(eventType),
		PreviousState: prevJSON,
		NewState:      newJSON,
		ChangeSetID:   changeSetID,
	}

	return a.LogEvent(ctx, record)
}

// LogRunEvent logs a run-related event
func (a *AuditTrailService) LogRunEvent(
	ctx context.Context,
	eventType AuditEventType,
	tenantID, runID, jobID uuid.UUID,
	details interface{},
) error {
	var detailsJSON json.RawMessage
	var err error

	if details != nil {
		detailsJSON, err = json.Marshal(details)
		if err != nil {
			return fmt.Errorf("marshal details: %w", err)
		}
	}

	record := &AuditRecord{
		TenantID:   tenantID,
		EventType:  eventType,
		TargetType: "run",
		TargetID:   runID,
		ActorID:    "system",
		ActorType:  "system",
		Action:     string(eventType),
		Details:    detailsJSON,
	}

	return a.LogEvent(ctx, record)
}

// AuditQuery defines query parameters for audit log retrieval
type AuditQuery struct {
	TenantID   *uuid.UUID
	TargetType string
	TargetID   *uuid.UUID
	EventTypes []AuditEventType
	ActorID    string
	FromTime   *time.Time
	ToTime     *time.Time
	Limit      int
	Offset     int
}

// GetAuditHistory retrieves audit records
func (a *AuditTrailService) GetAuditHistory(ctx context.Context, query AuditQuery) ([]AuditRecord, int, error) {
	// Would query database with filters
	return []AuditRecord{}, 0, nil
}

// GetEntityTimeline returns the full history of an entity
func (a *AuditTrailService) GetEntityTimeline(ctx context.Context, targetType string, targetID uuid.UUID) ([]AuditRecord, error) {
	query := AuditQuery{
		TargetType: targetType,
		TargetID:   &targetID,
		Limit:      100,
	}
	records, _, err := a.GetAuditHistory(ctx, query)
	return records, err
}

// GetRecentActivityForTenant returns recent audit activity
func (a *AuditTrailService) GetRecentActivityForTenant(ctx context.Context, tenantID uuid.UUID, limit int) ([]AuditRecord, error) {
	query := AuditQuery{
		TenantID: &tenantID,
		Limit:    limit,
	}
	records, _, err := a.GetAuditHistory(ctx, query)
	return records, err
}

// GetChangeSetAuditTrail returns all events for a change set
func (a *AuditTrailService) GetChangeSetAuditTrail(ctx context.Context, changeSetID uuid.UUID) ([]AuditRecord, error) {
	// Would filter by changeset_id
	return []AuditRecord{}, nil
}

// AuditStats provides aggregate audit statistics
type AuditStats struct {
	TotalEvents      int            `json:"total_events"`
	EventsByType     map[string]int `json:"events_by_type"`
	EventsByActor    map[string]int `json:"events_by_actor"`
	FailedRuns       int            `json:"failed_runs"`
	SuccessfulRuns   int            `json:"successful_runs"`
	PolicyViolations int            `json:"policy_violations"`
	TimeRange        struct {
		From time.Time `json:"from"`
		To   time.Time `json:"to"`
	} `json:"time_range"`
}

// GetAuditStats returns aggregate statistics
func (a *AuditTrailService) GetAuditStats(ctx context.Context, tenantID uuid.UUID, from, to time.Time) (*AuditStats, error) {
	stats := &AuditStats{
		EventsByType:  make(map[string]int),
		EventsByActor: make(map[string]int),
	}
	stats.TimeRange.From = from
	stats.TimeRange.To = to

	// Would aggregate from database
	return stats, nil
}
