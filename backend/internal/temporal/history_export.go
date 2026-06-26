package temporal

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"go.temporal.io/sdk/client"
)

// ============================================================================
// HISTORY EXPORT SERVICE
// Exports workflow execution history for audit, analytics, and BI
// ============================================================================

type HistoryExportService struct {
	client    client.Client
	namespace string
}

// NewHistoryExportService creates a new history export service
func NewHistoryExportService(c client.Client, namespace string) *HistoryExportService {
	return &HistoryExportService{
		client:    c,
		namespace: namespace,
	}
}

// HistoryExportRequest specifies what to export
type HistoryExportRequest struct {
	WorkflowID string `json:"workflow_id" binding:"required"`
	RunID      string `json:"run_id,omitempty"`
	Format     string `json:"format"` // "json", "csv", "parquet" (default: json)
}

// HistoryExportResponse wraps the exported history
type HistoryExportResponse struct {
	Status     string         `json:"status"`
	WorkflowID string         `json:"workflow_id"`
	RunID      string         `json:"run_id,omitempty"`
	Events     []HistoryEvent `json:"events"`
	Summary    HistorySummary `json:"summary"`
	Timestamp  time.Time      `json:"timestamp"`
}

// HistoryEvent represents a single workflow event
type HistoryEvent struct {
	EventID    int64                  `json:"event_id"`
	EventType  string                 `json:"event_type"`
	Timestamp  time.Time              `json:"timestamp"`
	Attributes map[string]interface{} `json:"attributes"`
}

// HistorySummary provides metadata about the export
type HistorySummary struct {
	TotalEvents      int64         `json:"total_events"`
	StartTime        time.Time     `json:"start_time"`
	EndTime          time.Time     `json:"end_time"`
	Duration         time.Duration `json:"duration"`
	Status           string        `json:"status"`
	ExportedAt       time.Time     `json:"exported_at"`
	TenantID         string        `json:"tenant_id,omitempty"`
	ParentWorkflowID string        `json:"parent_workflow_id,omitempty"`
}

// ExportHistory exports the complete execution history for a workflow
func (hes *HistoryExportService) ExportHistory(ctx context.Context, req HistoryExportRequest) (*HistoryExportResponse, error) {
	log.Printf("[HistoryExport] Exporting history for workflow %s (run: %s)", req.WorkflowID, req.RunID)

	// Get workflow execution history
	iter := hes.client.GetWorkflowHistory(ctx, req.WorkflowID, req.RunID, false, 0)

	// Convert history events to our format
	var events []HistoryEvent
	var startTime, endTime time.Time

	for iter.HasNext() {
		event, err := iter.Next()
		if err != nil {
			log.Printf("[HistoryExport] Error reading event: %v", err)
			break
		}

		if event == nil {
			continue
		}

		eventTime := event.GetEventTime()
		if eventTime != nil {
			et := eventTime.AsTime()
			if startTime.IsZero() {
				startTime = et
			}
			endTime = et

			events = append(events, HistoryEvent{
				EventID:    event.GetEventId(),
				EventType:  event.GetEventType().String(),
				Timestamp:  et,
				Attributes: extractEventAttributes(event),
			})
		}
	}

	resp := &HistoryExportResponse{
		Status:     "success",
		WorkflowID: req.WorkflowID,
		RunID:      req.RunID,
		Events:     events,
		Summary: HistorySummary{
			TotalEvents: int64(len(events)),
			StartTime:   startTime,
			EndTime:     endTime,
			Duration:    endTime.Sub(startTime),
			Status:      "exported",
			ExportedAt:  time.Now(),
		},
		Timestamp: time.Now(),
	}

	log.Printf("[HistoryExport] Exported %d events for workflow %s", len(events), req.WorkflowID)
	return resp, nil
}

