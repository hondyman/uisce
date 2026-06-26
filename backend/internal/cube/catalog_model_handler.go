package cube

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// CatalogModelHandler handles HTTP requests for catalog-driven Cube model management
type CatalogModelHandler struct {
	generator *CatalogModelGenerator
	security  *SecurityService
	db        *sqlx.DB
}

// NewCatalogModelHandler creates a new catalog model handler
func NewCatalogModelHandler(db *sqlx.DB) *CatalogModelHandler {
	return &CatalogModelHandler{
		generator: NewCatalogModelGenerator(db),
		security:  NewSecurityService(db),
		db:        db,
	}
}

// RegisterRoutes registers all catalog model routes
func (h *CatalogModelHandler) RegisterRoutes(r chi.Router) {
	// Core model management
	r.Get("/models/core", h.ListCoreModels)
	r.Post("/models/sync-catalog", h.SyncCoreModelsFromCatalog)
	r.Get("/models/core/{id}", h.GetCoreModel)
	r.Get("/models/core/{id}/yaml", h.GetCoreModelYAML)
	r.Delete("/models/core/{id}", h.DeleteCoreModel)

	// Custom model extensions
	r.Get("/models/custom", h.ListCustomModels)
	r.Post("/models/custom", h.CreateCustomModel)
	r.Get("/models/custom/{id}", h.GetCustomModel)
	r.Put("/models/custom/{id}", h.UpdateCustomModel)
	r.Delete("/models/custom/{id}", h.DeleteCustomModel)
	r.Get("/models/custom/{id}/yaml", h.GetCustomModelYAML)
	r.Get("/models/custom/{id}/merged-yaml", h.GetMergedModelYAML)

	// YAML generation
	r.Post("/models/generate-yaml", h.GenerateYAMLFromSpec)
	r.Post("/models/preview-yaml", h.PreviewYAML)
	r.Post("/models/validate-yaml", h.ValidateYAML)

	// Security policies
	r.Get("/security/policies", h.ListSecurityPolicies)
	r.Post("/security/policies", h.CreateSecurityPolicy)
	r.Get("/security/policies/{id}", h.GetSecurityPolicy)
	r.Put("/security/policies/{id}", h.UpdateSecurityPolicy)
	r.Delete("/security/policies/{id}", h.DeleteSecurityPolicy)
	r.Post("/security/evaluate", h.EvaluateSecurity)
	r.Get("/security/cache/stats", h.GetCacheStats)
	r.Post("/security/cache/invalidate", h.InvalidateCache)

	// Wizard sessions
	r.Post("/wizard/sessions", h.CreateWizardSession)
	r.Get("/wizard/sessions/{id}", h.GetWizardSession)
	r.Put("/wizard/sessions/{id}/steps/{step}", h.UpdateWizardStep)
	r.Post("/wizard/sessions/{id}/complete", h.CompleteWizardSession)
	r.Delete("/wizard/sessions/{id}", h.DeleteWizardSession)

	// Catalog browsing (for wizard)
	r.Get("/catalog/tables", h.ListCatalogTables)
	r.Get("/catalog/tables/{tableId}/columns", h.ListCatalogColumns)
	r.Get("/catalog/relationships", h.ListCatalogRelationships)
}

// --- Core Model Handlers ---

// ListCoreModels returns all core models for a tenant
func (h *CatalogModelHandler) ListCoreModels(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	if tenantID == "" || datasourceID == "" {
		writeError(w, http.StatusBadRequest, "tenant_id and datasource_id are required")
		return
	}

	models, err := h.generator.ListCoreModels(r.Context(), tenantID, datasourceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, models)
}

// SyncCoreModelsFromCatalog synchronizes core models from the metadata catalog
func (h *CatalogModelHandler) SyncCoreModelsFromCatalog(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	if tenantID == "" || datasourceID == "" {
		writeError(w, http.StatusBadRequest, "tenant_id and datasource_id are required")
		return
	}

	var req struct {
		Force bool `json:"force"` // Force re-sync even if models exist
	}
	json.NewDecoder(r.Body).Decode(&req)

	models, err := h.generator.GenerateCoreModelsFromCatalog(r.Context(), tenantID, datasourceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"synced_count": len(models),
		"models":       models,
	})
}

// GetCoreModel returns a single core model
func (h *CatalogModelHandler) GetCoreModel(w http.ResponseWriter, r *http.Request) {
	modelID := chi.URLParam(r, "id")
	id, err := uuid.Parse(modelID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid model ID")
		return
	}

	model, err := h.generator.GetCoreModel(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, model)
}

