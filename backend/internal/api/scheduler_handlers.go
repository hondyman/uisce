package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	si "github.com/hondyman/semlayer/backend/internal/scheduler_intelligence"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// SchedulerHandlers handles scheduler intelligence API requests
type SchedulerHandlers struct {
	service *si.Service
	logger  *zap.Logger
}

// NewSchedulerHandlers creates new scheduler handlers
func NewSchedulerHandlers(db *sqlx.DB, semanticClient si.SemanticClient, logger *zap.Logger) *SchedulerHandlers {
	return &SchedulerHandlers{
		service: si.NewService(db, semanticClient, logger),
		logger:  logger,
	}
}

// Service returns the underlying scheduler intelligence service
func (h *SchedulerHandlers) Service() *si.Service {
	return h.service
}

// RegisterRoutes registers all scheduler routes
func (h *SchedulerHandlers) RegisterRoutes(r chi.Router) {
	r.Route("/scheduler", func(r chi.Router) {
		// Jobs
		r.Get("/jobs", h.ListJobs)
		r.Post("/jobs", h.CreateJob)
		r.Get("/jobs/{id}", h.GetJob)
		r.Patch("/jobs/{id}", h.UpdateJob)
		r.Delete("/jobs/{id}", h.DeleteJob)
		r.Post("/jobs/{id}/run", h.TriggerJob)
		r.Get("/jobs/{id}/runs", h.GetJobRuns)

		// DAGs
		r.Get("/dags", h.ListDAGs)
		r.Post("/dags", h.CreateDAG)
		r.Get("/dags/{id}", h.GetDAG)
		r.Patch("/dags/{id}", h.UpdateDAG)
		r.Delete("/dags/{id}", h.DeleteDAG)
		r.Post("/dags/{id}/run", h.TriggerDAG)
		r.Get("/dags/{id}/runs", h.GetDAGRuns)

		// Runs
		r.Get("/runs/jobs/{id}", h.GetJobRun)
		r.Get("/runs/dags/{id}", h.GetDAGRun)

		// AI Suggestions
		r.Get("/ai/suggestions", h.GetAISuggestions)
		r.Post("/ai/suggestions/{id}/accept", h.AcceptAISuggestion)
		r.Post("/ai/suggestions/{id}/dismiss", h.DismissAISuggestion)

		// Stats
		r.Get("/stats", h.GetStats)
	})
}

// ============================================================================
// Job Handlers
// ============================================================================

// ListJobs returns a list of scheduled jobs
func (h *SchedulerHandlers) ListJobs(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		tenantID = r.URL.Query().Get("tenant_id")
	}
	if tenantID == "" {
		respondWithError(w, http.StatusBadRequest, "tenant_id is required")
		return
	}

	filters := si.JobListFilters{
		TenantID:     tenantID,
		DatasourceID: r.URL.Query().Get("datasource_id"),
		Category:     r.URL.Query().Get("category"),
	}

	if active := r.URL.Query().Get("is_active"); active != "" {
		b, _ := strconv.ParseBool(active)
		filters.IsActive = &b
	}

	if limit := r.URL.Query().Get("limit"); limit != "" {
		filters.Limit, _ = strconv.Atoi(limit)
	}
	if offset := r.URL.Query().Get("offset"); offset != "" {
		filters.Offset, _ = strconv.Atoi(offset)
	}

	jobs, total, err := h.service.ListJobs(r.Context(), filters)
	if err != nil {
		h.logger.Sugar().Error("Failed to list jobs", "error", err)
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"jobs":  jobs,
		"total": total,
	})
}

// CreateJob creates a new scheduled job
func (h *SchedulerHandlers) CreateJob(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		tenantID = r.URL.Query().Get("tenant_id")
	}
	if tenantID == "" {
		respondWithError(w, http.StatusBadRequest, "tenant_id is required")
		return
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid tenant_id")
		return
	}

	var req si.CreateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	changesetID, err := h.service.CreateJob(r.Context(), tenantUUID, req)
	if err != nil {
		h.logger.Sugar().Error("Failed to create job", "error", err)
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"changeset_id": changesetID,
		"status":       "pending_review",
	})
}

