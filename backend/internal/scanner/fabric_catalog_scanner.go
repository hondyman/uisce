package scanner

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/cube"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/hondyman/semlayer/backend/models"
)

// FabricStatus enum values
type FabricStatus string

const (
	StatusDraft     FabricStatus = "draft"
	StatusPublished FabricStatus = "published"
	StatusArchived  FabricStatus = "archived"
)

// FabricDefinition represents a semantic model definition
type FabricDefinition struct {
	ID                 uuid.UUID       `json:"id" db:"id"`
	TenantID           uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	TenantDatasourceID uuid.UUID       `json:"tenant_datasource_id" db:"tenant_datasource_id"`
	ModelKey           string          `json:"model_key" db:"model_key"`
	Version            int             `json:"version" db:"version"`
	Status             FabricStatus    `json:"status" db:"status"`
	IsCurrent          bool            `json:"is_current" db:"is_current"`
	Title              *string         `json:"title" db:"title"`
	Description        *string         `json:"description" db:"description"`
	SourceConfig       json.RawMessage `json:"source_config" db:"source_config"`
	ResolvedConfig     json.RawMessage `json:"resolved_config" db:"resolved_config"`
	CreatedBy          uuid.UUID       `json:"created_by" db:"created_by"`
	CreatedAt          time.Time       `json:"created_at" db:"created_at"`
	PublishedAt        *time.Time      `json:"published_at" db:"published_at"`
	ChecksumSHA256     []byte          `json:"checksum_sha256" db:"checksum_sha256"`
	UpdatedAt          *time.Time      `json:"updated_at" db:"updated_at"`
}