// GetCoreModelYAML returns the YAML representation of a core model
func (h *CatalogModelHandler) GetCoreModelYAML(w http.ResponseWriter, r *http.Request) {
	modelID := chi.URLParam(r, "id")
	id, err := uuid.Parse(modelID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid model ID")
		return
	}

	model, err := h.generator.GetCoreModel(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	yaml, err := h.generator.GenerateCubeYAML(r.Context(), model)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/x-yaml")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(yaml))
}

// DeleteCoreModel deletes a core model
func (h *CatalogModelHandler) DeleteCoreModel(w http.ResponseWriter, r *http.Request) {
	modelID := chi.URLParam(r, "id")
	tenantID := r.URL.Query().Get("tenant_id")

	id, err := uuid.Parse(modelID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid model ID")
		return
	}

	tid, err := uuid.Parse(tenantID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tenant ID")
		return
	}

	err = h.generator.DeleteCoreModel(r.Context(), tid, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// --- Custom Model Handlers ---

// ListCustomModels returns all custom models for a tenant
func (h *CatalogModelHandler) ListCustomModels(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	if tenantID == "" || datasourceID == "" {
		writeError(w, http.StatusBadRequest, "tenant_id and datasource_id are required")
		return
	}

	models, err := h.generator.ListCustomModels(r.Context(), tenantID, datasourceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, models)
}

// CreateCustomModel creates a new custom model extension
func (h *CatalogModelHandler) CreateCustomModel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID       string `json:"tenant_id"`
		DatasourceID   string `json:"datasource_id"`
		CoreModelID    string `json:"core_model_id,omitempty"`
		Name           string `json:"name"`
		Description    string `json:"description"`
		ExtensionType  string `json:"extension_type"` // extend, override, standalone
		CustomMeasures []struct {
			Name   string `json:"name"`
			Type   string `json:"type"`
			SQL    string `json:"sql"`
			Title  string `json:"title"`
			Format string `json:"format,omitempty"`
		} `json:"custom_measures,omitempty"`
		CustomDimensions []struct {
			Name  string `json:"name"`
			Type  string `json:"type"`
			SQL   string `json:"sql"`
			Title string `json:"title"`
		} `json:"custom_dimensions,omitempty"`
		CustomJoins []struct {
			Name         string `json:"name"`
			TargetCube   string `json:"target_cube"`
			Relationship string `json:"relationship"`
			SQL          string `json:"sql"`
		} `json:"custom_joins,omitempty"`
		PreAggregations []struct {
			Name          string   `json:"name"`
			Type          string   `json:"type"`
			Measures      []string `json:"measures"`
			Dimensions    []string `json:"dimensions"`
			TimeDimension string   `json:"time_dimension,omitempty"`
			Granularity   string   `json:"granularity,omitempty"`
			RefreshKey    string   `json:"refresh_key,omitempty"`
		} `json:"pre_aggregations,omitempty"`
		CreatedBy string `json:"created_by"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Build custom model
	customModel := &CustomCubeModel{
		TenantID:      uuid.MustParse(req.TenantID),
		DatasourceID:  uuid.MustParse(req.DatasourceID),
		Name:          req.Name,
		Description:   req.Description,
		ExtensionType: req.ExtensionType,
		CreatedBy:     uuid.MustParse(req.CreatedBy),
	}

	if req.CoreModelID != "" {
		coreID := uuid.MustParse(req.CoreModelID)
		customModel.CoreModelID = &coreID
	}

	// Build custom config
	config := CustomConfig{}

	for _, m := range req.CustomMeasures {
		config.Measures = append(config.Measures, CustomMeasure{
			Name:   m.Name,
			Type:   m.Type,
			SQL:    m.SQL,
			Title:  m.Title,
			Format: m.Format,
		})
	}

	for _, d := range req.CustomDimensions {
		config.Dimensions = append(config.Dimensions, CustomDimension{
			Name:  d.Name,
			Type:  d.Type,
			SQL:   d.SQL,
			Title: d.Title,
		})
	}

	for _, j := range req.CustomJoins {
		config.Joins = append(config.Joins, CustomJoin{
			Name:         j.Name,
			TargetCube:   j.TargetCube,
			Relationship: j.Relationship,
			SQL:          j.SQL,
		})
	}

	for _, p := range req.PreAggregations {
		config.PreAggregations = append(config.PreAggregations, CustomPreAgg{
			Name:          p.Name,
			Type:          p.Type,
			Measures:      p.Measures,
			Dimensions:    p.Dimensions,
			TimeDimension: p.TimeDimension,
			Granularity:   p.Granularity,
			RefreshKey:    p.RefreshKey,
		})
	}

	customModel.CustomConfig = config

	// Save
	err := h.generator.CreateCustomModel(r.Context(), customModel)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, customModel)
}

// GetCustomModel returns a single custom model
func (h *CatalogModelHandler) GetCustomModel(w http.ResponseWriter, r *http.Request) {
	modelID := chi.URLParam(r, "id")
	id, err := uuid.Parse(modelID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid model ID")
		return
	}

	model, err := h.generator.GetCustomModel(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, model)
}

// UpdateCustomModel updates an existing custom model
func (h *CatalogModelHandler) UpdateCustomModel(w http.ResponseWriter, r *http.Request) {
	modelID := chi.URLParam(r, "id")
	id, err := uuid.Parse(modelID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid model ID")
		return
	}

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Fetch existing model
	model, err := h.generator.GetCustomModel(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	// Apply updates
	if name, ok := updates["name"].(string); ok {
		model.Name = name
	}
	if desc, ok := updates["description"].(string); ok {
		model.Description = desc
	}
	if config, ok := updates["custom_config"].(map[string]interface{}); ok {
		configJSON, _ := json.Marshal(config)
		var customConfig CustomConfig
		json.Unmarshal(configJSON, &customConfig)
		model.CustomConfig = customConfig
	}

	model.Version++

	err = h.generator.UpdateCustomModel(r.Context(), model)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, model)
}

// DeleteCustomModel deletes a custom model
func (h *CatalogModelHandler) DeleteCustomModel(w http.ResponseWriter, r *http.Request) {
	modelID := chi.URLParam(r, "id")
	tenantID := r.URL.Query().Get("tenant_id")

	id, err := uuid.Parse(modelID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid model ID")
		return
	}

	tid, err := uuid.Parse(tenantID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tenant ID")
		return
	}

	err = h.generator.DeleteCustomModel(r.Context(), tid, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// GetCustomModelYAML returns the YAML for a custom model only
func (h *CatalogModelHandler) GetCustomModelYAML(w http.ResponseWriter, r *http.Request) {
	modelID := chi.URLParam(r, "id")
	id, err := uuid.Parse(modelID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid model ID")
		return
	}

	model, err := h.generator.GetCustomModel(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	yaml, err := h.generator.GenerateCustomYAML(r.Context(), model)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/x-yaml")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(yaml))
}

// GetMergedModelYAML returns the merged core + custom YAML
func (h *CatalogModelHandler) GetMergedModelYAML(w http.ResponseWriter, r *http.Request) {
	modelID := chi.URLParam(r, "id")
	id, err := uuid.Parse(modelID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid model ID")
		return
	}

	customModel, err := h.generator.GetCustomModel(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	// Get core model if this extends one
	var coreModel *CoreCubeModel
	if customModel.CoreModelID != nil {
		coreModel, err = h.generator.GetCoreModel(r.Context(), *customModel.CoreModelID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to get core model")
			return
		}
	}

	yaml, err := h.generator.MergeCoreAndCustomYAML(r.Context(), coreModel, customModel)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/x-yaml")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(yaml))
}

// --- YAML Generation Handlers ---

// GenerateYAMLFromSpec generates YAML from a model specification
func (h *CatalogModelHandler) GenerateYAMLFromSpec(w http.ResponseWriter, r *http.Request) {
	var spec struct {
		CubeName    string `json:"cube_name"`
		SQLTable    string `json:"sql_table"`
		DataSource  string `json:"data_source"`
		Description string `json:"description"`
		Measures    []struct {
			Name   string `json:"name"`
			Type   string `json:"type"`
			SQL    string `json:"sql"`
			Title  string `json:"title"`
			Format string `json:"format,omitempty"`
		} `json:"measures"`
		Dimensions []struct {
			Name       string `json:"name"`
			Type       string `json:"type"`
			SQL        string `json:"sql"`
			Title      string `json:"title"`
			PrimaryKey bool   `json:"primary_key,omitempty"`
		} `json:"dimensions"`
		Joins []struct {
			Name         string `json:"name"`
			TargetCube   string `json:"target_cube"`
			Relationship string `json:"relationship"`
			SQL          string `json:"sql"`
		} `json:"joins,omitempty"`
		PreAggregations []struct {
			Name          string   `json:"name"`
			Type          string   `json:"type"`
			Measures      []string `json:"measures"`
			Dimensions    []string `json:"dimensions"`
			TimeDimension string   `json:"time_dimension,omitempty"`
			Granularity   string   `json:"granularity,omitempty"`
		} `json:"pre_aggregations,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&spec); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	yaml := h.generator.GenerateYAMLFromSpec(spec.CubeName, spec.SQLTable, spec.DataSource, spec.Description,
		spec.Measures, spec.Dimensions, spec.Joins, spec.PreAggregations)

	w.Header().Set("Content-Type", "application/x-yaml")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(yaml))
}

