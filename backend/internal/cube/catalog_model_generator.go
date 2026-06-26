package cube

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"gopkg.in/yaml.v3"
)

// CatalogModelGenerator generates Cube.js models from metadata catalog
type CatalogModelGenerator struct {
	db  *sql.DB
	dbx *sqlx.DB
}

// CatalogNode represents a node in the catalog
type CatalogNode struct {
	ID           uuid.UUID  `db:"id"`
	TenantID     uuid.UUID  `db:"tenant_id"`
	DatasourceID *uuid.UUID `db:"tenant_datasource_id"`
	NodeTypeID   uuid.UUID  `db:"node_type_id"`
	NodeTypeName string     `db:"node_type_name"`
	NodeName     string     `db:"node_name"`
	DisplayName  string     `db:"display_name"`
	Description  string     `db:"description"`
	ParentID     *uuid.UUID `db:"parent_id"`
	SchemaPath   string     `db:"schema_path"`
	Properties   []byte     `db:"properties"`
	CreatedAt    time.Time  `db:"created_at"`
}

// CatalogColumn represents column metadata from catalog
type CatalogColumn struct {
	ID           uuid.UUID `db:"id"`
	TableNodeID  uuid.UUID `db:"table_node_id"`
	ColumnName   string    `db:"column_name"`
	DisplayName  string    `db:"display_name"`
	Description  string    `db:"description"`
	DataType     string    `db:"data_type"`
	IsNullable   bool      `db:"is_nullable"`
	IsPrimaryKey bool      `db:"is_primary_key"`
	IsForeignKey bool      `db:"is_foreign_key"`
	DefaultValue *string   `db:"default_value"`
	Properties   []byte    `db:"properties"`
}

// CoreModel represents a generated core Cube model
type CoreModel struct {
	ID             uuid.UUID       `db:"id" json:"id"`
	TenantID       uuid.UUID       `db:"tenant_id" json:"tenant_id"`
	DatasourceID   uuid.UUID       `db:"datasource_id" json:"datasource_id"`
	CatalogNodeID  *uuid.UUID      `db:"catalog_node_id" json:"catalog_node_id"`
	ModelName      string          `db:"model_name" json:"model_name"`
	DisplayName    string          `db:"display_name" json:"display_name"`
	Description    string          `db:"description" json:"description"`
	SQLTable       string          `db:"sql_table" json:"sql_table"`
	DataSource     string          `db:"data_source" json:"data_source"`
	GeneratedYAML  string          `db:"generated_yaml" json:"generated_yaml"`
	YAMLHash       string          `db:"yaml_hash" json:"yaml_hash"`
	RefreshKeySql  *string         `db:"refresh_key_sql" json:"refresh_key_sql"`
	PrimaryKeyCols json.RawMessage `db:"primary_key_columns" json:"primary_key_columns"`
	IsActive       bool            `db:"is_active" json:"is_active"`
	IsPublished    bool            `db:"is_published" json:"is_published"`
	Version        int             `db:"version" json:"version"`
	LastSyncedAt   time.Time       `db:"last_synced_at" json:"last_synced_at"`
}

// CoreMeasure represents a generated measure
type CoreMeasure struct {
	ID              uuid.UUID       `db:"id" json:"id"`
	CoreModelID     uuid.UUID       `db:"core_model_id" json:"core_model_id"`
	CatalogColumnID *uuid.UUID      `db:"catalog_column_id" json:"catalog_column_id"`
	MeasureName     string          `db:"measure_name" json:"measure_name"`
	DisplayName     string          `db:"display_name" json:"display_name"`
	Description     string          `db:"description" json:"description"`
	MeasureType     string          `db:"measure_type" json:"measure_type"`
	SQLExpression   string          `db:"sql_expression" json:"sql_expression"`
	DataType        string          `db:"data_type" json:"data_type"`
	FormatType      *string         `db:"format_type" json:"format_type"`
	FormatMeta      json.RawMessage `db:"format_meta" json:"format_meta"`
	IsVisible       bool            `db:"is_visible" json:"is_visible"`
}

// CoreDimension represents a generated dimension
type CoreDimension struct {
	ID              uuid.UUID       `db:"id" json:"id"`
	CoreModelID     uuid.UUID       `db:"core_model_id" json:"core_model_id"`
	CatalogColumnID *uuid.UUID      `db:"catalog_column_id" json:"catalog_column_id"`
	DimensionName   string          `db:"dimension_name" json:"dimension_name"`
	DisplayName     string          `db:"display_name" json:"display_name"`
	Description     string          `db:"description" json:"description"`
	DimensionType   string          `db:"dimension_type" json:"dimension_type"`
	SQLExpression   string          `db:"sql_expression" json:"sql_expression"`
	IsTimeDimension bool            `db:"is_time_dimension" json:"is_time_dimension"`
	Granularities   json.RawMessage `db:"granularities" json:"granularities"`
	PrimaryKey      bool            `db:"primary_key" json:"primary_key"`
	IsVisible       bool            `db:"is_visible" json:"is_visible"`
}

// CubeYAMLModel represents the Cube.js YAML structure
type CubeYAMLModel struct {
	Cubes []CubeDefinition `yaml:"cubes"`
}

// CubeDefinition represents a single Cube definition
type CubeDefinition struct {
	Name            string          `yaml:"name"`
	Title           string          `yaml:"title,omitempty"`
	Description     string          `yaml:"description,omitempty"`
	SQL             string          `yaml:"sql,omitempty"`
	SQLTable        string          `yaml:"sql_table,omitempty"`
	DataSource      string          `yaml:"data_source,omitempty"`
	RefreshKey      *RefreshKey     `yaml:"refresh_key,omitempty"`
	Measures        []MeasureYAML   `yaml:"measures,omitempty"`
	Dimensions      []DimensionYAML `yaml:"dimensions,omitempty"`
	Joins           []JoinYAML      `yaml:"joins,omitempty"`
	PreAggregations []PreAggYAML    `yaml:"pre_aggregations,omitempty"`
}

type RefreshKey struct {
	SQL   string `yaml:"sql,omitempty"`
	Every string `yaml:"every,omitempty"`
}