// ExportHistoryForAnalytics exports history in a format suitable for BI/analytics tools
// Flattened JSON with denormalized event data
func (hes *HistoryExportService) ExportHistoryForAnalytics(ctx context.Context, req HistoryExportRequest) ([]map[string]interface{}, error) {
	resp, err := hes.ExportHistory(ctx, req)
	if err != nil {
		return nil, err
	}

	// Flatten events with workflow metadata
	var records []map[string]interface{}
	for _, event := range resp.Events {
		record := map[string]interface{}{
			"workflow_id": req.WorkflowID,
			"run_id":      req.RunID,
			"event_id":    event.EventID,
			"event_type":  event.EventType,
			"timestamp":   event.Timestamp,
			"exported_at": resp.Timestamp,
		}

		// Flatten nested attributes
		for k, v := range event.Attributes {
			record[fmt.Sprintf("attr_%s", k)] = v
		}

		records = append(records, record)
	}

	log.Printf("[HistoryExport] Flattened %d records for analytics", len(records))
	return records, nil
}

// ============================================================================
// BATCH EXPORT
// ============================================================================

type BatchExportRequest struct {
	WorkflowIDPattern string     `json:"workflow_id_pattern"` // regex pattern or prefix
	Query             string     `json:"query"`               // Temporal query DSL
	StartTime         *time.Time `json:"start_time,omitempty"`
	EndTime           *time.Time `json:"end_time,omitempty"`
	Format            string     `json:"format"` // json, jsonl, csv
}

// ExportBatch exports history for multiple workflows
// This is useful for reporting and compliance audits
func (hes *HistoryExportService) ExportBatch(ctx context.Context, req BatchExportRequest) (map[string]interface{}, error) {
	log.Printf("[HistoryExport] Batch exporting workflows matching: %s", req.Query)

	// In production, list workflows using Temporal's ListWorkflow API
	// For now, return placeholder
	return map[string]interface{}{
		"status":      "queued",
		"query":       req.Query,
		"format":      req.Format,
		"total_count": 0,
		"exported_at": time.Now(),
	}, nil
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// extractEventAttributes extracts relevant attributes from a workflow event
func extractEventAttributes(event interface{}) map[string]interface{} {
	// This is a simplified version; in production, use reflection or type assertions
	// to extract event-specific fields
	attrs := make(map[string]interface{})

	// Convert event to JSON and back to map for generic attribute extraction
	if b, err := json.Marshal(event); err == nil {
		if err := json.Unmarshal(b, &attrs); err == nil {
			return attrs
		}
	}

	return attrs
}

// ============================================================================
// COMPLIANCE & AUDIT EXPORTS
// ============================================================================

type AuditTrailExport struct {
	AuditID    string                 `json:"audit_id"`
	WorkflowID string                 `json:"workflow_id"`
	RunID      string                 `json:"run_id"`
	ActorID    string                 `json:"actor_id,omitempty"`
	Action     string                 `json:"action"` // "created", "updated", "terminated", "signaled"
	Details    map[string]interface{} `json:"details"`
	Timestamp  time.Time              `json:"timestamp"`
	Reason     string                 `json:"reason,omitempty"`
	Status     string                 `json:"status"` // "success", "failed"
}

// ExportAuditTrail exports compliance-relevant audit trail
// Useful for regulatory requirements (SOX, HIPAA, etc.)
func (hes *HistoryExportService) ExportAuditTrail(ctx context.Context, workflowID, runID string) ([]AuditTrailExport, error) {
	resp, err := hes.ExportHistory(ctx, HistoryExportRequest{
		WorkflowID: workflowID,
		RunID:      runID,
	})
	if err != nil {
		return nil, err
	}

	// Filter and format for audit compliance
	var audit []AuditTrailExport
	for _, event := range resp.Events {
		trail := AuditTrailExport{
			AuditID:    fmt.Sprintf("%s-%d", workflowID, event.EventID),
			WorkflowID: workflowID,
			RunID:      runID,
			Action:     event.EventType,
			Details:    event.Attributes,
			Timestamp:  event.Timestamp,
			Status:     "success",
		}
		audit = append(audit, trail)
	}

	log.Printf("[AuditTrail] Exported %d audit records for workflow %s", len(audit), workflowID)
	return audit, nil
}