// PreviewYAML previews generated YAML without saving
func (h *CatalogModelHandler) PreviewYAML(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CoreModelID   string                 `json:"core_model_id,omitempty"`
		CustomConfig  map[string]interface{} `json:"custom_config,omitempty"`
		ExtensionType string                 `json:"extension_type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var yamlContent string
	var err error

	if req.CoreModelID != "" {
		coreID := uuid.MustParse(req.CoreModelID)
		coreModel, getErr := h.generator.GetCoreModel(r.Context(), coreID)
		if getErr != nil {
			writeError(w, http.StatusNotFound, "core model not found")
			return
		}

		// Build custom model for preview
		customModel := &CustomCubeModel{
			ExtensionType: req.ExtensionType,
		}
		if req.CustomConfig != nil {
			configJSON, _ := json.Marshal(req.CustomConfig)
			json.Unmarshal(configJSON, &customModel.CustomConfig)
		}

		yamlContent, err = h.generator.MergeCoreAndCustomYAML(r.Context(), coreModel, customModel)
	} else {
		yamlContent, err = h.generator.GenerateCubeYAML(r.Context(), &CoreCubeModel{})
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/x-yaml")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(yamlContent))
}

// ValidateYAML validates Cube YAML syntax
func (h *CatalogModelHandler) ValidateYAML(w http.ResponseWriter, r *http.Request) {
	var req struct {
		YAML string `json:"yaml"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	errors := h.generator.ValidateCubeYAML(req.YAML)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"valid":  len(errors) == 0,
		"errors": errors,
	})
}

