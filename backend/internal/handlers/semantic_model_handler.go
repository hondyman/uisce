package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/analytics"
	"github.com/hondyman/semlayer/backend/internal/cube"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/hondyman/semlayer/backend/internal/scanner"
	models "github.com/hondyman/semlayer/backend/models"
)

type SemanticModelHandler struct {
	Service *analytics.SemanticModelService
}

// NewSemanticModelHandler creates a new SemanticModelHandler
func NewSemanticModelHandler(service *analytics.SemanticModelService) *SemanticModelHandler {
	return &SemanticModelHandler{Service: service}
}

// GenerateDefaultModelRequest defines the structure for the request
type GenerateDefaultModelRequest struct {
	DatasourceID string `json:"datasource_id" binding:"required"`
}

// GenerateModelForTableRequest defines the structure for the request
type GenerateModelForTableRequest struct {
	DatasourceID string `json:"datasource_id" binding:"required"`
	TableName    string `json:"table_name" binding:"required"`
}

// ModelMetadataBatchRequest defines the structure for the /fabric/models/metadata endpoint.
type ModelMetadataBatchRequest struct {
	DatasourceID string   `json:"datasource_id" binding:"required"`
	TableNames   []string `json:"table_names" binding:"required"`
}

// ModelExistsBatchRequest defines the structure for the /fabric/models/exists endpoint.
type ModelExistsBatchRequest struct {
	DatasourceID string   `json:"datasource_id" binding:"required"`
	TableNames   []string `json:"table_names" binding:"required"`
}

// BatchGenerateItem is a single item in a batch request.
type BatchGenerateItem struct {
	models.GenerateModelsRequest
}

// BatchGenerateRequest is the request for the batch endpoint.
type BatchGenerateRequest struct {
	Items          []BatchGenerateItem `json:"items" binding:"required"`
	Concurrency    int                 `json:"concurrency"`   // default: 4
	StopOnError    bool                `json:"stop_on_error"` // default: false
	IdempotencyKey string              `json:"idempotency_key,omitempty"`
}

// BatchGenerateResult is the result for a single item in a batch response.
type BatchGenerateResult struct {
	Index   int         `json:"index"`
	Success bool        `json:"success"`
	Error   string      `json:"error,omitempty"`
	Payload interface{} `json:"payload,omitempty"` // mirrors single generate response
}

// ListModels retrieves all fabric definitions for a given datasource.
func (h *SemanticModelHandler) ListModels(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLogger().Sugar()
	datasourceIDStr := r.URL.Query().Get("datasource_id")
	if datasourceIDStr == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "datasource_id query parameter is required"})
		return
	}

	datasourceUUID, err := uuid.Parse(datasourceIDStr)
	if err != nil {
		logger.Warnf("Invalid datasource ID: %s", datasourceIDStr)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid datasource_id format"})
		return
	}

	// The original `ListModels` was simpler, but the frontend now expects the
	// detailed catalog nodes from the scanner. To fix the 404 without changing routing,
	// we'll adapt this existing endpoint to provide the data the frontend needs.

	// 1. We need the tenant_id to initialize the scanner. We can derive it from the datasource_id.
	var tenantID uuid.UUID
	err = h.Service.DB.Get(&tenantID, `
		SELECT t.id FROM public.tenants t
		JOIN public.tenant_instance ti ON t.id = ti.tenant_id
		JOIN public.tenant_product tp ON ti.id = tp.datasource_id
		JOIN public.tenant_product_datasource tpd ON tp.id = tpd.tenant_product_id
		WHERE tpd.id = $1 LIMIT 1
	`, datasourceUUID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": fmt.Sprintf("could not determine tenant_id for datasource %s: %v", datasourceUUID, err)})
		return
	}

	// 2. Create scanner and extract models, similar to the logic in the (unregistered) ModelCatalogHandler.
	catalogScanner, err := scanner.NewFabricCatalogScanner(h.Service.DB.DB, tenantID, datasourceUUID)
	if err != nil {
		logger.Errorf("Failed to create fabric catalog scanner: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to initialize catalog scanner"})
		return
	}

	catalogModels, err := catalogScanner.ExtractModels()
	if err != nil {
		logger.Errorf("Failed to extract models: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to retrieve models"})
		return
	}

	// 3. Return the data in the format expected by the useModelCatalog hook.
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"models": catalogModels,
		"count":  len(catalogModels),
	})
}

