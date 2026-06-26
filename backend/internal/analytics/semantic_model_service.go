package analytics

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/cube"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/hondyman/semlayer/backend/models"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// SemanticModelService provides methods for interacting with semantic models.
type SemanticModelService struct {
	DB *sqlx.DB
}

// NewSemanticModelService creates a new SemanticModelService.
func NewSemanticModelService(db *sqlx.DB) *SemanticModelService {
	return &SemanticModelService{DB: db}
}

var (
	NODE_TYPE_TABLE           = mustParseOrDefault("SEMLAYER_NODETYPE_TABLE", "49a50271-ae58-4d3e-ae1c-2f5b89d89192")
	NODE_TYPE_COLUMN          = mustParseOrDefault("SEMLAYER_NODETYPE_DATABASE_COLUMN", "a64c1011-16e8-4ddf-b447-363bf8e15c9a")
	NODE_TYPE_SEMANTIC_MODEL  = mustParseOrDefault("SEMLAYER_NODETYPE_SEMANTIC_MODEL", "c53f9e99-8d02-4dfb-bc1b-914747d35edb")
	NODE_TYPE_SEMANTIC_COLUMN = mustParseOrDefault("SEMLAYER_NODETYPE_SEMANTIC_COLUMN", "1439f761-606a-44cb-b4f8-7aa6b27a9bf5")
	EDGE_TYPE_TABLE_TO_MODEL  = mustParseOrDefault("SEMLAYER_EDGETYPE_MODEL_TABLE", "b2c3d4e5-6789-0123-4567-890123456789")
	EDGE_TYPE_SEMANTIC_MAP    = mustParseOrDefault("SEMLAYER_EDGETYPE_MAPPED_TO", "97d82101-2b84-47a6-9ec0-f930fe389c3c")
	catalogUUIDNamespace      = uuid.MustParse("1b671a64-40d5-491e-99b0-da01ff1f3341")
)

func mustParseOrDefault(envKey, fallback string) uuid.UUID {
	if val := strings.TrimSpace(os.Getenv(envKey)); val != "" {
		if parsed, err := uuid.Parse(val); err == nil {
			return parsed
		}
	}
	return uuid.MustParse(fallback)
}

func catalogDeterministicID(parts ...string) uuid.UUID {
	combined := strings.Builder{}
	for _, p := range parts {
		combined.WriteString(strings.ToLower(strings.TrimSpace(p)))
	}
	return uuid.NewSHA1(catalogUUIDNamespace, []byte(combined.String()))
}

type semanticCatalogItem struct {
	ID         uuid.UUID
	Kind       string
	Key        string
	SQL        string
	Props      map[string]interface{}
	ColumnNode *models.CatalogNode
}