type MeasureYAML struct {
	Name          string             `yaml:"name"`
	Title         string             `yaml:"title,omitempty"`
	Description   string             `yaml:"description,omitempty"`
	Type          string             `yaml:"type"`
	SQL           string             `yaml:"sql,omitempty"`
	Format        string             `yaml:"format,omitempty"`
	DrillMembers  []string           `yaml:"drill_members,omitempty"`
	Filters       []FilterYAML       `yaml:"filters,omitempty"`
	RollingWindow *RollingWindowYAML `yaml:"rolling_window,omitempty"`
}

type DimensionYAML struct {
	Name                string    `yaml:"name"`
	Title               string    `yaml:"title,omitempty"`
	Description         string    `yaml:"description,omitempty"`
	Type                string    `yaml:"type"`
	SQL                 string    `yaml:"sql"`
	PrimaryKey          bool      `yaml:"primary_key,omitempty"`
	Shown               *bool     `yaml:"shown,omitempty"`
	Case                *CaseYAML `yaml:"case,omitempty"`
	SuggestFilterValues *bool     `yaml:"suggest_filter_values,omitempty"`
	Granularities       []string  `yaml:"granularities,omitempty"` // For time dimensions
}

type JoinYAML struct {
	Name         string `yaml:"name"`
	SQL          string `yaml:"sql"`
	Relationship string `yaml:"relationship"` // one_to_one, one_to_many, many_to_one
}

type PreAggYAML struct {
	Name                 string      `yaml:"name"`
	Measures             []string    `yaml:"measures,omitempty"`
	Dimensions           []string    `yaml:"dimensions,omitempty"`
	TimeDimension        string      `yaml:"time_dimension,omitempty"`
	Granularity          string      `yaml:"granularity,omitempty"`
	PartitionGranularity string      `yaml:"partition_granularity,omitempty"`
	RefreshKey           *RefreshKey `yaml:"refresh_key,omitempty"`
	ScheduledRefresh     *bool       `yaml:"scheduled_refresh,omitempty"`
	External             *bool       `yaml:"external,omitempty"`
}

type FilterYAML struct {
	SQL string `yaml:"sql"`
}

type RollingWindowYAML struct {
	Trailing string `yaml:"trailing"`
	Offset   string `yaml:"offset,omitempty"`
}

type CaseYAML struct {
	When []WhenYAML `yaml:"when"`
	Else *ElseYAML  `yaml:"else,omitempty"`
}

type WhenYAML struct {
	SQL  string    `yaml:"sql"`
	Then ValueYAML `yaml:"then"`
}

type ElseYAML struct {
	Label string `yaml:"label"`
}

type ValueYAML struct {
	Label string `yaml:"label"`
}