// GenerateDefaultModel creates a default semantic model for a tenant's datasource
func (h *SemanticModelHandler) GenerateDefaultModel(w http.ResponseWriter, r *http.Request) {
	logging.GetLogger().Sugar().Info("[BACKEND_HANDLER] === ENTERED GenerateDefaultModel handler ===")

	// Add comprehensive panic recovery
	defer func() {
		if recoveryErr := recover(); recoveryErr != nil {
			logging.GetLogger().Sugar().Errorf("[BACKEND_HANDLER] *** PANIC RECOVERED ***: %v", recoveryErr)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":   fmt.Sprintf("Internal server panic: %v", recoveryErr),
				"success": false,
			})
		}
	}()

	var req GenerateDefaultModelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid request: datasource_id is required in the body"})
		return
	}

	datasourceUUID, err := uuid.Parse(req.DatasourceID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid datasource_id format"})
		return
	}

	// Test service availability
	if h.Service == nil {
		logging.GetLogger().Sugar().Error("[BACKEND_HANDLER] Error: Service is nil")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Service not initialized"})
		return
	}

	logging.GetLogger().Sugar().Info("[BACKEND_HANDLER] All validations passed, calling service...")

	// Call the updated service method with both parameters
	models, err := h.Service.GenerateDefaultSemanticModel(datasourceUUID)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("[BACKEND_HANDLER] *** SERVICE ERROR (Fabric) ***: %v", err)

		// Determine appropriate HTTP status based on error type
		var statusCode int
		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		} else if strings.Contains(err.Error(), "already exists") {
			statusCode = http.StatusConflict
		} else {
			statusCode = http.StatusInternalServerError
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   fmt.Sprintf("Failed to generate model: %v", err),
			"success": false,
		})
		return
	}

	// Validate model before responding
	if models == nil {
		logging.GetLogger().Sugar().Error("[BACKEND_HANDLER] Error: Service returned nil models slice")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Service returned empty models slice",
			"success": false,
		})
		return
	}

	// Log success for debugging
	logging.GetLogger().Sugar().Infof("[BACKEND_SERVICE] === SUCCESS === Generated %d fabric definitions", len(models))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": fmt.Sprintf("Default semantic models generated successfully: %d created.", len(models)),
		"models":  models,
		"success": true,
	})

	logging.GetLogger().Sugar().Info("[BACKEND_HANDLER] === RESPONSE SENT ===")
}

// GenerateModels creates semantic models based on a given scope (e.g., schema or tables).
func (h *SemanticModelHandler) GenerateModels(w http.ResponseWriter, r *http.Request) {
	var req models.GenerateModelsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid request: datasource_id and scope are required"})
		return
	}

	svcs := models.Services{
		SemanticModelService: h.Service,
	}

	resp, err := models.Generate(svcs, req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(err.Error(), "validation failed") {
			w.WriteHeader(http.StatusBadRequest)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// GenerateModelsBatch is the HTTP handler for batch generation requests.
func (h *SemanticModelHandler) GenerateModelsBatch(w http.ResponseWriter, r *http.Request) {
	var req BatchGenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid batch request"})
		return
	}
	if req.Concurrency <= 0 {
		req.Concurrency = 4
	}

	results := make([]BatchGenerateResult, len(req.Items))
	sem := make(chan struct{}, req.Concurrency)
	var wg sync.WaitGroup

	var stop int32

	for i, item := range req.Items {
		if atomic.LoadInt32(&stop) == 1 {
			results[i] = BatchGenerateResult{Index: i, Success: false, Error: "Batch stopped due to previous error"}
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, itm BatchGenerateItem) {
			defer wg.Done()
			defer func() { <-sem }()

			svcs := models.Services{
				SemanticModelService: h.Service,
			}

			payload, err := models.Generate(svcs, itm.GenerateModelsRequest)
			if err != nil {
				results[idx] = BatchGenerateResult{Index: idx, Success: false, Error: err.Error()}
				if req.StopOnError {
					atomic.StoreInt32(&stop, 1)
				}
				return
			}
			results[idx] = BatchGenerateResult{Index: idx, Success: true, Payload: payload}
		}(i, item)
	}
	wg.Wait()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"results":       results,
		"stopped_early": atomic.LoadInt32(&stop) == 1,
	})
}

