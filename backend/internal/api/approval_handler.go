package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

type ApprovalEventService struct {
	db *sql.DB
}

func NewApprovalEventService(db *sql.DB) *ApprovalEventService {
	return &ApprovalEventService{db: db}
}

func (s *ApprovalEventService) GetApprovalEvents(w http.ResponseWriter, r *http.Request) {
	workflowID := r.URL.Query().Get("workflowId")
	stepKey := r.URL.Query().Get("step")

	// Note: Assuming we have a table `approval_event_log` or we query `business_process_event`.
	// The current model.go shows `BPEvent` mapped to `business_process_event`.
	// We should filter by event_type relating to approval/escalation.
	// For this implementation, I will assume we use `business_process_event` and store details in JSONB.

	rows, err := s.db.QueryContext(r.Context(), `
        select created_at, event_type, details
        from business_process_event
        where bp_run_id = $1 and step_key = $2
        order by created_at asc
    `, workflowID, stepKey)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var history []map[string]interface{}
	var currentLevel int = 0

	for rows.Next() {
		var ts string
		var eventType string
		var detailsJSON []byte

		if err := rows.Scan(&ts, &eventType, &detailsJSON); err != nil {
			continue
		}

		var details map[string]interface{}
		_ = json.Unmarshal(detailsJSON, &details)

		// Map to UI expectation
		item := map[string]interface{}{
			"timestamp":    ts,
			"action":       eventType,
			"approverRole": details["approverRole"],
			"level":        details["escalationLevel"],
		}

		// Try to infer level if not explicit
		if lvl, ok := details["escalationLevel"].(float64); ok {
			if int(lvl) > currentLevel {
				currentLevel = int(lvl)
			}
		}

		history = append(history, item)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"escalationHistory":      history,
		"currentEscalationLevel": currentLevel,
	})
}
