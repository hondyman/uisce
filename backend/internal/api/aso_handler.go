package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/aso"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// ASOHandler handles ASO-related API endpoints
type ASOHandler struct {
	engine      aso.ASOEngine
	policyStore aso.ASOPolicyStore
	optRepo     aso.ASOOptimizationRepository
}

// NewASOHandler creates a new ASO handler
func NewASOHandler(
	engine aso.ASOEngine,
	policyStore aso.ASOPolicyStore,
	optRepo aso.ASOOptimizationRepository,
) *ASOHandler {
	return &ASOHandler{
		engine:      engine,
		policyStore: policyStore,
		optRepo:     optRepo,
	}
}

// RegisterASORoutes registers all ASO routes
func RegisterASORoutes(r chi.Router, h *ASOHandler) {
	r.Route("/aso", func(r chi.Router) {
		// Summary and dashboard
		r.Get("/summary", h.GetSummary)
		r.Get("/summary/{env}", h.GetSummaryByEnv)

		// Policies
		r.Get("/policies", h.ListPolicies)
		r.Get("/policies/{env}", h.ListPoliciesByEnv)
		r.Post("/policies", h.UpsertPolicy)
		r.Delete("/policies/{id}", h.DeletePolicy)

		// Optimizations
		r.Get("/optimizations", h.ListOptimizations)
		r.Get("/optimizations/{id}", h.GetOptimization)
		r.Post("/optimizations/{id}/apply", h.ApplyOptimization)
		r.Post("/optimizations/{id}/approve", h.ApproveOptimization)
		r.Post("/optimizations/{id}/reject", h.RejectOptimization)

		// Manual triggers
		r.Post("/evaluate/{env}", h.TriggerEvaluation)
		r.Post("/evaluate/{env}/{tenantId}", h.TriggerTenantEvaluation)
	})
}

// ============================================================================
// Summary Endpoints
// ============================================================================

// GetSummary returns ASO summaries for all environments
func (h *ASOHandler) GetSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	envs := []string{"dev", "staging", "prod"}
	summaries := make(map[string]*aso.ASOSummary)

	for _, env := range envs {
		summary, err := h.engine.GetSummary(ctx, env)
		if err != nil {
			continue
		}
		summaries[env] = summary
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summaries)
}