func (s *SemanticModelService) syncCatalogForGeneratedModel(ctx context.Context, tenantID, tenantDatasourceID uuid.UUID, tableNode models.CatalogNode, columns []models.CatalogNode, cube *cube.Cube, defn *models.FabricDefn) error {
	if s.DB == nil {
		return fmt.Errorf("db handle is nil")
	}

	tx, err := s.DB.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin catalog sync tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	now := time.Now()

	qualifiedModelKey := strings.TrimPrefix(defn.ModelKey, "/")
	modelQualifiedPath := fmt.Sprintf("/semantic_model/%s", qualifiedModelKey)
	modelNodeName := fmt.Sprintf("model.%s", strings.ReplaceAll(qualifiedModelKey, "/", "."))
	modelNodeID := catalogDeterministicID(tenantDatasourceID.String(), "semantic_model", defn.ModelKey)

	dimensionKeys := make([]string, 0, len(cube.Dimensions))
	for key := range cube.Dimensions {
		dimensionKeys = append(dimensionKeys, key)
	}
	sort.Strings(dimensionKeys)

	measureKeys := make([]string, 0, len(cube.Measures))
	for key := range cube.Measures {
		measureKeys = append(measureKeys, key)
	}
	sort.Strings(measureKeys)

	modelProps := map[string]interface{}{
		// Existing auto-generated properties
		"model_key":      defn.ModelKey,
		"fabric_defn_id": defn.ID.String(),
		"source_table":   tableNode.QualifiedPath,
		"generator":      "single-table",
		"dimension_keys": dimensionKeys,
		"measure_keys":   measureKeys,
		"column_count":   len(columns),
		"is_core":        tableNode.CoreID.Valid,
		"version":        defn.Version,
		"status":         defn.Status,
		
		// NEW: Basic Identification
		"technical_name": defn.ModelKey, // Use model_key as technical name
		
		// NEW: Core vs Custom Status
		"model_type": func() string {
			if tableNode.CoreID.Valid {
				return "core"
			}
			return "custom"
		}(),
		
		// NEW: Data Source Information
		"data_source_description": fmt.Sprintf("Generated from table: %s", tableNode.QualifiedPath),
		"schema_table_reference":  tableNode.QualifiedPath,
		
		// Existing timestamps
		"last_synced_at": now.Format(time.RFC3339Nano),
		"generated_by":   "semantic-model-service",
	}
	if defn.PublishedAt != nil {
		modelProps["published_at"] = defn.PublishedAt.Format(time.RFC3339Nano)
	}
	if defn.CreatedAt != nil {
		modelProps["created_at"] = defn.CreatedAt.Format(time.RFC3339Nano)
	}

	modelPropsJSON, _ := json.Marshal(modelProps)

	var coreID interface{}
	if tableNode.CoreID.Valid {
		coreID = tableNode.CoreID.UUID
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO public.catalog_node (
			id, tenant_id, tenant_datasource_id, node_type_id, node_name, qualified_path,
			description, parent_id, properties, core_id, is_alpha, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,NULL,$8,$9,$10,$11,$11)
		ON CONFLICT (tenant_datasource_id, node_type_id, qualified_path) DO UPDATE SET
			node_name = EXCLUDED.node_name,
			description = EXCLUDED.description,
			properties = EXCLUDED.properties,
			core_id = EXCLUDED.core_id,
			is_alpha = EXCLUDED.is_alpha,
			updated_at = EXCLUDED.updated_at;
	`,
		modelNodeID,
		tenantID,
		tenantDatasourceID,
		NODE_TYPE_SEMANTIC_MODEL,
		modelNodeName,
		modelQualifiedPath,
		fmt.Sprintf("Semantic model generated for %s", strings.TrimPrefix(tableNode.QualifiedPath, "/")),
		modelPropsJSON,
		coreID,
		tableNode.IsAlpha,
		now,
	)
	if err != nil {
		return fmt.Errorf("upsert semantic model node: %w", err)
	}

	// Ensure we have the persisted model node ID (may differ if an existing one already existed)
	if err = tx.GetContext(ctx, &modelNodeID, `
		SELECT id FROM public.catalog_node WHERE tenant_datasource_id = $1 AND node_type_id = $2 AND qualified_path = $3
	`, tenantDatasourceID, NODE_TYPE_SEMANTIC_MODEL, modelQualifiedPath); err != nil {
		return fmt.Errorf("load persisted model node id: %w", err)
	}

	// Update table properties to reflect model linkage
	tableProps := map[string]interface{}{}
	if len(tableNode.Properties) > 0 {
		_ = json.Unmarshal(tableNode.Properties, &tableProps)
	}
	tableProps["has_model"] = true
	tableProps["model_node_id"] = modelNodeID.String()
	tableProps["model_key"] = defn.ModelKey
	tableProps["last_model_sync_at"] = now.Format(time.RFC3339Nano)
	tablePropsJSON, _ := json.Marshal(tableProps)

	if _, err = tx.ExecContext(ctx, `
		UPDATE public.catalog_node
		SET properties = $1,
			updated_at = $2
		WHERE id = $3
	`, tablePropsJSON, now, tableNode.ID); err != nil {
		return fmt.Errorf("update table node properties: %w", err)
	}

	// Map of column names for lookup (case-insensitive)
	columnByName := make(map[string]models.CatalogNode, len(columns))
	for _, col := range columns {
		columnByName[strings.ToLower(col.NodeName)] = col
	}

	items := make([]semanticCatalogItem, 0, len(dimensionKeys)+len(measureKeys))

	for _, key := range dimensionKeys {
		d := cube.Dimensions[key]
		sqlExpr, _ := d["sql"].(string)
		columnName := extractBaseColumnName(sqlExpr)
		columnNode, found := columnByName[strings.ToLower(columnName)]
		var columnRef *models.CatalogNode
		if found {
			c := columnNode
			columnRef = &c
		}
		itemProps := map[string]interface{}{
			"sql":     sqlExpr,
			"type":    d["type"],
			"kind":    "dimension",
			"title":   d["title"],
			"is_core": tableNode.CoreID.Valid,
		}
		if meta, ok := d["meta"]; ok {
			itemProps["meta"] = meta
		}
		if description, ok := d["description"].(string); ok && description != "" {
			itemProps["description"] = description
		}

		items = append(items, semanticCatalogItem{
			ID:         catalogDeterministicID(tenantDatasourceID.String(), defn.ModelKey, "dimension", key),
			Kind:       "dimension",
			Key:        key,
			SQL:        sqlExpr,
			Props:      itemProps,
			ColumnNode: columnRef,
		})
	}

	for _, key := range measureKeys {
		m := cube.Measures[key]
		sqlExpr, _ := m["sql"].(string)
		columnName := extractBaseColumnName(sqlExpr)
		columnNode, found := columnByName[strings.ToLower(columnName)]
		var columnRef *models.CatalogNode
		if found {
			c := columnNode
			columnRef = &c
		}
		itemProps := map[string]interface{}{
			"sql":     sqlExpr,
			"type":    m["type"],
			"kind":    "measure",
			"title":   m["title"],
			"is_core": tableNode.CoreID.Valid,
		}
		if meta, ok := m["meta"]; ok {
			itemProps["meta"] = meta
		}

		items = append(items, semanticCatalogItem{
			ID:         catalogDeterministicID(tenantDatasourceID.String(), defn.ModelKey, "measure", key),
			Kind:       "measure",
			Key:        key,
			SQL:        sqlExpr,
			Props:      itemProps,
			ColumnNode: columnRef,
		})
	}

	semanticNodeQualified := func(key string) string {
		return fmt.Sprintf("%s/%s", modelQualifiedPath, key)
	}

	for _, item := range items {
		var parentID interface{} = modelNodeID
		semanticPropsJSON, _ := json.Marshal(item.Props)

		var semanticCore interface{}
		if item.ColumnNode != nil && item.ColumnNode.CoreID.Valid {
			semanticCore = item.ColumnNode.CoreID.UUID
		}

		title := item.Kind
		if len(title) > 0 {
			title = strings.ToUpper(title[:1]) + title[1:]
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO public.catalog_node (
				id, tenant_id, tenant_datasource_id, node_type_id, node_name, qualified_path,
				description, parent_id, properties, core_id, is_alpha, created_at, updated_at
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$12)
			ON CONFLICT (tenant_datasource_id, node_type_id, qualified_path) DO UPDATE SET
				node_name = EXCLUDED.node_name,
				description = EXCLUDED.description,
				properties = EXCLUDED.properties,
				core_id = EXCLUDED.core_id,
				is_alpha = EXCLUDED.is_alpha,
				updated_at = EXCLUDED.updated_at;
		`,
			item.ID,
			tenantID,
			tenantDatasourceID,
			NODE_TYPE_SEMANTIC_COLUMN,
			fmt.Sprintf("%s.%s", cube.Name, item.Key),
			semanticNodeQualified(item.Key),
			fmt.Sprintf("%s %s for %s", title, item.Key, cube.Name),
			parentID,
			semanticPropsJSON,
			semanticCore,
			tableNode.IsAlpha,
			now,
		)
		if err != nil {
			return fmt.Errorf("upsert semantic %s node %s: %w", item.Kind, item.Key, err)
		}
	}

	// Upsert table -> model edge
	edgeProps := map[string]interface{}{
		"fabric_defn_id": defn.ID.String(),
		"relationship":   "table_to_model",
	}
	edgePropsJSON, _ := json.Marshal(edgeProps)
	modelEdgeID := catalogDeterministicID(tableNode.ID.String(), modelNodeID.String(), EDGE_TYPE_TABLE_TO_MODEL.String())

	_, err = tx.ExecContext(ctx, `
		INSERT INTO public.catalog_edge (
			id, tenant_id, tenant_datasource_id, source_node_id, target_node_id,
			edge_type_id, relationship_type, properties, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$9)
		ON CONFLICT (id) DO UPDATE SET
			relationship_type = EXCLUDED.relationship_type,
			properties = EXCLUDED.properties,
			updated_at = EXCLUDED.updated_at;
	`,
		modelEdgeID,
		tenantID,
		tenantDatasourceID,
		tableNode.ID,
		modelNodeID,
		EDGE_TYPE_TABLE_TO_MODEL,
		"references",
		edgePropsJSON,
		now,
	)
	if err != nil {
		return fmt.Errorf("upsert table->model edge: %w", err)
	}

	for _, item := range items {
		if item.ColumnNode == nil || item.ColumnNode.ID == uuid.Nil {
			continue
		}

		props := map[string]interface{}{
			"mapping_kind": item.Kind,
			"sql":          item.SQL,
		}
		propsJSON, _ := json.Marshal(props)
		edgeID := catalogDeterministicID(item.ColumnNode.ID.String(), item.ID.String(), EDGE_TYPE_SEMANTIC_MAP.String())
		_, err = tx.ExecContext(ctx, `
			INSERT INTO public.catalog_edge (
				id, tenant_id, tenant_datasource_id, source_node_id, target_node_id,
				edge_type_id, relationship_type, properties, created_at, updated_at
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$9)
			ON CONFLICT (id) DO UPDATE SET
				relationship_type = EXCLUDED.relationship_type,
				properties = EXCLUDED.properties,
				updated_at = EXCLUDED.updated_at;
		`,
			edgeID,
			tenantID,
			tenantDatasourceID,
			item.ColumnNode.ID,
			item.ID,
			EDGE_TYPE_SEMANTIC_MAP,
			"mapped_to",
			propsJSON,
			now,
		)
		if err != nil {
			return fmt.Errorf("upsert column->semantic edge (%s -> %s): %w", item.ColumnNode.ID, item.ID, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit catalog sync tx: %w", err)
	}

	return nil
}

func extractBaseColumnName(expr string) string {
	if expr == "" {
		return ""
	}
	trimmed := strings.TrimSpace(expr)
	// Handle nested functions by stripping outermost call
	if idx := strings.Index(trimmed, "("); idx != -1 && strings.HasSuffix(trimmed, ")") {
		inner := trimmed[idx+1 : len(trimmed)-1]
		if inner != "" {
			return extractBaseColumnName(inner)
		}
	}
	// Remove table qualifiers
	if dot := strings.LastIndex(trimmed, "."); dot != -1 {
		trimmed = trimmed[dot+1:]
	}
	trimmed = strings.Trim(trimmed, "`\"")
	trimmed = strings.TrimSpace(trimmed)
	return trimmed
}

// ListModels retrieves all fabric definitions for a specific tenant datasource.
func (s *SemanticModelService) ListModels(tenantDatasourceID uuid.UUID) ([]*models.FabricDefn, error) {
	var models []*models.FabricDefn
	query := "SELECT * FROM public.fabric_defn WHERE tenant_datasource_id = $1 ORDER BY title asc"
	err := s.DB.Select(&models, query, tenantDatasourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list models for datasource %s: %w", tenantDatasourceID, err)
	}
	logging.GetLogger().Sugar().Infof("[BACKEND_SERVICE] Found %d models for datasource %s", len(models), tenantDatasourceID)
	return models, nil
}

// GetModelMetadata returns a map of tableName -> metadata for the given tables.
func (s *SemanticModelService) GetModelMetadata(tenantDatasourceID uuid.UUID, tableNames []string) (map[string]models.ModelMetadata, error) {
	metadataMap := make(map[string]models.ModelMetadata)
	if len(tableNames) == 0 {
		return metadataMap, nil
	}

	// Initialize all as non-existent, preserving original casing
	for _, t := range tableNames {
		metadataMap[t] = models.ModelMetadata{TableName: t, Exists: false}
	}

	// Create a map for case-insensitive lookup of original names
	lowerToOriginal := make(map[string]string)
	qualifiedPaths := make([]string, len(tableNames))
	for i, name := range tableNames {
		lower := strings.ToLower(name)
		// Convert "schema.table" to "/schema/table" to match model_key format
		qualified := "/" + strings.Replace(lower, ".", "/", 1)
		qualifiedPaths[i] = qualified
		lowerToOriginal[qualified] = name // Map qualified path back to original name
	}

	query := `
		SELECT 
			model_key AS table_name, 
			title,
			created_at, 
			published_at AS updated_at, 
			created_by
		FROM public.fabric_defn
		WHERE tenant_datasource_id = $1
		  AND model_key = ANY($2)
	`
	var rows []models.ModelMetadata
	err := s.DB.Select(&rows, query, tenantDatasourceID, pq.Array(qualifiedPaths))
	if err != nil {
		return nil, err
	}

	// Update the map with data for existing models, using original casing for keys
	for _, row := range rows {
		if originalName, ok := lowerToOriginal[row.TableName]; ok {
			row.Exists = true
			row.ModelKey = originalName
			row.TableName = originalName // Ensure the struct has the original casing too
			metadataMap[originalName] = row
		}
	}

	return metadataMap, nil
}

// DeleteModels removes fabric definitions for the given table names.
func (s *SemanticModelService) DeleteModels(tenantDatasourceID uuid.UUID, tableNames []string) error {
	if len(tableNames) == 0 {
		return nil
	}

	qualifiedPaths := make([]string, len(tableNames))
	for i, name := range tableNames {
		// Convert "schema.table" from frontend to "/schema/table" to match model_key format
		lower := strings.ToLower(name)
		qualifiedPaths[i] = "/" + strings.Replace(lower, ".", "/", 1)
	}

	query := `DELETE FROM public.fabric_defn WHERE tenant_datasource_id = $1 AND model_key = ANY($2)`
	_, err := s.DB.Exec(query, tenantDatasourceID, pq.Array(qualifiedPaths))
	if err != nil {
		return fmt.Errorf("failed to delete models for datasource %s: %w", tenantDatasourceID, err)
	}
	return nil
}

// GetModelDefinition retrieves a single fabric definition by its model key.
func (s *SemanticModelService) GetModelDefinition(tenantDatasourceID uuid.UUID, modelKey string) (*models.FabricDefn, error) {
	// First try current version
	var defn models.FabricDefn
	currentQuery := "SELECT * FROM public.fabric_defn WHERE tenant_datasource_id = $1 AND model_key = $2 AND is_current = true"
	err := s.DB.Get(&defn, currentQuery, tenantDatasourceID, modelKey)
	if err == nil {
		return &defn, nil
	}
	if err == sql.ErrNoRows {
		// No current row, will try fallback below
	} else {
		return nil, fmt.Errorf("failed to get model definition: %w", err)
	}

	// Fallback: get latest version (highest version) if no current row
	fallbackQuery := "SELECT * FROM public.fabric_defn WHERE tenant_datasource_id = $1 AND model_key = $2 ORDER BY version DESC LIMIT 1"
	err = s.DB.Get(&defn, fallbackQuery, tenantDatasourceID, modelKey)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("model with key '%s' not found for datasource %s", modelKey, tenantDatasourceID)
		}
		return nil, fmt.Errorf("failed to get model definition: %w", err)
	}

	// Optionally promote this version to current for future calls (best-effort)
	go func(id uuid.UUID) {
		_, promoteErr := s.DB.Exec("UPDATE public.fabric_defn SET is_current = true WHERE id = $1", id)
		if promoteErr != nil {
			logging.GetLogger().Sugar().Warnf("WARN: failed to promote model %s to current: %v", id, promoteErr)
		}
	}(defn.ID)

	return &defn, nil
}