// GenerateModelForTable creates a semantic model for a single table
func (h *SemanticModelHandler) GenerateModelForTable(w http.ResponseWriter, r *http.Request) {
	var req GenerateModelForTableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid request: datasource_id and table_name are required"})
		return
	}

	datasourceUUID, err := uuid.Parse(req.DatasourceID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid datasource_id format"})
		return
	}

	model, err := h.Service.GenerateSemanticModelForTable(datasourceUUID, req.TableName)
	if err != nil {
		var statusCode int
		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		} else if strings.Contains(err.Error(), "already exists") {
			statusCode = http.StatusConflict
		} else {
			statusCode = http.StatusInternalServerError
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   fmt.Sprintf("Failed to generate model for table: %v", err),
			"success": false,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Semantic model for table generated successfully.",
		"model":   model,
		"success": true,
	})
}

// ModelsMetadataBatch checks for the existence of models for a batch of tables and returns metadata.
func (h *SemanticModelHandler) ModelsMetadataBatch(w http.ResponseWriter, r *http.Request) {
	var req ModelMetadataBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid request format"})
		return
	}

	datasourceUUID, err := uuid.Parse(req.DatasourceID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid datasource_id"})
		return
	}

	metadataMap, err := h.Service.GetModelMetadata(datasourceUUID, req.TableNames)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"results": metadataMap})
}

// GetModelDefinition handles the GET request to fetch a single model definition.
func (h *SemanticModelHandler) GetModelDefinition(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	datasourceIDStr := query.Get("datasource_id")
	modelKey := query.Get("model_key")

	if datasourceIDStr == "" || modelKey == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "datasource_id and model_key query parameters are required"})
		return
	}

	datasourceID, err := uuid.Parse(datasourceIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "invalid datasource_id format"})
		return
	}

	// Normalize modelKey? Accept both with and without leading slash.
	originalModelKey := modelKey
	if !strings.HasPrefix(modelKey, "/") {
		modelKey = "/" + modelKey
	}

	defn, err := h.Service.GetModelDefinition(datasourceID, modelKey)
	if err != nil {
		// Log full context for debugging
		logging.GetLogger().Sugar().Errorf("[GetModelDefinition] lookup failed: datasource_id=%s original_key=%s normalized_key=%s error=%v", datasourceID, originalModelKey, modelKey, err)
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(err.Error(), "not found") {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error(), "datasource_id": datasourceID, "model_key": originalModelKey})
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{"error": "failed to retrieve model definition", "detail": err.Error(), "datasource_id": datasourceID, "model_key": originalModelKey})
		}
		return
	}

	// If the stored key differs and user omitted leading slash, let them know.
	if originalModelKey != modelKey {
		w.Header().Set("X-Normalized-Model-Key", modelKey)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(defn)
}

// ModelsExistBatch checks for the existence of models for a batch of tables.
func (h *SemanticModelHandler) ModelsExistBatch(w http.ResponseWriter, r *http.Request) {
	var req ModelExistsBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid request format"})
		return
	}

	datasourceUUID, err := uuid.Parse(req.DatasourceID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid datasource_id"})
		return
	}

	// The `ModelsExist` method was removed as it was redundant.
	// We can use `GetModelMetadata` which provides the same existence check plus more detail.
	metadataMap, err := h.Service.GetModelMetadata(datasourceUUID, req.TableNames)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	// To maintain the original contract of this endpoint (map[string]bool),
	// we transform the result.
	existsMap := make(map[string]bool)
	for tableName, metadata := range metadataMap {
		existsMap[tableName] = metadata.Exists
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"results": existsMap})
}

