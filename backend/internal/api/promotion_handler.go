package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/aso"
)

// PromotionHandler handles semantic promotion API endpoints
type PromotionHandler struct {
	promotionService aso.PromotionService
}

// NewPromotionHandler creates a new promotion handler
func NewPromotionHandler(promotionService aso.PromotionService) *PromotionHandler {
	return &PromotionHandler{
		promotionService: promotionService,
	}
}

// RegisterPromotionRoutes registers all promotion routes
func RegisterPromotionRoutes(r chi.Router, h *PromotionHandler) {
	r.Route("/promotions", func(r chi.Router) {
		// ChangeSet management
		r.Get("/changesets", h.ListChangeSets)
		r.Post("/changesets", h.CreateChangeSet)
		r.Get("/changesets/{id}", h.GetChangeSet)
		r.Post("/changesets/{id}/validate", h.ValidateChangeSet)
		r.Post("/changesets/{id}/approve", h.ApproveChangeSet)
		r.Post("/changesets/{id}/apply", h.ApplyChangeSet)
		r.Post("/changesets/{id}/reject", h.RejectChangeSet)

		// ASO-generated changesets
		r.Post("/changesets/from-aso", h.CreateASOChangeSet)
	})
}

// ============================================================================
// ChangeSet Endpoints
// ============================================================================

// ListChangeSets lists changesets with filters
func (h *PromotionHandler) ListChangeSets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	filter := aso.ChangeSetFilter{Limit: 50}

	if status := r.URL.Query().Get("status"); status != "" {
		s := aso.ChangeSetStatus(status)
		filter.Status = &s
	}
	if sourceEnv := r.URL.Query().Get("source_env"); sourceEnv != "" {
		filter.SourceEnv = &sourceEnv
	}
	if targetEnv := r.URL.Query().Get("target_env"); targetEnv != "" {
		filter.TargetEnv = &targetEnv
	}
	if asoSource := r.URL.Query().Get("aso_source"); asoSource == "true" {
		b := true
		filter.ASOSource = &b
	}

	changesets, err := h.promotionService.ListChangeSets(ctx, filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(changesets)
}

// CreateChangeSetRequest is the request body for creating a changeset
type CreateChangeSetRequest struct {
	TenantID    *string              `json:"tenant_id,omitempty"`
	SourceEnv   string               `json:"source_env"`
	TargetEnv   string               `json:"target_env"`
	Description string               `json:"description"`
	Changes     []aso.SemanticChange `json:"changes"`
}

// CreateChangeSet creates a new changeset
func (h *PromotionHandler) CreateChangeSet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateChangeSetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	actor := r.Header.Get("X-User-ID")
	if actor == "" {
		actor = "api_user"
	}

	var tenantID *uuid.UUID
	if req.TenantID != nil {
		id, err := uuid.Parse(*req.TenantID)
		if err == nil {
			tenantID = &id
		}
	}

	cs := &aso.SemanticChangeSet{
		TenantID:    tenantID,
		SourceEnv:   req.SourceEnv,
		TargetEnv:   req.TargetEnv,
		Description: req.Description,
		Changes:     req.Changes,
		CreatedBy:   actor,
	}

	if err := h.promotionService.CreateChangeSet(ctx, cs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(cs)
}

// GetChangeSet retrieves a changeset by ID
func (h *PromotionHandler) GetChangeSet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid changeset ID", http.StatusBadRequest)
		return
	}

	cs, err := h.promotionService.GetChangeSet(ctx, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cs)
}

// ValidateChangeSet runs ASO validation on a changeset
func (h *PromotionHandler) ValidateChangeSet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid changeset ID", http.StatusBadRequest)
		return
	}

	result, err := h.promotionService.ValidateChangeSet(ctx, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// ApproveChangeSet approves a changeset
func (h *PromotionHandler) ApproveChangeSet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid changeset ID", http.StatusBadRequest)
		return
	}

	approver := r.Header.Get("X-User-ID")
	if approver == "" {
		approver = "api_user"
	}

	if err := h.promotionService.ApproveChangeSet(ctx, id, approver); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cs, _ := h.promotionService.GetChangeSet(ctx, id)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cs)
}

// ApplyChangeSet applies a changeset
func (h *PromotionHandler) ApplyChangeSet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid changeset ID", http.StatusBadRequest)
		return
	}

	applier := r.Header.Get("X-User-ID")
	if applier == "" {
		applier = "api_user"
	}

	if err := h.promotionService.ApplyChangeSet(ctx, id, applier); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cs, _ := h.promotionService.GetChangeSet(ctx, id)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cs)
}

// RejectChangeSetRequest is the request body for rejecting
type RejectChangeSetRequest struct {
	Reason string `json:"reason"`
}

// RejectChangeSet rejects a changeset
func (h *PromotionHandler) RejectChangeSet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid changeset ID", http.StatusBadRequest)
		return
	}

	var req RejectChangeSetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.Reason = "No reason provided"
	}

	rejector := r.Header.Get("X-User-ID")
	if rejector == "" {
		rejector = "api_user"
	}

	if err := h.promotionService.RejectChangeSet(ctx, id, rejector, req.Reason); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	cs, _ := h.promotionService.GetChangeSet(ctx, id)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cs)
}

// CreateASOChangeSetRequest is the request for creating from ASO optimizations
type CreateASOChangeSetRequest struct {
	Env             string   `json:"env"`
	OptimizationIDs []string `json:"optimization_ids"`
}

// CreateASOChangeSet creates a changeset from ASO optimizations
func (h *PromotionHandler) CreateASOChangeSet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateASOChangeSetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	creator := r.Header.Get("X-User-ID")
	if creator == "" {
		creator = "api_user"
	}

	var optIDs []uuid.UUID
	for _, idStr := range req.OptimizationIDs {
		id, err := uuid.Parse(idStr)
		if err == nil {
			optIDs = append(optIDs, id)
		}
	}

	cs, err := h.promotionService.CreateASOChangeSet(ctx, req.Env, optIDs, creator)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(cs)
}