// ListExtensionModels returns the current extension models for a datasource.
// An "extension" is identified by source_config.generator == "extension".
func (s *SemanticModelService) ListExtensionModels(tenantDatasourceID uuid.UUID) ([]*models.FabricDefn, error) {
	var out []*models.FabricDefn
	query := `
		SELECT * FROM public.fabric_defn
		WHERE tenant_datasource_id = $1
		  AND is_current = true
		  AND (source_config->>'generator') = 'extension'
		ORDER BY model_key ASC`
	if err := s.DB.Select(&out, query, tenantDatasourceID); err != nil {
		return nil, fmt.Errorf("failed to list extension models for datasource %s: %w", tenantDatasourceID, err)
	}
	return out, nil
}

// SaveExtensionModelRequest specifies the payload to save or update an extension model.
type SaveExtensionModelRequest struct {
	BaseModelKey string    `json:"base_model_key"`
	ModelKey     string    `json:"model_key"` // optional; defaults to BaseModelKey+"_ext"
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	Status       string    `json:"status"` // draft|published
	CoreVersion  *int      `json:"core_version,omitempty"`
	ModelObject  cube.Cube `json:"model_object"`
	ActorID      uuid.UUID `json:"actor_id"`
}

// SaveExtensionModel upserts an extension FabricDefn row for a base model, versioning via is_current flag.
// Returns the saved definition and any validation issues detected against the current core base.
func (s *SemanticModelService) SaveExtensionModel(tenantDatasourceID uuid.UUID, req SaveExtensionModelRequest) (*models.FabricDefn, []cube.ValidationIssue, error) {
	// The base model key can be in the request or inside the model object itself.
	baseKeyFromObject, _ := req.ModelObject.Extends.(string)
	if req.BaseModelKey == "" && baseKeyFromObject != "" {
		req.BaseModelKey = baseKeyFromObject
	} else if req.BaseModelKey == "" {
		return nil, nil, fmt.Errorf("base_model_key is required")
	}

	extensionCube := req.ModelObject

	// Normalize model keys to include leading '/'
	baseKey := req.BaseModelKey
	if !strings.HasPrefix(baseKey, "/") {
		baseKey = "/" + baseKey
	}
	extKey := req.ModelKey
	if extKey == "" {
		extKey = baseKey + "_ext"
	} else if !strings.HasPrefix(extKey, "/") {
		extKey = "/" + extKey
	}

	// Resolve tenant_id for the datasource
	var tenantID uuid.UUID
	err := s.DB.Get(&tenantID, `
		SELECT t.id FROM public.tenants t
		JOIN public.tenant_instance ti ON t.id = ti.tenant_id
		JOIN public.tenant_product tp ON ti.id = tp.datasource_id
		JOIN public.tenant_product_datasource tpd ON tp.id = tpd.tenant_product_id
		WHERE tpd.id = $1 LIMIT 1
	`, tenantDatasourceID)
	if err != nil {
		return nil, nil, fmt.Errorf("could not determine tenant_id for datasource %s: %w", tenantDatasourceID, err)
	}

	// Load base model defn (current or latest version)
	baseDefn, err := s.GetModelDefinition(tenantDatasourceID, baseKey)
	if err != nil {
		return nil, nil, fmt.Errorf("base model not found for key %s: %w", baseKey, err)
	}
	var baseConfig models.ResolvedModelConfig
	if err := json.Unmarshal(baseDefn.ResolvedConfig, &baseConfig); err != nil {
		return nil, nil, fmt.Errorf("failed to parse base resolved_config: %w", err)
	}
	if len(baseConfig.Cubes) == 0 {
		return nil, nil, fmt.Errorf("base model has no cubes to extend")
	}
	baseCube := baseConfig.Cubes[0]

	// Ensure extension has proper inheritance metadata
	if extensionCube.Extends == nil {
		extensionCube.Extends = baseCube.Name
	}
	if extensionCube.Metadata == nil {
		extensionCube.Metadata = map[string]any{}
	}
	extensionCube.Metadata["inherits_from"] = baseCube.Name
	if req.CoreVersion != nil {
		extensionCube.Metadata["core_version"] = *req.CoreVersion
	} else {
		// Stamp with the base FabricDefn.Version if not provided
		extensionCube.Metadata["core_version"] = baseDefn.Version
	}

	// Validate against base cube
	issues := cube.ValidateExtension(baseCube, extensionCube)

	// Prevent extending itself: extKey should not equal baseKey
	if strings.EqualFold(strings.TrimSpace(extKey), strings.TrimSpace(baseKey)) || strings.EqualFold(strings.TrimPrefix(extKey, "/"), strings.TrimPrefix(baseKey, "/")) {
		return nil, nil, fmt.Errorf("invalid extends: a model cannot extend itself (%s)", baseKey)
	}

	// Prune dimensions/measures that reference simple column identifiers which do not exist in the datasource catalog.
	// This is a conservative cleanup: only prune when the 'sql' value is a naked identifier (no ${}, no function calls).
	colsMap, errCols := s.GatherColumnsMapForDatasource(tenantDatasourceID)
	if errCols == nil {
		prunedIssues := s.PruneMissingColumnsFromExtension(&extensionCube, colsMap, baseCube.Name)
		if len(prunedIssues) > 0 {
			issues = append(issues, prunedIssues...)
		}
		// Add FK-based join membership warnings
		fkIssues := s.ValidateJoinsWithCatalogFKs(tenantDatasourceID, &extensionCube)
		if len(fkIssues) > 0 {
			issues = append(issues, fkIssues...)
		}
	} else {
		// best-effort: log but don't fail
		// fmt.Printf could be noisy in library code; rely on logger at call site if needed.
	}

	// Build resolved config with merged cube (base + extension)
	mergedCube, mergeIssues := cube.MergeCube(baseCube, extensionCube)
	if len(mergeIssues) > 0 {
		issues = append(issues, mergeIssues...)
	}
	resolved := models.ResolvedModelConfig{
		ModelKey: extKey, // The key for the custom/extension model
		Cubes:    []cube.Cube{mergedCube},
	}
	resolvedJSON, err := json.Marshal(resolved)
	if err != nil {
		return nil, issues, fmt.Errorf("failed to marshal extension resolved_config: %w", err)
	}

	// Determine next version and flip current flag on prior ext row if exists
	var currentVersion sql.NullInt64
	err = s.DB.Get(&currentVersion, `SELECT version FROM public.fabric_defn WHERE tenant_datasource_id = $1 AND model_key = $2 AND is_current = true LIMIT 1`, tenantDatasourceID, extKey)
	nextVersion := 1
	if err == nil && currentVersion.Valid {
		// Mark previous current as not current
		_, _ = s.DB.Exec(`UPDATE public.fabric_defn SET is_current = false WHERE tenant_datasource_id = $1 AND model_key = $2 AND is_current = true`, tenantDatasourceID, extKey)
		nextVersion = int(currentVersion.Int64) + 1
	}

	status := req.Status
	if status == "" {
		status = models.StatusDraft
	}

	srcCfg := map[string]any{
		"generator":     "extension",
		"inherits_from": baseKey,
		"core_version":  extensionCube.Metadata["core_version"],
	}

	defn := models.FabricDefn{
		TenantID:           tenantID,
		TenantDatasourceID: tenantDatasourceID,
		ModelKey:           extKey,
		Version:            nextVersion,
		Status:             status,
		Title:              req.Title,
		Description:        req.Description,
		SourceConfig:       models.MustJSONB(srcCfg),
		ResolvedConfig:     models.JSONB(resolvedJSON),
		CreatedBy:          req.ActorID,
		IsCurrent:          true,
	}

	q := `
		INSERT INTO public.fabric_defn (tenant_id, tenant_datasource_id, model_key, version, status, title, description, source_config, resolved_config, created_by, is_current)
		VALUES (:tenant_id, :tenant_datasource_id, :model_key, :version, :status, :title, :description, :source_config, :resolved_config, :created_by, :is_current)
		RETURNING *`
	rows, err := s.DB.NamedQuery(q, &defn)
	if err != nil {
		return nil, issues, fmt.Errorf("failed to save extension model: %w", err)
	}
	if rows.Next() {
		if err := rows.StructScan(&defn); err != nil {
			rows.Close()
			return nil, issues, fmt.Errorf("failed to scan saved extension: %w", err)
		}
	}
	rows.Close()

	return &defn, issues, nil
}

// ExtensionCompatibility summarizes an extension vs. its base.
type ExtensionCompatibility struct {
	ExtensionModelKey   string                 `json:"extension_model_key"`
	BaseModelKey        string                 `json:"base_model_key"`
	BaseCubeName        string                 `json:"base_cube_name"`
	BaseVersion         int                    `json:"base_version"`
	ExtensionCoreTarget *int                   `json:"extension_core_version_target,omitempty"`
	VersionMismatch     bool                   `json:"version_mismatch"`
	Status              string                 `json:"status"`
	Issues              []cube.ValidationIssue `json:"issues"`
	ExtensionChanges    map[string]any         `json:"extension_changes,omitempty"`
}