// --- Security Policy Handlers ---

// ListSecurityPolicies returns all security policies for a tenant
func (h *CatalogModelHandler) ListSecurityPolicies(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		writeError(w, http.StatusBadRequest, "tenant_id is required")
		return
	}

	tid, err := uuid.Parse(tenantID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tenant ID")
		return
	}

	policies, err := h.security.ListPolicies(r.Context(), tid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, policies)
}

// CreateSecurityPolicy creates a new ABAC policy
func (h *CatalogModelHandler) CreateSecurityPolicy(w http.ResponseWriter, r *http.Request) {
	var policy ABACPolicy
	if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	err := h.security.CreatePolicy(r.Context(), &policy)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, policy)
}

// GetSecurityPolicy returns a single security policy
func (h *CatalogModelHandler) GetSecurityPolicy(w http.ResponseWriter, r *http.Request) {
	policyID := chi.URLParam(r, "id")
	tenantID := r.URL.Query().Get("tenant_id")

	pid, err := uuid.Parse(policyID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid policy ID")
		return
	}

	tid, err := uuid.Parse(tenantID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tenant ID")
		return
	}

	policy, err := h.security.GetPolicy(r.Context(), tid, pid)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, policy)
}

// UpdateSecurityPolicy updates an existing security policy
func (h *CatalogModelHandler) UpdateSecurityPolicy(w http.ResponseWriter, r *http.Request) {
	policyID := chi.URLParam(r, "id")

	pid, err := uuid.Parse(policyID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid policy ID")
		return
	}

	var policy ABACPolicy
	if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	policy.ID = pid
	err = h.security.UpdatePolicy(r.Context(), &policy)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, policy)
}

