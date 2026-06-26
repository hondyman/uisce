package handlers

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/db/charts"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/hondyman/semlayer/backend/internal/scanner"
)

var (
	NODE_TYPE_SEMANTIC_MODEL = uuid.MustParse("c53f9e99-8d02-4dfb-bc1b-914747d35edb")
)

// ModelCatalogHandler handles HTTP requests for the model catalog
type ModelCatalogHandler struct {
	db *sql.DB
}

// NewModelCatalogHandler creates a new model catalog handler
func NewModelCatalogHandler(db *sql.DB) *ModelCatalogHandler {
	return &ModelCatalogHandler{
		db: db,
	}
}

// respondJSON is a helper method to send JSON responses
func (h *ModelCatalogHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// GetModelsRequest represents the request parameters for getting models
type GetModelsRequest struct {
	TenantID           string `uri:"tenant_id" binding:"required,uuid"`
	TenantDatasourceID string `uri:"datasource_id" binding:"required,uuid"`
}

// CreateCustomModelRequest represents the request body for creating a custom model
type CreateCustomModelRequest struct {
	BaseModelKey string `json:"base_model_key" binding:"required"`
}

// CloneModelRequest represents the request body for cloning a model
type CloneModelRequest struct {
	// ID of model (core or custom) to clone
	ModelID string `json:"model_id" binding:"required"`
}

// resolveUserUUID attempts to parse the X-User-ID header into a UUID.
// If the header contains a non-UUID identifier we derive a deterministic
// UUID so downstream code always receives a valid value.
func resolveUserUUID(userIDStr string) (uuid.UUID, bool, error) {
	trimmed := strings.TrimSpace(userIDStr)
	if trimmed == "" {
		return uuid.Nil, false, fmt.Errorf("user id missing")
	}
	if parsed, err := uuid.Parse(trimmed); err == nil {
		return parsed, false, nil
	}
	derived := uuid.NewSHA1(uuid.NameSpaceOID, []byte(trimmed))
	return derived, true, nil
}

// UpdateModelRequest represents the request body for updating a model
type UpdateModelRequest struct {
	Title          *string         `json:"title,omitempty"`
	DisplayName    *string         `json:"display_name,omitempty"` // alias for title
	Description    *string         `json:"description,omitempty"`
	ResolvedConfig json.RawMessage `json:"resolved_config,omitempty"`
	Status         *string         `json:"status,omitempty"`
}

// CreateGeneratedModelRequest represents the request body for creating a model from generated JSON
type CreateGeneratedModelRequest struct {
	TableName       string          `json:"table_name" binding:"required"`
	Schema          string          `json:"schema" binding:"required"`
	ModelDefinition json.RawMessage `json:"model_definition" binding:"required"`
}

// GetModelsResponse represents the response for getting models
type GetModelsResponse struct {
	Models []scanner.ModelCatalogNode `json:"models"`
	Count  int                        `json:"count"`
}

// retryInsertGeneratedModel attempts to insert a generated model record using the provided tenant UUID.
// It returns nil on success or an error if insertion still fails.
func retryInsertGeneratedModel(h *ModelCatalogHandler, tenantID uuid.UUID, datasourceID uuid.UUID, modelKey string, sourceConfigBytes, resolvedConfigBytes []byte, createdBy uuid.UUID) error {
	checksum := sha256.Sum256(resolvedConfigBytes)
	insertQuery := `INSERT INTO fabric_defn (
		tenant_id, tenant_datasource_id, model_key, "version", status, is_current,
		title, description, source_config, resolved_config, created_by, created_at, checksum_sha256
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,now(),$12) RETURNING id`

	_, err := h.db.Exec(insertQuery,
		tenantID, datasourceID, modelKey, 1, "published", true,
		fmt.Sprintf("Generated model for %s", modelKey),
		fmt.Sprintf("Auto-generated model (retry) for %s", modelKey),
		json.RawMessage(sourceConfigBytes), json.RawMessage(resolvedConfigBytes), createdBy.String(), checksum[:],
	)
	return err
}

// GetModels retrieves all models for a tenant datasource
// GET /api/tenants/:tenant_id/datasources/:datasource_id/models
func (h *ModelCatalogHandler) GetModels(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLogger().Sugar()

	tenantIDStr := r.URL.Query().Get("tenant_id")
	datasourceIDStr := r.URL.Query().Get("datasource_id")

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		logger.Warnf("Invalid tenant ID: %s", tenantIDStr)
		h.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid tenant ID"})
		return
	}

	datasourceID, err := uuid.Parse(datasourceIDStr)
	if err != nil {
		logger.Warnf("Invalid datasource ID: %s", datasourceIDStr)
		h.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid datasource ID"})
		return
	}

	// Create scanner and extract models
	catalogScanner, err := scanner.NewFabricCatalogScanner(h.db, tenantID, datasourceID)
	if err != nil {
		logger.Errorf("Failed to create fabric catalog scanner: %v", err)
		h.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to initialize catalog scanner"})
		return
	}

	models, err := catalogScanner.ExtractModels()
	if err != nil {
		logger.Errorf("Failed to extract models: %v", err)
		h.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve models"})
		return
	}

	logger.Infof("Retrieved %d models for tenant %s, datasource %s", len(models), tenantID, datasourceID)

	response := GetModelsResponse{
		Models: models,
		Count:  len(models),
	}

	h.respondJSON(w, http.StatusOK, response)
}

