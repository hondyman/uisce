package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/hondyman/uisce/backend/internal/services"
	"github.com/hondyman/uisce/backend/models"
)

// AccessIntelligenceHandler handles API requests for the unified access intelligence service.
type AccessIntelligenceHandler struct {
	service      *services.AccessIntelligenceService
	securityDeps SecurityContextDeps
}

// NewAccessIntelligenceHandler creates a new AccessIntelligenceHandler.
func NewAccessIntelligenceHandler(service *services.AccessIntelligenceService, securityDeps SecurityContextDeps) *AccessIntelligenceHandler {
	return &AccessIntelligenceHandler{service: service, securityDeps: securityDeps}
}

// RegisterRoutes registers the routes for AccessIntelligenceHandler.
func (h *AccessIntelligenceHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/access-intelligence", func(r chi.Router) {
		r.Get("/claims/effective", h.HandleGetEffectiveClaims)
		r.Post("/claims/grant", h.HandleGrantClaim)
		r.Post("/bundles/assign", h.HandleAssignBundle)
		r.Post("/evaluate", h.HandleEvaluateAccess)
		r.Post("/cache/refresh", h.HandleRefreshClaimsCache)
		r.Get("/decisions/{id}/trace", h.HandleGetDecisionTrace)
		r.Get("/decisions/{id}/explanation", h.HandleGetDecisionExplanation)
		r.Post("/simulate", h.HandleSimulateAccess)
		r.Get("/cockpit/snapshot", h.HandleGetGovernanceCockpitSnapshot)
	})
}

// HandleGetEffectiveClaims retrieves all effective claims for a user.
func (h *AccessIntelligenceHandler) HandleGetEffectiveClaims(w http.ResponseWriter, r *http.Request) {
	secCtx, ctx, err := SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	userID, ok := security.RequireUser(w, r)
	if !ok {
		return
	}
	tenantID := secCtx.TenantID
	if tenantID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "tenant_id is required"})
		return
	}
	claims, err := h.service.GetEffectiveClaims(ctx, userID, tenantID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to get effective claims"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(claims)
}

// HandleGrantClaim grants a claim to a user.
func (h *AccessIntelligenceHandler) HandleGrantClaim(w http.ResponseWriter, r *http.Request) {
	var req models.GrantClaimRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid request payload"})
		return
	}
	actorID := "current_admin" // In a real app, get this from the auth context
	claim, conflict, err := h.service.GrantClaim(r.Context(), req, actorID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to grant claim", "details": err.Error()})
		return
	}
	if conflict != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Claim conflict detected", "conflict": conflict})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(claim)
}

// HandleAssignBundle assigns a claim bundle to a user.
func (h *AccessIntelligenceHandler) HandleAssignBundle(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID   string `json:"user_id" binding:"required"`
		BundleID string `json:"bundle_id" binding:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid request payload"})
		return
	}
	actorID := "current_admin" // In a real app, get this from the auth context
	err := h.service.AssignBundle(r.Context(), req.UserID, req.BundleID, actorID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to assign bundle"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "assigned"})
}

// HandleEvaluateAccess performs a real-time access check.
func (h *AccessIntelligenceHandler) HandleEvaluateAccess(w http.ResponseWriter, r *http.Request) {
	var req models.EvaluateAccessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid request payload"})
		return
	}
	response, err := h.service.EvaluateAccess(r.Context(), req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to evaluate access"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleRefreshClaimsCache invalidates a user's claims cache.
func (h *AccessIntelligenceHandler) HandleRefreshClaimsCache(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID   string `json:"user_id" binding:"required"`
		TenantID string `json:"tenant_id" binding:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid request payload"})
		return
	}
	err := h.service.RefreshClaimsCache(r.Context(), req.UserID, req.TenantID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to refresh cache"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "cache_invalidated"})
}

// HandleGetDecisionTrace retrieves a decision trace.
func (h *AccessIntelligenceHandler) HandleGetDecisionTrace(w http.ResponseWriter, r *http.Request) {
	decisionIDStr := chi.URLParam(r, "id")
	decisionID, err := uuid.Parse(decisionIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid decision ID format"})
		return
	}
	trace, err := h.service.GetDecisionTrace(r.Context(), decisionID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to get decision trace"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(trace)
}

// HandleGetDecisionExplanation retrieves a decision explanation.
func (h *AccessIntelligenceHandler) HandleGetDecisionExplanation(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]interface{}{"error": "Not implemented"})
}

// HandleSimulateAccess performs a what-if access simulation.
func (h *AccessIntelligenceHandler) HandleSimulateAccess(w http.ResponseWriter, r *http.Request) {
	var req models.SimulateAccessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid request payload"})
		return
	}
	response, err := h.service.SimulateAccess(r.Context(), req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to simulate access"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleGetGovernanceCockpitSnapshot retrieves the snapshot for the governance cockpit.
func (h *AccessIntelligenceHandler) HandleGetGovernanceCockpitSnapshot(w http.ResponseWriter, r *http.Request) {
	secCtx, ctx, err := SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	tenantID := secCtx.TenantID
	if tenantID == "" {
		tenantID = "default_tenant" // Or get from auth context
	}
	snapshot, err := h.service.GetGovernanceCockpitSnapshot(ctx, tenantID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to get governance cockpit snapshot"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snapshot)
}
