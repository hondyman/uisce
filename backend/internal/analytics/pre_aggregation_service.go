package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/jmoiron/sqlx"
)

// PreAggregationService manages pre-aggregation definitions and materializations.
type PreAggregationService struct {
	db               *sqlx.DB
	boResolver       *BOContextResolver
	semanticGraphSvc *SemanticGraphService
}

func NewPreAggregationService(db *sqlx.DB, boResolver *BOContextResolver, semanticGraphSvc *SemanticGraphService) *PreAggregationService {
	return &PreAggregationService{
		db:               db,
		boResolver:       boResolver,
		semanticGraphSvc: semanticGraphSvc,
	}
}

// UpsertPreAggregation creates or updates a pre-aggregation node in the catalog.
func (s *PreAggregationService) UpsertPreAggregation(ctx context.Context, req models.UpsertPreAggRequest) (*models.PreAggDescriptor, error) {
	// Build properties JSON
	props := models.PreAggProperties{
		BOName:                 req.BOName,
		TenantID:               req.TenantID,
		Dialect:                "starrocks",
		RefreshStrategy:        req.RefreshStrategy,
		RefreshIntervalMinutes: req.RefreshIntervalMinutes,
		GovernanceStatus:       "draft",
		TargetDatabase:         fmt.Sprintf("tenant_%s", req.TenantID),
	}
	propsJSON, _ := json.Marshal(props)

	// Build config JSON
	cfg := models.PreAggConfig{
		Terms:           req.Terms,
		Calculations:    req.Calculations,
		Filters:         req.Filters,
		GroupBy:         req.GroupBy,
		Materialization: req.Materialization,
	}
	cfgJSON, _ := json.Marshal(cfg)

	// Get pre_aggregation node type ID
	var nodeTypeID string
	err := s.db.GetContext(ctx, &nodeTypeID, `
		SELECT id FROM catalog_node_type WHERE catalog_type_name = 'pre_aggregation' LIMIT 1
	`)
	if err != nil {
		return nil, fmt.Errorf("pre_aggregation node type not found: %w", err)
	}

	// Upsert node
	nodeID := uuid.New()
	qualifiedPath := fmt.Sprintf("pre_aggregation/%s/%s", req.TenantID, req.Name)

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO catalog_node (id, node_name, description, node_type_id, tenant_id, qualified_path, properties, config, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
		ON CONFLICT (tenant_id, node_type_id, node_name) DO UPDATE SET
			description = EXCLUDED.description,
			properties = EXCLUDED.properties,
			config = EXCLUDED.config,
			updated_at = NOW()
		RETURNING id
	`, nodeID, req.Name, req.Description, nodeTypeID, req.TenantID, qualifiedPath, propsJSON, cfgJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert pre-aggregation node: %w", err)
	}

	// TODO: Create edges (PREAGG_FOR_BO, PREAGG_USES_TERM, PREAGG_USES_CALC)

	return &models.PreAggDescriptor{
		ID:                     nodeID,
		TenantID:               req.TenantID,
		BOName:                 req.BOName,
		Name:                   req.Name,
		Description:            req.Description,
		TargetDatabase:         props.TargetDatabase,
		TargetName:             req.Materialization.TargetName,
		Dialect:                props.Dialect,
		RefreshStrategy:        req.RefreshStrategy,
		RefreshIntervalMinutes: req.RefreshIntervalMinutes,
		GovernanceStatus:       props.GovernanceStatus,
	}, nil
}

// GenerateDDL builds the StarRocks DDL for a pre-aggregation node.
func (s *PreAggregationService) GenerateDDL(ctx context.Context, preAggID uuid.UUID, dialect string) (string, error) {
	// 1. Load pre_aggregation node
	var node struct {
		NodeName   string          `db:"node_name"`
		Properties json.RawMessage `db:"properties"`
		Config     json.RawMessage `db:"config"`
	}
	err := s.db.GetContext(ctx, &node, `
		SELECT node_name, properties, config FROM catalog_node WHERE id = $1
	`, preAggID)
	if err != nil {
		return "", fmt.Errorf("pre-aggregation node not found: %w", err)
	}

	props, err := models.ParsePreAggProperties(node.Properties)
	if err != nil {
		return "", err
	}

	cfg, err := models.ParsePreAggConfig(node.Config)
	if err != nil {
		return "", err
	}

	// 2. Get BO ID by name
	var boID string
	err = s.db.GetContext(ctx, &boID, `
		SELECT n.id FROM catalog_node n
		JOIN catalog_node_type nt ON n.node_type_id = nt.id
		WHERE nt.catalog_type_name = 'business_object' AND n.node_name = $1 AND n.tenant_id = $2
		LIMIT 1
	`, props.BOName, props.TenantID)
	if err != nil {
		return "", fmt.Errorf("BO '%s' not found: %w", props.BOName, err)
	}

	// 3. Generate base BO SQL using BOContextResolver
	boIDParsed, err := uuid.Parse(boID)
	if err != nil {
		return "", fmt.Errorf("invalid BO ID: %w", err)
	}
	tenantIDParsed, _ := uuid.Parse(props.TenantID)
	boCtx, err := s.boResolver.GetBOContext(props.BOName, tenantIDParsed, uuid.Nil, dialect)
	if err != nil {
		return "", fmt.Errorf("failed to get BO context: %w", err)
	}
	_ = boIDParsed // For future use

	boSQL, err := s.boResolver.GenerateBOSQL(*boCtx, cfg.Terms, cfg.Calculations)
	if err != nil {
		return "", fmt.Errorf("failed to generate BO SQL: %w", err)
	}

	// 4. Build SELECT list from terms + calculations
	selectCols := make([]string, 0, len(cfg.GroupBy)+len(cfg.Calculations))
	for _, col := range cfg.GroupBy {
		selectCols = append(selectCols, col)
	}
	for _, calc := range cfg.Calculations {
		// Resolve calculation expression
		// For now, assume calc name = column name in BO SQL
		// TODO: Resolve actual calc DSL -> SQL
		selectCols = append(selectCols, calc)
	}

	// 5. Build WHERE clause from filters
	var whereClause string
	if len(cfg.Filters) > 0 {
		filters := make([]string, 0, len(cfg.Filters))
		for _, f := range cfg.Filters {
			filters = append(filters, f.Expression)
		}
		whereClause = "WHERE " + strings.Join(filters, " AND ")
	}

	// 6. Build GROUP BY clause
	groupByClause := ""
	if len(cfg.GroupBy) > 0 {
		groupByClause = "GROUP BY " + strings.Join(cfg.GroupBy, ", ")
	}

	// 7. Construct final DDL
	var ddl string
	switch cfg.Materialization.Type {
	case "materialized_view":
		ddl = fmt.Sprintf(`CREATE MATERIALIZED VIEW %s.%s
BUILD IMMEDIATE
REFRESH ASYNC
AS
SELECT
    %s
FROM (
    %s
) t
%s
%s;`,
			props.TargetDatabase,
			cfg.Materialization.TargetName,
			strings.Join(selectCols, ",\n    "),
			boSQL,
			whereClause,
			groupByClause,
		)
	case "table":
		ddl = fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.%s AS
SELECT
    %s
FROM (
    %s
) t
%s
%s;`,
			props.TargetDatabase,
			cfg.Materialization.TargetName,
			strings.Join(selectCols, ",\n    "),
			boSQL,
			whereClause,
			groupByClause,
		)
	default:
		return "", fmt.Errorf("unsupported materialization type: %s", cfg.Materialization.Type)
	}

	return ddl, nil
}

