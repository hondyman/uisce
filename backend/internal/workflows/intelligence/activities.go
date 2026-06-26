package intelligence

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/audit"
	"github.com/hondyman/semlayer/backend/internal/guardrails"
	"github.com/hondyman/semlayer/backend/pkg/multitenancy"
)

type LogEventInput struct {
	TenantID   string                 `json:"tenant_id"` // Added for isolation
	RunID      uuid.UUID              `json:"run_id"`
	Seq        int64                  `json:"seq"`
	EventType  string                 `json:"event_type"`
	Payload    map[string]interface{} `json:"payload"`
	ParentHash string                 `json:"parent_hash"`
}

type IntelligenceActivities struct {
	logger *audit.EventLogger
	tm     *multitenancy.TenantManager
}

func NewIntelligenceActivities(logger *audit.EventLogger, tm *multitenancy.TenantManager) *IntelligenceActivities {
	return &IntelligenceActivities{logger: logger, tm: tm}
}

func (a *IntelligenceActivities) LogEventActivity(ctx context.Context, input LogEventInput) (string, error) {
	// Resolve Tenant DB
	db, err := a.tm.GetDB(input.TenantID)
	if err != nil {
		return "", err
	}

	// Create a scoped logger for this tenant DB
	// Note: EventLogger is lightweight, so creating it here is fine.
	// Alternatively, EventLogger could accept DB in LogEvent.
	tenantLogger := audit.NewEventLogger(db)

	evt := audit.Event{
		EventID:    uuid.New(),
		RunID:      input.RunID,
		Seq:        input.Seq,
		EventType:  input.EventType,
		Payload:    input.Payload,
		ParentHash: input.ParentHash,
		Timestamp:  time.Now(),
	}

	return tenantLogger.LogEvent(ctx, evt)
}

func (a *IntelligenceActivities) EvaluateGuardrailsActivity(ctx context.Context, content string) (guardrails.Outcome, error) {
	// In a real app, allowedTopics might come from config or context
	return guardrails.Evaluate(content, []string{}), nil
}

// StartAdviceSession initializes the workflow run in the structured DB and logs the event
func (a *IntelligenceActivities) StartAdviceSession(ctx context.Context, input LogEventInput, clientID, objective, policyVersion string) (string, error) {
	// Resolve Tenant DB
	db, err := a.tm.GetDB(input.TenantID)
	if err != nil {
		return "", err
	}

	// 1. Structured Write
	// TODO: Replace SQL with Hasura GraphQL mutation:
	// mutation InsertWorkflowRun($object: workflow_runs_insert_input!) {
	//   insert_workflow_runs_one(object: $object) {
	//     run_id
	//     status
	//   }
	// }
	// Variables: {"object": {"run_id": "...", "client_id": "...", "objective": "...",
	//   "policy_version": "...", "status": "initiated"}}
	// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
	_, err = db.ExecContext(ctx, `
		INSERT INTO workflow_runs (run_id, client_id, objective, policy_version, status)
		VALUES ($1, $2, $3, $4, 'initiated')
	`, input.RunID, clientID, objective, policyVersion)
	if err != nil {
		return "", err
	}

	// 2. Immutable Log
	return a.LogEventActivity(ctx, input)
}

// RecordGuardrailOutcome records hits in the structured DB and logs the event
func (a *IntelligenceActivities) RecordGuardrailOutcome(ctx context.Context, input LogEventInput, outcome guardrails.Outcome) (string, error) {
	// Resolve Tenant DB
	db, err := a.tm.GetDB(input.TenantID)
	if err != nil {
		return "", err
	}

	// 1. Structured Write (Hits)
	// In a real app, we'd iterate and insert each hit. For now, we'll just update status.
	status := "review"
	if !outcome.RequiresHuman {
		status = "approved" // Auto-approved if no hits
	}

	// TODO: Replace SQL with Hasura GraphQL mutation:
	// mutation UpdateWorkflowRunStatus($runId: uuid!, $status: String!) {
	//   update_workflow_runs_by_pk(pk_columns: {run_id: $runId}, _set: {status: $status}) {
	//     run_id
	//     status
	//   }
	// }
	// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
	_, err = db.ExecContext(ctx, `
		UPDATE workflow_runs SET status = $1 WHERE run_id = $2
	`, status, input.RunID)
	if err != nil {
		return "", err
	}

	// 2. Immutable Log
	return a.LogEventActivity(ctx, input)
}

// FetchEntityData, CheckDrift, RenderPageAsImage would also be here

// Package-level activity references for Temporal workflow registration.
// These are used as activity identifiers in workflow definitions.
// The actual implementations are methods on IntelligenceActivities struct.
var (
	LogEventActivity           = "LogEventActivity"
	EvaluateGuardrailsActivity = "EvaluateGuardrailsActivity"
)