// ListExtensions lists current extension models for a datasource.
func (h *SemanticModelHandler) ListExtensions(w http.ResponseWriter, r *http.Request) {
	datasourceIDStr := r.URL.Query().Get("datasource_id")
	if datasourceIDStr == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "datasource_id query parameter is required"})
		return
	}
	dsID, err := uuid.Parse(datasourceIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid datasource_id format"})
		return
	}
	items, err := h.Service.ListExtensionModels(dsID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

// SaveExtensionRequest is the incoming payload to create/update an extension.
type SaveExtensionRequest struct {
	BaseModelKey string         `json:"base_model_key"`
	ModelKey     string         `json:"model_key"`
	Title        string         `json:"title"`
	Description  string         `json:"description"`
	Status       string         `json:"status"`
	CoreVersion  *int           `json:"core_version,omitempty"`
	ModelObject  map[string]any `json:"model_object" binding:"required"`
	ActorID      string         `json:"actor_id,omitempty"`
}

// SaveExtension creates or updates an extension model with validation and version stamping.
func (h *SemanticModelHandler) SaveExtension(w http.ResponseWriter, r *http.Request) {
	var req SaveExtensionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid request body"})
		return
	}
	datasourceIDStr := r.URL.Query().Get("datasource_id")
	if datasourceIDStr == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "datasource_id query parameter is required"})
		return
	}
	dsID, err := uuid.Parse(datasourceIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid datasource_id format"})
		return
	}
	var actor uuid.UUID
	if req.ActorID != "" {
		if a, err := uuid.Parse(req.ActorID); err == nil {
			actor = a
		}
	}
	// Convert generic map payload to cube.Cube
	// Marshal then unmarshal for robust shape conversion.
	b, err := json.Marshal(req.ModelObject)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid model_object payload"})
		return
	}
	var ext cube.Cube
	if err := json.Unmarshal(b, &ext); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid model_object structure"})
		return
	}

	// DEBUG: log decoded extension shape for troubleshooting validation issues
	// This is temporary and can be removed after debugging.
	if logger := logging.GetLogger(); logger != nil {
		if bs, err := json.MarshalIndent(ext, "", "  "); err == nil {
			logger.Sugar().Infof("[DEBUG ValidateModel] decoded ext: %s", string(bs))
		} else {
			logger.Sugar().Infof("[DEBUG ValidateModel] decoded ext (marshal failed): %v", err)
		}
	}

	saved, issues, err := h.Service.SaveExtensionModel(dsID, analytics.SaveExtensionModelRequest{
		BaseModelKey: req.BaseModelKey,
		ModelKey:     req.ModelKey,
		Title:        req.Title,
		Description:  req.Description,
		Status:       req.Status,
		CoreVersion:  req.CoreVersion,
		ModelObject:  ext,
		ActorID:      actor,
	})
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(err.Error(), "base model not found") {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"model": saved, "issues": issues})
}