// GetExtensionsCompatibilityReport validates all current extensions against current core and reports issues.
func (s *SemanticModelService) GetExtensionsCompatibilityReport(tenantDatasourceID uuid.UUID) ([]ExtensionCompatibility, []cube.ValidationIssue, error) {
	// Load all current core (published) models and all current extensions (any status)
	var defns []models.FabricDefn
	q := `SELECT * FROM public.fabric_defn WHERE tenant_datasource_id = $1 AND is_current = true`
	if err := s.DB.Select(&defns, q, tenantDatasourceID); err != nil {
		return nil, nil, fmt.Errorf("failed to load fabric_defn rows: %w", err)
	}

	coreCubes := map[string]cube.Cube{}
	coreByModelKey := map[string]models.FabricDefn{}
	extCubes := map[string]cube.Cube{}
	// Track extension rows for per-extension base lookup
	var extRows []models.FabricDefn

	for _, d := range defns {
		var cfg models.ResolvedModelConfig
		if err := json.Unmarshal(d.ResolvedConfig, &cfg); err != nil {
			// skip invalid rows
			continue
		}
		if len(cfg.Cubes) == 0 {
			continue
		}
		c := cfg.Cubes[0]
		isExtension := false
		// Prefer source_config.generator flag
		var src map[string]any
		_ = json.Unmarshal(d.SourceConfig, &src)
		if g, ok := src["generator"].(string); ok && g == "extension" {
			isExtension = true
		}
		if !isExtension && c.Extends != nil {
			isExtension = true
		}
		if isExtension {
			extCubes[c.Name] = c
			extRows = append(extRows, d)
		} else if d.Status == models.StatusPublished || hasTag(c, "core") || (c.Metadata != nil && c.Metadata["read_only"] == true) || strings.HasSuffix(c.Name, "_core") {
			coreCubes[c.Name] = c
			coreByModelKey[d.ModelKey] = d
		} else {
			// Treat as core fallback
			coreCubes[c.Name] = c
			coreByModelKey[d.ModelKey] = d
		}
	}

	merged, issues := cube.ComposeCatalog(coreCubes, extCubes)
	_ = merged // merged not used further here, but compose yields global issues we return

	// Build per-extension compatibility summary
	var report []ExtensionCompatibility
	for _, extRow := range extRows {
		var cfg models.ResolvedModelConfig
		if err := json.Unmarshal(extRow.ResolvedConfig, &cfg); err != nil || len(cfg.Cubes) == 0 {
			continue
		}
		extCube := cfg.Cubes[0]
		inheritsFrom := ""
		if s, ok := extCube.Metadata["inherits_from"].(string); ok && s != "" {
			inheritsFrom = s
		} else if s, ok := extCube.Extends.(string); ok {
			inheritsFrom = s
		}
		var baseModelKey string
		var baseCubeName string
		var baseVersion int
		var target *int
		var mismatch bool
		var perIssues []cube.ValidationIssue
		var extChanges map[string]any

		for mk, coreDef := range coreByModelKey {
			var coreCfg models.ResolvedModelConfig
			if err := json.Unmarshal(coreDef.ResolvedConfig, &coreCfg); err == nil && len(coreCfg.Cubes) > 0 {
				if coreCfg.Cubes[0].Name == inheritsFrom {
					baseModelKey = mk
					baseCubeName = coreCfg.Cubes[0].Name
					baseVersion = coreDef.Version
					v := baseVersion
					target = &v
					break
				}
			}
		}

		// Extract extension changes from merged catalog if available
		if resolvedCube, ok := merged[extCube.Name]; ok {
			if resolvedCube.Metadata != nil {
				if ch, ok := resolvedCube.Metadata["extension_changes"].(map[string]any); ok {
					extChanges = ch
				}
			}
		}
		report = append(report, ExtensionCompatibility{
			ExtensionModelKey:   extRow.ModelKey,
			BaseModelKey:        baseModelKey,
			BaseCubeName:        baseCubeName,
			BaseVersion:         baseVersion,
			ExtensionCoreTarget: target,
			VersionMismatch:     mismatch,
			Status:              extRow.Status,
			Issues:              perIssues,
			ExtensionChanges:    extChanges,
		})
	}

	return report, issues, nil
}

// SuggestJoinsFromChart extracts potential joins from foreign key relationships in the ERD chart,
// optionally filtering them to a specific scope of tables.
func (s *SemanticModelService) SuggestJoinsFromChart(datasourceID uuid.UUID, scopeTables []string) ([]models.JoinSuggestion, error) {
	// This implementation queries the catalog directly for robustness, instead of parsing chart data.
	// This avoids dependency on the chart generation process having all necessary details.
	query := `
		SELECT 
			ce.properties, 
			source_node.qualified_path as source_table_path,
			target_node.qualified_path as target_table_path
		FROM public.catalog_edge ce
		JOIN public.catalog_node source_node ON ce.source_node_id = source_node.id
		JOIN public.catalog_node target_node ON ce.target_node_id = target_node.id
		WHERE ce.tenant_datasource_id = $1 
		  AND ce.relationship_type = 'foreign_key'
		  AND source_node.node_type_id = '49a50271-ae58-4d3e-ae1c-2f5b89d89192' -- table type
		  AND target_node.node_type_id = '49a50271-ae58-4d3e-ae1c-2f5b89d89192' -- table type
	`

	type fkEdge struct {
		Properties      json.RawMessage `db:"properties"`
		SourceTablePath string          `db:"source_table_path"`
		TargetTablePath string          `db:"target_table_path"`
	}

	var edges []fkEdge
	if err := s.DB.Select(&edges, query, datasourceID); err != nil {
		return nil, fmt.Errorf("failed to query foreign key edges from catalog: %w", err)
	}

	scopeSet := make(map[string]struct{})
	for _, t := range scopeTables {
		// The scopeTables are in "schema.table" format. The paths are "/schema/table".
		// Let's normalize for comparison.
		qualifiedPath := "/" + strings.Replace(strings.ToLower(t), ".", "/", 1)
		scopeSet[qualifiedPath] = struct{}{}
	}
	filterByScope := len(scopeSet) > 0

	var suggestions []models.JoinSuggestion
	for _, edge := range edges {
		sourceTableQualified := strings.Replace(strings.TrimPrefix(edge.SourceTablePath, "/"), "/", ".", 1)
		targetTableQualified := strings.Replace(strings.TrimPrefix(edge.TargetTablePath, "/"), "/", ".", 1)

		// If filtering, check if both tables are in scope
		if filterByScope {
			_, sourceInScope := scopeSet[strings.ToLower(edge.SourceTablePath)]
			_, targetInScope := scopeSet[strings.ToLower(edge.TargetTablePath)]
			if !sourceInScope && !targetInScope {
				continue
			}
		}

		var props struct {
			Columns []struct {
				SourceColumn string `json:"source_column"`
				TargetColumn string `json:"target_column"`
			} `json:"columns"`
		}

		if err := json.Unmarshal(edge.Properties, &props); err != nil {
			logging.GetLogger().Sugar().Warnf("Warning: failed to unmarshal FK edge properties for %s -> %s: %v", edge.SourceTablePath, edge.TargetTablePath, err)
			continue
		}

		// A single FK constraint can have multiple columns (composite key).
		// We create a join suggestion for each column pair.
		for _, colPair := range props.Columns {
			if sourceTableQualified != "" && targetTableQualified != "" && colPair.SourceColumn != "" && colPair.TargetColumn != "" {
				suggestions = append(suggestions, models.JoinSuggestion{
					FromTable: sourceTableQualified,
					ToTable:   targetTableQualified,
					FromCol:   colPair.SourceColumn,
					ToCol:     colPair.TargetColumn,
					JoinType:  "inner",
					Source:    "foreign_key",
				})
			}
		}
	}
	return suggestions, nil
}