// CreateCustomModel creates a new custom model inheriting from a base model
// POST /api/tenants/:tenant_id/datasources/:datasource_id/models/custom
func (h *ModelCatalogHandler) CreateCustomModel(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLogger().Sugar()

	tenantIDStr := r.URL.Query().Get("tenant_id")
	datasourceIDStr := r.URL.Query().Get("datasource_id")

	var bodyReq CreateCustomModelRequest
	if err := json.NewDecoder(r.Body).Decode(&bodyReq); err != nil {
		logger.Warnf("Invalid request body: %v", err)
		h.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		logger.Warnf("Invalid tenant ID: %s", tenantIDStr)
		h.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid tenant ID"})
		return
	}

	datasourceID, err := uuid.Parse(datasourceIDStr)
	if err != nil {
		logger.Warnf("Invalid datasource ID: %s", datasourceIDStr)
		h.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid datasource ID"})
		return
	}

	// Get user ID from context (you'll need to implement authentication middleware)
	// For now, we'll assume it's passed in a header or context
	userIDStr := r.Header.Get("X-User-ID")
	userUUID, derived, err := resolveUserUUID(userIDStr)
	if err != nil {
		logger.Warn("User ID not found in request")
		h.respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "Authentication required"})
		return
	}
	if derived {
		logger.Warnf("Derived UUID for non-UUID user identifier: %s", userIDStr)
	}

	// Create scanner
	catalogScanner, err := scanner.NewFabricCatalogScanner(h.db, tenantID, datasourceID)
	if err != nil {
		logger.Errorf("Failed to create fabric catalog scanner: %v", err)
		h.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to initialize catalog scanner"})
		return
	}

	// Create the custom model
	customModel, err := catalogScanner.CreateCustomModel(bodyReq.BaseModelKey, userUUID)
	if err != nil {
		logger.Errorf("Failed to create custom model: %v", err)
		if err.Error() == fmt.Sprintf("custom model already exists: %s_custom", bodyReq.BaseModelKey) {
			h.respondJSON(w, http.StatusConflict, map[string]string{"error": "Custom model already exists"})
			return
		}
		if err.Error() == fmt.Sprintf("core model not found: %s", bodyReq.BaseModelKey) {
			h.respondJSON(w, http.StatusNotFound, map[string]string{"error": "Base model not found"})
			return
		}
		h.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create custom model"})
		return
	}

	logger.Infof("Created custom model %s for user %s", customModel.ModelKey, userUUID)

	// Create catalog_node for the custom model
	customNodeName := fmt.Sprintf("model.%s", customModel.ModelKey)
	customQualified := fmt.Sprintf("/semantic_model/%s", customModel.ModelKey)
	var customNodeID string
	insertCustomNode := `INSERT INTO public.catalog_node (id, tenant_id, tenant_datasource_id, node_type_id, node_name, qualified_path, properties, description, created_at, updated_at) VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, '{}'::jsonb, $6, now(), now()) RETURNING id`
	if err := h.db.QueryRow(insertCustomNode, tenantID, datasourceID.String(), charts.MODEL_NODE_TYPE_ID, customNodeName, customQualified, fmt.Sprintf("Custom semantic model for %s", customModel.ModelKey)).Scan(&customNodeID); err != nil {
		logger.Warnf("Failed to insert custom model catalog_node: %v", err)
	} else {
		logger.Infof("Created custom model catalog_node %s for %s", customNodeID, customQualified)
	}

	// Find core model's catalog_node ID
	coreQualified := fmt.Sprintf("/semantic_model/%s", bodyReq.BaseModelKey)
	var coreNodeID string
	if err := h.db.QueryRow(`SELECT id FROM public.catalog_node WHERE tenant_datasource_id = $1 AND node_type_id = $2 AND qualified_path = $3 LIMIT 1`, datasourceID.String(), charts.MODEL_NODE_TYPE_ID, coreQualified).Scan(&coreNodeID); err != nil {
		logger.Warnf("Failed to find core model catalog_node for %s: %v", coreQualified, err)
	} else {
		// Create edge from core to custom
		edgeTypeID := uuid.MustParse("3be9d6ae-1598-4628-a3dd-b606921a9193")
		edgeProps := map[string]interface{}{
			"relationship": "core_to_custom",
		}
		edgePropsJSON, _ := json.Marshal(edgeProps)
		edgeID := uuid.New()
		_, err = h.db.Exec(`
			INSERT INTO public.catalog_edge (
				id, tenant_id, tenant_datasource_id, source_node_id, target_node_id,
				edge_type_id, edge_type_name, relationship_type, properties, created_at, updated_at
			) VALUES ($1,$2,$3,$4,$5,$6, COALESCE((SELECT edge_type_name FROM catalog_edge_type WHERE id = $6), 'has_semantic'), $7,$8,$9,$9)
			ON CONFLICT (tenant_datasource_id, source_node_id, edge_type_name, target_node_id) DO UPDATE SET
				relationship_type = EXCLUDED.relationship_type,
				properties = EXCLUDED.properties,
				updated_at = EXCLUDED.updated_at;
		`,
			edgeID,
			tenantID,
			datasourceID,
			coreNodeID,
			customNodeID,
			edgeTypeID,
			"has_semantic",
			edgePropsJSON,
			time.Now(),
		)
		if err != nil {
			logger.Warnf("Failed to create edge from core to custom model: %v", err)
		} else {
			logger.Infof("Created edge from core %s to custom %s", coreNodeID, customNodeID)
		}
	}

	h.respondJSON(w, http.StatusCreated, customModel)
}