// ApplyMaterialization executes the DDL against StarRocks.
func (s *PreAggregationService) ApplyMaterialization(ctx context.Context, preAggID uuid.UUID) error {
	ddl, err := s.GenerateDDL(ctx, preAggID, "starrocks")
	if err != nil {
		return err
	}

	// TODO: Execute DDL against StarRocks using a separate connection pool
	// For now, log/stub
	_ = ddl
	// starrocksDB.ExecContext(ctx, ddl)

	return nil
}

// Refresh triggers a refresh of the materialized view.
func (s *PreAggregationService) Refresh(ctx context.Context, preAggID uuid.UUID) error {
	// Load node to get target info
	var node struct {
		Properties json.RawMessage `db:"properties"`
		Config     json.RawMessage `db:"config"`
	}
	err := s.db.GetContext(ctx, &node, `
		SELECT properties, config FROM catalog_node WHERE id = $1
	`, preAggID)
	if err != nil {
		return err
	}

	props, _ := models.ParsePreAggProperties(node.Properties)
	cfg, _ := models.ParsePreAggConfig(node.Config)

	refreshSQL := fmt.Sprintf("REFRESH MATERIALIZED VIEW %s.%s;", props.TargetDatabase, cfg.Materialization.TargetName)

	// TODO: Execute refresh against StarRocks
	_ = refreshSQL

	return nil
}