// GetJob returns a single job by ID
func (h *SchedulerHandlers) GetJob(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid job id")
		return
	}

	job, err := h.service.GetJob(r.Context(), id)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "job not found")
		return
	}

	respondJSON(w, http.StatusOK, job)
}

// UpdateJob updates an existing job
func (h *SchedulerHandlers) UpdateJob(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid job id")
		return
	}

	var req si.UpdateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	changesetID, err := h.service.UpdateJob(r.Context(), id, req)
	if err != nil {
		h.logger.Sugar().Error("Failed to update job", "error", err)
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"changeset_id": changesetID,
		"status":       "pending_review",
	})
}

// DeleteJob deletes a job
func (h *SchedulerHandlers) DeleteJob(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid job id")
		return
	}

	changesetID, err := h.service.DeleteJob(r.Context(), id)
	if err != nil {
		h.logger.Sugar().Error("Failed to delete job", "error", err)
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"changeset_id": changesetID,
		"status":       "pending_review",
	})
}

// TriggerJob triggers a job run
func (h *SchedulerHandlers) TriggerJob(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid job id")
		return
	}

	var req si.TriggerJobRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondWithError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
			return
		}
	}

	// Extract user ID from context/header if available
	var triggeredBy *uuid.UUID
	if userID := r.Header.Get("X-User-ID"); userID != "" {
		if parsed, err := uuid.Parse(userID); err == nil {
			triggeredBy = &parsed
		}
	}

	run, err := h.service.TriggerJob(r.Context(), id, triggeredBy, req.Parameters)
	if err != nil {
		h.logger.Sugar().Error("Failed to trigger job", "error", err)
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusAccepted, run)
}

// GetJobRuns returns runs for a job
func (h *SchedulerHandlers) GetJobRuns(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")

	filters := si.JobRunListFilters{
		JobID: idStr,
	}

	if limit := r.URL.Query().Get("limit"); limit != "" {
		filters.Limit, _ = strconv.Atoi(limit)
	}
	if status := r.URL.Query().Get("status"); status != "" {
		filters.Status = status
	}

	runs, err := h.service.ListJobRuns(r.Context(), filters)
	if err != nil {
		h.logger.Sugar().Error("Failed to list job runs", "error", err)
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, runs)
}

// GetJobRun returns a single job run by ID
func (h *SchedulerHandlers) GetJobRun(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid run id")
		return
	}

	run, err := h.service.GetJobRun(r.Context(), id)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "run not found")
		return
	}

	respondJSON(w, http.StatusOK, run)
}

// ============================================================================
// DAG Handlers
// ============================================================================

// ListDAGs returns a list of DAGs
func (h *SchedulerHandlers) ListDAGs(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		tenantID = r.URL.Query().Get("tenant_id")
	}
	if tenantID == "" {
		respondWithError(w, http.StatusBadRequest, "tenant_id is required")
		return
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid tenant_id")
		return
	}

	activeOnly := r.URL.Query().Get("active_only") == "true"

	dags, err := h.service.ListDAGs(r.Context(), tenantUUID, activeOnly)
	if err != nil {
		h.logger.Sugar().Error("Failed to list DAGs", "error", err)
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, dags)
}

// CreateDAG creates a new DAG
func (h *SchedulerHandlers) CreateDAG(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		tenantID = r.URL.Query().Get("tenant_id")
	}
	if tenantID == "" {
		respondWithError(w, http.StatusBadRequest, "tenant_id is required")
		return
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid tenant_id")
		return
	}

	var req si.CreateDAGRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	changesetID, err := h.service.CreateDAG(r.Context(), tenantUUID, req)
	if err != nil {
		h.logger.Sugar().Error("Failed to create DAG", "error", err)
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"changeset_id": changesetID,
		"status":       "pending_review",
	})
}

// GetDAG returns a single DAG by ID
func (h *SchedulerHandlers) GetDAG(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid DAG id")
		return
	}

	dag, err := h.service.GetDAG(r.Context(), id)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "DAG not found")
		return
	}

	respondJSON(w, http.StatusOK, dag)
}

