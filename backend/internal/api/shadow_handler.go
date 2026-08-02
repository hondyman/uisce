package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/rules"
	"github.com/hondyman/uisce/backend/internal/shadow"
)

type ShadowHandler struct {
	replayEngine *shadow.ReplayEngine
}

func NewShadowHandler(engine *shadow.ReplayEngine) *ShadowHandler {
	return &ShadowHandler{replayEngine: engine}
}

type StartShadowJobRequest struct {
	DraftRuleID uuid.UUID      `json:"draft_rule_id"`
	RuleName    string         `json:"rule_name"`
	RuleNode    rules.RuleNode `json:"rule_node"`
	CreatedBy   string         `json:"created_by"`
}

func (h *ShadowHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/shadow", func(r chi.Router) {
		r.Post("/jobs", h.HandleStartJob)
		r.Get("/jobs/{jobID}/report", h.HandleGetReport)
		r.Post("/jobs/{jobID}/cancel", h.HandleCancelJob)
		r.Post("/jobs/{jobID}/complete", h.HandleCompleteJob)
	})
}

func (h *ShadowHandler) HandleStartJob(w http.ResponseWriter, r *http.Request) {
	tenantIDStr := r.Header.Get("X-Tenant-ID")
	if tenantIDStr == "" {
		http.Error(w, "X-Tenant-ID header is required", http.StatusBadRequest)
		return
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		http.Error(w, "invalid X-Tenant-ID header", http.StatusBadRequest)
		return
	}

	var req StartShadowJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.RuleName == "" {
		http.Error(w, "rule_name is required", http.StatusBadRequest)
		return
	}
	if req.CreatedBy == "" {
		http.Error(w, "created_by is required", http.StatusBadRequest)
		return
	}

	job, err := h.replayEngine.StartShadowJob(r.Context(), tenantID, req.DraftRuleID, req.RuleName, &req.RuleNode, req.CreatedBy)
	if err != nil {
		http.Error(w, "failed to start shadow job: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"job_id":       job.JobID,
		"tenant_id":    job.TenantID,
		"draft_rule_id": job.DraftRuleID,
		"rule_name":    job.RuleName,
		"status":       job.Status,
		"started_at":   job.TotalEvaluated,
	})
}

func (h *ShadowHandler) HandleGetReport(w http.ResponseWriter, r *http.Request) {
	jobIDStr := chi.URLParam(r, "jobID")
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		http.Error(w, "invalid job_id URL parameter", http.StatusBadRequest)
		return
	}

	report, err := h.replayEngine.GetImpactReport(r.Context(), jobID)
	if err != nil {
		http.Error(w, "failed to fetch impact report: "+err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

func (h *ShadowHandler) HandleCancelJob(w http.ResponseWriter, r *http.Request) {
	jobIDStr := chi.URLParam(r, "jobID")
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		http.Error(w, "invalid job_id URL parameter", http.StatusBadRequest)
		return
	}

	if err := h.replayEngine.CancelJob(r.Context(), jobID); err != nil {
		http.Error(w, "failed to cancel job: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "CANCELLED"})
}

func (h *ShadowHandler) HandleCompleteJob(w http.ResponseWriter, r *http.Request) {
	jobIDStr := chi.URLParam(r, "jobID")
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		http.Error(w, "invalid job_id URL parameter", http.StatusBadRequest)
		return
	}

	if err := h.replayEngine.CompleteJob(r.Context(), jobID); err != nil {
		http.Error(w, "failed to complete job: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "COMPLETED"})
}