// ListByBO returns all pre-aggregations for a given BO and tenant.
func (s *PreAggregationService) ListByBO(ctx context.Context, tenantID, boName string) ([]models.PreAggDescriptor, error) {
	var nodes []struct {
		ID          uuid.UUID       `db:"id"`
		NodeName    string          `db:"node_name"`
		Description string          `db:"description"`
		Properties  json.RawMessage `db:"properties"`
		Config      json.RawMessage `db:"config"`
		CreatedAt   string          `db:"created_at"`
		UpdatedAt   string          `db:"updated_at"`
	}

	err := s.db.SelectContext(ctx, &nodes, `
		SELECT n.id, n.node_name, COALESCE(n.description, '') as description, n.properties, n.config, n.created_at, n.updated_at
		FROM catalog_node n
		JOIN catalog_node_type nt ON n.node_type_id = nt.id
		WHERE nt.catalog_type_name = 'pre_aggregation'
		  AND n.tenant_id = $1
		  AND n.properties->>'bo_name' = $2
	`, tenantID, boName)
	if err != nil {
		return nil, err
	}

	result := make([]models.PreAggDescriptor, 0, len(nodes))
	for _, n := range nodes {
		props, _ := models.ParsePreAggProperties(n.Properties)
		cfg, _ := models.ParsePreAggConfig(n.Config)
		result = append(result, models.PreAggDescriptor{
			ID:                     n.ID,
			TenantID:               props.TenantID,
			BOName:                 props.BOName,
			Name:                   n.NodeName,
			Description:            n.Description,
			TargetDatabase:         props.TargetDatabase,
			TargetName:             cfg.Materialization.TargetName,
			Dialect:                props.Dialect,
			RefreshStrategy:        props.RefreshStrategy,
			RefreshIntervalMinutes: props.RefreshIntervalMinutes,
			GovernanceStatus:       props.GovernanceStatus,
			// Lifecycle fields
			LifecycleStatus:      props.LifecycleStatus,
			LastMaterializedAt:   props.LastMaterializedAt,
			LastRefreshedAt:      props.LastRefreshedAt,
			LastRefreshStatus:    props.LastRefreshStatus,
			LastRefreshError:     props.LastRefreshError,
			NextScheduledRefresh: props.NextScheduledRefresh,
			RowCount:             props.RowCount,
			SizeBytes:            props.SizeBytes,
		})
	}

	return result, nil
}

// GenerateCubeSchema generates Cube.js schema for all pre-aggregations of a tenant.
func (s *PreAggregationService) GenerateCubeSchema(ctx context.Context, tenantID string) (*models.CubeSchema, error) {
	var nodes []struct {
		NodeName   string          `db:"node_name"`
		Properties json.RawMessage `db:"properties"`
		Config     json.RawMessage `db:"config"`
	}

	err := s.db.SelectContext(ctx, &nodes, `
		SELECT n.node_name, n.properties, n.config
		FROM catalog_node n
		JOIN catalog_node_type nt ON n.node_type_id = nt.id
		WHERE nt.catalog_type_name = 'pre_aggregation'
		  AND n.tenant_id = $1
		  AND n.properties->>'governance_status' IN ('published', 'draft')
	`, tenantID)
	if err != nil {
		return nil, err
	}

	schema := &models.CubeSchema{
		Cubes: make([]models.CubeDefinition, 0, len(nodes)),
	}

	for _, n := range nodes {
		props, _ := models.ParsePreAggProperties(n.Properties)
		cfg, _ := models.ParsePreAggConfig(n.Config)

		// Generate cube name (PascalCase from snake_case)
		cubeName := preAggToPascalCase(n.NodeName)

		// Build SQL
		sql := fmt.Sprintf("SELECT * FROM %s.%s", props.TargetDatabase, cfg.Materialization.TargetName)

		// Build measures from calculations
		measures := make(map[string]models.CubeMeasure)
		for _, calc := range cfg.Calculations {
			measures[preAggToCamelCase(calc)] = models.CubeMeasure{
				SQL:  calc,
				Type: "number",
			}
		}

		// Build dimensions from terms/group_by
		dimensions := make(map[string]models.CubeDimension)
		for _, term := range cfg.GroupBy {
			dimType := "string"
			// Heuristic: if name contains "date" or "time", treat as time dimension
			if strings.Contains(strings.ToLower(term), "date") || strings.Contains(strings.ToLower(term), "time") {
				dimType = "time"
			}
			dimensions[preAggToCamelCase(term)] = models.CubeDimension{
				SQL:  term,
				Type: dimType,
			}
		}

		schema.Cubes = append(schema.Cubes, models.CubeDefinition{
			Name:       cubeName,
			SQL:        sql,
			Measures:   measures,
			Dimensions: dimensions,
		})
	}

	return schema, nil
}