// DeleteSecurityPolicy deletes a security policy
func (h *CatalogModelHandler) DeleteSecurityPolicy(w http.ResponseWriter, r *http.Request) {
	policyID := chi.URLParam(r, "id")
	tenantID := r.URL.Query().Get("tenant_id")

	pid, err := uuid.Parse(policyID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid policy ID")
		return
	}

	tid, err := uuid.Parse(tenantID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tenant ID")
		return
	}

	err = h.security.DeletePolicy(r.Context(), tid, pid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// EvaluateSecurity evaluates security policies for a given context
func (h *CatalogModelHandler) EvaluateSecurity(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SecurityContext SecurityContext `json:"security_context"`
		Cubes           []string        `json:"cubes,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	decision, err := h.security.EvaluateSecurity(r.Context(), req.SecurityContext, req.Cubes)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, decision)
}

// GetCacheStats returns security cache statistics
func (h *CatalogModelHandler) GetCacheStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.security.GetCacheStats(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

// InvalidateCache invalidates the security cache for a tenant
func (h *CatalogModelHandler) InvalidateCache(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		writeError(w, http.StatusBadRequest, "tenant_id is required")
		return
	}

	err := h.security.InvalidateCache(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "cache invalidated"})
}

// --- Wizard Session Handlers ---

// WizardSession represents a model builder wizard session
type WizardSession struct {
	ID            uuid.UUID              `json:"id" db:"id"`
	TenantID      uuid.UUID              `json:"tenant_id" db:"tenant_id"`
	DatasourceID  uuid.UUID              `json:"datasource_id" db:"datasource_id"`
	SessionType   string                 `json:"session_type" db:"session_type"` // core, custom, extension
	CurrentStep   int                    `json:"current_step" db:"current_step"`
	TotalSteps    int                    `json:"total_steps" db:"total_steps"`
	SessionData   map[string]interface{} `json:"session_data" db:"session_data"`
	Status        string                 `json:"status" db:"status"` // in_progress, completed, cancelled
	CreatedBy     uuid.UUID              `json:"created_by" db:"created_by"`
	CreatedAt     string                 `json:"created_at" db:"created_at"`
	UpdatedAt     string                 `json:"updated_at" db:"updated_at"`
	CompletedAt   *string                `json:"completed_at,omitempty" db:"completed_at"`
	ResultModelID *uuid.UUID             `json:"result_model_id,omitempty" db:"result_model_id"`
}

// CreateWizardSession creates a new wizard session
func (h *CatalogModelHandler) CreateWizardSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID     string `json:"tenant_id"`
		DatasourceID string `json:"datasource_id"`
		SessionType  string `json:"session_type"`
		CreatedBy    string `json:"created_by"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	session := &WizardSession{
		ID:           uuid.New(),
		TenantID:     uuid.MustParse(req.TenantID),
		DatasourceID: uuid.MustParse(req.DatasourceID),
		SessionType:  req.SessionType,
		CurrentStep:  1,
		TotalSteps:   6, // Default wizard steps
		SessionData:  make(map[string]interface{}),
		Status:       "in_progress",
		CreatedBy:    uuid.MustParse(req.CreatedBy),
	}

	sessionDataJSON, _ := json.Marshal(session.SessionData)

	_, err := h.db.ExecContext(r.Context(), `
		INSERT INTO cube_model_builder_sessions (id, tenant_id, datasource_id, session_type, current_step, total_steps, session_data, status, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, session.ID, session.TenantID, session.DatasourceID, session.SessionType, session.CurrentStep,
		session.TotalSteps, sessionDataJSON, session.Status, session.CreatedBy)

	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, session)
}

// GetWizardSession returns a wizard session
func (h *CatalogModelHandler) GetWizardSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	id, err := uuid.Parse(sessionID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid session ID")
		return
	}

	var session struct {
		ID            uuid.UUID       `db:"id"`
		TenantID      uuid.UUID       `db:"tenant_id"`
		DatasourceID  uuid.UUID       `db:"datasource_id"`
		SessionType   string          `db:"session_type"`
		CurrentStep   int             `db:"current_step"`
		TotalSteps    int             `db:"total_steps"`
		SessionData   json.RawMessage `db:"session_data"`
		Status        string          `db:"status"`
		CreatedBy     uuid.UUID       `db:"created_by"`
		CreatedAt     string          `db:"created_at"`
		UpdatedAt     string          `db:"updated_at"`
		CompletedAt   *string         `db:"completed_at"`
		ResultModelID *uuid.UUID      `db:"result_model_id"`
	}

	err = h.db.GetContext(r.Context(), &session, `
		SELECT id, tenant_id, datasource_id, session_type, current_step, total_steps, 
		       session_data, status, created_by, created_at, updated_at, completed_at, result_model_id
		FROM cube_model_builder_sessions WHERE id = $1
	`, id)

	if err != nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	// Parse session data
	var sessionData map[string]interface{}
	json.Unmarshal(session.SessionData, &sessionData)

	// Also get step data
	var steps []struct {
		StepNumber int             `db:"step_number"`
		StepType   string          `db:"step_type"`
		StepData   json.RawMessage `db:"step_data"`
		Completed  bool            `db:"completed"`
	}

	h.db.SelectContext(r.Context(), &steps, `
		SELECT step_number, step_type, step_data, completed
		FROM cube_model_builder_steps WHERE session_id = $1 ORDER BY step_number
	`, id)

	stepsData := make([]map[string]interface{}, len(steps))
	for i, s := range steps {
		var data map[string]interface{}
		json.Unmarshal(s.StepData, &data)
		stepsData[i] = map[string]interface{}{
			"step_number": s.StepNumber,
			"step_type":   s.StepType,
			"step_data":   data,
			"completed":   s.Completed,
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":              session.ID,
		"tenant_id":       session.TenantID,
		"datasource_id":   session.DatasourceID,
		"session_type":    session.SessionType,
		"current_step":    session.CurrentStep,
		"total_steps":     session.TotalSteps,
		"session_data":    sessionData,
		"status":          session.Status,
		"created_by":      session.CreatedBy,
		"created_at":      session.CreatedAt,
		"updated_at":      session.UpdatedAt,
		"completed_at":    session.CompletedAt,
		"result_model_id": session.ResultModelID,
		"steps":           stepsData,
	})
}

// UpdateWizardStep updates a specific step in the wizard
func (h *CatalogModelHandler) UpdateWizardStep(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	stepStr := chi.URLParam(r, "step")

	id, err := uuid.Parse(sessionID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid session ID")
		return
	}

	stepNum, err := strconv.Atoi(stepStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid step number")
		return
	}

	var req struct {
		StepType  string                 `json:"step_type"`
		StepData  map[string]interface{} `json:"step_data"`
		Completed bool                   `json:"completed"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	stepDataJSON, _ := json.Marshal(req.StepData)

	// Upsert step data
	_, err = h.db.ExecContext(r.Context(), `
		INSERT INTO cube_model_builder_steps (session_id, step_number, step_type, step_data, completed)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (session_id, step_number) DO UPDATE SET
			step_type = EXCLUDED.step_type,
			step_data = EXCLUDED.step_data,
			completed = EXCLUDED.completed,
			updated_at = NOW()
	`, id, stepNum, req.StepType, stepDataJSON, req.Completed)

	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Update session current step
	if req.Completed {
		_, _ = h.db.ExecContext(r.Context(), `
			UPDATE cube_model_builder_sessions SET current_step = $1, updated_at = NOW() WHERE id = $2
		`, stepNum+1, id)
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "step updated"})
}

// CompleteWizardSession completes a wizard session and generates the model
func (h *CatalogModelHandler) CompleteWizardSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	id, err := uuid.Parse(sessionID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid session ID")
		return
	}

	// Get session and all steps
	var session struct {
		ID           uuid.UUID       `db:"id"`
		TenantID     uuid.UUID       `db:"tenant_id"`
		DatasourceID uuid.UUID       `db:"datasource_id"`
		SessionType  string          `db:"session_type"`
		SessionData  json.RawMessage `db:"session_data"`
		CreatedBy    uuid.UUID       `db:"created_by"`
	}

	err = h.db.GetContext(r.Context(), &session, `
		SELECT id, tenant_id, datasource_id, session_type, session_data, created_by
		FROM cube_model_builder_sessions WHERE id = $1
	`, id)

	if err != nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	var steps []struct {
		StepNumber int             `db:"step_number"`
		StepType   string          `db:"step_type"`
		StepData   json.RawMessage `db:"step_data"`
	}

	h.db.SelectContext(r.Context(), &steps, `
		SELECT step_number, step_type, step_data
		FROM cube_model_builder_steps WHERE session_id = $1 ORDER BY step_number
	`, id)

	// Consolidate step data
	allStepData := make(map[string]interface{})
	for _, s := range steps {
		var data map[string]interface{}
		json.Unmarshal(s.StepData, &data)
		allStepData[s.StepType] = data
	}

	// Generate model based on session type
	var resultModelID uuid.UUID

	switch session.SessionType {
	case "core":
		// Sync from catalog
		models, err := h.generator.GenerateCoreModelsFromCatalog(r.Context(), session.TenantID.String(), session.DatasourceID.String())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if len(models) > 0 {
			resultModelID = models[0].ID
		}

	case "custom", "extension":
		// Create custom model from wizard data
		customModel := h.buildCustomModelFromWizard(session.TenantID, session.DatasourceID, session.CreatedBy, allStepData)
		err = h.generator.CreateCustomModel(r.Context(), customModel)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		resultModelID = customModel.ID
	}

	// Update session as completed
	_, err = h.db.ExecContext(r.Context(), `
		UPDATE cube_model_builder_sessions 
		SET status = 'completed', completed_at = NOW(), result_model_id = $1, updated_at = NOW()
		WHERE id = $2
	`, resultModelID, id)

	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":          "completed",
		"result_model_id": resultModelID,
	})
}