// GenerateDefaultSemanticModel creates a default semantic model for a tenant's datasource.
func (s *SemanticModelService) GenerateDefaultSemanticModel(tenantDatasourceID uuid.UUID) ([]*models.FabricDefn, error) {
	fmt.Printf("[BACKEND_SERVICE] Entered GenerateDefaultSemanticModel (Fabric) for tenant_datasource_id: %s\n", tenantDatasourceID)

	var datasource models.TenantProductDatasource
	var tenantProduct struct {
		TenantInstanceID uuid.UUID `db:"datasource_id"`
	}
	var tenantInstance struct {
		TenantID uuid.UUID `db:"tenant_id"`
	}

	// 1. Find the datasource to get its name and traverse up to get the tenant_id
	err := s.DB.Get(&datasource, "SELECT * FROM public.tenant_product_datasource WHERE id = $1", tenantDatasourceID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("datasource with id %s not found", tenantDatasourceID)
		}
		return nil, fmt.Errorf("database error finding datasource: %w", err)
	}

	err = s.DB.Get(&tenantProduct, "SELECT datasource_id FROM public.tenant_product WHERE id = $1", datasource.AlphaProductID)
	if err != nil {
		return nil, fmt.Errorf("could not find tenant instance for tenant product %s: %w", datasource.AlphaProductID, err)
	}

	err = s.DB.Get(&tenantInstance, "SELECT tenant_id FROM public.tenant_instance WHERE id = $1", tenantProduct.TenantInstanceID)
	if err != nil {
		return nil, fmt.Errorf("could not find tenant for tenant instance %s: %w", tenantProduct.TenantInstanceID, err)
	}

	tenantID := tenantInstance.TenantID
	fmt.Printf("[BACKEND_SERVICE] Found tenant_id: %s\n", tenantID)

	// 3. Load tables and columns for this datasource from the catalog.
	// Explicitly list columns to avoid errors if the table schema changes (e.g., adding schema_hash).
	var catalogNodes []models.CatalogNode
	columnsToSelect := "id, core_id, tenant_id, tenant_datasource_id, node_type_id, node_name, qualified_path, parent_id, properties, created_at, updated_at, is_alpha"

	// We filter by the unique tenant_datasource_id.
	err = s.DB.Select(&catalogNodes, fmt.Sprintf("SELECT %s FROM public.catalog_node WHERE tenant_datasource_id = $1 AND core_id IS NULL", columnsToSelect), tenantDatasourceID)
	if err != nil {
		return nil, fmt.Errorf("error loading catalog nodes for datasource %s: %w", tenantDatasourceID, err)
	}
	if len(catalogNodes) == 0 {
		return nil, fmt.Errorf("no schema information found in catalog for datasource %s. Please run a scan first", tenantDatasourceID)
	}

	// 4. Organize nodes into tables and columns.
	tables := make(map[uuid.UUID]models.CatalogNode)
	columnsByTable := make(map[uuid.UUID][]models.CatalogNode)
	for _, node := range catalogNodes {
		if node.NodeTypeID == NODE_TYPE_TABLE {
			tables[node.ID] = node
		} else if node.NodeTypeID == NODE_TYPE_COLUMN && node.ParentID.Valid {
			columnsByTable[node.ParentID.UUID] = append(columnsByTable[node.ParentID.UUID], node)
		}
	}
	fmt.Printf("[BACKEND_SERVICE] Found %d tables in catalog for datasource: %s\n", len(tables), tenantDatasourceID)

	var createdModels []*models.FabricDefn
	modelsCreatedCount := 0

	// Pre-fetch existing models for this datasource to avoid N+1 queries in the loop.
	existingModelKeys := make(map[string]bool)
	var existingDefns []models.FabricDefn
	// We don't check for error here, as an empty result is not an error.
	if err := s.DB.Select(&existingDefns, "SELECT * FROM public.fabric_defn WHERE tenant_id = $1 AND tenant_datasource_id = $2", tenantID, tenantDatasourceID); err != nil {
		return nil, fmt.Errorf("failed to select existing definitions: %w", err)
	}
	for _, defn := range existingDefns {
		existingModelKeys[defn.ModelKey] = true
	}

	// 5. Process each table as a separate model.
	for tableID, tableNode := range tables {
		tableName := tableNode.NodeName
		// Use the qualified path of the table node as the unique key for the model.
		modelKey := tableNode.QualifiedPath
		if modelKey == "" {
			fmt.Printf("[BACKEND_SERVICE] Skipping table '%s' due to empty qualified_path\n", tableName)
			continue
		}

		fmt.Printf("[BACKEND_SERVICE] Processing table '%s' as a new model with key '%s'\n", tableName, modelKey)

		// Check if a model already exists for this specific table.
		if _, exists := existingModelKeys[modelKey]; exists {
			fmt.Printf("[BACKEND_SERVICE] Model for table '%s' (key: %s) already exists. Skipping.\n", tableName, modelKey)
			continue // Skip if model already exists
		}

		// Create the semantic model config for this single table.
		newModel, err := s.createFabricDefnFromTable(tenantID, tenantDatasourceID, tableNode, columnsByTable[tableID])
		if err != nil {
			fmt.Printf("[BACKEND_SERVICE] Failed to create fabric definition for table '%s': %v. Skipping.\n", tableName, err)
			continue
		}

		createdModels = append(createdModels, newModel)
		modelsCreatedCount++
		existingModelKeys[newModel.ModelKey] = true // Add to map to avoid duplicates in same run
	}

	if modelsCreatedCount == 0 {
		return nil, fmt.Errorf("no new models were created; either no tables were found or models already exist for all tables")
	}

	fmt.Printf("[BACKEND_SERVICE] Successfully created %d new fabric definitions.\n", modelsCreatedCount)
	return createdModels, nil
}

// GenerateModels creates semantic models for a given scope (e.g., a schema or a list of tables).
// This function aligns with the blueprint's goal of a centralized, discoverable modeling process.
func (s *SemanticModelService) GenerateModels(tenantDatasourceID uuid.UUID, scope map[string]interface{}) ([]*models.FabricDefn, error) {
	fmt.Printf("[BACKEND_SERVICE] Entered GenerateModels for tenant_datasource_id: %s, NODE_TYPE_TABLE: %s\n", tenantDatasourceID, NODE_TYPE_TABLE)

	// Get TenantID for the definition record. This is a common prerequisite.
	var tenantID uuid.UUID
	err := s.DB.Get(&tenantID, `
		SELECT t.id FROM public.tenants t
		JOIN public.tenant_instance ti ON t.id = ti.tenant_id
		JOIN public.tenant_product tp ON ti.id = tp.datasource_id
		JOIN public.tenant_product_datasource tpd ON tp.id = tpd.tenant_product_id
		WHERE tpd.id = $1 LIMIT 1
	`, tenantDatasourceID)
	if err != nil {
		return nil, fmt.Errorf("could not determine tenant_id for datasource %s: %w", tenantDatasourceID, err)
	}

	// 1. Determine which tables to process based on the scope from the request.
	var tableNodes []models.CatalogNode
	var names []string
	scopeType, _ := scope["type"].(string)
	// Explicitly list columns to avoid errors if the table schema changes (e.g., adding schema_hash).
	columnsToSelect := "id, core_id, tenant_id, tenant_datasource_id, node_type_id, node_name, qualified_path, parent_id, properties, created_at, updated_at, is_alpha"
	switch scopeType {
	case "schema":
		schemaName, ok := scope["name"].(string)
		if !ok || schemaName == "" {
			return nil, fmt.Errorf("schema name is required for scope type 'schema'")
		}
		// Query for all tables within the given schema, selecting specific columns.
		query := fmt.Sprintf(`
			SELECT %s FROM public.catalog_node 
			WHERE tenant_datasource_id = $1 
			AND node_type_id = $2 -- table type
			AND properties->>'schema' = $3`, columnsToSelect)
		err = s.DB.Select(&tableNodes, query, tenantDatasourceID, NODE_TYPE_TABLE, schemaName)

	case "tables":
		namesVal, ok := scope["names"]
		if !ok {
			return nil, fmt.Errorf("scope for 'tables' must include 'names' field")
		}

		// Handle both []string (from internal calls) and []interface{} (from JSON)
		switch v := namesVal.(type) {
		case []string:
			names = v
		case []interface{}:
			for _, item := range v {
				if str, ok := item.(string); ok {
					names = append(names, str)
				}
			}
		default:
			return nil, fmt.Errorf("field 'names' in scope must be an array of strings")
		}

		if len(names) == 0 {
			return nil, fmt.Errorf("table names array is required for scope type 'tables' and cannot be empty")
		}

		lowerNames := make([]string, len(names))
		for i, name := range names {
			lowerNames[i] = strings.ToLower(name)
		}

		prefixedNames := make([]string, len(lowerNames))
		for i, name := range lowerNames {
			// Convert "schema.table" from frontend to "/schema/table" for DB query
			prefixedNames[i] = "/" + strings.Replace(name, ".", "/", 1)
		}

		fmt.Printf("[BACKEND_SERVICE] GenerateModels: lowerNames %v, prefixedNames %v\n", lowerNames, prefixedNames)

		query := fmt.Sprintf(`
			SELECT %s FROM public.catalog_node 
			WHERE tenant_datasource_id = $1 
			AND node_type_id = $2 
			AND LOWER(qualified_path) = ANY($3)`, columnsToSelect)
		err = s.DB.Select(&tableNodes, query, tenantDatasourceID, NODE_TYPE_TABLE, pq.Array(prefixedNames))

	default:
		return nil, fmt.Errorf("invalid scope type: '%s'", scopeType)
	}

	if err != nil {
		return nil, fmt.Errorf("error fetching tables for scope '%s': %w", scopeType, err)
	}

	fmt.Printf("[BACKEND_SERVICE] GenerateModels: Queried for tables %v, found %d tableNodes\n", names, len(tableNodes))
	for _, node := range tableNodes {
		fmt.Printf("[BACKEND_SERVICE] Found table node: %s, qualified_path: %s\n", node.NodeName, node.QualifiedPath)
	}

	if len(tableNodes) == 0 {
		// Debug: check total table nodes for this datasource
		var totalTables int
		err = s.DB.Get(&totalTables, "SELECT COUNT(*) FROM public.catalog_node WHERE tenant_datasource_id = $1 AND node_type_id = $2", tenantDatasourceID, NODE_TYPE_TABLE)
		if err != nil {
			fmt.Printf("[BACKEND_SERVICE] Error counting total tables: %v\n", err)
		} else {
			fmt.Printf("[BACKEND_SERVICE] Total table nodes in catalog for datasource %s: %d\n", tenantDatasourceID, totalTables)
			if totalTables > 0 {
				var sampleNodes []models.CatalogNode
				err = s.DB.Select(&sampleNodes, "SELECT node_name, qualified_path FROM public.catalog_node WHERE tenant_datasource_id = $1 AND node_type_id = $2 LIMIT 5", tenantDatasourceID, NODE_TYPE_TABLE)
				if err != nil {
					fmt.Printf("[BACKEND_SERVICE] Error getting sample nodes: %v\n", err)
				} else {
					for _, node := range sampleNodes {
						fmt.Printf("[BACKEND_SERVICE] Sample table node: node_name='%s', qualified_path='%s'\n", node.NodeName, node.QualifiedPath)
					}
				}
			}
		}
		return nil, fmt.Errorf("no tables found for the given scope")
	}

	// 2. Pre-fetch existing models for this datasource to avoid N+1 queries.
	existingModelKeys := make(map[string]bool)
	var existingDefns []models.FabricDefn
	if err := s.DB.Select(&existingDefns, "SELECT model_key FROM public.fabric_defn WHERE tenant_id = $1 AND tenant_datasource_id = $2", tenantID, tenantDatasourceID); err != nil {
		return nil, fmt.Errorf("failed to select existing model keys: %w", err)
	}
	for _, defn := range existingDefns {
		existingModelKeys[defn.ModelKey] = true
	}

	// 3. Process each table, generate its model, and save it.
	var createdModels []*models.FabricDefn
	for _, tableNode := range tableNodes {
		// Use the single-table generation logic, which is now a reusable part of the service.
		// We pass tenantID to avoid redundant lookups inside the function.
		newModel, err := s.generateAndSaveModelForTable(tenantID, tenantDatasourceID, tableNode, existingModelKeys)
		if err != nil {
			// Log the error but continue processing other tables.
			fmt.Printf("[BACKEND_SERVICE] Skipping model generation for table '%s': %v\n", tableNode.NodeName, err)
			continue
		}
		if newModel != nil {
			createdModels = append(createdModels, newModel)
			// Add the new model key to our map to prevent duplicates within the same batch.
			existingModelKeys[newModel.ModelKey] = true
		}
	}

	fmt.Printf("[BACKEND_SERVICE] Successfully created %d new fabric definitions from scope.\n", len(createdModels))
	return createdModels, nil
}