// ModelCatalogNode represents a model in the catalog with inheritance info
type ModelCatalogNode struct {
	ID                uuid.UUID              `json:"id"`
	ModelKey          string                 `json:"model_key"`
	DisplayName       string                 `json:"display_name"`
	Description       string                 `json:"description"`
	Status            FabricStatus           `json:"status"`
	Version           int                    `json:"version"`
	IsCurrent         bool                   `json:"is_current"`
	IsCore            bool                   `json:"is_core"`
	IsCustom          bool                   `json:"is_custom"`
	CanEdit           bool                   `json:"can_edit"`
	ParentModelKey    *string                `json:"parent_model_key,omitempty"`
	CoreModelExists   bool                   `json:"core_model_exists"`
	CustomModelExists bool                   `json:"custom_model_exists"`
	SourceConfig      json.RawMessage        `json:"source_config"`
	ResolvedConfig    json.RawMessage        `json:"resolved_config"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         *time.Time             `json:"updated_at"`
	PublishedAt       *time.Time             `json:"published_at"`
	Metadata          map[string]interface{} `json:"metadata"`
}

// FabricCatalogScanner extracts fabric model metadata
type FabricCatalogScanner struct {
	db                 *sql.DB
	tenantID           uuid.UUID
	tenantDatasourceID uuid.UUID
	models             []ModelCatalogNode
}

// NewFabricCatalogScanner creates a new fabric catalog scanner
func NewFabricCatalogScanner(db *sql.DB, tenantID, tenantDatasourceID uuid.UUID) (*FabricCatalogScanner, error) {
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("could not ping database: %w", err)
	}

	return &FabricCatalogScanner{
		db:                 db,
		tenantID:           tenantID,
		tenantDatasourceID: tenantDatasourceID,
		models:             []ModelCatalogNode{},
	}, nil
}

// ExtractModels retrieves all fabric models for the datasource and creates catalog nodes
func (s *FabricCatalogScanner) ExtractModels() ([]ModelCatalogNode, error) {
	logger := logging.GetLogger().Sugar()

	// Get all models (all versions) for this tenant and datasource and prefer latest versions
	query := `
		SELECT 
			id, tenant_id, tenant_datasource_id, model_key, version, status, 
			is_current, title, description, source_config, resolved_config,
			created_by, created_at, published_at, checksum_sha256, updated_at
		FROM fabric_defn 
		WHERE tenant_id = $1 AND tenant_datasource_id = $2
		ORDER BY model_key, version DESC
	`

	rows, err := s.db.Query(query, s.tenantID, s.tenantDatasourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query fabric definitions: %w", err)
	}
	defer rows.Close()

	fabricModels := make([]FabricDefinition, 0)

	for rows.Next() {
		var model FabricDefinition
		err := rows.Scan(
			&model.ID, &model.TenantID, &model.TenantDatasourceID,
			&model.ModelKey, &model.Version, &model.Status, &model.IsCurrent,
			&model.Title, &model.Description, &model.SourceConfig, &model.ResolvedConfig,
			&model.CreatedBy, &model.CreatedAt, &model.PublishedAt,
			&model.ChecksumSHA256, &model.UpdatedAt,
		)
		if err != nil {
			logger.Warnf("Error scanning fabric model: %v", err)
			continue
		}

		fabricModels = append(fabricModels, model)
	}

	logger.Infof("Found %d fabric models for datasource %s", len(fabricModels), s.tenantDatasourceID)

	// Process models and create catalog nodes with inheritance logic
	catalogNodes := s.processModelsWithInheritance(fabricModels)

	s.models = catalogNodes
	return catalogNodes, nil
}

// processModelsWithInheritance processes fabric models and creates catalog nodes with inheritance
func (s *FabricCatalogScanner) processModelsWithInheritance(fabricModels []FabricDefinition) []ModelCatalogNode {
	logger := logging.GetLogger().Sugar()

	// Group models by base key (remove _custom suffix if present)
	modelGroups := make(map[string][]FabricDefinition)

	for _, model := range fabricModels {
		baseKey := s.getBaseModelKey(model.ModelKey)
		modelGroups[baseKey] = append(modelGroups[baseKey], model)
	}

	catalogNodes := make([]ModelCatalogNode, 0)

	for baseKey, models := range modelGroups {
		nodes := s.createCatalogNodesForGroup(baseKey, models)
		catalogNodes = append(catalogNodes, nodes...)
	}

	logger.Infof("Created %d catalog nodes from %d model groups", len(catalogNodes), len(modelGroups))

	return catalogNodes
}

// getBaseModelKey removes _custom suffix to get the base model key
func (s *FabricCatalogScanner) getBaseModelKey(modelKey string) string {
	// Support multiple custom/clone variants by trimming at first occurrence of _custom
	if idx := strings.Index(modelKey, "_custom"); idx != -1 {
		return modelKey[:idx]
	}
	return modelKey
}

// isCustomModelKey checks if a model key represents a custom model
func (s *FabricCatalogScanner) isCustomModelKey(modelKey string) bool {
	return strings.Contains(modelKey, "_custom")
}

// createCatalogNodesForGroup creates catalog nodes for a group of related models
func (s *FabricCatalogScanner) createCatalogNodesForGroup(baseKey string, models []FabricDefinition) []ModelCatalogNode {
	logger := logging.GetLogger().Sugar()

	var coreModel *FabricDefinition
	customModels := make([]*FabricDefinition, 0)

	for i := range models {
		m := models[i]
		if s.isCustomModelKey(m.ModelKey) {
			// ensure unique by model_key (latest version only)
			dup := false
			for _, existing := range customModels {
				if existing.ModelKey == m.ModelKey {
					dup = true
					break
				}
			}
			if !dup {
				customModels = append(customModels, &m)
			}
		} else if coreModel == nil {
			coreModel = &m
		}
	}

	catalogNodes := make([]ModelCatalogNode, 0)
	if coreModel != nil {
		coreNode := s.createCoreModelNode(*coreModel, len(customModels) > 0)
		catalogNodes = append(catalogNodes, coreNode)
		logger.Debugf("Created core model node: %s", coreModel.ModelKey)
	}

	if len(customModels) == 0 && coreModel != nil {
		potential := s.createCustomModelNode(baseKey, coreModel, nil)
		catalogNodes = append(catalogNodes, potential)
		logger.Debugf("Created potential custom model node: %s_custom", baseKey)
	} else {
		for _, cm := range customModels {
			cn := s.createCustomModelNode(baseKey, coreModel, cm)
			// Mark clone variants (non canonical key)
			if cm.ModelKey != baseKey+"_custom" {
				if cn.Metadata == nil {
					cn.Metadata = map[string]interface{}{}
				}
				cn.Metadata["is_clone_variant"] = true
			}
			catalogNodes = append(catalogNodes, cn)
			logger.Debugf("Added custom/clone model node: %s", cm.ModelKey)
		}
	}
	return catalogNodes
}

// createCoreModelNode creates a catalog node for a core model
func (s *FabricCatalogScanner) createCoreModelNode(model FabricDefinition, hasCustom bool) ModelCatalogNode {
	displayName := s.getDisplayName(model)
	description := s.getDescription(model)

	// Check if this is a core model (has core_id or special indicator)
	isCore := s.isCoreModel(model)

	metadata := map[string]interface{}{
		"generator":          s.extractGenerator(model.SourceConfig),
		"has_custom_version": hasCustom,
		"table_count":        s.extractTableCount(model.ResolvedConfig),
		"measure_count":      s.extractMeasureCount(model.ResolvedConfig),
		"dimension_count":    s.extractDimensionCount(model.ResolvedConfig),
	}

	return ModelCatalogNode{
		ID:                model.ID,
		ModelKey:          model.ModelKey,
		DisplayName:       displayName,
		Description:       description,
		Status:            model.Status,
		Version:           model.Version,
		IsCurrent:         model.IsCurrent,
		IsCore:            isCore,
		IsCustom:          false,
		CanEdit:           !isCore, // Core models cannot be edited
		ParentModelKey:    nil,
		CoreModelExists:   true,
		CustomModelExists: hasCustom,
		SourceConfig:      model.SourceConfig,
		ResolvedConfig:    model.ResolvedConfig,
		CreatedAt:         model.CreatedAt,
		UpdatedAt:         model.UpdatedAt,
		PublishedAt:       model.PublishedAt,
		Metadata:          metadata,
	}
}

// createCustomModelNode creates a catalog node for a custom model (existing or potential)
func (s *FabricCatalogScanner) createCustomModelNode(baseKey string, coreModel *FabricDefinition, customModel *FabricDefinition) ModelCatalogNode {
	customKey := baseKey + "_custom"

	if customModel != nil {
		// Existing custom model
		displayName := s.getDisplayName(*customModel)
		description := s.getDescription(*customModel)

		// Merge resolved config if we have both core and custom models
		resolvedConfig := customModel.ResolvedConfig
		if coreModel != nil {
			// Parse the core and custom resolved configs
			var coreConfig, customConfig models.ResolvedModelConfig
			if err := json.Unmarshal(coreModel.ResolvedConfig, &coreConfig); err == nil {
				if err := json.Unmarshal(customModel.ResolvedConfig, &customConfig); err == nil {
					// If custom config has the old format (direct extends/dimensions), convert it
					if len(customConfig.Cubes) == 0 {
						// Convert old format to new format
						customCube := cube.Cube{
							Name:       customModel.ModelKey,
							Extends:    baseKey,
							Dimensions: s.convertOldDimensionsFormat(customModel.ResolvedConfig),
							Measures:   s.convertOldMeasuresFormat(customModel.ResolvedConfig),
						}
						customConfig = models.ResolvedModelConfig{
							ModelKey: customModel.ModelKey,
							Cubes:    []cube.Cube{customCube},
						}
					}

					// Merge the cubes
					if len(coreConfig.Cubes) > 0 && len(customConfig.Cubes) > 0 {
						mergedCube, _ := cube.MergeCube(coreConfig.Cubes[0], customConfig.Cubes[0])
						mergedConfig := models.ResolvedModelConfig{
							ModelKey: customModel.ModelKey,
							Cubes:    []cube.Cube{mergedCube},
						}
						mergedConfigJSON, _ := json.Marshal(mergedConfig)
						resolvedConfig = mergedConfigJSON
					}
				}
			}
		}

		metadata := map[string]interface{}{
			"generator":       "custom",
			"inherits_from":   baseKey,
			"table_count":     s.extractTableCount(resolvedConfig),
			"measure_count":   s.extractMeasureCount(resolvedConfig),
			"dimension_count": s.extractDimensionCount(resolvedConfig),
		}

		return ModelCatalogNode{
			ID:                customModel.ID,
			ModelKey:          customModel.ModelKey,
			DisplayName:       displayName,
			Description:       description,
			Status:            customModel.Status,
			Version:           customModel.Version,
			IsCurrent:         customModel.IsCurrent,
			IsCore:            false,
			IsCustom:          true,
			CanEdit:           true,
			ParentModelKey:    &baseKey,
			CoreModelExists:   coreModel != nil,
			CustomModelExists: true,
			SourceConfig:      customModel.SourceConfig,
			ResolvedConfig:    resolvedConfig,
			CreatedAt:         customModel.CreatedAt,
			UpdatedAt:         customModel.UpdatedAt,
			PublishedAt:       customModel.PublishedAt,
			Metadata:          metadata,
		}
	} else {
		// Potential custom model (doesn't exist yet)
		now := time.Now()
		displayName := ""
		description := "Custom semantic model (not yet created)"

		if coreModel != nil {
			coreDisplayName := s.getDisplayName(*coreModel)
			displayName = coreDisplayName
			if !strings.HasSuffix(displayName, " (Custom)") {
				displayName += " (Custom)"
			}
			description = "Custom version of " + coreDisplayName
		}

		metadata := map[string]interface{}{
			"generator":       "custom",
			"inherits_from":   baseKey,
			"can_create":      coreModel != nil,
			"table_count":     0,
			"measure_count":   0,
			"dimension_count": 0,
		}

		// Generate a placeholder ID for the potential custom model
		placeholderID := uuid.New()

		return ModelCatalogNode{
			ID:                placeholderID,
			ModelKey:          customKey,
			DisplayName:       displayName,
			Description:       description,
			Status:            StatusDraft,
			Version:           0,
			IsCurrent:         false,
			IsCore:            false,
			IsCustom:          true,
			CanEdit:           true,
			ParentModelKey:    &baseKey,
			CoreModelExists:   coreModel != nil,
			CustomModelExists: false,
			SourceConfig:      json.RawMessage(`{"generator": "custom"}`),
			ResolvedConfig:    json.RawMessage(`{}`),
			CreatedAt:         now,
			UpdatedAt:         nil,
			PublishedAt:       nil,
			Metadata:          metadata,
		}
	}
}

// Helper functions to extract information from models

func (s *FabricCatalogScanner) getDisplayName(model FabricDefinition) string {
	if model.Title != nil && *model.Title != "" {
		return *model.Title
	}
	// Extract table name from model key (e.g., "/public/customers" -> "customers")
	parts := strings.Split(strings.Trim(model.ModelKey, "/"), "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return model.ModelKey
}

func (s *FabricCatalogScanner) getDescription(model FabricDefinition) string {
	if model.Description != nil {
		return *model.Description
	}
	return "Semantic model"
}

func (s *FabricCatalogScanner) isCoreModel(model FabricDefinition) bool {
	// Check if this model has indicators that it's a core model
	// This could be based on source_config, special naming, or other metadata
	var sourceConfig map[string]interface{}
	if err := json.Unmarshal(model.SourceConfig, &sourceConfig); err == nil {
		if generator, ok := sourceConfig["generator"].(string); ok {
			return generator == "core" || generator == "single-table"
		}
	}
	return false
}

func (s *FabricCatalogScanner) extractGenerator(sourceConfig json.RawMessage) string {
	var config map[string]interface{}
	if err := json.Unmarshal(sourceConfig, &config); err == nil {
		if generator, ok := config["generator"].(string); ok {
			return generator
		}
	}
	return "unknown"
}

func (s *FabricCatalogScanner) extractTableCount(resolvedConfig json.RawMessage) int {
	var config map[string]interface{}
	if err := json.Unmarshal(resolvedConfig, &config); err == nil {
		if cubes, ok := config["cubes"].([]interface{}); ok {
			return len(cubes)
		}
	}
	return 0
}

func (s *FabricCatalogScanner) extractMeasureCount(resolvedConfig json.RawMessage) int {
	var config map[string]interface{}
	if err := json.Unmarshal(resolvedConfig, &config); err == nil {
		if cubes, ok := config["cubes"].([]interface{}); ok {
			count := 0
			for _, cube := range cubes {
				if cubeMap, ok := cube.(map[string]interface{}); ok {
					if measures, ok := cubeMap["measures"].(map[string]interface{}); ok {
						count += len(measures)
					}
				}
			}
			return count
		}
	}
	return 0
}

func (s *FabricCatalogScanner) extractDimensionCount(resolvedConfig json.RawMessage) int {
	var config map[string]interface{}
	if err := json.Unmarshal(resolvedConfig, &config); err == nil {
		if cubes, ok := config["cubes"].([]interface{}); ok {
			count := 0
			for _, cube := range cubes {
				if cubeMap, ok := cube.(map[string]interface{}); ok {
					if dimensions, ok := cubeMap["dimensions"].(map[string]interface{}); ok {
						count += len(dimensions)
					}
				}
			}
			return count
		}
	}
	return 0
}

// CreateCustomModel creates a new custom model inheriting from a core model
func (s *FabricCatalogScanner) CreateCustomModel(baseModelKey string, userID uuid.UUID) (*ModelCatalogNode, error) {
	logger := logging.GetLogger().Sugar()

	// First, get the core model
	coreModel, err := s.getCoreModel(baseModelKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get core model: %w", err)
	}

	if coreModel == nil {
		return nil, fmt.Errorf("core model not found: %s", baseModelKey)
	}

	// Check if custom model already exists
	customKey := baseModelKey + "_custom"
	if s.customModelExists(customKey) {
		return nil, fmt.Errorf("custom model already exists: %s", customKey)
	}

	coreDisplayName := s.getDisplayName(*coreModel)
	customTitle := coreDisplayName
	if !strings.HasSuffix(customTitle, " (Custom)") {
		customTitle += " (Custom)"
	}
	description := "Custom version of " + coreDisplayName

	// Create the custom model by copying the core model
	customModel := &FabricDefinition{
		ID:                 uuid.New(),
		TenantID:           s.tenantID,
		TenantDatasourceID: s.tenantDatasourceID,
		ModelKey:           customKey,
		Version:            1,
		Status:             StatusDraft,
		IsCurrent:          true,
		Title:              &customTitle,
		Description:        &description,
		SourceConfig:       json.RawMessage(`{"generator": "custom", "inherits_from": "` + baseModelKey + `"}`),
		ResolvedConfig:     coreModel.ResolvedConfig, // Start with core model config
		CreatedBy:          userID,
		CreatedAt:          time.Now(),
		PublishedAt:        nil,
		ChecksumSHA256:     nil,
		UpdatedAt:          nil,
	}

	// Insert the custom model into the database (compute checksum of resolved_config)
	checksum := sha256.Sum256([]byte(customModel.ResolvedConfig))
	query := `
		INSERT INTO fabric_defn (
			id, tenant_id, tenant_datasource_id, model_key, version, status, 
			is_current, title, description, source_config, resolved_config, created_by, checksum_sha256
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err = s.db.Exec(query,
		customModel.ID, customModel.TenantID, customModel.TenantDatasourceID,
		customModel.ModelKey, customModel.Version, customModel.Status,
		customModel.IsCurrent, customModel.Title, customModel.Description,
		customModel.SourceConfig, customModel.ResolvedConfig, customModel.CreatedBy, checksum[:],
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create custom model: %w", err)
	}

	logger.Infof("Created custom model: %s", customKey)

	// Convert to catalog node
	catalogNode := s.createCustomModelNode(baseModelKey, coreModel, customModel)

	return &catalogNode, nil
}

func (s *FabricCatalogScanner) getCoreModel(modelKey string) (*FabricDefinition, error) {
	query := `
		SELECT 
			id, tenant_id, tenant_datasource_id, model_key, version, status, 
			is_current, title, description, source_config, resolved_config,
			created_by, created_at, published_at, checksum_sha256, updated_at
		FROM fabric_defn 
		WHERE tenant_id = $1 AND tenant_datasource_id = $2 AND model_key = $3 AND is_current = true
	`

	var model FabricDefinition
	err := s.db.QueryRow(query, s.tenantID, s.tenantDatasourceID, modelKey).Scan(
		&model.ID, &model.TenantID, &model.TenantDatasourceID,
		&model.ModelKey, &model.Version, &model.Status, &model.IsCurrent,
		&model.Title, &model.Description, &model.SourceConfig, &model.ResolvedConfig,
		&model.CreatedBy, &model.CreatedAt, &model.PublishedAt,
		&model.ChecksumSHA256, &model.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &model, nil
}

func (s *FabricCatalogScanner) customModelExists(customKey string) bool {
	query := `
		SELECT 1 FROM fabric_defn 
		WHERE tenant_id = $1 AND tenant_datasource_id = $2 AND model_key = $3 AND is_current = true
	`

	var exists int
	err := s.db.QueryRow(query, s.tenantID, s.tenantDatasourceID, customKey).Scan(&exists)
	return err == nil
}

// CloneModel creates a new custom clone (distinct key) from an existing model (core or custom)
func (s *FabricCatalogScanner) CloneModel(modelID uuid.UUID, userID uuid.UUID) (*ModelCatalogNode, error) {
	logger := logging.GetLogger().Sugar()

	query := `SELECT id, tenant_id, tenant_datasource_id, model_key, version, status, is_current, title, description, source_config, resolved_config, created_by, created_at, published_at, checksum_sha256, updated_at FROM fabric_defn WHERE id = $1 AND tenant_id = $2 AND tenant_datasource_id = $3`
	var base FabricDefinition
	if err := s.db.QueryRow(query, modelID, s.tenantID, s.tenantDatasourceID).Scan(
		&base.ID, &base.TenantID, &base.TenantDatasourceID, &base.ModelKey, &base.Version, &base.Status,
		&base.IsCurrent, &base.Title, &base.Description, &base.SourceConfig, &base.ResolvedConfig,
		&base.CreatedBy, &base.CreatedAt, &base.PublishedAt, &base.ChecksumSHA256, &base.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("base model not found: %w", err)
	}

	baseKey := s.getBaseModelKey(base.ModelKey)

	// find unique key
	var cloneKey string
	for i := 0; i < 20; i++ {
		candidate := fmt.Sprintf("%s_custom_%04d", baseKey, rand.Intn(10000))
		if !s.customModelExists(candidate) {
			cloneKey = candidate
			break
		}
	}
	if cloneKey == "" {
		return nil, fmt.Errorf("unable to generate unique clone key for %s", baseKey)
	}

	titleVal := "Clone of " + baseKey
	if base.Title != nil && *base.Title != "" {
		titleVal = "Clone of " + *base.Title
	}
	descVal := "Cloned variant of " + base.ModelKey
	now := time.Now()

	clone := &FabricDefinition{
		ID:                 uuid.New(),
		TenantID:           s.tenantID,
		TenantDatasourceID: s.tenantDatasourceID,
		ModelKey:           cloneKey,
		Version:            1,
		Status:             StatusDraft,
		IsCurrent:          true,
		Title:              &titleVal,
		Description:        &descVal,
		SourceConfig:       json.RawMessage(`{"generator":"custom","inherits_from":"` + base.ModelKey + `"}`),
		ResolvedConfig:     base.ResolvedConfig,
		CreatedBy:          userID,
		CreatedAt:          now,
		PublishedAt:        nil,
		ChecksumSHA256:     nil,
		UpdatedAt:          nil,
	}

	// compute checksum of resolved_config for clone
	ch := sha256.Sum256([]byte(clone.ResolvedConfig))
	insert := `INSERT INTO fabric_defn (id, tenant_id, tenant_datasource_id, model_key, version, status, is_current, title, description, source_config, resolved_config, created_by, checksum_sha256) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`
	if _, err := s.db.Exec(insert, clone.ID, clone.TenantID, clone.TenantDatasourceID, clone.ModelKey, clone.Version, clone.Status, clone.IsCurrent, clone.Title, clone.Description, clone.SourceConfig, clone.ResolvedConfig, clone.CreatedBy, ch[:]); err != nil {
		return nil, fmt.Errorf("failed to insert clone: %w", err)
	}

	var coreModel *FabricDefinition
	if !s.isCustomModelKey(base.ModelKey) {
		coreModel = &base
	} else {
		cm, _ := s.getCoreModel(baseKey)
		coreModel = cm
	}
	node := s.createCustomModelNode(baseKey, coreModel, clone)
	if node.Metadata == nil {
		node.Metadata = map[string]interface{}{}
	}
	node.Metadata["is_clone_variant"] = true
	logger.Infof("Cloned model %s -> %s", base.ModelKey, clone.ModelKey)
	return &node, nil
}

// convertOldDimensionsFormat converts old format dimensions to cube.Cube format
func (s *FabricCatalogScanner) convertOldDimensionsFormat(resolvedConfig json.RawMessage) map[string]map[string]any {
	var config map[string]interface{}
	if err := json.Unmarshal(resolvedConfig, &config); err != nil {
		return nil
	}

	if dims, ok := config["dimensions"].([]interface{}); ok {
		result := make(map[string]map[string]any)
		for _, dim := range dims {
			if dimMap, ok := dim.(map[string]interface{}); ok {
				if name, ok := dimMap["name"].(string); ok {
					result[name] = dimMap
				}
			}
		}
		return result
	}

	return nil
}

// convertOldMeasuresFormat converts old format measures to cube.Cube format
func (s *FabricCatalogScanner) convertOldMeasuresFormat(resolvedConfig json.RawMessage) map[string]map[string]any {
	var config map[string]interface{}
	if err := json.Unmarshal(resolvedConfig, &config); err != nil {
		return nil
	}

	if measures, ok := config["measures"].([]interface{}); ok {
		result := make(map[string]map[string]any)
		for _, measure := range measures {
			if measureMap, ok := measure.(map[string]interface{}); ok {
				if name, ok := measureMap["name"].(string); ok {
					result[name] = measureMap
				}
			}
		}
		return result
	}

	return nil
}
