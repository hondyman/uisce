package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/logging"
	"github.com/hondyman/uisce/backend/internal/metadata"
	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
	"github.com/jmoiron/sqlx"
)

// RebaseHandler handles 3-way graph rebase & conflict resolution endpoints.
type RebaseHandler struct {
	rebaseService *metadata.GoldCopyRebaseService
}

// NewRebaseHandler creates a new RebaseHandler instance.
func NewRebaseHandler(db *sqlx.DB) *RebaseHandler {
	return &RebaseHandler{
		rebaseService: metadata.NewGoldCopyRebaseService(db),
	}
}

// RegisterRoutes registers 3-way rebase governance endpoints with Chi.
func (h *RebaseHandler) RegisterRoutes(r chi.Router) {
	r.Post("/governance/rebase/dry-run", h.DryRunRebase)
	r.Post("/governance/rebase/apply", h.ApplyRebase)
	r.Get("/governance/rebase/conflicts", h.ListConflicts)
	r.Post("/governance/rebase/conflicts/{id}/resolve", h.ResolveConflict)
}

// getTenantUUIDFromRequest resolves the tenant governing 3-way rebase
// operations (dry-run/apply mutate graph state). It NEVER trusts the raw
// X-Tenant-ID header: the header is only honored when a valid JWT is
// present and ValidateTenantAccess confirms that JWT is entitled to the
// requested tenant (including core/global admins, who may select any
// tenant). See getTenantIDFromRequest in connections_routes.go for the
// same pattern.
func getTenantUUIDFromRequest(r *http.Request) uuid.UUID {
	claims, err := jwtmiddleware.ValidateTokenFromRequest(r)
	if err != nil || claims == nil {
		return uuid.Nil
	}

	headerTid := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	if headerTid == "" {
		if tid, err := uuid.Parse(claims.TenantID); err == nil {
			return tid
		}
		return uuid.Nil
	}

	if verr := jwtmiddleware.ValidateTenantAccess(claims, headerTid); verr != nil {
		return uuid.Nil
	}
	if tid, err := uuid.Parse(headerTid); err == nil {
		return tid
	}
	return uuid.Nil
}

// DryRunRebase previews the prospective 3-way merge outcome without mutations.
func (h *RebaseHandler) DryRunRebase(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID := getTenantUUIDFromRequest(r)
	if tenantID == uuid.Nil {
		http.Error(w, `{"error":"valid tenant_id is required"}`, http.StatusBadRequest)
		return
	}

	results, err := h.rebaseService.DryRunRebase(r.Context(), tenantID)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed dry-run rebase for tenant %s: %v", tenantID, err)
		http.Error(w, `{"error":"failed executing dry-run rebase"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"tenant_id":       tenantID,
		"dry_run":         true,
		"total_processed": len(results),
		"results":         results,
	})
}

// ApplyRebase executes the batch merge and records audit events.
func (h *RebaseHandler) ApplyRebase(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID := getTenantUUIDFromRequest(r)
	if tenantID == uuid.Nil {
		http.Error(w, `{"error":"valid tenant_id is required"}`, http.StatusBadRequest)
		return
	}

	results, err := h.rebaseService.ApplyRebase(r.Context(), tenantID)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed applying rebase for tenant %s: %v", tenantID, err)
		http.Error(w, `{"error":"failed applying rebase"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"tenant_id":       tenantID,
		"dry_run":         false,
		"total_processed": len(results),
		"results":         results,
	})
}

// ListConflicts retrieves active collisions from catalog_rebase_conflict_ledger.
func (h *RebaseHandler) ListConflicts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID := getTenantUUIDFromRequest(r)
	if tenantID == uuid.Nil {
		http.Error(w, `{"error":"valid tenant_id is required"}`, http.StatusBadRequest)
		return
	}

	status := r.URL.Query().Get("status")
	conflicts, err := h.rebaseService.ListConflicts(r.Context(), tenantID, status)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed listing rebase conflicts for tenant %s: %v", tenantID, err)
		http.Error(w, `{"error":"failed fetching rebase conflicts"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"tenant_id": tenantID,
		"conflicts": conflicts,
		"count":     len(conflicts),
	})
}

// ResolveConflict resolves a conflict ledger entry.
func (h *RebaseHandler) ResolveConflict(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID := getTenantUUIDFromRequest(r)
	if tenantID == uuid.Nil {
		http.Error(w, `{"error":"valid tenant_id is required"}`, http.StatusBadRequest)
		return
	}

	conflictIDStr := chi.URLParam(r, "id")
	conflictID, err := uuid.Parse(conflictIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid conflict_id"}`, http.StatusBadRequest)
		return
	}

	var req struct {
		Resolution string `json:"resolution"` // RESOLVED_TENANT_OVERRIDE or RESOLVED_GOLD_COPY_ADOPTED
		ResolvedBy string `json:"resolved_by"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Resolution == "" {
		http.Error(w, `{"error":"resolution is required (RESOLVED_TENANT_OVERRIDE | RESOLVED_GOLD_COPY_ADOPTED)"}`, http.StatusBadRequest)
		return
	}

	if req.ResolvedBy == "" {
		req.ResolvedBy = "governance_officer"
	}

	err = h.rebaseService.ResolveConflict(r.Context(), tenantID, conflictID, req.Resolution, req.ResolvedBy)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed resolving conflict %s: %v", conflictID, err)
		http.Error(w, `{"error":"failed resolving conflict"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "resolved",
		"conflict_id": conflictID,
		"resolution":  req.Resolution,
	})
}