// generateAndSaveModelForTable is a helper that contains the core logic for generating a model from a single table node.
// It's designed to be called by other service methods like GenerateModels or GenerateSemanticModelForTable.
func (s *SemanticModelService) generateAndSaveModelForTable(tenantID, tenantDatasourceID uuid.UUID, tableNode models.CatalogNode, existingModelKeys map[string]bool) (*models.FabricDefn, error) {
	modelKey := tableNode.QualifiedPath
	if modelKey == "" {
		return nil, fmt.Errorf("table '%s' has an empty qualified_path", tableNode.NodeName)
	}

	if _, exists := existingModelKeys[modelKey]; exists {
		return nil, fmt.Errorf("model for table '%s' (key: %s) already exists", tableNode.NodeName, modelKey)
	}

	// This reuses the logic from the original GenerateSemanticModelForTable.
	// We pass the tableNode and its columns to avoid re-querying.
	columnsToSelect := "id, core_id, tenant_id, tenant_datasource_id, node_type_id, node_name, qualified_path, parent_id, properties, created_at, updated_at, is_alpha"
	var columns []models.CatalogNode
	err := s.DB.Select(&columns, fmt.Sprintf("SELECT %s FROM public.catalog_node WHERE tenant_datasource_id = $1 AND parent_id = $2 AND node_type_id = $3", columnsToSelect), tenantDatasourceID, tableNode.ID, NODE_TYPE_COLUMN)
	if err != nil {
		return nil, fmt.Errorf("error loading columns for table %s: %w", tableNode.NodeName, err)
	}

	return s.createFabricDefnFromTable(tenantID, tenantDatasourceID, tableNode, columns)
}

// GenerateSemanticModelForTable creates a semantic model for a single table.
func (s *SemanticModelService) GenerateSemanticModelForTable(tenantDatasourceID uuid.UUID, tableName string) (*models.FabricDefn, error) {
	fmt.Printf("[BACKEND_SERVICE] Entered GenerateSemanticModelForTable for table: %s, tenant_datasource_id: %s\n", tableName, tenantDatasourceID)

	var datasource models.TenantProductDatasource
	err := s.DB.Get(&datasource, "SELECT * FROM public.tenant_product_datasource WHERE id = $1", tenantDatasourceID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("datasource with id %s not found", tenantDatasourceID)
		}
		return nil, fmt.Errorf("database error finding datasource: %w", err)
	}

	var tenantProduct struct {
		TenantInstanceID uuid.UUID `db:"datasource_id"`
	}
	err = s.DB.Get(&tenantProduct, "SELECT datasource_id FROM public.tenant_product WHERE id = $1", datasource.AlphaProductID)
	if err != nil {
		return nil, fmt.Errorf("could not find tenant instance for tenant product %s: %w", datasource.AlphaProductID, err)
	}

	var tenantInstance struct {
		TenantID uuid.UUID `db:"tenant_id"`
	}
	err = s.DB.Get(&tenantInstance, "SELECT tenant_id FROM public.tenant_instance WHERE id = $1", tenantProduct.TenantInstanceID)
	if err != nil {
		return nil, fmt.Errorf("could not find tenant for tenant instance %s: %w", tenantProduct.TenantInstanceID, err)
	}
	tenantID := tenantInstance.TenantID

	columnsToSelect := "id, core_id, tenant_id, tenant_datasource_id, node_type_id, node_name, qualified_path, parent_id, properties, created_at, updated_at, is_alpha"
	var tableNode models.CatalogNode
	err = s.DB.Get(&tableNode, fmt.Sprintf("SELECT %s FROM public.catalog_node WHERE tenant_datasource_id = $1 AND node_name = $2 AND node_type_id = $3", columnsToSelect), tenantDatasourceID, tableName, NODE_TYPE_TABLE)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("table '%s' not found in catalog for datasource %s", tableName, tenantDatasourceID)
		}
		return nil, fmt.Errorf("error loading table node: %w", err)
	}

	// For a single table generation, the set of existing models is small.
	existingModelKeys := make(map[string]bool)
	var existingDefn models.FabricDefn
	err = s.DB.Get(&existingDefn, "SELECT model_key FROM public.fabric_defn WHERE tenant_id = $1 AND tenant_datasource_id = $2 AND model_key = $3", tenantID, tenantDatasourceID, tableNode.QualifiedPath)
	if err == nil {
		existingModelKeys[existingDefn.ModelKey] = true
	} else if err != sql.ErrNoRows {
		return nil, fmt.Errorf("error checking for existing model: %w", err)
	}

	return s.generateAndSaveModelForTable(tenantID, tenantDatasourceID, tableNode, existingModelKeys)
}

// GatherColumnsMapForDatasource returns a set-like map of simple column names available across the datasource catalog.
func (s *SemanticModelService) GatherColumnsMapForDatasource(tenantDatasourceID uuid.UUID) (map[string]struct{}, error) {
	cols := make(map[string]struct{})
	// Query catalog columns for this datasource
	query := `SELECT node_name FROM public.catalog_node WHERE tenant_datasource_id = $1 AND node_type_id = $2`
	var names []string
	if err := s.DB.Select(&names, query, tenantDatasourceID, NODE_TYPE_COLUMN); err != nil {
		return nil, err
	}
	for _, n := range names {
		cols[strings.ToLower(n)] = struct{}{}
	}
	return cols, nil
}

// PruneMissingColumnsFromExtension inspects the extension cube and removes dimensions/measures/joins whose 'sql' is a simple column
// reference that does not exist in the catalog. It returns ValidationIssues describing what was removed.
func (s *SemanticModelService) PruneMissingColumnsFromExtension(ext *cube.Cube, colsMap map[string]struct{}, baseCubeName string) []cube.ValidationIssue {
	issues := []cube.ValidationIssue{}
	// Helper to check if sql is a simple identifier like "col" or "table.col"
	isSimpleIdentifier := func(sql string) (string, bool) {
		sql = strings.TrimSpace(sql)
		if sql == "" {
			return "", false
		}
		// Ignore expressions containing ${ or ( or whitespace beyond dots
		if strings.Contains(sql, "${") || strings.ContainsAny(sql, `( +-*\/%)`) {
			return "", false
		}
		// If quoted or contains a dot, take the last segment as column
		parts := strings.Split(sql, ".")
		col := parts[len(parts)-1]
		col = strings.Trim(col, " \"`[]")
		if col == "" {
			return "", false
		}
		return strings.ToLower(col), true
	}

	// Dimensions
	for name, dim := range ext.Dimensions {
		if sqlRaw, ok := dim["sql"].(string); ok {
			if col, ok2 := isSimpleIdentifier(sqlRaw); ok2 {
				if _, exists := colsMap[col]; !exists {
					// remove this dimension
					delete(ext.Dimensions, name)
					issues = append(issues, cube.ValidationIssue{Level: "warning", Code: "MISSING_COLUMN_PRUNED", Message: fmt.Sprintf("dimension '%s' removed: column '%s' not found in catalog for base '%s'", name, col, baseCubeName)})
				}
			}
		}
	}
	// Measures
	for name, mea := range ext.Measures {
		if sqlRaw, ok := mea["sql"].(string); ok {
			if col, ok2 := isSimpleIdentifier(sqlRaw); ok2 {
				// strip any aggregate wrapper like SUM(col) — our heuristic only prunes plain identifiers
				if _, exists := colsMap[col]; !exists {
					delete(ext.Measures, name)
					issues = append(issues, cube.ValidationIssue{Level: "warning", Code: "MISSING_COLUMN_PRUNED", Message: fmt.Sprintf("measure '%s' removed: column '%s' not found in catalog for base '%s'", name, col, baseCubeName)})
				}
			}
		}
	}
	// Joins: check basic occurrence of column references
	for name, j := range ext.Joins {
		if sqlRaw, ok := j["sql"].(string); ok {
			// attempt to find a simple identifier after ${CUBE}. or the first token
			// naive parse
			token := strings.Split(sqlRaw, "=")
			candidate := strings.TrimSpace(token[0])
			// take last segment
			parts := strings.Split(candidate, ".")
			col := strings.Trim(parts[len(parts)-1], " \"`[]")
			col = strings.ToLower(col)
			if col != "" {
				if _, exists := colsMap[col]; !exists {
					delete(ext.Joins, name)
					issues = append(issues, cube.ValidationIssue{Level: "warning", Code: "MISSING_COLUMN_PRUNED", Message: fmt.Sprintf("join '%s' removed: column '%s' not found in catalog for base '%s'", name, col, baseCubeName)})
				}
			}
		}
	}
	return issues
}