// ValidateModel accepts a model_object payload and runs validation against the current base model
// without saving. It returns validation issues and any pruning actions that would be taken.
func (h *SemanticModelHandler) ValidateModel(w http.ResponseWriter, r *http.Request) {
	var req SaveExtensionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid request body"})
		return
	}
	datasourceIDStr := r.URL.Query().Get("datasource_id")
	if datasourceIDStr == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "datasource_id query parameter is required"})
		return
	}
	dsID, err := uuid.Parse(datasourceIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid datasource_id format"})
		return
	}

	// Convert generic map payload to cube.Cube
	b, err := json.Marshal(req.ModelObject)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid model_object payload"})
		return
	}
	var ext cube.Cube
	if err := json.Unmarshal(b, &ext); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid model_object structure"})
		return
	}

	// Resolve base key
	baseKey := req.BaseModelKey
	if baseKey == "" {
		if s, ok := ext.Extends.(string); ok {
			baseKey = s
		} else if s, ok := ext.Metadata["inherits_from"].(string); ok {
			baseKey = s
		}
	}
	if baseKey == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "base_model_key is required or must be present in model_object"})
		return
	}
	if !strings.HasPrefix(baseKey, "/") {
		baseKey = "/" + baseKey
	}

	baseDefn, err := h.Service.GetModelDefinition(dsID, baseKey)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "base model not found"})
		return
	}
	var baseConfig models.ResolvedModelConfig
	if err := json.Unmarshal(baseDefn.ResolvedConfig, &baseConfig); err != nil || len(baseConfig.Cubes) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "failed to parse base model resolved_config"})
		return
	}
	baseCube := baseConfig.Cubes[0]

	// Run extension validation
	issues := cube.ValidateExtension(baseCube, ext)

	// Prune-finding: gather columns and run pruning heuristics (without mutating DB)
	colsMap, errCols := h.Service.GatherColumnsMapForDatasource(dsID)
	pruning := []cube.ValidationIssue{}
	if errCols == nil {
		pruning = h.Service.PruneMissingColumnsFromExtension(&ext, colsMap, baseCube.Name)
	}
	// Add FK-based join membership checks
	fkIssues := h.Service.ValidateJoinsWithCatalogFKs(dsID, &ext)

	// combine
	all := append(issues, pruning...)
	all = append(all, fkIssues...)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"issues": all})
}

// ExtensionsCompatibilityReport validates all extension models against current core and returns a report.
func (h *SemanticModelHandler) ExtensionsCompatibilityReport(w http.ResponseWriter, r *http.Request) {
	datasourceIDStr := r.URL.Query().Get("datasource_id")
	if datasourceIDStr == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "datasource_id query parameter is required"})
		return
	}
	dsID, err := uuid.Parse(datasourceIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid datasource_id format"})
		return
	}
	report, globalIssues, err := h.Service.GetExtensionsCompatibilityReport(dsID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"report": report, "issues": globalIssues})
}

// AddCalculationRequest defines the payload for adding a calculation to a semantic model.
type AddCalculationRequest struct {
	CalculationID   string            `json:"calculation_id" binding:"required"`
	ArgumentMapping map[string]string `json:"argument_mapping"`
	OutputName      string            `json:"output_name" binding:"required"`
	IsPublic        bool              `json:"is_public"`
}

// AddCalculation associates a calculation with a semantic model.
func (h *SemanticModelHandler) AddCalculation(w http.ResponseWriter, r *http.Request) {
	semanticModelIDStr := chi.URLParam(r, "id")
	if semanticModelIDStr == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "semantic model id is required"})
		return
	}
	semanticModelID, err := uuid.Parse(semanticModelIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "invalid semantic model id"})
		return
	}

	var req AddCalculationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "invalid request body"})
		return
	}

	calculationID, err := uuid.Parse(req.CalculationID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "invalid calculation id"})
		return
	}

	// TODO: Get user ID from context
	userID := uuid.Nil

	smc, err := h.Service.AddCalculation(r.Context(), semanticModelID, calculationID, req.ArgumentMapping, req.OutputName, req.IsPublic, userID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(smc)
}

// RemoveCalculation removes a calculation association from a semantic model.
func (h *SemanticModelHandler) RemoveCalculation(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "calc_id")
	if idStr == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "calculation association id is required"})
		return
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "invalid calculation association id"})
		return
	}

	if err := h.Service.RemoveCalculation(r.Context(), id); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetCalculations retrieves all calculations associated with a semantic model.
func (h *SemanticModelHandler) GetCalculations(w http.ResponseWriter, r *http.Request) {
	semanticModelIDStr := chi.URLParam(r, "id")
	if semanticModelIDStr == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "semantic model id is required"})
		return
	}
	semanticModelID, err := uuid.Parse(semanticModelIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "invalid semantic model id"})
		return
	}

	calcs, err := h.Service.GetCalculations(r.Context(), semanticModelID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(calcs)
}