// GetByID returns a single pre-aggregation by its ID.
func (s *PreAggregationService) GetByID(ctx context.Context, id uuid.UUID) (*models.PreAggDescriptor, error) {
	var node struct {
		ID          uuid.UUID       `db:"id"`
		NodeName    string          `db:"node_name"`
		Description string          `db:"description"`
		Properties  json.RawMessage `db:"properties"`
		Config      json.RawMessage `db:"config"`
		CreatedAt   string          `db:"created_at"`
		UpdatedAt   string          `db:"updated_at"`
	}

	err := s.db.GetContext(ctx, &node, `
		SELECT n.id, n.node_name, COALESCE(n.description, '') as description, n.properties, n.config, n.created_at, n.updated_at
		FROM catalog_node n
		JOIN catalog_node_type nt ON n.node_type_id = nt.id
		WHERE nt.catalog_type_name = 'pre_aggregation'
		  AND n.id = $1
	`, id)
	if err != nil {
		return nil, fmt.Errorf("pre-aggregation not found: %w", err)
	}

	props, _ := models.ParsePreAggProperties(node.Properties)
	cfg, _ := models.ParsePreAggConfig(node.Config)

	return &models.PreAggDescriptor{
		ID:                     node.ID,
		TenantID:               props.TenantID,
		BOName:                 props.BOName,
		Name:                   node.NodeName,
		Description:            node.Description,
		TargetDatabase:         props.TargetDatabase,
		TargetName:             cfg.Materialization.TargetName,
		Dialect:                props.Dialect,
		RefreshStrategy:        props.RefreshStrategy,
		RefreshIntervalMinutes: props.RefreshIntervalMinutes,
		GovernanceStatus:       props.GovernanceStatus,
		LifecycleStatus:        props.LifecycleStatus,
		LastMaterializedAt:     props.LastMaterializedAt,
		LastRefreshedAt:        props.LastRefreshedAt,
		LastRefreshStatus:      props.LastRefreshStatus,
		LastRefreshError:       props.LastRefreshError,
		NextScheduledRefresh:   props.NextScheduledRefresh,
		RowCount:               props.RowCount,
		SizeBytes:              props.SizeBytes,
		GroupBy:                cfg.GroupBy,
		Measures:               cfg.Calculations,
		UsageCount:             props.UsageCount,
		AvgLatencyReductionMs:  props.AvgLatencyReductionMs,
	}, nil
}

// ExistsForPattern checks if a pre-aggregation already exists for the given pattern.
func (s *PreAggregationService) ExistsForPattern(ctx context.Context, tenantID, datasource string, groupBy []string) (bool, error) {
	// Convert groupBy to JSON for comparison
	groupByJSON, err := json.Marshal(groupBy)
	if err != nil {
		return false, err
	}

	var count int
	err = s.db.GetContext(ctx, &count, `
		SELECT COUNT(*)
		FROM catalog_node n
		JOIN catalog_node_type nt ON n.node_type_id = nt.id
		WHERE nt.catalog_type_name = 'pre_aggregation'
		  AND n.tenant_id = $1
		  AND n.properties->>'bo_name' = $2
		  AND n.config->'group_by' = $3::jsonb
	`, tenantID, datasource, string(groupByJSON))

	if err != nil {
		return false, fmt.Errorf("failed to check pre-agg existence: %w", err)
	}

	return count > 0, nil
}