// DeleteWizardSession deletes/cancels a wizard session
func (h *CatalogModelHandler) DeleteWizardSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	id, err := uuid.Parse(sessionID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid session ID")
		return
	}

	// Delete steps first
	_, _ = h.db.ExecContext(r.Context(), `DELETE FROM cube_model_builder_steps WHERE session_id = $1`, id)

	// Delete session
	_, err = h.db.ExecContext(r.Context(), `DELETE FROM cube_model_builder_sessions WHERE id = $1`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *CatalogModelHandler) buildCustomModelFromWizard(tenantID, datasourceID, createdBy uuid.UUID, stepData map[string]interface{}) *CustomCubeModel {
	model := &CustomCubeModel{
		ID:           uuid.New(),
		TenantID:     tenantID,
		DatasourceID: datasourceID,
		CreatedBy:    createdBy,
		Version:      1,
	}

	// Extract from step data
	if sourceStep, ok := stepData["source_selection"].(map[string]interface{}); ok {
		if name, ok := sourceStep["cube_name"].(string); ok {
			model.Name = name
		}
		if desc, ok := sourceStep["description"].(string); ok {
			model.Description = desc
		}
		if coreID, ok := sourceStep["core_model_id"].(string); ok && coreID != "" {
			id := uuid.MustParse(coreID)
			model.CoreModelID = &id
			model.ExtensionType = "extend"
		} else {
			model.ExtensionType = "standalone"
		}
	}

	config := CustomConfig{}

	// Extract measures
	if measuresStep, ok := stepData["measures_config"].(map[string]interface{}); ok {
		if measures, ok := measuresStep["measures"].([]interface{}); ok {
			for _, m := range measures {
				if mMap, ok := m.(map[string]interface{}); ok {
					measure := CustomMeasure{
						Name:  getString(mMap, "name"),
						Type:  getString(mMap, "type"),
						SQL:   getString(mMap, "sql"),
						Title: getString(mMap, "title"),
					}
					config.Measures = append(config.Measures, measure)
				}
			}
		}
	}

	// Extract dimensions
	if dimsStep, ok := stepData["dimensions_config"].(map[string]interface{}); ok {
		if dims, ok := dimsStep["dimensions"].([]interface{}); ok {
			for _, d := range dims {
				if dMap, ok := d.(map[string]interface{}); ok {
					dim := CustomDimension{
						Name:  getString(dMap, "name"),
						Type:  getString(dMap, "type"),
						SQL:   getString(dMap, "sql"),
						Title: getString(dMap, "title"),
					}
					config.Dimensions = append(config.Dimensions, dim)
				}
			}
		}
	}

	// Extract joins
	if joinsStep, ok := stepData["relationships_config"].(map[string]interface{}); ok {
		if joins, ok := joinsStep["joins"].([]interface{}); ok {
			for _, j := range joins {
				if jMap, ok := j.(map[string]interface{}); ok {
					join := CustomJoin{
						Name:         getString(jMap, "name"),
						TargetCube:   getString(jMap, "target_cube"),
						Relationship: getString(jMap, "relationship"),
						SQL:          getString(jMap, "sql"),
					}
					config.Joins = append(config.Joins, join)
				}
			}
		}
	}

	// Extract pre-aggregations
	if preAggStep, ok := stepData["preagg_config"].(map[string]interface{}); ok {
		if preAggs, ok := preAggStep["pre_aggregations"].([]interface{}); ok {
			for _, p := range preAggs {
				if pMap, ok := p.(map[string]interface{}); ok {
					preAgg := CustomPreAgg{
						Name:          getString(pMap, "name"),
						Type:          getString(pMap, "type"),
						TimeDimension: getString(pMap, "time_dimension"),
						Granularity:   getString(pMap, "granularity"),
					}
					if measures, ok := pMap["measures"].([]interface{}); ok {
						for _, m := range measures {
							preAgg.Measures = append(preAgg.Measures, fmt.Sprintf("%v", m))
						}
					}
					if dims, ok := pMap["dimensions"].([]interface{}); ok {
						for _, d := range dims {
							preAgg.Dimensions = append(preAgg.Dimensions, fmt.Sprintf("%v", d))
						}
					}
					config.PreAggregations = append(config.PreAggregations, preAgg)
				}
			}
		}
	}

	model.CustomConfig = config
	return model
}