// CloneModel creates a new clone from an existing model (core or custom)
// POST /api/tenants/:tenant_id/datasources/:datasource_id/models/clone
func (h *ModelCatalogHandler) CloneModel(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLogger().Sugar()

	tenantIDStr := r.URL.Query().Get("tenant_id")
	datasourceIDStr := r.URL.Query().Get("datasource_id")

	var bodyReq CloneModelRequest
	if err := json.NewDecoder(r.Body).Decode(&bodyReq); err != nil {
		logger.Warnf("Invalid request body: %v", err)
		h.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		logger.Warnf("Invalid tenant ID: %s", tenantIDStr)
		h.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid tenant ID"})
		return
	}
	datasourceID, err := uuid.Parse(datasourceIDStr)
	if err != nil {
		logger.Warnf("Invalid datasource ID: %s", datasourceIDStr)
		h.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid datasource ID"})
		return
	}
	modelUUID, err := uuid.Parse(bodyReq.ModelID)
	if err != nil {
		logger.Warnf("Invalid model ID: %s", bodyReq.ModelID)
		h.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid model ID"})
		return
	}

	userIDStr := r.Header.Get("X-User-ID")
	userUUID, derived, err := resolveUserUUID(userIDStr)
	if err != nil {
		logger.Warn("User ID not found in request")
		h.respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "Authentication required"})
		return
	}
	if derived {
		logger.Warnf("Derived UUID for non-UUID user identifier: %s", userIDStr)
	}

	catalogScanner, err := scanner.NewFabricCatalogScanner(h.db, tenantID, datasourceID)
	if err != nil {
		logger.Errorf("Failed to create fabric catalog scanner: %v", err)
		h.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to initialize catalog scanner"})
		return
	}

	clonedNode, err := catalogScanner.CloneModel(modelUUID, userUUID)
	if err != nil {
		logger.Errorf("Failed to clone model: %v", err)
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(strings.ToLower(err.Error()), "base model not found") {
			h.respondJSON(w, http.StatusNotFound, map[string]string{"error": "Base model not found"})
			return
		}
		h.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to clone model"})
		return
	}

	logger.Infof("Cloned model %s for user %s", bodyReq.ModelID, userUUID)
	h.respondJSON(w, http.StatusCreated, clonedNode)
}

// GetModel retrieves a specific model by ID
// GET /api/tenants/:tenant_id/datasources/:datasource_id/models/:model_id
func (h *ModelCatalogHandler) GetModel(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLogger().Sugar()

	tenantIDStr := r.URL.Query().Get("tenant_id")
	datasourceIDStr := r.URL.Query().Get("datasource_id")
	modelID := chi.URLParam(r, "model_id")

	modelUUID, err := uuid.Parse(modelID)
	if err != nil {
		logger.Warnf("Invalid model ID: %s", modelID)
		h.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid model ID"})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		logger.Warnf("Invalid tenant ID: %s", tenantIDStr)
		h.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid tenant ID"})
		return
	}

	datasourceID, err := uuid.Parse(datasourceIDStr)
	if err != nil {
		logger.Warnf("Invalid datasource ID: %s", datasourceIDStr)
		h.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid datasource ID"})
		return
	}

	// Query the specific model
	query := `
		SELECT 
			id, tenant_id, tenant_datasource_id, model_key, version, status, 
			is_current, title, description, source_config, resolved_config,
			created_by, created_at, published_at, checksum_sha256, updated_at
		FROM fabric_defn 
		WHERE id = $1 AND tenant_id = $2 AND tenant_datasource_id = $3
	`

	var model scanner.FabricDefinition
	err = h.db.QueryRow(query, modelUUID, tenantID, datasourceID).Scan(
		&model.ID, &model.TenantID, &model.TenantDatasourceID,
		&model.ModelKey, &model.Version, &model.Status, &model.IsCurrent,
		&model.Title, &model.Description, &model.SourceConfig, &model.ResolvedConfig,
		&model.CreatedBy, &model.CreatedAt, &model.PublishedAt,
		&model.ChecksumSHA256, &model.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		logger.Warnf("Model not found: %s", modelID)
		h.respondJSON(w, http.StatusNotFound, map[string]string{"error": "Model not found"})
		return
	}
	if err != nil {
		logger.Errorf("Failed to query model: %v", err)
		h.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve model"})
		return
	}

	h.respondJSON(w, http.StatusOK, model)
}