// GetSummaryByEnv returns ASO summary for a specific environment
func (h *ASOHandler) GetSummaryByEnv(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	env := chi.URLParam(r, "env")

	summary, err := h.engine.GetSummary(ctx, env)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// ============================================================================
// Policy Endpoints
// ============================================================================

// ListPolicies returns all policies
func (h *ASOHandler) ListPolicies(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	policies, err := h.policyStore.ListAllPolicies(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(policies)
}

// ListPoliciesByEnv returns policies for an environment
func (h *ASOHandler) ListPoliciesByEnv(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	env := chi.URLParam(r, "env")

	policies, err := h.policyStore.ListPolicies(ctx, env)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(policies)
}

// UpsertPolicyRequest is the request body for creating/updating a policy
type UpsertPolicyRequest struct {
	Env                    string     `json:"env"`
	TenantID               *uuid.UUID `json:"tenant_id,omitempty"`
	Enabled                bool       `json:"enabled"`
	Mode                   string     `json:"mode"`
	MaxNewPreAggsPerDay    int        `json:"max_new_preaggs_per_day"`
	MaxChangesPerDay       int        `json:"max_changes_per_day"`
	MinScoreForNewPreAgg   float64    `json:"min_score_for_new_preagg"`
	MinUsageForRetirement  int        `json:"min_usage_for_retirement"`
	HotPathThresholdMs     int        `json:"hot_path_threshold_ms"`
	LookbackWindowSeconds  int        `json:"lookback_window_seconds"`
	PrewarmEnabled         bool       `json:"prewarm_enabled"`
	PrewarmLeadTimeMinutes int        `json:"prewarm_lead_time_minutes"`
}

// UpsertPolicy creates or updates a policy
func (h *ASOHandler) UpsertPolicy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req UpsertPolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get actor from context
	actor := r.Header.Get("X-User-ID")
	if actor == "" {
		actor = "api_user"
	}

	policy := &aso.ASOPolicy{
		Env:                    req.Env,
		TenantID:               req.TenantID,
		Enabled:                req.Enabled,
		Mode:                   aso.ASOMode(req.Mode),
		MaxNewPreAggsPerDay:    req.MaxNewPreAggsPerDay,
		MaxChangesPerDay:       req.MaxChangesPerDay,
		MinScoreForNewPreAgg:   req.MinScoreForNewPreAgg,
		MinUsageForRetirement:  req.MinUsageForRetirement,
		HotPathThresholdMs:     req.HotPathThresholdMs,
		LookbackWindowSeconds:  req.LookbackWindowSeconds,
		PrewarmEnabled:         req.PrewarmEnabled,
		PrewarmLeadTimeMinutes: req.PrewarmLeadTimeMinutes,
		UpdatedBy:              actor,
	}

	if err := h.policyStore.UpsertPolicy(ctx, policy); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(policy)
}

// DeletePolicy deletes a tenant policy override
func (h *ASOHandler) DeletePolicy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid policy ID", http.StatusBadRequest)
		return
	}

	if err := h.policyStore.DeletePolicy(ctx, id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ============================================================================
// Optimization Endpoints
// ============================================================================

// ListOptimizations returns optimizations with filters
func (h *ASOHandler) ListOptimizations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query params
	filter := aso.OptimizationFilter{
		Limit:  50,
		Offset: 0,
	}

	if env := r.URL.Query().Get("env"); env != "" {
		filter.Env = &env
	}
	if status := r.URL.Query().Get("status"); status != "" {
		s := aso.OptimizationStatus(status)
		filter.Status = &s
	}
	if optType := r.URL.Query().Get("type"); optType != "" {
		t := aso.OptimizationType(optType)
		filter.Type = &t
	}
	if targetType := r.URL.Query().Get("target_type"); targetType != "" {
		tt := aso.TargetType(targetType)
		filter.TargetType = &tt
	}
	if limit := r.URL.Query().Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			filter.Limit = l
		}
	}
	if offset := r.URL.Query().Get("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil {
			filter.Offset = o
		}
	}

	opts, err := h.optRepo.List(ctx, filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(opts)
}

// GetOptimization returns a single optimization with full details
func (h *ASOHandler) GetOptimization(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid optimization ID", http.StatusBadRequest)
		return
	}

	opt, err := h.optRepo.GetByID(ctx, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if opt == nil {
		http.Error(w, "Optimization not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(opt)
}

// ApplyOptimization triggers application of an optimization
func (h *ASOHandler) ApplyOptimization(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid optimization ID", http.StatusBadRequest)
		return
	}

	actor := r.Header.Get("X-User-ID")
	if actor == "" {
		actor = "api_user"
	}

	// Permission check
	opt, err := h.optRepo.GetByID(ctx, id)
	if err != nil {
		http.Error(w, "Optimization not found", http.StatusNotFound)
		return
	}

	// Fetch policy if possible, otherwise nil (defaults to manual apply rules)
	var policy *aso.ASOPolicy
	// Try to fetch effective policy for this optimization's context
	policy, _ = h.policyStore.GetPolicy(ctx, opt.Env, opt.TenantID)

	authCtx := h.getAuthContext(r)
	result := aso.AuthorizeOptimizationAction(&authCtx, "apply", opt, policy)
	if !result.Allowed {
		http.Error(w, "Unauthorized: "+result.Reason, http.StatusForbidden)
		return
	}

	if err := h.engine.ApplyOptimization(ctx, id, actor); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Return updated optimization
	refreshedOpt, _ := h.optRepo.GetByID(ctx, id)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(refreshedOpt)
}

// ApproveOptimization approves an optimization for later application
func (h *ASOHandler) ApproveOptimization(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid optimization ID", http.StatusBadRequest)
		return
	}

	actor := r.Header.Get("X-User-ID")
	if actor == "" {
		actor = "api_user"
	}

	// Permission check
	opt, err := h.optRepo.GetByID(ctx, id)
	if err != nil {
		http.Error(w, "Optimization not found", http.StatusNotFound)
		return
	}

	authCtx := h.getAuthContext(r)
	result := aso.AuthorizeOptimizationAction(&authCtx, "approve", opt, nil)
	if !result.Allowed {
		http.Error(w, "Unauthorized: "+result.Reason, http.StatusForbidden)
		return
	}

	if err := h.optRepo.UpdateStatus(ctx, id, aso.OptStatusApproved, actor, ""); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	refreshedOpt, _ := h.optRepo.GetByID(ctx, id)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(refreshedOpt)
}

