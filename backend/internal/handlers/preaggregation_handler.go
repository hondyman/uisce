package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/hondyman/uisce/backend/internal/logging"
	"github.com/hondyman/uisce/backend/internal/models"
)

// PreAggregationHandler exposes CRUD + DDL generation for pre-aggregation
// catalog nodes (the StarRocks hot-tier rollup definitions used to serve
// calculated semantic terms without recomputing them per-query).
type PreAggregationHandler struct {
	svc *analytics.PreAggregationService
}

func NewPreAggregationHandler(svc *analytics.PreAggregationService) *PreAggregationHandler {
	return &PreAggregationHandler{svc: svc}
}

// RegisterRoutes adds the pre-aggregation routes to the given chi.Router.
func (h *PreAggregationHandler) RegisterRoutes(r chi.Router) {
	r.Route("/preaggregations", func(r chi.Router) {
		r.Post("/", h.handleUpsert)
		r.Get("/", h.handleListByBO)
		r.Get("/{id}", h.handleGetByID)
		r.Put("/{id}", h.handleUpdate)
		r.Delete("/{id}", h.handleDelete)
		r.Post("/{id}/ddl", h.handleGenerateDDL)
		r.Post("/{id}/refresh", h.handleRefresh)
	})
}

func (h *PreAggregationHandler) handleUpsert(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := mustTenantID(r)
	if !ok {
		http.Error(w, "tenant_id is required", http.StatusUnauthorized)
		return
	}
	var req models.UpsertPreAggRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	req.TenantID = tenantID.String()

	desc, err := h.svc.UpsertPreAggregation(r.Context(), req)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("preaggregation: upsert failed: %v", err)
		http.Error(w, "failed to upsert pre-aggregation", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(desc)
}

func (h *PreAggregationHandler) handleListByBO(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := mustTenantID(r)
	if !ok {
		http.Error(w, "tenant_id is required", http.StatusUnauthorized)
		return
	}
	boName := r.URL.Query().Get("bo_name")
	list, err := h.svc.ListByBO(r.Context(), tenantID.String(), boName)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("preaggregation: list failed: %v", err)
		http.Error(w, "failed to list pre-aggregations", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"preAggregations": list})
}

func (h *PreAggregationHandler) handleGetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	desc, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "pre-aggregation not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(desc)
}

func (h *PreAggregationHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := mustTenantID(r)
	if !ok {
		http.Error(w, "tenant_id is required", http.StatusUnauthorized)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var req models.UpsertPreAggRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	req.TenantID = tenantID.String()

	desc, err := h.svc.Update(r.Context(), id, req)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("preaggregation: update failed: %v", err)
		http.Error(w, "failed to update pre-aggregation", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(desc)
}

func (h *PreAggregationHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		logging.GetLogger().Sugar().Errorf("preaggregation: delete failed: %v", err)
		http.Error(w, "failed to delete pre-aggregation", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *PreAggregationHandler) handleGenerateDDL(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	dialect := r.URL.Query().Get("dialect")
	if dialect == "" {
		dialect = "starrocks"
	}
	ddl, err := h.svc.GenerateDDL(r.Context(), id, dialect)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("preaggregation: generate DDL failed: %v", err)
		http.Error(w, "failed to generate DDL", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"ddl": ddl})
}

func (h *PreAggregationHandler) handleRefresh(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := h.svc.Refresh(r.Context(), id); err != nil {
		logging.GetLogger().Sugar().Errorf("preaggregation: refresh failed: %v", err)
		http.Error(w, "failed to refresh pre-aggregation", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