// UpdateModel updates a specific model
// PATCH /api/tenants/:tenant_id/datasources/:datasource_id/models/:model_id
func (h *ModelCatalogHandler) UpdateModel(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLogger().Sugar()

	tenantIDStr := r.URL.Query().Get("tenant_id")
	datasourceIDStr := r.URL.Query().Get("datasource_id")
	modelIDStr := chi.URLParam(r, "model_id")

	var bodyReq UpdateModelRequest
	if err := json.NewDecoder(r.Body).Decode(&bodyReq); err != nil {
		logger.Warnf("Invalid request body: %v", err)
		h.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	modelUUID, err := uuid.Parse(modelIDStr)
	if err != nil {
		logger.Warnf("Invalid model ID: %s", modelIDStr)
		h.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid model ID"})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		logger.Warnf("Invalid tenant ID: %s", tenantIDStr)
		h.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid tenant ID"})
		return
	}

	datasourceID, err := uuid.Parse(datasourceIDStr)
	if err != nil {
		logger.Warnf("Invalid datasource ID: %s", datasourceIDStr)
		h.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid datasource ID"})
		return
	}

	// Build dynamic update query
	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	// Handle title/display_name (display_name takes precedence if both provided)
	var titleToUpdate *string
	if bodyReq.DisplayName != nil {
		titleToUpdate = bodyReq.DisplayName
	} else if bodyReq.Title != nil {
		titleToUpdate = bodyReq.Title
	}
	if titleToUpdate != nil {
		setParts = append(setParts, fmt.Sprintf("title = $%d", argIndex))
		args = append(args, *titleToUpdate)
		argIndex++
	}

	if bodyReq.Description != nil {
		setParts = append(setParts, fmt.Sprintf("description = $%d", argIndex))
		args = append(args, *bodyReq.Description)
		argIndex++
	}

	// Track whether resolved_config will be updated, to compute checksum
	var resolvedConfigProvided bool
	if bodyReq.ResolvedConfig != nil {
		setParts = append(setParts, fmt.Sprintf("resolved_config = $%d", argIndex))
		args = append(args, bodyReq.ResolvedConfig)
		argIndex++
		resolvedConfigProvided = true
	}

	if bodyReq.Status != nil {
		// Validate allowed statuses
		status := strings.ToLower(strings.TrimSpace(*bodyReq.Status))
		if status != "draft" && status != "published" && status != "archived" {
			h.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid status value"})
			return
		}
		setParts = append(setParts, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, status)
		argIndex++
		// If publishing, set published_at timestamp to now
		if status == "published" {
			setParts = append(setParts, fmt.Sprintf("published_at = $%d", argIndex))
			args = append(args, time.Now())
			argIndex++
		} else if status == "draft" {
			// Moving back to draft clears the published_at timestamp
			setParts = append(setParts, "published_at = NULL")
		}
	}

	if len(setParts) == 0 {
		h.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "No fields to update"})
		return
	}

	// If resolved_config is being updated, compute checksum and set checksum_sha256
	if resolvedConfigProvided {
		// We store raw bytes of the JSON in checksum to avoid semantic diffs
		// Compute SHA256 over the exact payload we will write
		// Note: bodyReq.ResolvedConfig is json.RawMessage, already bytes
		checksum := sha256.Sum256([]byte(bodyReq.ResolvedConfig))
		setParts = append(setParts, fmt.Sprintf("checksum_sha256 = $%d", argIndex))
		// Store as bytea
		args = append(args, checksum[:])
		argIndex++
	}

	// Always update the updated_at timestamp
	setParts = append(setParts, fmt.Sprintf("updated_at = $%d", argIndex))
	args = append(args, time.Now())
	argIndex++

	// Add WHERE clause parameters
	args = append(args, modelUUID, tenantID, datasourceID)

	query := fmt.Sprintf(`
		UPDATE fabric_defn 
		SET %s
		WHERE id = $%d AND tenant_id = $%d AND tenant_datasource_id = $%d
		RETURNING id, tenant_id, tenant_datasource_id, model_key, version, status, 
			is_current, title, description, source_config, resolved_config,
			created_by, created_at, published_at, checksum_sha256, updated_at
	`,
		strings.Join(setParts, ", "),
		argIndex, argIndex+1, argIndex+2,
	)

	var updatedModel scanner.FabricDefinition
	err = h.db.QueryRow(query, args...).Scan(
		&updatedModel.ID, &updatedModel.TenantID, &updatedModel.TenantDatasourceID,
		&updatedModel.ModelKey, &updatedModel.Version, &updatedModel.Status,
		&updatedModel.IsCurrent, &updatedModel.Title, &updatedModel.Description,
		&updatedModel.SourceConfig, &updatedModel.ResolvedConfig,
		&updatedModel.CreatedBy, &updatedModel.CreatedAt, &updatedModel.PublishedAt,
		&updatedModel.ChecksumSHA256, &updatedModel.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		logger.Warnf("Model not found for update: %s", modelIDStr)
		h.respondJSON(w, http.StatusNotFound, map[string]string{"error": "Model not found"})
		return
	}
	if err != nil {
		logger.Errorf("Failed to update model: %v", err)
		h.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update model"})
		return
	}

	logger.Infof("Updated model %s", modelIDStr)

	h.respondJSON(w, http.StatusOK, updatedModel)
}