// RejectOptimizationRequest is the request body for rejecting
type RejectOptimizationRequest struct {
	Reason string `json:"reason"`
}

// RejectOptimization rejects an optimization
func (h *ASOHandler) RejectOptimization(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid optimization ID", http.StatusBadRequest)
		return
	}

	var req RejectOptimizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.Reason = "No reason provided"
	}

	actor := r.Header.Get("X-User-ID")
	if actor == "" {
		actor = "api_user"
	}

	// Permission check
	opt, err := h.optRepo.GetByID(ctx, id)
	if err != nil {
		http.Error(w, "Optimization not found", http.StatusNotFound)
		return
	}

	authCtx := h.getAuthContext(r)
	result := aso.AuthorizeOptimizationAction(&authCtx, "reject", opt, nil)
	if !result.Allowed {
		http.Error(w, "Unauthorized: "+result.Reason, http.StatusForbidden)
		return
	}

	if err := h.optRepo.MarkRejected(ctx, id, actor, req.Reason); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	refreshedOpt, _ := h.optRepo.GetByID(ctx, id)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(refreshedOpt)
}

func (h *ASOHandler) getAuthContext(r *http.Request) aso.AuthContext {
	userID := r.Header.Get("X-User-ID")
	rolesStr := r.Header.Get("X-User-Roles")   // assume comma separated
	tenantIDStr := jwtmiddleware.GetClaimsFromContext(r).TenantID // assume single tenant context for now

	// Simplified role parsing
	var roles []string
	if rolesStr != "" {
		// In a real app, use strings.Split(rolesStr, ",")
		// For now, assuming single role or handled by middleware
		roles = []string{rolesStr}
	} else if userID == "admin" {
		roles = []string{string(aso.RoleGoldCopyAdmin)}
	} else {
		roles = []string{string(aso.RoleTenantOps)} // Default
	}

	var tenantID *uuid.UUID
	if tenantIDStr != "" {
		if id, err := uuid.Parse(tenantIDStr); err == nil {
			tenantID = &id
		}
	}

	var tenantIDs []uuid.UUID
	if tenantID != nil {
		tenantIDs = []uuid.UUID{*tenantID}
	}

	return aso.AuthContext{
		UserID:    userID,
		Roles:     roles,
		TenantID:  tenantID,
		TenantIDs: tenantIDs,
	}
}

// ============================================================================
// Manual Trigger Endpoints
// ============================================================================

// TriggerEvaluation triggers ASO evaluation for all tenants in an environment
func (h *ASOHandler) TriggerEvaluation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	env := chi.URLParam(r, "env")

	// This would ideally trigger a Temporal workflow
	opts, err := h.engine.EvaluateAllTenants(ctx, env)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"optimizations_found": len(opts),
		"env":                 env,
	})
}

// TriggerTenantEvaluation triggers ASO evaluation for a specific tenant
func (h *ASOHandler) TriggerTenantEvaluation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	env := chi.URLParam(r, "env")
	tenantID := chi.URLParam(r, "tenantId")

	opts, err := h.engine.EvaluateTenant(ctx, env, tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"optimizations_found": len(opts),
		"env":                 env,
		"tenant_id":           tenantID,
	})
}