// ListByDatasource returns all pre-aggregations for a given datasource and tenant.
func (s *PreAggregationService) ListByDatasource(ctx context.Context, tenantID, datasource string) ([]models.PreAggDescriptor, error) {
	var nodes []struct {
		ID          uuid.UUID       `db:"id"`
		NodeName    string          `db:"node_name"`
		Description string          `db:"description"`
		Properties  json.RawMessage `db:"properties"`
		Config      json.RawMessage `db:"config"`
	}

	err := s.db.SelectContext(ctx, &nodes, `
		SELECT n.id, n.node_name, COALESCE(n.description, '') as description, n.properties, n.config
		FROM catalog_node n
		JOIN catalog_node_type nt ON n.node_type_id = nt.id
		WHERE nt.catalog_type_name = 'pre_aggregation'
		  AND n.tenant_id = $1
		  AND (n.properties->>'datasource' = $2 OR n.properties->>'bo_name' = $2)
	`, tenantID, datasource)
	if err != nil {
		return nil, err
	}

	result := make([]models.PreAggDescriptor, 0, len(nodes))
	for _, n := range nodes {
		props, _ := models.ParsePreAggProperties(n.Properties)
		cfg, _ := models.ParsePreAggConfig(n.Config)
		result = append(result, models.PreAggDescriptor{
			ID:                     n.ID,
			TenantID:               props.TenantID,
			BOName:                 props.BOName,
			Name:                   n.NodeName,
			Description:            n.Description,
			TargetDatabase:         props.TargetDatabase,
			TargetName:             cfg.Materialization.TargetName,
			Dialect:                props.Dialect,
			RefreshStrategy:        props.RefreshStrategy,
			RefreshIntervalMinutes: props.RefreshIntervalMinutes,
			GovernanceStatus:       props.GovernanceStatus,
			LifecycleStatus:        props.LifecycleStatus,
			GroupBy:                cfg.GroupBy,
			Measures:               cfg.Calculations,
		})
	}

	return result, nil
}

// Update updates an existing pre-aggregation.
func (s *PreAggregationService) Update(ctx context.Context, id uuid.UUID, req models.UpsertPreAggRequest) (*models.PreAggDescriptor, error) {
	// Build updated properties
	props := models.PreAggProperties{
		BOName:                 req.BOName,
		TenantID:               req.TenantID,
		Dialect:                "starrocks",
		RefreshStrategy:        req.RefreshStrategy,
		RefreshIntervalMinutes: req.RefreshIntervalMinutes,
		GovernanceStatus:       "draft",
		TargetDatabase:         fmt.Sprintf("tenant_%s", req.TenantID),
	}
	propsJSON, _ := json.Marshal(props)

	// Build updated config
	cfg := models.PreAggConfig{
		Terms:           req.Terms,
		Calculations:    req.Calculations,
		Filters:         req.Filters,
		GroupBy:         req.GroupBy,
		Materialization: req.Materialization,
	}
	cfgJSON, _ := json.Marshal(cfg)

	_, err := s.db.ExecContext(ctx, `
		UPDATE catalog_node SET
			description = $2,
			properties = $3,
			config = $4,
			updated_at = NOW()
		WHERE id = $1
	`, id, req.Description, propsJSON, cfgJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to update pre-aggregation: %w", err)
	}

	return s.GetByID(ctx, id)
}

// Delete removes a pre-aggregation from the catalog.
func (s *PreAggregationService) Delete(ctx context.Context, id uuid.UUID) error {
	// First, delete related edges
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM catalog_edge WHERE source_node_id = $1 OR target_node_id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("failed to delete pre-aggregation edges: %w", err)
	}

	// Then delete the node
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM catalog_node WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("failed to delete pre-aggregation: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("pre-aggregation not found")
	}

	return nil
}

// Disable marks a pre-aggregation as disabled without deleting it.
func (s *PreAggregationService) Disable(ctx context.Context, id uuid.UUID) error {
	// Load current properties
	var propsRaw json.RawMessage
	err := s.db.GetContext(ctx, &propsRaw, `
		SELECT properties FROM catalog_node WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("pre-aggregation not found: %w", err)
	}

	props, err := models.ParsePreAggProperties(propsRaw)
	if err != nil {
		return err
	}

	// Update status to disabled
	props.LifecycleStatus = models.LifecycleFailed // Using failed as a proxy for disabled
	props.GovernanceStatus = "deprecated"

	propsJSON, _ := json.Marshal(props)

	_, err = s.db.ExecContext(ctx, `
		UPDATE catalog_node SET
			properties = $2,
			updated_at = NOW()
		WHERE id = $1
	`, id, propsJSON)
	if err != nil {
		return fmt.Errorf("failed to disable pre-aggregation: %w", err)
	}

	return nil
}

// Helpers

func preAggToPascalCase(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}

func preAggToCamelCase(s string) string {
	pascal := preAggToPascalCase(s)
	if len(pascal) > 0 {
		return strings.ToLower(pascal[:1]) + pascal[1:]
	}
	return pascal
}