// ValidateJoinsWithCatalogFKs inspects extension joins and cross-checks them against catalog foreign key edges
// to ensure that referenced columns exist on the referenced tables. This is a conservative validation that
// adds warnings when a join condition references a column that is not part of any FK relationship between
// the involved tables according to the catalog. It does not mutate the extension cube.
func (s *SemanticModelService) ValidateJoinsWithCatalogFKs(tenantDatasourceID uuid.UUID, ext *cube.Cube) []cube.ValidationIssue {
	issues := []cube.ValidationIssue{}
	if len(ext.Joins) == 0 {
		return issues
	}

	// Load FK edges once for the datasource and build a quick lookup map by qualified table name.
	type fkEdge struct {
		Properties      json.RawMessage `db:"properties"`
		SourceTablePath string          `db:"source_table_path"`
		TargetTablePath string          `db:"target_table_path"`
	}
	query := `
		SELECT 
			ce.properties, 
			source_node.qualified_path as source_table_path,
			target_node.qualified_path as target_table_path
		FROM public.catalog_edge ce
		JOIN public.catalog_node source_node ON ce.source_node_id = source_node.id
		JOIN public.catalog_node target_node ON ce.target_node_id = target_node.id
		WHERE ce.tenant_datasource_id = $1 
		  AND ce.relationship_type = 'foreign_key'
		  AND source_node.node_type_id = '49a50271-ae58-4d3e-ae1c-2f5b89d89192' 
		  AND target_node.node_type_id = '49a50271-ae58-4d3e-ae1c-2f5b89d89192'`

	var edges []fkEdge
	if err := s.DB.Select(&edges, query, tenantDatasourceID); err != nil {
		// best-effort: proceed with no edges; downstream will warn that joins aren't backed by FK
		edges = nil
	}

	// Build a map: pair of tables (unordered) -> set of allowed column pairs
	type colPair struct{ from, to string }
	fkMap := make(map[[2]string]map[colPair]struct{})
	addPair := func(a, b, colA, colB string) {
		key := [2]string{strings.ToLower(a), strings.ToLower(b)}
		if a > b {
			key = [2]string{strings.ToLower(b), strings.ToLower(a)}
			colA, colB = colB, colA
		}
		if fkMap[key] == nil {
			fkMap[key] = make(map[colPair]struct{})
		}
		fkMap[key][colPair{from: strings.ToLower(colA), to: strings.ToLower(colB)}] = struct{}{}
	}

	for _, e := range edges {
		var props struct {
			Columns []struct {
				SourceColumn string `json:"source_column"`
				TargetColumn string `json:"target_column"`
			} `json:"columns"`
		}
		if err := json.Unmarshal(e.Properties, &props); err != nil {
			continue
		}
		a := strings.Replace(strings.TrimPrefix(e.SourceTablePath, "/"), "/", ".", 1)
		b := strings.Replace(strings.TrimPrefix(e.TargetTablePath, "/"), "/", ".", 1)
		for _, c := range props.Columns {
			if c.SourceColumn != "" && c.TargetColumn != "" {
				addPair(a, b, c.SourceColumn, c.TargetColumn)
			}
		}
	}

	// Helper to extract simple table/column tokens from a join sql like "${CUBE}.id = ${other}.other_id"
	parseJoin := func(sql string) (leftTable, leftCol, rightTable, rightCol string, ok bool) {
		parts := strings.Split(sql, "=")
		if len(parts) != 2 {
			return
		}
		norm := func(s string) (tbl, col string) {
			s = strings.TrimSpace(s)
			s = strings.Trim(s, "`\"[]")
			// Replace ${X}.Y -> X.Y
			s = strings.ReplaceAll(s, "${", "")
			s = strings.ReplaceAll(s, "}", "")
			// Collapse multiple dots
			toks := strings.Split(s, ".")
			if len(toks) >= 2 {
				tbl = strings.Join(toks[:len(toks)-1], ".")
				col = toks[len(toks)-1]
			} else if len(toks) == 1 {
				col = toks[0]
			}
			tbl = strings.TrimSpace(tbl)
			col = strings.TrimSpace(col)
			return
		}
		lt, lc := norm(parts[0])
		rt, rc := norm(parts[1])
		if lc == "" || rc == "" {
			return
		}
		return lt, lc, rt, rc, true
	}

	for joinName, j := range ext.Joins {
		raw, _ := j["sql"].(string)
		if raw == "" {
			continue
		}
		lt, lc, rt, rc, ok := parseJoin(raw)
		if !ok {
			continue // non-simple expression; skip
		}
		// Without full cube-to-table mapping, we can only validate when both sides include table names
		if lt == "" || rt == "" {
			continue
		}
		key := [2]string{strings.ToLower(lt), strings.ToLower(rt)}
		if lt > rt {
			key = [2]string{strings.ToLower(rt), strings.ToLower(lt)}
			lc, rc = rc, lc
		}
		allowed := fkMap[key]
		if allowed == nil {
			issues = append(issues, cube.ValidationIssue{Level: "warning", Code: "JOIN_NOT_IN_CATALOG_FK", Message: fmt.Sprintf("join '%s' not backed by any catalog FK between '%s' and '%s'", joinName, lt, rt)})
			continue
		}
		// Look for matching column pair in either direction
		_, okPair := allowed[colPair{from: strings.ToLower(lc), to: strings.ToLower(rc)}]
		if !okPair {
			issues = append(issues, cube.ValidationIssue{Level: "warning", Code: "JOIN_COLUMNS_NOT_IN_FK", Message: fmt.Sprintf("join '%s' uses columns '%s' and '%s' which are not part of an FK between '%s' and '%s'", joinName, lc, rc, lt, rt)})
		}
	}
	return issues
}

