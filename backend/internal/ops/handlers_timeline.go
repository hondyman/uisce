package ops

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// GetTimeline returns recent events from the ops timeline
func (h *Handler) GetTimeline(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	sinceStr := q.Get("since")
	limitStr := q.Get("limit")

	// Parse since duration (e.g., "1h", "24h", "7d")
	since := time.Now().Add(-1 * time.Hour)
	if sinceStr != "" {
		if d, err := time.ParseDuration(sinceStr); err == nil {
			since = time.Now().Add(-d)
		}
	}

	limit := 200
	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 && v <= 1000 {
			limit = v
		}
	}

	events, err := h.store.ListEvents(r.Context(), since, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(TimelineResponse{
		Events: events,
		Total:  len(events),
	})
}

// GetIncident returns a specific incident with all its related events
func (h *Handler) GetIncident(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "incidentID")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid incidentID", http.StatusBadRequest)
		return
	}

	inc, events, err := h.store.GetIncident(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(IncidentResponse{
		Incident: inc,
		Events:   events,
	})
}

// CloseIncidentRequest is the request body for closing an incident
type CloseIncidentRequest struct {
	Summary   *string `json:"summary,omitempty"`
	RootCause *string `json:"root_cause,omitempty"`
}

// CloseIncident closes an incident and optionally records root cause
func (h *Handler) CloseIncident(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "incidentID")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid incidentID", http.StatusBadRequest)
		return
	}

	var req CloseIncidentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	if err := h.store.CloseIncident(r.Context(), id, req.Summary, req.RootCause); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"closed": true})
}

// ExecuteAction executes an ops action on an incident with RBAC, rate limiting, and audit logging
// POST /admin/ops/incidents/{incidentID}/execute-action
func (h *Handler) ExecuteAction(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "incidentID")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid incidentID", http.StatusBadRequest)
		return
	}

	var req ExecuteActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	// Extract user context from request headers
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "anonymous"
	}
	userRole := r.Header.Get("X-User-Role")
	sourceIP := r.RemoteAddr

	// === Phase 2.4b Security Checks ===

	// 1. RBAC: Check user role
	if !h.hasOpsManagerRole(userRole) {
		http.Error(w, "Unauthorized: ops_manager role required", http.StatusForbidden)
		return
	}

	// 2. Rate Limiting: Check if user is within limit
	if !h.rateLimiter.IsAllowed(userID) {
		remaining := h.rateLimiter.GetRemaining(userID)
		w.Header().Set("X-RateLimit-Remaining", "0")
		http.Error(w, fmt.Sprintf("Rate limit exceeded: max 10 actions per minute (remaining: %d)", remaining), http.StatusTooManyRequests)
		return
	}

	// 3. Parameter Validation: Validate action parameters
	if err := h.paramValidator.Validate(req.ActionType, req.Parameters); err != nil {
		http.Error(w, fmt.Sprintf("Invalid parameters: %v", err), http.StatusBadRequest)
		return
	}

	// === Execute Action ===
	startTime := time.Now()
	result, err := h.actionExecutor.ExecuteAction(r.Context(), id, req.ActionType, req.Parameters, req.Region) // Phase 3.3: Pass region
	durationMs := time.Since(startTime).Milliseconds()

	// 4. Response Sanitization: Remove sensitive data
	sanitized := h.responseSanitizer.Sanitize(result.Result)
	result.Result = sanitized

	// 5. Audit Logging: Log the action
	status := "success"
	if err != nil {
		status = "failed"
	}
	auditLog, _ := h.auditLogger.LogAction(userID, userRole, req.ActionType, sourceIP, id, status, req.Parameters, sanitized, nil, durationMs)

	// === Return Response ===
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":        err.Error(),
			"audit_log_id": auditLog.ID,
			"timestamp":    time.Now(),
			"user_id":      userID,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", h.rateLimiter.GetRemaining(userID)))
	w.Header().Set("X-Audit-Log-ID", auditLog.ID.String())
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

// hasOpsManagerRole checks if the user has the ops_manager role
func (h *Handler) hasOpsManagerRole(role string) bool {
	// Split by comma in case of multiple roles
	roles := strings.Split(role, ",")
	for _, r := range roles {
		if strings.TrimSpace(r) == "ops_manager" || strings.TrimSpace(r) == "admin" {
			return true
		}
	}
	return false
}

// ComputeRCA computes intelligent root cause analysis for an incident
// GET /admin/ops/incidents/{incidentID}/rca
func (h *Handler) ComputeRCA(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "incidentID")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid incidentID", http.StatusBadRequest)
		return
	}

	// Get incident and all related events
	_, events, err := h.store.GetIncident(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Compute RCA using correlation engine
	rca := h.rcaEngine.ComputeRCA(events)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(rca)
}

// GetSimilarIncidents finds historically similar incidents
// GET /admin/ops/incidents/{incidentID}/similar
func (h *Handler) GetSimilarIncidents(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "incidentID")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid incidentID", http.StatusBadRequest)
		return
	}

	// Get current incident events
	_, _, err = h.store.GetIncident(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// For now, return an empty list of similar incidents
	// In production, this would query historical incidents from store
	// and use PatternMatcher.FindSimilarIncidents()

	similarities := []map[string]interface{}{}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"incident_id":  id.String(),
		"similarities": similarities,
	})
}

// GetIncidentPattern returns the pattern fingerprint for an incident
// GET /admin/ops/incidents/{incidentID}/pattern
func (h *Handler) GetIncidentPattern(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "incidentID")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid incidentID", http.StatusBadRequest)
		return
	}

	// Get incident events
	_, events, err := h.store.GetIncident(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Create pattern from events
	pattern := h.patternMatcher.CreateIncidentPattern(events)
	if pattern == nil {
		http.Error(w, "unable to create pattern", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(pattern)
}
