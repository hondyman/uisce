package workflow

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"go.temporal.io/sdk/client"
)

// ReplayService provides workflow replay capabilities for regulatory compliance
type ReplayService struct {
	db             *sqlx.DB
	temporalClient client.Client
}

// NewReplayService creates a new workflow replay service
func NewReplayService(db *sqlx.DB, temporalClient client.Client) *ReplayService {
	return &ReplayService{
		db:             db,
		temporalClient: temporalClient,
	}
}

// WorkflowExecution represents a complete workflow execution with all events
type WorkflowExecution struct {
	WorkflowID      string                 `json:"workflow_id"`
	RunID           string                 `json:"run_id"`
	WorkflowType    string                 `json:"workflow_type"`
	StartTime       time.Time              `json:"start_time"`
	CloseTime       *time.Time             `json:"close_time,omitempty"`
	Status          string                 `json:"status"`
	ExecutionTime   int64                  `json:"execution_time_ms"`
	Events          []WorkflowEvent        `json:"events"`
	Inputs          map[string]interface{} `json:"inputs"`
	Result          map[string]interface{} `json:"result,omitempty"`
	AIModelVersions []string               `json:"ai_model_versions"`
	PolicyVersions  []string               `json:"policy_versions"`
}

// WorkflowEvent represents a single event in workflow history
type WorkflowEvent struct {
	EventID      int64                  `json:"event_id"`
	EventType    string                 `json:"event_type"`
	Timestamp    time.Time              `json:"timestamp"`
	Attributes   map[string]interface{} `json:"attributes"`
	ActorID      string                 `json:"actor_id,omitempty"`      // Human or system
	DecisionMade string                 `json:"decision_made,omitempty"` // For AI decisions
}

// ReplayWorkflow reconstructs the complete execution of a workflow
func (s *ReplayService) ReplayWorkflow(ctx context.Context, workflowID string, runID string) (*WorkflowExecution, error) {
	// Get workflow history from Temporal
	workflowRun := s.temporalClient.GetWorkflow(ctx, workflowID, runID)

	var workflowResult interface{}
	err := workflowRun.Get(ctx, &workflowResult)
	if err != nil && err.Error() != "workflow execution still running" {
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	// Get workflow description
	describe, err := s.temporalClient.DescribeWorkflowExecution(ctx, workflowID, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to describe workflow: %w", err)
	}

	// Get complete event history
	iter := s.temporalClient.GetWorkflowHistory(ctx, workflowID, runID, false, 0)

	events := []WorkflowEvent{}
	for iter.HasNext() {
		event, err := iter.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to get next event: %w", err)
		}

		// Convert Temporal event to our format
		workflowEvent := WorkflowEvent{
			EventID:   event.GetEventId(),
			EventType: event.GetEventType().String(),
			Timestamp: event.GetEventTime().AsTime(),
		}

		// Extract attributes based on event type
		attributes := make(map[string]interface{})

		// Parse event-specific attributes
		switch event.GetEventType().String() {
		case "WorkflowExecutionStarted":
			attrs := event.GetWorkflowExecutionStartedEventAttributes()
			if attrs != nil {
				if attrs.Input != nil && len(attrs.Input.Payloads) > 0 {
					attributes["input"] = attrs.Input.String()
				}
				attributes["workflow_type"] = attrs.WorkflowType.GetName()
			}
		case "ActivityTaskScheduled":
			attrs := event.GetActivityTaskScheduledEventAttributes()
			if attrs != nil {
				attributes["activity_type"] = attrs.ActivityType.GetName()
				if attrs.Input != nil && len(attrs.Input.Payloads) > 0 {
					attributes["input"] = attrs.Input.String()
				}
			}
		case "ActivityTaskCompleted":
			attrs := event.GetActivityTaskCompletedEventAttributes()
			if attrs != nil && attrs.Result != nil && len(attrs.Result.Payloads) > 0 {
				attributes["result"] = attrs.Result.String()
			}
		case "WorkflowExecutionCompleted":
			attrs := event.GetWorkflowExecutionCompletedEventAttributes()
			if attrs != nil && attrs.Result != nil && len(attrs.Result.Payloads) > 0 {
				attributes["result"] = attrs.Result.String()
			}
		}

		workflowEvent.Attributes = attributes
		events = append(events, workflowEvent)
	}

	// Get AI model and policy versions from our audit log
	aiVersions, policyVersions := s.getVersionInfo(ctx, workflowID)

	execution := &WorkflowExecution{
		WorkflowID:      workflowID,
		RunID:           runID,
		WorkflowType:    describe.WorkflowExecutionInfo.Type.Name,
		StartTime:       describe.WorkflowExecutionInfo.StartTime.AsTime(),
		CloseTime:       nil,
		Status:          describe.WorkflowExecutionInfo.Status.String(),
		Events:          events,
		AIModelVersions: aiVersions,
		PolicyVersions:  policyVersions,
	}

	if describe.WorkflowExecutionInfo.CloseTime != nil {
		closeTime := describe.WorkflowExecutionInfo.CloseTime.AsTime()
		execution.CloseTime = &closeTime
		execution.ExecutionTime = closeTime.Sub(execution.StartTime).Milliseconds()
	}

	// Parse inputs and results
	if len(events) > 0 {
		if inputStr, ok := events[0].Attributes["input"].(string); ok {
			json.Unmarshal([]byte(inputStr), &execution.Inputs)
		}
	}

	if workflowResult != nil {
		resultBytes, _ := json.Marshal(workflowResult)
		json.Unmarshal(resultBytes, &execution.Result)
	}

	// Store replay in audit log
	s.auditReplay(ctx, workflowID, runID)

	return execution, nil
}