// --- Catalog Browsing Handlers ---

// ListCatalogTables lists tables from the metadata catalog
func (h *CatalogModelHandler) ListCatalogTables(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	if tenantID == "" || datasourceID == "" {
		writeError(w, http.StatusBadRequest, "tenant_id and datasource_id are required")
		return
	}

	var tables []struct {
		ID          uuid.UUID `db:"id" json:"id"`
		Name        string    `db:"name" json:"name"`
		DisplayName string    `db:"display_name" json:"display_name"`
		Description string    `db:"description" json:"description"`
		Schema      string    `db:"schema_name" json:"schema"`
		IsCore      bool      `db:"is_core" json:"is_core"`
	}

	// Query catalog_node for table-type nodes
	err := h.db.SelectContext(r.Context(), &tables, `
		SELECT cn.id, cn.name, 
		       COALESCE(cn.properties->>'display_name', cn.name) as display_name,
		       COALESCE(cn.properties->>'description', '') as description,
		       COALESCE(cn.properties->>'schema', 'public') as schema_name,
		       COALESCE((cn.properties->>'is_core')::boolean, false) as is_core
		FROM catalog_node cn
		JOIN catalog_node_type cnt ON cn.node_type_id = cnt.id
		WHERE cn.tenant_id = $1 
		  AND cnt.name IN ('table', 'view', 'semantic_model')
		  AND (cn.properties->>'datasource_id' = $2 OR cn.properties->>'datasource_id' IS NULL)
		ORDER BY cn.name
	`, tenantID, datasourceID)

	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, tables)
}