// DeleteModel deletes a specific model (only custom models can be deleted)
// DELETE /api/tenants/:tenant_id/datasources/:datasource_id/models/:model_id
func (h *ModelCatalogHandler) DeleteModel(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLogger().Sugar()

	tenantIDStr := r.URL.Query().Get("tenant_id")
	datasourceIDStr := r.URL.Query().Get("datasource_id")
	modelID := chi.URLParam(r, "model_id")

	logger.Infof("DeleteModel called with tenantID=%s, datasourceID=%s, modelID=%s", tenantIDStr, datasourceIDStr, modelID)

	modelUUID, err := uuid.Parse(modelID)
	if err != nil {
		logger.Errorf("Invalid model ID: %s, error: %v", modelID, err)
		h.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid model ID"})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		logger.Errorf("Invalid tenant ID: %s, error: %v", tenantIDStr, err)
		h.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid tenant ID"})
		return
	}

	datasourceID, err := uuid.Parse(datasourceIDStr)
	if err != nil {
		logger.Errorf("Invalid datasource ID: %s, error: %v", datasourceIDStr, err)
		h.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid datasource ID"})
		return
	}

	logger.Infof("Parsed IDs successfully: modelUUID=%s, tenantID=%s, datasourceID=%s", modelUUID, tenantID, datasourceID)

	// First, check if the model exists and is a custom model
	checkQuery := `
		SELECT model_key, source_config
		FROM fabric_defn 
		WHERE id = $1 AND tenant_id = $2 AND tenant_datasource_id = $3
	`

	var modelKey string
	var sourceConfig json.RawMessage
	logger.Infof("Executing check query for model %s", modelUUID)
	err = h.db.QueryRow(checkQuery, modelUUID, tenantID, datasourceID).Scan(&modelKey, &sourceConfig)

	if err == sql.ErrNoRows {
		logger.Warnf("Model not found for deletion: %s", modelID)
		h.respondJSON(w, http.StatusNotFound, map[string]string{"error": "Model not found"})
		return
	}
	if err != nil {
		logger.Errorf("Failed to check model: %v", err)
		h.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to check model"})
		return
	}

	logger.Infof("Found model %s with key %s, sourceConfig length: %d", modelID, modelKey, len(sourceConfig))

	// Check if it's a custom model (ends with _custom or has custom generator)
	var sourceConfigMap map[string]interface{}
	isCustom := false
	if sourceConfig != nil {
		if json.Unmarshal(sourceConfig, &sourceConfigMap) == nil {
			if generator, ok := sourceConfigMap["generator"].(string); ok && generator == "custom" {
				isCustom = true
			}
		}
	}

	logger.Infof("Model %s: isCustom=%v, hasSuffix=%v", modelKey, isCustom, strings.HasSuffix(modelKey, "_custom"))

	// If it's a custom model, delete just that model (existing behavior)
	if isCustom || strings.HasSuffix(modelKey, "_custom") {
		logger.Infof("Deleting custom model %s", modelKey)
		deleteQuery := `
			DELETE FROM fabric_defn 
			WHERE id = $1 AND tenant_id = $2 AND tenant_datasource_id = $3
		`

		result, err := h.db.Exec(deleteQuery, modelUUID, tenantID, datasourceID)
		if err != nil {
			logger.Errorf("Failed to delete custom model: %v", err)
			h.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to delete model"})
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			logger.Errorf("Failed to get rows affected: %v", err)
			h.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to verify deletion"})
			return
		}

		if rowsAffected == 0 {
			logger.Warnf("No rows affected when deleting model: %s", modelID)
			h.respondJSON(w, http.StatusNotFound, map[string]string{"error": "Model not found"})
			return
		}

		// Delete the catalog node for the custom model
		qualifiedPath := fmt.Sprintf("/semantic_model/%s", modelKey)
		_, err = h.db.Exec(`DELETE FROM public.catalog_node WHERE tenant_id = $1 AND tenant_datasource_id = $2 AND node_type_id = $3 AND qualified_path = $4`, tenantID, datasourceID, NODE_TYPE_SEMANTIC_MODEL, qualifiedPath)
		if err != nil {
			logger.Warnf("Failed to delete catalog node for custom model %s: %v", modelKey, err)
		}

		logger.Infof("Deleted custom model %s", modelID)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Otherwise it's a core model. Allow deletion of core model and any associated custom models.
	logger.Infof("Processing core model deletion for %s", modelKey)

	// Find associated custom models by JSONB inherits_from or conventional suffix
	// Note: We previously attempted to use a parent_model_key column which does not exist.
	// Custom and cloned models record their base in source_config->>'inherits_from'
	// and are marked with generator = 'custom'. We also match the conventional "_custom" suffix.
	customQuery := `
				SELECT id, model_key FROM fabric_defn
				WHERE tenant_id = $1 AND tenant_datasource_id = $2
					AND (
						(source_config->>'generator' = 'custom' AND source_config->>'inherits_from' = $3)
						OR model_key = $4
					)
		`
	customKey := modelKey + "_custom"

	logger.Infof("Looking for associated custom models with parent_model_key=%s or model_key=%s", modelKey, customKey)
	rows, err := h.db.Query(customQuery, tenantID, datasourceID, modelKey, customKey)
	if err != nil {
		logger.Errorf("Failed to query associated custom models for core model %s: %v", modelKey, err)
		h.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to check associated custom models"})
		return
	}
	defer rows.Close()

	var customIDs []string
	var customKeys []string
	for rows.Next() {
		var cid, ckey string
		if err := rows.Scan(&cid, &ckey); err == nil {
			customIDs = append(customIDs, cid)
			customKeys = append(customKeys, ckey)
		}
	}
	if err := rows.Err(); err != nil {
		logger.Errorf("Row iteration error when finding custom models for %s: %v", modelKey, err)
		h.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to inspect associated custom models"})
		return
	}

	logger.Infof("Found %d associated custom models for core model %s", len(customIDs), modelKey)

	// Begin transaction to delete customs (if any) and core atomically
	logger.Infof("Starting transaction for deleting core model %s", modelKey)
	tx, err := h.db.Begin()
	if err != nil {
		logger.Errorf("Failed to start transaction for deleting core model %s: %v", modelKey, err)
		h.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to delete model"})
		return
	}

	// Ensure transaction is rolled back if we return early
	defer func() {
		if tx != nil {
			logger.Infof("Rolling back transaction for model %s", modelKey)
			_ = tx.Rollback()
		}
	}()

	// Delete associated custom models first
	logger.Infof("Deleting %d associated custom models", len(customIDs))
	for _, cid := range customIDs {
		logger.Infof("Deleting associated custom model %s", cid)
		if _, err := tx.Exec(`DELETE FROM fabric_defn WHERE id = $1 AND tenant_id = $2 AND tenant_datasource_id = $3`, cid, tenantID, datasourceID); err != nil {
			logger.Errorf("Failed to delete associated custom model %s for core %s: %v", cid, modelKey, err)
			h.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to delete associated custom models"})
			return
		}
	}

	// Delete the core model itself
	logger.Infof("Deleting core model %s", modelKey)
	if _, err := tx.Exec(`DELETE FROM fabric_defn WHERE id = $1 AND tenant_id = $2 AND tenant_datasource_id = $3`, modelUUID, tenantID, datasourceID); err != nil {
		logger.Errorf("Failed to delete core model %s: %v", modelKey, err)
		h.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to delete core model"})
		return
	}

	// Delete catalog nodes for the core model and associated customs
	coreQualifiedPath := fmt.Sprintf("/semantic_model/%s", modelKey)
	if _, err := tx.Exec(`DELETE FROM public.catalog_node WHERE tenant_id = $1 AND tenant_datasource_id = $2 AND node_type_id = $3 AND qualified_path = $4`, tenantID, datasourceID, NODE_TYPE_SEMANTIC_MODEL, coreQualifiedPath); err != nil {
		logger.Warnf("Failed to delete catalog node for core model %s: %v", modelKey, err)
	}

	for _, ckey := range customKeys {
		customQualifiedPath := fmt.Sprintf("/semantic_model/%s", ckey)
		if _, err := tx.Exec(`DELETE FROM public.catalog_node WHERE tenant_id = $1 AND tenant_datasource_id = $2 AND node_type_id = $3 AND qualified_path = $4`, tenantID, datasourceID, NODE_TYPE_SEMANTIC_MODEL, customQualifiedPath); err != nil {
			logger.Warnf("Failed to delete catalog node for custom model %s: %v", ckey, err)
		}
	}

	logger.Infof("Committing transaction for model %s", modelKey)
	if err := tx.Commit(); err != nil {
		logger.Errorf("Failed to commit transaction when deleting core model %s: %v", modelKey, err)
		h.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to delete model"})
		return
	}

	// Clear the transaction reference since it was committed successfully
	tx = nil

	logger.Infof("Deleted core model %s and %d associated custom model(s)", modelKey, len(customIDs))
	w.WriteHeader(http.StatusNoContent)
}

// CreateGeneratedModel creates a new core model from generated Cube.js JSON
// POST /api/models/generated?tenant_id=<uuid>&datasource_id=<uuid>
func (h *ModelCatalogHandler) CreateGeneratedModel(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLogger().Sugar()

	tenantIDStr := r.URL.Query().Get("tenant_id")
	datasourceIDStr := r.URL.Query().Get("datasource_id")

	var bodyReq CreateGeneratedModelRequest
	if err := json.NewDecoder(r.Body).Decode(&bodyReq); err != nil {
		logger.Warnf("Invalid request body: %v", err)
		h.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		logger.Warnf("Invalid tenant_id: %v", err)
		h.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid tenant_id"})
		return
	}

	datasourceID, err := uuid.Parse(datasourceIDStr)
	if err != nil {
		logger.Warnf("Invalid datasource_id: %v", err)
		h.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid datasource_id"})
		return
	}

	// For now, we'll store the generated model as a simple record
	// In a full implementation, you'd want to parse and validate the Cube.js JSON
	// and create proper database records

	logger.Infof("Creating generated model for table %s.%s", bodyReq.Schema, bodyReq.TableName)

	// Build a canonical model_key in the same format as existing models: /<schema>/<table>
	modelKey := fmt.Sprintf("/%s/%s", bodyReq.Schema, bodyReq.TableName)

	// Determine created_by (optional X-User-ID header). Use zero UUID if absent.
	createdByStr := r.Header.Get("X-User-ID")
	var createdBy uuid.UUID
	if createdByStr == "" {
		createdBy = uuid.Nil
	} else {
		if cb, err := uuid.Parse(createdByStr); err == nil {
			createdBy = cb
		} else {
			// If header is malformed, treat as anonymous
			createdBy = uuid.Nil
		}
	}

	ensureCustomModel := func() {
		catalogScanner, err := scanner.NewFabricCatalogScanner(h.db, tenantID, datasourceID)
		if err != nil {
			logger.Warnf("Failed to initialize catalog scanner for custom model creation: %v", err)
			return
		}

		if _, cerr := catalogScanner.CreateCustomModel(modelKey, createdBy); cerr != nil {
			if strings.Contains(strings.ToLower(cerr.Error()), "custom model already exists") {
				logger.Infof("Custom model already exists for generated core %s", modelKey)
			} else {
				logger.Warnf("Failed to auto-create custom model for generated core %s: %v", modelKey, cerr)
			}
		} else {
			logger.Infof("Auto-created custom model for generated core %s", modelKey)
		}
	}

	// Prepare source_config and resolved_config JSONB values
	// Use generator 'single-table' so created models are treated like other core models
	sourceConfig := map[string]interface{}{
		"generator": "single-table",
		"table":     fmt.Sprintf("%s.%s", bodyReq.Schema, bodyReq.TableName),
	}
	sourceConfigBytes, _ := json.Marshal(sourceConfig)

	resolvedConfigBytes := []byte(bodyReq.ModelDefinition)

	// Compute checksum over resolved_config
	checksum := sha256.Sum256(resolvedConfigBytes)

	// Insert into fabric_defn: minimal required fields
	insertQuery := `INSERT INTO fabric_defn (
		tenant_id, tenant_datasource_id, model_key, "version", status, is_current,
		title, description, source_config, resolved_config, created_by, created_at, checksum_sha256
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,now(),$12) RETURNING id, model_key, "version"`

	var createdID string
	var createdModelKey string
	var createdVersion int

	// Use version=1, status=published and is_current=true so it appears as a core model
	err = h.db.QueryRow(insertQuery,
		tenantID, datasourceID, modelKey, 1, "published", true,
		fmt.Sprintf("Generated model for %s.%s", bodyReq.Schema, bodyReq.TableName),
		fmt.Sprintf("Auto-generated model from Model Generator for %s.%s", bodyReq.Schema, bodyReq.TableName),
		json.RawMessage(sourceConfigBytes), json.RawMessage(resolvedConfigBytes), createdBy.String(), checksum[:],
	).Scan(&createdID, &createdModelKey, &createdVersion)

	if err != nil {
		// If the insert failed due to a missing tenant FK, try to resolve the
		// tenant from the provided datasource and retry the insert once.
		if strings.Contains(err.Error(), "fk_fabric_defn_tenant") || strings.Contains(err.Error(), "23503") {
			logger.Warnf("Persist failed due to missing tenant FK; attempting to resolve tenant for datasource %s", datasourceID.String())
			var resolvedTenant string
			// Attempt to find the tenant via tenant_product_datasource -> tenant_product -> tenant_instance
			tenantLookup := `SELECT ti.tenant_id
				FROM public.tenant_product_datasource tpd
				JOIN public.tenant_product tp ON tpd.tenant_product_id = tp.id
				JOIN public.tenant_instance ti ON tp.datasource_id = ti.id
				WHERE tpd.id = $1 LIMIT 1`
			if lerr := h.db.QueryRow(tenantLookup, datasourceID.String()).Scan(&resolvedTenant); lerr == nil && resolvedTenant != "" {
				if rt, perr := uuid.Parse(resolvedTenant); perr == nil {
					// Retry the insert with the resolved tenant
					logger.Infof("Retrying fabric_defn insert with resolved tenant %s", rt.String())
					if rerr := retryInsertGeneratedModel(h, rt, datasourceID, modelKey, sourceConfigBytes, resolvedConfigBytes, createdBy); rerr == nil {
						// Successful retry; proceed to link catalog_node below by setting createdID/createdModelKey/createdVersion
						// Fetch the persisted row we just created
						var rid string
						var rmk string
						var rver int
						fetchQuery := `SELECT id, model_key, "version" FROM fabric_defn WHERE tenant_id = $1 AND tenant_datasource_id = $2 AND model_key = $3 ORDER BY "version" DESC LIMIT 1`
						if ferr := h.db.QueryRow(fetchQuery, rt, datasourceID, modelKey).Scan(&rid, &rmk, &rver); ferr == nil {
							createdID = rid
							createdModelKey = rmk
							createdVersion = rver
						}
					} else {
						logger.Warnf("Retry insert with resolved tenant failed: %v", rerr)
					}
				}
			} else {
				logger.Warnf("Failed to lookup tenant for datasource %s: %v", datasourceID.String(), lerr)
			}
		}

		// Detect duplicate key (model already exists). Instead of failing, try to
		// find the existing model record, link the catalog_node to it, and
		// return success so the frontend can reflect the saved core model.
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			logger.Warnf("Generated model already exists: %v", err)

			// Fetch the existing fabric_defn id and version for this model_key
			var existingID string
			var existingVersion int
			fetchQuery := `SELECT id, "version" FROM fabric_defn WHERE tenant_id = $1 AND tenant_datasource_id = $2 AND model_key = $3 ORDER BY "version" DESC LIMIT 1`
			if ferr := h.db.QueryRow(fetchQuery, tenantID, datasourceID, modelKey).Scan(&existingID, &existingVersion); ferr != nil {
				logger.Warnf("Failed to lookup existing generated model after duplicate error: %v", ferr)
				// Fall back to returning conflict so client knows it already exists
				h.respondJSON(w, http.StatusConflict, map[string]string{"error": "Generated model already exists"})
				return
			}

			// Update catalog_node to link this table to the existing core model
			// Instead of setting core_id to the fabric_defn id (which violates the FK to catalog_node),
			// ensure there's a semantic/model catalog_node and link the table to that node.
			qualifiedPath := fmt.Sprintf("%s.%s", bodyReq.Schema, bodyReq.TableName)
			modelNodeID := ""
			modelNodeName := fmt.Sprintf("model.%s", bodyReq.TableName)
			modelQualified := fmt.Sprintf("/semantic_model/%s", bodyReq.TableName)
			if err := h.db.QueryRow(`SELECT id FROM public.catalog_node WHERE tenant_datasource_id = $1 AND node_type_id = $2 AND qualified_path = $3 LIMIT 1`, datasourceID.String(), charts.MODEL_NODE_TYPE_ID, modelQualified).Scan(&modelNodeID); err != nil {
				// Create semantic model node
				insertModelNode := `INSERT INTO public.catalog_node (id, tenant_id, tenant_datasource_id, node_type_id, node_name, qualified_path, properties, description, created_at, updated_at) VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, '{}'::jsonb, $6, now(), now()) RETURNING id`
				if ierr := h.db.QueryRow(insertModelNode, tenantID, datasourceID.String(), charts.MODEL_NODE_TYPE_ID, modelNodeName, modelQualified, fmt.Sprintf("Semantic model for %s.%s", bodyReq.Schema, bodyReq.TableName)).Scan(&modelNodeID); ierr != nil {
					logger.Warnf("Failed to insert semantic model catalog_node on duplicate path: %v", ierr)
				} else {
					logger.Infof("Created semantic model catalog_node %s for duplicate path %s", modelNodeID, modelQualified)
				}
			} else {
				logger.Infof("Found existing semantic model catalog_node %s for duplicate path %s", modelNodeID, modelQualified)
			}

			if modelNodeID != "" {
				// Find the view node_id for this table, then upsert catalog_node with that id and set core_id.
				var viewNodeID string
				if verr := h.db.QueryRow(`SELECT node_id FROM public.catalog_node_vw WHERE tenant_datasource_id = $1 AND node_type_id = $2 AND qualified_path = $3 LIMIT 1`, datasourceID.String(), charts.TABLE_NODE_TYPE_ID, qualifiedPath).Scan(&viewNodeID); verr != nil || viewNodeID == "" {
					logger.Warnf("No catalog_node_vw entry found for %s (ds=%s), cannot link core_id: %v", qualifiedPath, datasourceID.String(), verr)
				} else {
					upsert := `INSERT INTO public.catalog_node (id, tenant_id, tenant_datasource_id, node_type_id, node_name, qualified_path, properties, description, core_id, created_at, updated_at)
						VALUES ($1,$2,$3,$4,$5,$6,'{}'::jsonb,$7,$8,now(),now())
						ON CONFLICT (id) DO UPDATE SET core_id=EXCLUDED.core_id, updated_at=now()`
					if _, uerr := h.db.Exec(upsert, viewNodeID, tenantID, datasourceID.String(), charts.TABLE_NODE_TYPE_ID, bodyReq.TableName, qualifiedPath, fmt.Sprintf("Table node for %s", qualifiedPath), modelNodeID); uerr != nil {
						logger.Warnf("Failed to upsert catalog_node core_id for %s (id=%s): %v", qualifiedPath, viewNodeID, uerr)
					} else {
						logger.Infof("Linked table node %s to semantic model node %s via upsert", viewNodeID, modelNodeID)
					}
				}
			} else {
				logger.Warnf("No semantic model node id available to link for duplicate path %s", qualifiedPath)
			}

			response := map[string]interface{}{
				"success":       true,
				"message":       "Generated model already exists",
				"id":            existingID,
				"model_key":     modelKey,
				"version":       existingVersion,
				"table":         bodyReq.TableName,
				"schema":        bodyReq.Schema,
				"tenant_id":     tenantID.String(),
				"datasource_id": datasourceID.String(),
			}

			// Ensure custom model exists even when the core model already existed.
			ensureCustomModel()

			h.respondJSON(w, http.StatusOK, response)
			return
		}
		logger.Errorf("Failed to persist generated model: %v", err)
		h.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to save generated model"})
		return
	}

	logger.Infof("Persisted generated model id=%s key=%s version=%d", createdID, createdModelKey, createdVersion)

	// Ensure a corresponding custom model exists for the newly generated core model.
	ensureCustomModel()

	// Also update catalog_node to link this table to the new core model (core_id)
	// The catalog_node.qualified_path uses 'schema.table' format, convert modelKey accordingly
	qualifiedPath := fmt.Sprintf("%s.%s", bodyReq.Schema, bodyReq.TableName)
	// Ensure a semantic model catalog_node exists for this generated model and link the table to it.
	// The catalog_node.core_id is a FK to another catalog_node (semantic model node), not to fabric_defn.
	// Create or reuse a semantic model node with node_name = model.<table> and qualified_path = /semantic_model/<table>
	var modelNodeID string
	modelNodeName := fmt.Sprintf("model.%s", bodyReq.TableName)
	modelQualified := fmt.Sprintf("/semantic_model/%s", bodyReq.TableName)
	// Try to find an existing semantic model node for this datasource
	if err := h.db.QueryRow(`SELECT id FROM public.catalog_node WHERE tenant_datasource_id = $1 AND node_type_id = $2 AND qualified_path = $3 LIMIT 1`, datasourceID.String(), charts.MODEL_NODE_TYPE_ID, modelQualified).Scan(&modelNodeID); err != nil {
		// Not found - insert a new catalog_node record for the semantic model.
		insertModelNode := `INSERT INTO public.catalog_node (id, tenant_id, tenant_datasource_id, node_type_id, node_name, qualified_path, properties, description, created_at, updated_at) VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, '{}'::jsonb, $6, now(), now()) RETURNING id`
		if ierr := h.db.QueryRow(insertModelNode, tenantID, datasourceID.String(), charts.MODEL_NODE_TYPE_ID, modelNodeName, modelQualified, fmt.Sprintf("Semantic model for %s.%s", bodyReq.Schema, bodyReq.TableName)).Scan(&modelNodeID); ierr != nil {
			logger.Warnf("Failed to insert semantic model catalog_node: %v", ierr)
		} else {
			logger.Infof("Created semantic model catalog_node %s for %s", modelNodeID, modelQualified)
		}
	} else {
		logger.Infof("Found existing semantic model catalog_node %s for %s", modelNodeID, modelQualified)
	}

	// Finally, upsert the table catalog_node by id from the view and set core_id to the semantic model node
	if modelNodeID != "" {
		var viewNodeID string
		if verr := h.db.QueryRow(`SELECT node_id FROM public.catalog_node_vw WHERE tenant_datasource_id = $1 AND node_type_id = $2 AND qualified_path = $3 LIMIT 1`, datasourceID.String(), charts.TABLE_NODE_TYPE_ID, qualifiedPath).Scan(&viewNodeID); verr != nil || viewNodeID == "" {
			logger.Warnf("No catalog_node_vw entry found for %s (ds=%s), cannot link core_id: %v", qualifiedPath, datasourceID.String(), verr)
		} else {
			upsert := `INSERT INTO public.catalog_node (id, tenant_id, tenant_datasource_id, node_type_id, node_name, qualified_path, properties, description, core_id, created_at, updated_at)
				VALUES ($1,$2,$3,$4,$5,$6,'{}'::jsonb,$7,$8,now(),now())
				ON CONFLICT (id) DO UPDATE SET core_id=EXCLUDED.core_id, updated_at=now()`
			if _, err := h.db.Exec(upsert, viewNodeID, tenantID, datasourceID.String(), charts.TABLE_NODE_TYPE_ID, bodyReq.TableName, qualifiedPath, fmt.Sprintf("Table node for %s", qualifiedPath), modelNodeID); err != nil {
				logger.Warnf("Failed to upsert catalog_node core_id for %s (id=%s): %v", qualifiedPath, viewNodeID, err)
			} else {
				logger.Infof("Linked table node %s to semantic model node %s via upsert", viewNodeID, modelNodeID)
			}
		}
	} else {
		logger.Warnf("No semantic model node id available to link for %s", qualifiedPath)
	}

	response := map[string]interface{}{
		"success":       true,
		"message":       "Generated model saved successfully",
		"id":            createdID,
		"model_key":     createdModelKey,
		"version":       createdVersion,
		"table":         bodyReq.TableName,
		"schema":        bodyReq.Schema,
		"tenant_id":     tenantID.String(),
		"datasource_id": datasourceID.String(),
	}

	h.respondJSON(w, http.StatusCreated, response)
}

// RegisterRoutes registers all model catalog routes
func (h *ModelCatalogHandler) RegisterRoutes(router chi.Router) {
	router.Get("/models", h.GetModels)
	router.Post("/models/custom", h.CreateCustomModel)
	router.Post("/models/clone", h.CloneModel)
	router.Post("/models/generated", h.CreateGeneratedModel)
	router.Get("/models/{model_id}", h.GetModel)
	router.Patch("/models/{model_id}", h.UpdateModel)
	router.Delete("/models/{model_id}", h.DeleteModel)
}