// UpdateDAG updates an existing DAG
func (h *SchedulerHandlers) UpdateDAG(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid DAG id")
		return
	}

	var req si.UpdateDAGRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	changesetID, err := h.service.UpdateDAG(r.Context(), id, req)
	if err != nil {
		h.logger.Sugar().Error("Failed to update DAG", "error", err)
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"changeset_id": changesetID,
		"status":       "pending_review",
	})
}

// DeleteDAG deletes a DAG
func (h *SchedulerHandlers) DeleteDAG(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid DAG id")
		return
	}

	changesetID, err := h.service.DeleteDAG(r.Context(), id)
	if err != nil {
		h.logger.Sugar().Error("Failed to delete DAG", "error", err)
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"changeset_id": changesetID,
		"status":       "pending_review",
	})
}

// TriggerDAG triggers a DAG run
func (h *SchedulerHandlers) TriggerDAG(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid DAG id")
		return
	}

	var triggeredBy *uuid.UUID
	if userID := r.Header.Get("X-User-ID"); userID != "" {
		if parsed, err := uuid.Parse(userID); err == nil {
			triggeredBy = &parsed
		}
	}

	run, err := h.service.TriggerDAG(r.Context(), id, triggeredBy)
	if err != nil {
		h.logger.Sugar().Error("Failed to trigger DAG", "error", err)
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusAccepted, run)
}

// GetDAGRuns returns runs for a DAG
func (h *SchedulerHandlers) GetDAGRuns(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid DAG id")
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		limit, _ = strconv.Atoi(l)
	}

	runs, err := h.service.ListDAGRuns(r.Context(), id, limit)
	if err != nil {
		h.logger.Sugar().Error("Failed to list DAG runs", "error", err)
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, runs)
}

// GetDAGRun returns a single DAG run by ID
func (h *SchedulerHandlers) GetDAGRun(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid run id")
		return
	}

	run, err := h.service.GetDAGRun(r.Context(), id)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "run not found")
		return
	}

	respondJSON(w, http.StatusOK, run)
}

// ============================================================================
// AI Suggestion Handlers
// ============================================================================

// GetAISuggestions returns pending AI suggestions
func (h *SchedulerHandlers) GetAISuggestions(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		tenantID = r.URL.Query().Get("tenant_id")
	}
	if tenantID == "" {
		respondWithError(w, http.StatusBadRequest, "tenant_id is required")
		return
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid tenant_id")
		return
	}

	suggestions, err := h.service.GetPendingAISuggestions(r.Context(), tenantUUID)
	if err != nil {
		h.logger.Sugar().Error("Failed to get AI suggestions", "error", err)
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, suggestions)
}

// AcceptAISuggestion accepts an AI suggestion
func (h *SchedulerHandlers) AcceptAISuggestion(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid suggestion id")
		return
	}

	if err := h.service.AcceptAISuggestion(r.Context(), id); err != nil {
		h.logger.Sugar().Error("Failed to accept suggestion", "error", err)
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "suggestion accepted"})
}

// DismissAISuggestion dismisses an AI suggestion
func (h *SchedulerHandlers) DismissAISuggestion(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid suggestion id")
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if r.ContentLength > 0 {
		json.NewDecoder(r.Body).Decode(&req)
	}

	if err := h.service.DismissAISuggestion(r.Context(), id, req.Reason); err != nil {
		h.logger.Sugar().Error("Failed to dismiss suggestion", "error", err)
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "suggestion dismissed"})
}

// ============================================================================
// Stats Handler
// ============================================================================

// GetStats returns scheduler statistics
func (h *SchedulerHandlers) GetStats(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		tenantID = r.URL.Query().Get("tenant_id")
	}
	if tenantID == "" {
		respondWithError(w, http.StatusBadRequest, "tenant_id is required")
		return
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid tenant_id")
		return
	}

	stats, err := h.service.GetJobStats(r.Context(), tenantUUID)
	if err != nil {
		h.logger.Sugar().Error("Failed to get stats", "error", err)
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, stats)
}