// ListCatalogColumns lists columns for a catalog table
func (h *CatalogModelHandler) ListCatalogColumns(w http.ResponseWriter, r *http.Request) {
	tableID := chi.URLParam(r, "tableId")

	tid, err := uuid.Parse(tableID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid table ID")
		return
	}

	var columns []struct {
		ID          uuid.UUID `db:"id" json:"id"`
		Name        string    `db:"name" json:"name"`
		DisplayName string    `db:"display_name" json:"display_name"`
		DataType    string    `db:"data_type" json:"data_type"`
		Description string    `db:"description" json:"description"`
		IsPK        bool      `db:"is_pk" json:"is_primary_key"`
		IsFK        bool      `db:"is_fk" json:"is_foreign_key"`
		IsCore      bool      `db:"is_core" json:"is_core"`
	}

	// Query columns linked to this table
	err = h.db.SelectContext(r.Context(), &columns, `
		SELECT cn.id, cn.name,
		       COALESCE(cn.properties->>'display_name', cn.name) as display_name,
		       COALESCE(cn.properties->>'data_type', 'string') as data_type,
		       COALESCE(cn.properties->>'description', '') as description,
		       COALESCE((cn.properties->>'is_primary_key')::boolean, false) as is_pk,
		       COALESCE((cn.properties->>'is_foreign_key')::boolean, false) as is_fk,
		       COALESCE((cn.properties->>'is_core')::boolean, false) as is_core
		FROM catalog_node cn
		JOIN catalog_node_type cnt ON cn.node_type_id = cnt.id
		JOIN catalog_edge ce ON ce.target_node_id = cn.id
		WHERE ce.source_node_id = $1
		  AND cnt.name IN ('column', 'semantic_column')
		ORDER BY cn.name
	`, tid)

	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, columns)
}

// ListCatalogRelationships lists relationships between catalog tables
func (h *CatalogModelHandler) ListCatalogRelationships(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	_ = r.URL.Query().Get("datasource_id") // May be used for filtering

	if tenantID == "" {
		writeError(w, http.StatusBadRequest, "tenant_id is required")
		return
	}

	var relationships []struct {
		ID           uuid.UUID `db:"id" json:"id"`
		SourceTable  string    `db:"source_table" json:"source_table"`
		SourceColumn string    `db:"source_column" json:"source_column"`
		TargetTable  string    `db:"target_table" json:"target_table"`
		TargetColumn string    `db:"target_column" json:"target_column"`
		RelationType string    `db:"relation_type" json:"relation_type"`
	}

	// Query foreign key relationships from catalog
	err := h.db.SelectContext(r.Context(), &relationships, `
		SELECT ce.id,
		       src.name as source_table,
		       COALESCE(ce.properties->>'source_column', '') as source_column,
		       tgt.name as target_table,
		       COALESCE(ce.properties->>'target_column', '') as target_column,
		       COALESCE(cet.name, 'foreign_key') as relation_type
		FROM catalog_edge ce
		JOIN catalog_node src ON ce.source_node_id = src.id
		JOIN catalog_node tgt ON ce.target_node_id = tgt.id
		JOIN catalog_edge_type cet ON ce.edge_type_id = cet.id
		WHERE src.tenant_id = $1
		  AND cet.name IN ('foreign_key', 'references', 'join')
		ORDER BY src.name, tgt.name
	`, tenantID)

	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, relationships)
}

// --- Helper Functions ---

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
