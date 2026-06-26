package workflow

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"go.temporal.io/sdk/client"
)

// ReplayHandler handles HTTP requests for workflow replay
type ReplayHandler struct {
	replayService *ReplayService
}

// NewReplayHandler creates a new replay handler
func NewReplayHandler(db *sqlx.DB, temporalClient client.Client) *ReplayHandler {
	return &ReplayHandler{
		replayService: NewReplayService(db, temporalClient),
	}
}

// ReplayWorkflow handles GET /api/workflows/{workflowId}/replay
func (h *ReplayHandler) ReplayWorkflow(w http.ResponseWriter, r *http.Request) {
	workflowID := chi.URLParam(r, "workflowId")
	runID := r.URL.Query().Get("run_id") // Optional, uses latest if not provided

	execution, err := h.replayService.ReplayWorkflow(r.Context(), workflowID, runID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(execution)
}

// SearchWorkflows handles GET /api/workflows/search
func (h *ReplayHandler) SearchWorkflows(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	criteria := WorkflowSearchCriteria{
		WorkflowType: query.Get("workflow_type"),
		Status:       query.Get("status"),
	}

	if startTimeStr := query.Get("start_time"); startTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			criteria.StartTime = t
		}
	}
	if endTimeStr := query.Get("end_time"); endTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			criteria.EndTime = t
		}
	}

	summaries, err := h.replayService.SearchWorkflows(r.Context(), criteria)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"workflows": summaries,
		"total":     len(summaries),
	})
}

// RegisterRoutes registers workflow replay routes
func (h *ReplayHandler) RegisterRoutes(r chi.Router) {
	r.Route("/workflows", func(r chi.Router) {
		r.Get("/{workflowId}/replay", h.ReplayWorkflow)
		r.Get("/search", h.SearchWorkflows)
	})
}