// createFabricDefnFromTable contains the logic to build and insert a FabricDefn.
func (s *SemanticModelService) createFabricDefnFromTable(tenantID, tenantDatasourceID uuid.UUID, tableNode models.CatalogNode, columns []models.CatalogNode) (*models.FabricDefn, error) {
	tableName := tableNode.NodeName
	modelKey := tableNode.QualifiedPath
	if modelKey == "" {
		return nil, fmt.Errorf("skipping table '%s' due to empty qualified_path", tableName)
	}

	// Create a single cube for this table
	c := cube.Cube{
		Name:       tableName,
		SQL:        modelKey,
		SQLTable:   strings.Replace(strings.TrimPrefix(modelKey, "/"), "/", ".", 1),
		Dimensions: make(map[string]map[string]any),
		Measures:   make(map[string]map[string]any),
		Joins:      make(map[string]map[string]any),
	}

	// Provide optional defaults to match Cube fields
	title := cleanColumnName(tableName)
	c.Title = title
	c.Description = fmt.Sprintf("Auto-generated semantic model for table %s.", tableNode.NodeName)
	pub := true
	c.Public = &pub

	// Add default COUNT measure
	c.Measures["count"] = map[string]any{
		"type": "count",
		"sql":  "*",
	}

	for _, columnNode := range columns {
		var props models.CatalogNodeProperties
		if err := json.Unmarshal(columnNode.Properties, &props); err != nil {
			fmt.Printf("[BACKEND_SERVICE] Error unmarshaling column properties for %s: %v\n", columnNode.NodeName, err)
			continue
		}
		dataType := props.DataType
		columnName := columnNode.NodeName

		if columnName == "" || dataType == "" {
			continue
		}
		if isSystemColumn(columnName) {
			continue
		}

		// Add as dimension
		dimensionIdentifier := cleanColumnNameForIdentifier(columnName)
		c.Dimensions[dimensionIdentifier] = map[string]any{
			"sql":   columnName,
			"type":  inferDimensionType(dataType),
			"title": cleanColumnName(columnName), // Add a human-friendly title
			"meta":  models.ExplainMeta("auto_dimension", tableName, columnName),
		}

		// Add as measure if numeric
		if isNumeric(dataType) {
			measureIdentifier := fmt.Sprintf("sum_%s", cleanColumnNameForIdentifier(columnName))
			c.Measures[measureIdentifier] = map[string]any{
				"sql":   fmt.Sprintf("SUM(%s)", columnName),
				"type":  "sum",
				"title": fmt.Sprintf("Sum of %s", cleanColumnName(columnName)), // Add a human-friendly title
				"meta":  models.ExplainMeta("auto_sum_numeric", tableName, columnName),
			}
		}
	}

	// Add joins based on foreign keys discovered in the catalog.
	qualifiedTableName := strings.Replace(strings.TrimPrefix(modelKey, "/"), "/", ".", 1)
	joinSuggestions, err := s.SuggestJoinsFromChart(tenantDatasourceID, []string{qualifiedTableName})
	if err != nil {
		// Log the error but don't fail the model generation, as joins are an enhancement.
		fmt.Printf("[BACKEND_SERVICE] Warning: could not suggest joins for table '%s': %v\n", tableName, err)
	} else {
		for _, j := range joinSuggestions {
			var targetCubeName, thisTableCol, otherTableCol, relationship string

			// Determine which part of the join belongs to the current table and set relationship.
			if j.FromTable == qualifiedTableName {
				targetCubeName = j.ToTable
				thisTableCol = j.FromCol
				otherTableCol = j.ToCol
				relationship = "many_to_one" // The "from" table has the FK, pointing to the "to" table's PK.
			} else if j.ToTable == qualifiedTableName {
				targetCubeName = j.FromTable
				thisTableCol = j.ToCol
				otherTableCol = j.FromCol
				relationship = "one_to_many" // The current table is the "to" table, being pointed at.
			} else {
				continue // This join doesn't involve the current table.
			}

			// The join key should be the simple name of the cube we are joining to.
			joinKey := strings.Split(targetCubeName, ".")[1]

			c.Joins[joinKey] = map[string]any{
				"sql":          fmt.Sprintf("${CUBE}.%s = ${%s}.%s", thisTableCol, joinKey, otherTableCol),
				"relationship": relationship,
			}
		}
	}

	resolvedConfig := models.ResolvedModelConfig{
		ModelKey: modelKey,
		Cubes:    []cube.Cube{c},
	}
	resolvedJSON, err := json.Marshal(resolvedConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config for table '%s': %w", tableName, err)
	}

	newDefn := models.FabricDefn{
		TenantID:           tenantID,
		TenantDatasourceID: tenantDatasourceID,
		ModelKey:           modelKey,
		Version:            1,
		Status:             models.StatusDraft,
		Title:              tableName,
		Description:        fmt.Sprintf("Auto-generated semantic model for table %s.", tableNode.NodeName),
		SourceConfig:       models.MustJSONB(map[string]interface{}{"generator": "single-table"}),
		ResolvedConfig:     models.JSONB(resolvedJSON),
		CreatedBy:          uuid.Nil,
	}

	query := `
		INSERT INTO public.fabric_defn (tenant_id, tenant_datasource_id, model_key, version, status, title, description, source_config, resolved_config, created_by)
		VALUES (:tenant_id, :tenant_datasource_id, :model_key, :version, :status, :title, :description, :source_config, :resolved_config, :created_by)
		RETURNING *
	`
	rows, err := s.DB.NamedQuery(query, &newDefn)
	if err != nil {
		return nil, fmt.Errorf("failed to create fabric definition for table '%s': %w", tableName, err)
	}
	if rows.Next() {
		if err := rows.StructScan(&newDefn); err != nil {
			rows.Close()
			return nil, fmt.Errorf("failed to scan new fabric definition: %w", err)
		}
	}
	rows.Close()

	if err := s.syncCatalogForGeneratedModel(context.Background(), tenantID, tenantDatasourceID, tableNode, columns, &c, &newDefn); err != nil {
		return nil, fmt.Errorf("failed to synchronize catalog for table '%s': %w", tableName, err)
	}

	// After creating the core model, ensure a custom (_custom) extension model exists
	customKey := modelKey + "_custom"

	// Check DB for existing custom model; best-effort creation if missing
	var existsFlag bool
	err = s.DB.Get(&existsFlag, "SELECT EXISTS(SELECT 1 FROM public.fabric_defn WHERE tenant_datasource_id=$1 AND model_key=$2)", tenantDatasourceID, customKey)
	if err != nil {
		// Log and attempt creation (best-effort)
		fmt.Printf("[BACKEND_SERVICE] Warning: error checking existence of custom model %s: %v\n", customKey, err)
		existsFlag = false
	}
	if !existsFlag {
		go func() {
			// Build a lightweight extension cube that inherits from the core cube.
			extCube := c
			// Mark this cube as an extension of the core cube
			extCube.Extends = c.Name
			if extCube.Metadata == nil {
				extCube.Metadata = map[string]any{}
			}
			extCube.Metadata["inherits_from"] = c.Name
			extCube.Metadata["core_version"] = newDefn.Version

			resolvedExt := models.ResolvedModelConfig{
				ModelKey: customKey,
				Cubes:    []cube.Cube{extCube},
			}
			resolvedExtJSON, _ := json.Marshal(resolvedExt)

			extDefn := models.FabricDefn{
				TenantID:           tenantID,
				TenantDatasourceID: tenantDatasourceID,
				ModelKey:           customKey,
				Version:            1,
				Status:             models.StatusDraft,
				Title:              tableName + " (custom)",
				Description:        fmt.Sprintf("Auto-created custom model for table %s.", tableNode.NodeName),
				SourceConfig:       models.MustJSONB(map[string]any{"generator": "extension", "inherits_from": modelKey, "core_version": newDefn.Version}),
				ResolvedConfig:     models.JSONB(resolvedExtJSON),
				CreatedBy:          uuid.Nil,
				IsCurrent:          true,
			}

			q2 := `
				INSERT INTO public.fabric_defn (tenant_id, tenant_datasource_id, model_key, version, status, title, description, source_config, resolved_config, created_by, is_current)
				VALUES (:tenant_id, :tenant_datasource_id, :model_key, :version, :status, :title, :description, :source_config, :resolved_config, :created_by, :is_current)
				RETURNING *
			`
			rows2, err2 := s.DB.NamedQuery(q2, &extDefn)
			if err2 != nil {
				fmt.Printf("[BACKEND_SERVICE] Warning: failed to create custom model %s: %v\n", customKey, err2)
				return
			}
			if rows2.Next() {
				if errScan := rows2.StructScan(&extDefn); errScan != nil {
					fmt.Printf("[BACKEND_SERVICE] Warning: failed to scan created custom model %s: %v\n", customKey, errScan)
				}
			}
			rows2.Close()
			fmt.Printf("[BACKEND_SERVICE] Auto-created custom model: %s\n", customKey)
		}()
	}

	return &newDefn, nil
}

func inferDimensionType(dataType string) string {
	if isDateTime(dataType) {
		return "time"
	}
	// For cube.js, both text and numbers can be 'string' type dimensions.
	// Specific numeric handling is for measures.
	return "string"
}

func cleanColumnNameForIdentifier(columnName string) string {
	return strings.ReplaceAll(strings.ToLower(columnName), " ", "_")
}

// Helper functions remain the same
func cleanColumnName(columnName string) string {
	parts := strings.Split(columnName, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(string(part[0])) + strings.ToLower(part[1:])
		}
	}
	return strings.Join(parts, " ")
}

func isSystemColumn(columnName string) bool {
	systemColumns := []string{"id", "created_at", "updated_at", "deleted_at", "version", "uuid"}
	lowerName := strings.ToLower(columnName)

	for _, sysCol := range systemColumns {
		if lowerName == sysCol || strings.HasSuffix(lowerName, "_"+sysCol) {
			return true
		}
	}
	return false
}

func isNumeric(dataType string) bool {
	dt := strings.ToUpper(dataType)
	numericTypes := []string{"INT", "INTEGER", "BIGINT", "SMALLINT", "TINYINT",
		"FLOAT", "DOUBLE", "DECIMAL", "NUMERIC", "REAL", "MONEY"}

	for _, numType := range numericTypes {
		if strings.Contains(dt, numType) {
			return true
		}
	}
	return false
}

func isDateTime(dataType string) bool {
	dt := strings.ToUpper(dataType)
	dateTimeTypes := []string{"DATE", "TIME", "TIMESTAMP", "DATETIME", "TIMESTAMPTZ"}

	for _, dateTimeType := range dateTimeTypes {
		if strings.Contains(dt, dateTimeType) {
			return true
		}
	}
	return false
}

// AddCalculation associates a calculation with a semantic model.
func (s *SemanticModelService) AddCalculation(ctx context.Context, semanticModelID, calculationID uuid.UUID, mapping map[string]string, outputName string, isPublic bool, userID uuid.UUID) (*models.SemanticModelCalculation, error) {
	mappingJSON, err := json.Marshal(mapping)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal argument mapping: %w", err)
	}

	smc := models.SemanticModelCalculation{
		SemanticModelID: semanticModelID,
		CalculationID:   calculationID,
		ArgumentMapping: mappingJSON,
		OutputName:      outputName,
		IsPublic:        isPublic,
		CreatedBy:       &userID,
		UpdatedBy:       &userID,
	}

	query := `
		INSERT INTO semantic_model_calculations (
			semantic_model_id, calculation_id, argument_mapping, output_name, is_public, created_by, updated_by
		) VALUES (
			:semantic_model_id, :calculation_id, :argument_mapping, :output_name, :is_public, :created_by, :updated_by
		) RETURNING *
	`

	rows, err := s.DB.NamedQueryContext(ctx, query, smc)
	if err != nil {
		return nil, fmt.Errorf("failed to add calculation to model: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		if err := rows.StructScan(&smc); err != nil {
			return nil, fmt.Errorf("failed to scan semantic model calculation: %w", err)
		}
	}

	return &smc, nil
}

// RemoveCalculation removes a calculation association from a semantic model.
func (s *SemanticModelService) RemoveCalculation(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM semantic_model_calculations WHERE id = $1`
	_, err := s.DB.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to remove calculation from model: %w", err)
	}
	return nil
}

// GetCalculations retrieves all calculations associated with a semantic model.
func (s *SemanticModelService) GetCalculations(ctx context.Context, semanticModelID uuid.UUID) ([]models.SemanticModelCalculation, error) {
	var calcs []models.SemanticModelCalculation
	query := `
		SELECT smc.*, c.name as calculation_name
		FROM semantic_model_calculations smc
		JOIN calculations c ON smc.calculation_id = c.id
		WHERE smc.semantic_model_id = $1
		ORDER BY smc.created_at DESC
	`
	err := s.DB.SelectContext(ctx, &calcs, query, semanticModelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get calculations for model: %w", err)
	}
	return calcs, nil
}
