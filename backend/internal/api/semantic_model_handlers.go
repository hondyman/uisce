package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/services"
	"go.uber.org/zap"
)

// ============================================================================
// SEMANTIC MODEL API HANDLERS
// ============================================================================

type SemanticModelHandler struct {
	service *services.SemanticModelInheritanceService
	logger  *zap.Logger
}

func NewSemanticModelHandler(service *services.SemanticModelInheritanceService) *SemanticModelHandler {
	logger, _ := zap.NewProduction()
	return &SemanticModelHandler{
		service: service,
		logger:  logger,
	}
}

func (h *SemanticModelHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/semantic-models", func(r chi.Router) {
		r.Get("/core", h.GetCoreModels)
		r.Get("/tenant", h.GetTenantModels)
		r.Post("/provision", h.ProvisionModel)
		r.Get("/{id}", h.GetModelDetails)
		r.Post("/{id}/sync", h.SyncWithBO)
		r.Post("/{id}/dimensions", h.AddDimension)
		r.Put("/dimensions/{dimId}", h.OverrideDimension)
	})
}

// GetCoreModels returns all core semantic models (templates)
func (h *SemanticModelHandler) GetCoreModels(w http.ResponseWriter, r *http.Request) {
	models, err := h.service.GetCoreModels(r.Context())
	if err != nil {
		h.logger.Error("Failed to get core models", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models)
}

// GetTenantModels returns all custom models for a tenant
func (h *SemanticModelHandler) GetTenantModels(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		http.Error(w, "tenant_id required", http.StatusBadRequest)
		return
	}

	models, err := h.service.GetTenantModels(r.Context(), tenantID)
	if err != nil {
		h.logger.Error("Failed to get tenant models", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models)
}

// ProvisionModel creates a custom model from a core template
func (h *SemanticModelHandler) ProvisionModel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID     string  `json:"tenant_id"`
		CoreCubeID   string  `json:"core_cube_id"`
		DatasourceID *string `json:"datasource_id,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	modelID, err := h.service.ProvisionTenantModel(r.Context(), req.TenantID, req.CoreCubeID, req.DatasourceID)
	if err != nil {
		h.logger.Error("Failed to provision model", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"id": modelID})
}

// GetModelDetails returns a model with all dimensions and measures
func (h *SemanticModelHandler) GetModelDetails(w http.ResponseWriter, r *http.Request) {
	modelID := chi.URLParam(r, "id")

	model, dimensions, measures, err := h.service.GetModelWithInheritance(r.Context(), modelID)
	if err != nil {
		h.logger.Error("Failed to get model details", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"model":      model,
		"dimensions": dimensions,
		"measures":   measures,
	})
}

// SyncWithBO synchronizes a model with its business object
func (h *SemanticModelHandler) SyncWithBO(w http.ResponseWriter, r *http.Request) {
	modelID := chi.URLParam(r, "id")

	count, err := h.service.SyncModelWithBO(r.Context(), modelID)
	if err != nil {
		h.logger.Error("Failed to sync model", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"synced_fields": count,
		"message":       "Model synced successfully",
	})
}

// AddDimension adds a custom dimension to a model
func (h *SemanticModelHandler) AddDimension(w http.ResponseWriter, r *http.Request) {
	modelID := chi.URLParam(r, "id")

	var dim services.SemanticDimension
	if err := json.NewDecoder(r.Body).Decode(&dim); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dimID, err := h.service.AddCustomDimension(r.Context(), modelID, dim)
	if err != nil {
		h.logger.Error("Failed to add dimension", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"id": dimID})
}

// OverrideDimension overrides an inherited dimension
func (h *SemanticModelHandler) OverrideDimension(w http.ResponseWriter, r *http.Request) {
	dimID := chi.URLParam(r, "dimId")

	var req struct {
		SQL   string `json:"sql"`
		Label string `json:"label"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.service.OverrideDimension(r.Context(), dimID, req.SQL, req.Label); err != nil {
		h.logger.Error("Failed to override dimension", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Dimension overridden successfully"})
}