// CustomCubeModel represents a custom Cube model extension
type CustomCubeModel struct {
	ID            uuid.UUID    `db:"id" json:"id"`
	TenantID      uuid.UUID    `db:"tenant_id" json:"tenant_id"`
	DatasourceID  uuid.UUID    `db:"datasource_id" json:"datasource_id"`
	CoreModelID   *uuid.UUID   `db:"core_model_id" json:"core_model_id,omitempty"`
	Name          string       `db:"name" json:"name"`
	Description   string       `db:"description" json:"description"`
	ExtensionType string       `db:"extension_type" json:"extension_type"` // extend, override, standalone
	CustomConfig  CustomConfig `db:"-" json:"custom_config"`
	CustomYAML    string       `db:"custom_yaml" json:"custom_yaml,omitempty"`
	IsActive      bool         `db:"is_active" json:"is_active"`
	Version       int          `db:"version" json:"version"`
	CreatedBy     uuid.UUID    `db:"created_by" json:"created_by"`
	CreatedAt     time.Time    `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time    `db:"updated_at" json:"updated_at"`
}

// CoreCubeModel is an alias for CoreModel for handler compatibility
type CoreCubeModel = CoreModel

// CustomConfig holds the custom configuration for a model
type CustomConfig struct {
	Measures        []CustomMeasure   `json:"measures,omitempty"`
	Dimensions      []CustomDimension `json:"dimensions,omitempty"`
	Joins           []CustomJoin      `json:"joins,omitempty"`
	PreAggregations []CustomPreAgg    `json:"pre_aggregations,omitempty"`
}

// CustomMeasure represents a custom measure
type CustomMeasure struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	SQL    string `json:"sql"`
	Title  string `json:"title"`
	Format string `json:"format,omitempty"`
}

// CustomDimension represents a custom dimension
type CustomDimension struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	SQL        string `json:"sql"`
	Title      string `json:"title"`
	PrimaryKey bool   `json:"primary_key,omitempty"`
}

// CustomJoin represents a custom join
type CustomJoin struct {
	Name         string `json:"name"`
	TargetCube   string `json:"target_cube"`
	Relationship string `json:"relationship"`
	SQL          string `json:"sql"`
}

// CustomPreAgg represents a custom pre-aggregation
type CustomPreAgg struct {
	Name          string   `json:"name"`
	Type          string   `json:"type"`
	Measures      []string `json:"measures,omitempty"`
	Dimensions    []string `json:"dimensions,omitempty"`
	TimeDimension string   `json:"time_dimension,omitempty"`
	Granularity   string   `json:"granularity,omitempty"`
	RefreshKey    string   `json:"refresh_key,omitempty"`
}

// NewCatalogModelGenerator creates a new generator
func NewCatalogModelGenerator(db *sqlx.DB) *CatalogModelGenerator {
	return &CatalogModelGenerator{db: db.DB, dbx: db}
}

// GenerateFromCatalog generates Cube models from catalog metadata for a tenant/datasource
func (g *CatalogModelGenerator) GenerateFromCatalog(ctx context.Context, tenantID, datasourceID uuid.UUID) ([]CoreModel, error) {
	// 1. Get all semantic model nodes from catalog
	semanticModels, err := g.getSemanticModelNodes(ctx, tenantID, datasourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get semantic models: %w", err)
	}

	// 2. Get table nodes if no semantic models exist (fallback to physical catalog)
	if len(semanticModels) == 0 {
		semanticModels, err = g.getTableNodes(ctx, tenantID, datasourceID)
		if err != nil {
			return nil, fmt.Errorf("failed to get table nodes: %w", err)
		}
	}

	var models []CoreModel

	// 3. Generate Cube model for each catalog node
	for _, node := range semanticModels {
		model, err := g.generateModelFromNode(ctx, node, tenantID, datasourceID)
		if err != nil {
			// Log error but continue with other models
			fmt.Printf("Warning: failed to generate model for %s: %v\n", node.NodeName, err)
			continue
		}
		models = append(models, *model)
	}

	return models, nil
}

// getSemanticModelNodes retrieves semantic model nodes from catalog
func (g *CatalogModelGenerator) getSemanticModelNodes(ctx context.Context, tenantID, datasourceID uuid.UUID) ([]CatalogNode, error) {
	query := `
		SELECT cn.id, cn.tenant_id, cn.tenant_datasource_id, cn.node_type_id,
			   cnt.type_name as node_type_name, cn.node_name, 
			   COALESCE(cn.display_name, cn.node_name) as display_name,
			   COALESCE(cn.description, '') as description,
			   cn.parent_id, cn.schema_path, cn.properties, cn.created_at
		FROM public.catalog_node cn
		JOIN public.catalog_node_type cnt ON cn.node_type_id = cnt.id
		WHERE cn.tenant_id = $1
		  AND (cn.tenant_datasource_id = $2 OR cn.tenant_datasource_id IS NULL)
		  AND cnt.type_name = 'semantic_model'
		  AND cn.is_active = true
		ORDER BY cn.node_name`

	rows, err := g.db.QueryContext(ctx, query, tenantID, datasourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []CatalogNode
	for rows.Next() {
		var n CatalogNode
		err := rows.Scan(&n.ID, &n.TenantID, &n.DatasourceID, &n.NodeTypeID,
			&n.NodeTypeName, &n.NodeName, &n.DisplayName, &n.Description,
			&n.ParentID, &n.SchemaPath, &n.Properties, &n.CreatedAt)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, n)
	}
	return nodes, nil
}

// getTableNodes retrieves table nodes from catalog (fallback if no semantic models)
func (g *CatalogModelGenerator) getTableNodes(ctx context.Context, tenantID, datasourceID uuid.UUID) ([]CatalogNode, error) {
	query := `
		SELECT cn.id, cn.tenant_id, cn.tenant_datasource_id, cn.node_type_id,
			   cnt.type_name as node_type_name, cn.node_name,
			   COALESCE(cn.display_name, cn.node_name) as display_name,
			   COALESCE(cn.description, '') as description,
			   cn.parent_id, cn.schema_path, cn.properties, cn.created_at
		FROM public.catalog_node cn
		JOIN public.catalog_node_type cnt ON cn.node_type_id = cnt.id
		WHERE cn.tenant_id = $1
		  AND (cn.tenant_datasource_id = $2 OR cn.tenant_datasource_id IS NULL)
		  AND cnt.type_name = 'table'
		  AND cn.is_active = true
		ORDER BY cn.node_name`

	rows, err := g.db.QueryContext(ctx, query, tenantID, datasourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []CatalogNode
	for rows.Next() {
		var n CatalogNode
		err := rows.Scan(&n.ID, &n.TenantID, &n.DatasourceID, &n.NodeTypeID,
			&n.NodeTypeName, &n.NodeName, &n.DisplayName, &n.Description,
			&n.ParentID, &n.SchemaPath, &n.Properties, &n.CreatedAt)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, n)
	}
	return nodes, nil
}

// getColumnsForNode retrieves column catalog nodes for a table/model
func (g *CatalogModelGenerator) getColumnsForNode(ctx context.Context, parentNodeID uuid.UUID) ([]CatalogColumn, error) {
	// First try semantic_column type, then fall back to column type
	query := `
		SELECT cn.id, cn.parent_id as table_node_id, cn.node_name as column_name,
			   COALESCE(cn.display_name, cn.node_name) as display_name,
			   COALESCE(cn.description, '') as description,
			   COALESCE(cn.properties->>'data_type', 'string') as data_type,
			   COALESCE((cn.properties->>'is_nullable')::boolean, true) as is_nullable,
			   COALESCE((cn.properties->>'is_primary_key')::boolean, false) as is_primary_key,
			   COALESCE((cn.properties->>'is_foreign_key')::boolean, false) as is_foreign_key,
			   cn.properties->>'default_value' as default_value,
			   cn.properties
		FROM public.catalog_node cn
		JOIN public.catalog_node_type cnt ON cn.node_type_id = cnt.id
		WHERE cn.parent_id = $1
		  AND cnt.type_name IN ('semantic_column', 'column')
		  AND cn.is_active = true
		ORDER BY 
			COALESCE((cn.properties->>'ordinal_position')::int, 999),
			cn.node_name`

	rows, err := g.db.QueryContext(ctx, query, parentNodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []CatalogColumn
	for rows.Next() {
		var c CatalogColumn
		err := rows.Scan(&c.ID, &c.TableNodeID, &c.ColumnName, &c.DisplayName,
			&c.Description, &c.DataType, &c.IsNullable, &c.IsPrimaryKey,
			&c.IsForeignKey, &c.DefaultValue, &c.Properties)
		if err != nil {
			return nil, err
		}
		columns = append(columns, c)
	}
	return columns, nil
}

// generateModelFromNode creates a Cube model from a catalog node
func (g *CatalogModelGenerator) generateModelFromNode(ctx context.Context, node CatalogNode, tenantID, datasourceID uuid.UUID) (*CoreModel, error) {
	// Get columns for this node
	columns, err := g.getColumnsForNode(ctx, node.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	// Parse node properties
	var props map[string]interface{}
	if len(node.Properties) > 0 {
		json.Unmarshal(node.Properties, &props)
	}

	// Determine SQL table from properties or schema path
	sqlTable := g.determineSQLTable(node, props)

	// Determine data source
	dataSource := g.determineDataSource(props)

	// Generate measures and dimensions
	measures := g.generateMeasures(columns, node.NodeName)
	dimensions := g.generateDimensions(columns, node.NodeName)

	// Build Cube definition
	cubeDef := CubeDefinition{
		Name:        toCamelCase(node.NodeName),
		Title:       node.DisplayName,
		Description: node.Description,
		SQLTable:    sqlTable,
		DataSource:  dataSource,
		Measures:    measures,
		Dimensions:  dimensions,
		RefreshKey: &RefreshKey{
			Every: "1 hour",
		},
	}

	// Generate YAML
	yamlModel := CubeYAMLModel{
		Cubes: []CubeDefinition{cubeDef},
	}

	yamlBytes, err := yaml.Marshal(yamlModel)
	if err != nil {
		return nil, fmt.Errorf("failed to generate YAML: %w", err)
	}

	yamlStr := string(yamlBytes)
	yamlHash := hashYAML(yamlStr)

	// Build primary key columns
	var pkCols []string
	for _, col := range columns {
		if col.IsPrimaryKey {
			pkCols = append(pkCols, col.ColumnName)
		}
	}
	pkColsJSON, _ := json.Marshal(pkCols)

	model := &CoreModel{
		ID:             uuid.New(),
		TenantID:       tenantID,
		DatasourceID:   datasourceID,
		CatalogNodeID:  &node.ID,
		ModelName:      toCamelCase(node.NodeName),
		DisplayName:    node.DisplayName,
		Description:    node.Description,
		SQLTable:       sqlTable,
		DataSource:     dataSource,
		GeneratedYAML:  yamlStr,
		YAMLHash:       yamlHash,
		PrimaryKeyCols: pkColsJSON,
		IsActive:       true,
		IsPublished:    false,
		Version:        1,
		LastSyncedAt:   time.Now(),
	}

	return model, nil
}

// generateMeasures creates Cube measures from columns
func (g *CatalogModelGenerator) generateMeasures(columns []CatalogColumn, modelName string) []MeasureYAML {
	measures := []MeasureYAML{
		// Always add a count measure
		{
			Name:        "count",
			Title:       "Count",
			Description: fmt.Sprintf("Total count of %s records", modelName),
			Type:        "count",
		},
	}

	for _, col := range columns {
		// Generate sum/avg measures for numeric columns
		if isNumericDataType(col.DataType) && !col.IsPrimaryKey && !col.IsForeignKey {
			// Sum measure
			measures = append(measures, MeasureYAML{
				Name:        fmt.Sprintf("total_%s", toSnakeCase(col.ColumnName)),
				Title:       fmt.Sprintf("Total %s", col.DisplayName),
				Description: fmt.Sprintf("Sum of %s", col.Description),
				Type:        "sum",
				SQL:         fmt.Sprintf("${CUBE}.%s", col.ColumnName),
				Format:      inferFormat(col),
			})

			// Average measure
			measures = append(measures, MeasureYAML{
				Name:        fmt.Sprintf("avg_%s", toSnakeCase(col.ColumnName)),
				Title:       fmt.Sprintf("Average %s", col.DisplayName),
				Description: fmt.Sprintf("Average of %s", col.Description),
				Type:        "avg",
				SQL:         fmt.Sprintf("${CUBE}.%s", col.ColumnName),
				Format:      inferFormat(col),
			})
		}
	}

	return measures
}

// generateDimensions creates Cube dimensions from columns
func (g *CatalogModelGenerator) generateDimensions(columns []CatalogColumn, modelName string) []DimensionYAML {
	var dimensions []DimensionYAML

	for _, col := range columns {
		dim := DimensionYAML{
			Name:        toSnakeCase(col.ColumnName),
			Title:       col.DisplayName,
			Description: col.Description,
			Type:        mapToCubeType(col.DataType),
			SQL:         fmt.Sprintf("${CUBE}.%s", col.ColumnName),
			PrimaryKey:  col.IsPrimaryKey,
		}

		// Time dimension handling
		if isTimeType(col.DataType) {
			dim.Type = "time"
			// Add standard granularities for time dimensions
			dim.Granularities = []string{"day", "week", "month", "quarter", "year"}
		}

		// Set suggest_filter_values for string dimensions
		if dim.Type == "string" && !col.IsPrimaryKey {
			suggestTrue := true
			dim.SuggestFilterValues = &suggestTrue
		}

		dimensions = append(dimensions, dim)
	}

	return dimensions
}

// SaveCoreModel saves a generated core model to the database
func (g *CatalogModelGenerator) SaveCoreModel(ctx context.Context, model *CoreModel) error {
	query := `
		INSERT INTO public.cube_core_models (
			id, tenant_id, datasource_id, catalog_node_id, model_name,
			display_name, description, sql_table, data_source,
			generated_yaml, yaml_hash, primary_key_columns,
			is_active, is_published, version, last_synced_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		)
		ON CONFLICT (tenant_id, datasource_id, model_name) DO UPDATE SET
			catalog_node_id = EXCLUDED.catalog_node_id,
			display_name = EXCLUDED.display_name,
			description = EXCLUDED.description,
			sql_table = EXCLUDED.sql_table,
			data_source = EXCLUDED.data_source,
			generated_yaml = EXCLUDED.generated_yaml,
			yaml_hash = EXCLUDED.yaml_hash,
			primary_key_columns = EXCLUDED.primary_key_columns,
			version = cube_core_models.version + 1,
			last_synced_at = EXCLUDED.last_synced_at,
			updated_at = NOW()
		RETURNING id, version`

	err := g.db.QueryRowContext(ctx, query,
		model.ID, model.TenantID, model.DatasourceID, model.CatalogNodeID,
		model.ModelName, model.DisplayName, model.Description,
		model.SQLTable, model.DataSource, model.GeneratedYAML, model.YAMLHash,
		model.PrimaryKeyCols, model.IsActive, model.IsPublished,
		model.Version, model.LastSyncedAt,
	).Scan(&model.ID, &model.Version)

	return err
}

// SyncFromCatalog performs a full sync from catalog to core models
func (g *CatalogModelGenerator) SyncFromCatalog(ctx context.Context, tenantID, datasourceID uuid.UUID) (int, int, error) {
	models, err := g.GenerateFromCatalog(ctx, tenantID, datasourceID)
	if err != nil {
		return 0, 0, err
	}

	created := 0
	updated := 0

	for _, model := range models {
		// Check if model exists
		var existingHash string
		err := g.db.QueryRowContext(ctx,
			`SELECT yaml_hash FROM cube_core_models WHERE tenant_id = $1 AND datasource_id = $2 AND model_name = $3`,
			tenantID, datasourceID, model.ModelName,
		).Scan(&existingHash)

		if err == sql.ErrNoRows {
			// New model
			if err := g.SaveCoreModel(ctx, &model); err != nil {
				return created, updated, fmt.Errorf("failed to save new model %s: %w", model.ModelName, err)
			}
			created++
		} else if err == nil {
			// Existing model - check if changed
			if existingHash != model.YAMLHash {
				if err := g.SaveCoreModel(ctx, &model); err != nil {
					return created, updated, fmt.Errorf("failed to update model %s: %w", model.ModelName, err)
				}
				updated++
			}
		} else {
			return created, updated, fmt.Errorf("failed to check existing model: %w", err)
		}
	}

	return created, updated, nil
}

// GetMergedYAML returns the final YAML for a model (core + custom merged)
func (g *CatalogModelGenerator) GetMergedYAML(ctx context.Context, tenantID, datasourceID uuid.UUID, modelName string) (string, error) {
	// First check for custom model
	var customYAML, coreYAML sql.NullString
	var extensionMode string

	err := g.db.QueryRowContext(ctx, `
		SELECT cm.custom_yaml, cm.extension_mode, core.generated_yaml
		FROM cube_custom_models cm
		LEFT JOIN cube_core_models core ON cm.extends_core_model_id = core.id
		WHERE cm.tenant_id = $1 AND cm.datasource_id = $2 AND cm.model_name = $3 AND cm.is_active = true`,
		tenantID, datasourceID, modelName,
	).Scan(&customYAML, &extensionMode, &coreYAML)

	if err == sql.ErrNoRows {
		// No custom model, return core only
		var yaml string
		err := g.db.QueryRowContext(ctx, `
			SELECT generated_yaml FROM cube_core_models
			WHERE tenant_id = $1 AND datasource_id = $2 AND model_name = $3 AND is_active = true`,
			tenantID, datasourceID, modelName,
		).Scan(&yaml)
		return yaml, err
	}

	if err != nil {
		return "", err
	}

	// Handle extension modes
	switch extensionMode {
	case "standalone":
		return customYAML.String, nil
	case "override":
		if customYAML.Valid {
			return customYAML.String, nil
		}
		return coreYAML.String, nil
	case "extend":
		fallthrough
	default:
		// Merge custom into core
		return g.mergeYAML(coreYAML.String, customYAML.String)
	}
}

// mergeYAML merges custom YAML into core YAML
func (g *CatalogModelGenerator) mergeYAML(coreYAML, customYAML string) (string, error) {
	if customYAML == "" {
		return coreYAML, nil
	}
	if coreYAML == "" {
		return customYAML, nil
	}

	var core, custom CubeYAMLModel
	if err := yaml.Unmarshal([]byte(coreYAML), &core); err != nil {
		return "", fmt.Errorf("failed to parse core YAML: %w", err)
	}
	if err := yaml.Unmarshal([]byte(customYAML), &custom); err != nil {
		return "", fmt.Errorf("failed to parse custom YAML: %w", err)
	}

	// Simple merge: add custom measures/dimensions, override by name
	if len(core.Cubes) > 0 && len(custom.Cubes) > 0 {
		coreCube := &core.Cubes[0]
		customCube := custom.Cubes[0]

		// Merge measures
		measureMap := make(map[string]MeasureYAML)
		for _, m := range coreCube.Measures {
			measureMap[m.Name] = m
		}
		for _, m := range customCube.Measures {
			measureMap[m.Name] = m // Custom overrides core
		}
		coreCube.Measures = nil
		for _, m := range measureMap {
			coreCube.Measures = append(coreCube.Measures, m)
		}

		// Merge dimensions
		dimMap := make(map[string]DimensionYAML)
		for _, d := range coreCube.Dimensions {
			dimMap[d.Name] = d
		}
		for _, d := range customCube.Dimensions {
			dimMap[d.Name] = d // Custom overrides core
		}
		coreCube.Dimensions = nil
		for _, d := range dimMap {
			coreCube.Dimensions = append(coreCube.Dimensions, d)
		}

		// Add custom joins
		coreCube.Joins = append(coreCube.Joins, customCube.Joins...)

		// Merge pre-aggregations
		coreCube.PreAggregations = append(coreCube.PreAggregations, customCube.PreAggregations...)
	}

	merged, err := yaml.Marshal(core)
	return string(merged), err
}

// Helper functions

func (g *CatalogModelGenerator) determineSQLTable(node CatalogNode, props map[string]interface{}) string {
	// Try properties first
	if props != nil {
		if table, ok := props["sql_table"].(string); ok && table != "" {
			return table
		}
		if schema, ok := props["schema_name"].(string); ok && schema != "" {
			return fmt.Sprintf("%s.%s", schema, node.NodeName)
		}
	}

	// Parse from schema path: /public/orders -> public.orders
	if node.SchemaPath != "" {
		parts := strings.Split(strings.Trim(node.SchemaPath, "/"), "/")
		if len(parts) >= 2 {
			return fmt.Sprintf("%s.%s", parts[0], parts[1])
		}
	}

	return node.NodeName
}

func (g *CatalogModelGenerator) determineDataSource(props map[string]interface{}) string {
	if props != nil {
		if ds, ok := props["data_source"].(string); ok && ds != "" {
			return ds
		}
	}
	return "default"
}

func hashYAML(yaml string) string {
	h := sha256.New()
	h.Write([]byte(yaml))
	return hex.EncodeToString(h.Sum(nil))
}

func toCamelCase(s string) string {
	// Convert snake_case or kebab-case to CamelCase
	re := regexp.MustCompile(`[_-]([a-z])`)
	result := re.ReplaceAllStringFunc(s, func(m string) string {
		return strings.ToUpper(string(m[1]))
	})
	// Capitalize first letter
	if len(result) > 0 {
		return strings.ToUpper(string(result[0])) + result[1:]
	}
	return result
}

func toSnakeCase(s string) string {
	// Convert CamelCase to snake_case
	re := regexp.MustCompile(`([A-Z])`)
	result := re.ReplaceAllString(s, "_$1")
	result = strings.ToLower(strings.TrimPrefix(result, "_"))
	// Replace spaces and hyphens with underscores
	result = strings.ReplaceAll(result, " ", "_")
	result = strings.ReplaceAll(result, "-", "_")
	return result
}

func isNumericDataType(dataType string) bool {
	numericTypes := []string{
		"int", "integer", "bigint", "smallint", "tinyint",
		"decimal", "numeric", "float", "double", "real",
		"money", "number", "int4", "int8", "float4", "float8",
	}
	lower := strings.ToLower(dataType)
	for _, t := range numericTypes {
		if strings.Contains(lower, t) {
			return true
		}
	}
	return false
}

func isTimeType(dataType string) bool {
	timeTypes := []string{
		"date", "time", "timestamp", "datetime", "timestamptz",
	}
	lower := strings.ToLower(dataType)
	for _, t := range timeTypes {
		if strings.Contains(lower, t) {
			return true
		}
	}
	return false
}

func mapToCubeType(dataType string) string {
	lower := strings.ToLower(dataType)

	if isTimeType(lower) {
		return "time"
	}
	if isNumericDataType(lower) {
		return "number"
	}
	if strings.Contains(lower, "bool") {
		return "boolean"
	}
	if strings.Contains(lower, "geo") || strings.Contains(lower, "point") {
		return "geo"
	}
	return "string"
}

func inferFormat(col CatalogColumn) string {
	lower := strings.ToLower(col.ColumnName)
	if strings.Contains(lower, "amount") || strings.Contains(lower, "price") ||
		strings.Contains(lower, "cost") || strings.Contains(lower, "revenue") ||
		strings.Contains(lower, "total") {
		return "currency"
	}
	if strings.Contains(lower, "percent") || strings.Contains(lower, "rate") ||
		strings.Contains(lower, "ratio") {
		return "percent"
	}
	return "number"
}

// ============================================================================
// CRUD METHODS FOR HANDLER COMPATIBILITY
// ============================================================================

// ListCoreModels returns all core models for a tenant/datasource
func (g *CatalogModelGenerator) ListCoreModels(ctx context.Context, tenantID, datasourceID string) ([]CoreModel, error) {
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant_id: %w", err)
	}
	did, err := uuid.Parse(datasourceID)
	if err != nil {
		return nil, fmt.Errorf("invalid datasource_id: %w", err)
	}

	query := `
		SELECT id, tenant_id, datasource_id, catalog_node_id, model_name, display_name,
		       description, sql_table, data_source, generated_yaml, yaml_hash,
		       refresh_key_sql, primary_key_columns, is_active, is_published, version, last_synced_at
		FROM cube_core_models
		WHERE tenant_id = $1 AND datasource_id = $2 AND is_active = true
		ORDER BY model_name`

	rows, err := g.db.QueryContext(ctx, query, tid, did)
	if err != nil {
		return nil, fmt.Errorf("failed to list core models: %w", err)
	}
	defer rows.Close()

	var models []CoreModel
	for rows.Next() {
		var m CoreModel
		err := rows.Scan(
			&m.ID, &m.TenantID, &m.DatasourceID, &m.CatalogNodeID, &m.ModelName, &m.DisplayName,
			&m.Description, &m.SQLTable, &m.DataSource, &m.GeneratedYAML, &m.YAMLHash,
			&m.RefreshKeySql, &m.PrimaryKeyCols, &m.IsActive, &m.IsPublished, &m.Version, &m.LastSyncedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan core model: %w", err)
		}
		models = append(models, m)
	}
	return models, nil
}

// GenerateCoreModelsFromCatalog generates and saves core models from catalog
func (g *CatalogModelGenerator) GenerateCoreModelsFromCatalog(ctx context.Context, tenantID, datasourceID string) ([]CoreModel, error) {
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant_id: %w", err)
	}
	did, err := uuid.Parse(datasourceID)
	if err != nil {
		return nil, fmt.Errorf("invalid datasource_id: %w", err)
	}

	models, err := g.GenerateFromCatalog(ctx, tid, did)
	if err != nil {
		return nil, err
	}

	// Save each model
	for i := range models {
		if err := g.SaveCoreModel(ctx, &models[i]); err != nil {
			return nil, fmt.Errorf("failed to save model %s: %w", models[i].ModelName, err)
		}
	}

	return models, nil
}

// GetCoreModel retrieves a single core model by ID
func (g *CatalogModelGenerator) GetCoreModel(ctx context.Context, id uuid.UUID) (*CoreModel, error) {
	query := `
		SELECT id, tenant_id, datasource_id, catalog_node_id, model_name, display_name,
		       description, sql_table, data_source, generated_yaml, yaml_hash,
		       refresh_key_sql, primary_key_columns, is_active, is_published, version, last_synced_at
		FROM cube_core_models
		WHERE id = $1`

	var m CoreModel
	err := g.db.QueryRowContext(ctx, query, id).Scan(
		&m.ID, &m.TenantID, &m.DatasourceID, &m.CatalogNodeID, &m.ModelName, &m.DisplayName,
		&m.Description, &m.SQLTable, &m.DataSource, &m.GeneratedYAML, &m.YAMLHash,
		&m.RefreshKeySql, &m.PrimaryKeyCols, &m.IsActive, &m.IsPublished, &m.Version, &m.LastSyncedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("core model not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get core model: %w", err)
	}
	return &m, nil
}

// DeleteCoreModel deletes a core model
func (g *CatalogModelGenerator) DeleteCoreModel(ctx context.Context, tenantID, modelID uuid.UUID) error {
	result, err := g.db.ExecContext(ctx,
		`DELETE FROM cube_core_models WHERE id = $1 AND tenant_id = $2`,
		modelID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to delete core model: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("core model not found")
	}
	return nil
}

// GenerateCubeYAML generates YAML for a core model
func (g *CatalogModelGenerator) GenerateCubeYAML(ctx context.Context, model *CoreModel) (string, error) {
	if model.GeneratedYAML != "" {
		return model.GeneratedYAML, nil
	}
	// Generate from model data
	cube := CubeDefinition{
		Name:        model.ModelName,
		Title:       model.DisplayName,
		Description: model.Description,
		SQLTable:    model.SQLTable,
		DataSource:  model.DataSource,
	}
	yamlModel := CubeYAMLModel{Cubes: []CubeDefinition{cube}}
	yamlBytes, err := yaml.Marshal(yamlModel)
	return string(yamlBytes), err
}

// ListCustomModels returns all custom models for a tenant/datasource
func (g *CatalogModelGenerator) ListCustomModels(ctx context.Context, tenantID, datasourceID string) ([]CustomCubeModel, error) {
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant_id: %w", err)
	}
	did, err := uuid.Parse(datasourceID)
	if err != nil {
		return nil, fmt.Errorf("invalid datasource_id: %w", err)
	}

	query := `
		SELECT id, tenant_id, datasource_id, core_model_id, name, description,
		       extension_type, custom_yaml, is_active, version, created_by, created_at, updated_at
		FROM cube_custom_models
		WHERE tenant_id = $1 AND datasource_id = $2 AND is_active = true
		ORDER BY name`

	rows, err := g.db.QueryContext(ctx, query, tid, did)
	if err != nil {
		return nil, fmt.Errorf("failed to list custom models: %w", err)
	}
	defer rows.Close()

	var models []CustomCubeModel
	for rows.Next() {
		var m CustomCubeModel
		err := rows.Scan(
			&m.ID, &m.TenantID, &m.DatasourceID, &m.CoreModelID, &m.Name, &m.Description,
			&m.ExtensionType, &m.CustomYAML, &m.IsActive, &m.Version, &m.CreatedBy, &m.CreatedAt, &m.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan custom model: %w", err)
		}
		// Parse custom config from YAML if present
		if m.CustomYAML != "" {
			_ = json.Unmarshal([]byte(m.CustomYAML), &m.CustomConfig)
		}
		models = append(models, m)
	}
	return models, nil
}

// CreateCustomModel creates a new custom model
func (g *CatalogModelGenerator) CreateCustomModel(ctx context.Context, model *CustomCubeModel) error {
	model.ID = uuid.New()
	model.IsActive = true
	model.Version = 1
	model.CreatedAt = time.Now()
	model.UpdatedAt = time.Now()

	// Serialize custom config to YAML
	configJSON, _ := json.Marshal(model.CustomConfig)
	model.CustomYAML = string(configJSON)

	query := `
		INSERT INTO cube_custom_models (
			id, tenant_id, datasource_id, core_model_id, name, description,
			extension_type, custom_yaml, is_active, version, created_by, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`

	_, err := g.db.ExecContext(ctx, query,
		model.ID, model.TenantID, model.DatasourceID, model.CoreModelID, model.Name, model.Description,
		model.ExtensionType, model.CustomYAML, model.IsActive, model.Version, model.CreatedBy, model.CreatedAt, model.UpdatedAt,
	)
	return err
}

// GetCustomModel retrieves a single custom model by ID
func (g *CatalogModelGenerator) GetCustomModel(ctx context.Context, id uuid.UUID) (*CustomCubeModel, error) {
	query := `
		SELECT id, tenant_id, datasource_id, core_model_id, name, description,
		       extension_type, custom_yaml, is_active, version, created_by, created_at, updated_at
		FROM cube_custom_models
		WHERE id = $1`

	var m CustomCubeModel
	err := g.db.QueryRowContext(ctx, query, id).Scan(
		&m.ID, &m.TenantID, &m.DatasourceID, &m.CoreModelID, &m.Name, &m.Description,
		&m.ExtensionType, &m.CustomYAML, &m.IsActive, &m.Version, &m.CreatedBy, &m.CreatedAt, &m.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("custom model not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get custom model: %w", err)
	}
	// Parse custom config
	if m.CustomYAML != "" {
		_ = json.Unmarshal([]byte(m.CustomYAML), &m.CustomConfig)
	}
	return &m, nil
}

// UpdateCustomModel updates an existing custom model
func (g *CatalogModelGenerator) UpdateCustomModel(ctx context.Context, model *CustomCubeModel) error {
	model.UpdatedAt = time.Now()

	// Serialize custom config to YAML
	configJSON, _ := json.Marshal(model.CustomConfig)
	model.CustomYAML = string(configJSON)

	query := `
		UPDATE cube_custom_models SET
			name = $2, description = $3, extension_type = $4, custom_yaml = $5,
			version = $6, updated_at = $7
		WHERE id = $1`

	result, err := g.db.ExecContext(ctx, query,
		model.ID, model.Name, model.Description, model.ExtensionType, model.CustomYAML,
		model.Version, model.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update custom model: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("custom model not found")
	}
	return nil
}

// DeleteCustomModel deletes a custom model
func (g *CatalogModelGenerator) DeleteCustomModel(ctx context.Context, tenantID, modelID uuid.UUID) error {
	result, err := g.db.ExecContext(ctx,
		`DELETE FROM cube_custom_models WHERE id = $1 AND tenant_id = $2`,
		modelID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to delete custom model: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("custom model not found")
	}
	return nil
}

// GenerateCustomYAML generates YAML for a custom model
func (g *CatalogModelGenerator) GenerateCustomYAML(ctx context.Context, model *CustomCubeModel) (string, error) {
	cube := CubeDefinition{
		Name:        model.Name,
		Description: model.Description,
	}

	// Add custom measures
	for _, m := range model.CustomConfig.Measures {
		cube.Measures = append(cube.Measures, MeasureYAML{
			Name:   m.Name,
			Title:  m.Title,
			Type:   m.Type,
			SQL:    m.SQL,
			Format: m.Format,
		})
	}

	// Add custom dimensions
	for _, d := range model.CustomConfig.Dimensions {
		cube.Dimensions = append(cube.Dimensions, DimensionYAML{
			Name:       d.Name,
			Title:      d.Title,
			Type:       d.Type,
			SQL:        d.SQL,
			PrimaryKey: d.PrimaryKey,
		})
	}

	// Add custom joins
	for _, j := range model.CustomConfig.Joins {
		cube.Joins = append(cube.Joins, JoinYAML{
			Name:         j.Name,
			SQL:          j.SQL,
			Relationship: j.Relationship,
		})
	}

	// Add custom pre-aggregations
	for _, p := range model.CustomConfig.PreAggregations {
		cube.PreAggregations = append(cube.PreAggregations, PreAggYAML{
			Name:          p.Name,
			Measures:      p.Measures,
			Dimensions:    p.Dimensions,
			TimeDimension: p.TimeDimension,
			Granularity:   p.Granularity,
		})
	}

	yamlModel := CubeYAMLModel{Cubes: []CubeDefinition{cube}}
	yamlBytes, err := yaml.Marshal(yamlModel)
	return string(yamlBytes), err
}

// MergeCoreAndCustomYAML merges core and custom YAML
func (g *CatalogModelGenerator) MergeCoreAndCustomYAML(ctx context.Context, coreModel *CoreModel, customModel *CustomCubeModel) (string, error) {
	var coreYAML, customYAML string
	var err error

	if coreModel != nil {
		coreYAML, err = g.GenerateCubeYAML(ctx, coreModel)
		if err != nil {
			return "", err
		}
	}

	if customModel != nil {
		customYAML, err = g.GenerateCustomYAML(ctx, customModel)
		if err != nil {
			return "", err
		}
	}

	return g.mergeYAML(coreYAML, customYAML)
}

// GenerateYAMLFromSpec generates YAML from a model specification
func (g *CatalogModelGenerator) GenerateYAMLFromSpec(
	cubeName, sqlTable, dataSource, description string,
	measures interface{}, dimensions interface{}, joins interface{}, preAggregations interface{},
) string {
	cube := CubeDefinition{
		Name:        cubeName,
		SQLTable:    sqlTable,
		DataSource:  dataSource,
		Description: description,
	}

	// Convert measures
	if m, ok := measures.([]struct {
		Name   string `json:"name"`
		Type   string `json:"type"`
		SQL    string `json:"sql"`
		Title  string `json:"title"`
		Format string `json:"format,omitempty"`
	}); ok {
		for _, measure := range m {
			cube.Measures = append(cube.Measures, MeasureYAML{
				Name:   measure.Name,
				Title:  measure.Title,
				Type:   measure.Type,
				SQL:    measure.SQL,
				Format: measure.Format,
			})
		}
	}

	// Convert dimensions
	if d, ok := dimensions.([]struct {
		Name       string `json:"name"`
		Type       string `json:"type"`
		SQL        string `json:"sql"`
		Title      string `json:"title"`
		PrimaryKey bool   `json:"primary_key,omitempty"`
	}); ok {
		for _, dim := range d {
			cube.Dimensions = append(cube.Dimensions, DimensionYAML{
				Name:       dim.Name,
				Title:      dim.Title,
				Type:       dim.Type,
				SQL:        dim.SQL,
				PrimaryKey: dim.PrimaryKey,
			})
		}
	}

	// Convert joins
	if j, ok := joins.([]struct {
		Name         string `json:"name"`
		TargetCube   string `json:"target_cube"`
		Relationship string `json:"relationship"`
		SQL          string `json:"sql"`
	}); ok {
		for _, join := range j {
			cube.Joins = append(cube.Joins, JoinYAML{
				Name:         join.Name,
				SQL:          join.SQL,
				Relationship: join.Relationship,
			})
		}
	}

	// Convert pre-aggregations
	if p, ok := preAggregations.([]struct {
		Name          string   `json:"name"`
		Type          string   `json:"type"`
		Measures      []string `json:"measures"`
		Dimensions    []string `json:"dimensions"`
		TimeDimension string   `json:"time_dimension,omitempty"`
		Granularity   string   `json:"granularity,omitempty"`
	}); ok {
		for _, preAgg := range p {
			cube.PreAggregations = append(cube.PreAggregations, PreAggYAML{
				Name:          preAgg.Name,
				Measures:      preAgg.Measures,
				Dimensions:    preAgg.Dimensions,
				TimeDimension: preAgg.TimeDimension,
				Granularity:   preAgg.Granularity,
			})
		}
	}

	yamlModel := CubeYAMLModel{Cubes: []CubeDefinition{cube}}
	yamlBytes, _ := yaml.Marshal(yamlModel)
	return string(yamlBytes)
}

// ValidateCubeYAML validates YAML syntax and structure
func (g *CatalogModelGenerator) ValidateCubeYAML(yamlContent string) []string {
	var errors []string

	var model CubeYAMLModel
	if err := yaml.Unmarshal([]byte(yamlContent), &model); err != nil {
		errors = append(errors, fmt.Sprintf("YAML parse error: %v", err))
		return errors
	}

	if len(model.Cubes) == 0 {
		errors = append(errors, "No cubes defined in YAML")
		return errors
	}

	for i, cube := range model.Cubes {
		if cube.Name == "" {
			errors = append(errors, fmt.Sprintf("Cube %d: name is required", i+1))
		}
		if cube.SQLTable == "" && cube.SQL == "" {
			errors = append(errors, fmt.Sprintf("Cube %s: either sql or sql_table is required", cube.Name))
		}
		for j, m := range cube.Measures {
			if m.Name == "" {
				errors = append(errors, fmt.Sprintf("Cube %s, measure %d: name is required", cube.Name, j+1))
			}
			if m.Type == "" {
				errors = append(errors, fmt.Sprintf("Cube %s, measure %s: type is required", cube.Name, m.Name))
			}
		}
		for j, d := range cube.Dimensions {
			if d.Name == "" {
				errors = append(errors, fmt.Sprintf("Cube %s, dimension %d: name is required", cube.Name, j+1))
			}
			if d.SQL == "" {
				errors = append(errors, fmt.Sprintf("Cube %s, dimension %s: sql is required", cube.Name, d.Name))
			}
		}
	}

	return errors
}