// getVersionInfo retrieves AI model and policy versions used in workflow
// TODO: Migrate to Hasura GraphQL query with JSONB operators:
//
//	query GetVersionInfo($workflow_id: String!) {
//	  audit_events(
//	    where: {
//	      resource_id: {_eq: $workflow_id},
//	      _or: [
//	        {metadata: {_contains: {ai_model_version: {}}}},
//	        {metadata: {_contains: {policy_version: {}}}}
//	      ]
//	    },
//	    distinct_on: [metadata]
//	  ) {
//	    metadata
//	  }
//	}
//
// Note: Uses JSONB ->> operator and DISTINCT for version extraction
func (s *ReplayService) getVersionInfo(ctx context.Context, workflowID string) ([]string, []string) {
	aiVersions := []string{}
	policyVersions := []string{}

	// Query audit log for versions
	query := `
		SELECT DISTINCT 
			metadata->>'ai_model_version' as ai_version,
			metadata->>'policy_version' as policy_version
		FROM audit_events
		WHERE resource_id = $1
		AND (metadata->>'ai_model_version' IS NOT NULL 
		     OR metadata->>'policy_version' IS NOT NULL)
	`

	rows, err := s.db.QueryContext(ctx, query, workflowID)
	if err != nil {
		return aiVersions, policyVersions
	}
	defer rows.Close()

	for rows.Next() {
		var aiVer, policyVer sql.NullString
		rows.Scan(&aiVer, &policyVer)

		if aiVer.Valid && aiVer.String != "" {
			aiVersions = append(aiVersions, aiVer.String)
		}
		if policyVer.Valid && policyVer.String != "" {
			policyVersions = append(policyVersions, policyVer.String)
		}
	}

	return aiVersions, policyVersions
}

// auditReplay logs the workflow replay for compliance
// TODO: Migrate to Hasura GraphQL mutation:
//
//	mutation AuditReplay($object: audit_events_insert_input!) {
//	  insert_audit_events_one(object: $object) {
//	    event_id
//	    tenant_id
//	    user_id
//	    action
//	    resource
//	    resource_id
//	    metadata
//	    created_at
//	  }
//	}
//
// Note: Logs workflow replay for regulatory compliance (SEC, FINRA)
func (s *ReplayService) auditReplay(ctx context.Context, workflowID, runID string) error {
	query := `
		INSERT INTO audit_events (
			tenant_id, user_id, action, resource, resource_id, 
			metadata, created_at
		) VALUES (
			$1, $2, 'WORKFLOW_REPLAY', 'WORKFLOW', $3,
			$4, NOW()
		)
	`

	metadata := map[string]interface{}{
		"run_id":      runID,
		"replay_time": time.Now(),
		"purpose":     "REGULATORY_INQUIRY",
	}
	metadataJSON, _ := json.Marshal(metadata)

	_, err := s.db.ExecContext(ctx, query,
		"system", // tenantID from context
		"system", // userID from context
		workflowID,
		metadataJSON,
	)

	return err
}

// SearchWorkflows finds workflows matching criteria for replay
// TODO: Migrate to Hasura GraphQL query with dynamic where conditions:
//
//	query SearchWorkflows($workflow_type: String, $start_time: timestamptz, $end_time: timestamptz) {
//	  temporal_workflows(
//	    where: {
//	      workflow_type: {_eq: $workflow_type},
//	      start_time: {_gte: $start_time, _lte: $end_time}
//	    },
//	    order_by: {start_time: desc},
//	    limit: 100
//	  ) {
//	    workflow_id
//	    run_id
//	    workflow_type
//	    start_time
//	    close_time
//	    status
//	  }
//	}
//
// Note: Dynamic query construction for flexible workflow search
func (s *ReplayService) SearchWorkflows(ctx context.Context, criteria WorkflowSearchCriteria) ([]WorkflowSummary, error) {
	// Build dynamic query based on criteria
	query := `
		SELECT 
			workflow_id,
			run_id,
			workflow_type,
			start_time,
			close_time,
			status
		FROM temporal_workflows
		WHERE 1=1
	`

	args := []interface{}{}
	argCount := 1

	if criteria.WorkflowType != "" {
		query += fmt.Sprintf(" AND workflow_type = $%d", argCount)
		args = append(args, criteria.WorkflowType)
		argCount++
	}

	if !criteria.StartTime.IsZero() {
		query += fmt.Sprintf(" AND start_time >= $%d", argCount)
		args = append(args, criteria.StartTime)
		argCount++
	}

	if !criteria.EndTime.IsZero() {
		query += fmt.Sprintf(" AND start_time <= $%d", argCount)
		args = append(args, criteria.EndTime)
		argCount++
	}

	query += " ORDER BY start_time DESC LIMIT 100"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	summaries := []WorkflowSummary{}
	for rows.Next() {
		var summary WorkflowSummary
		err := rows.Scan(
			&summary.WorkflowID,
			&summary.RunID,
			&summary.WorkflowType,
			&summary.StartTime,
			&summary.CloseTime,
			&summary.Status,
		)
		if err != nil {
			continue
		}
		summaries = append(summaries, summary)
	}

	return summaries, nil
}

// WorkflowSearchCriteria defines search parameters
type WorkflowSearchCriteria struct {
	WorkflowType string
	StartTime    time.Time
	EndTime      time.Time
	Status       string
}

// WorkflowSummary is a lightweight workflow representation
type WorkflowSummary struct {
	WorkflowID   string     `json:"workflow_id"`
	RunID        string     `json:"run_id"`
	WorkflowType string     `json:"workflow_type"`
	StartTime    time.Time  `json:"start_time"`
	CloseTime    *time.Time `json:"close_time,omitempty"`
	Status       string     `json:"status"`
}
