package analytics

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/hondyman/uisce/backend/internal/rules/vm"
	"github.com/jmoiron/sqlx"

	_ "github.com/go-sql-driver/mysql" // StarRocks uses MySQL protocol
)

// PreAggregationService manages pre-aggregation definitions and materializations.
type PreAggregationService struct {
	db               *sqlx.DB
	boResolver       *BOContextResolver
	semanticGraphSvc *SemanticGraphService
	starrocksDB      *sql.DB // MySQL-protocol connection to StarRocks FE (9030); nil if unavailable
}

func NewPreAggregationService(db *sqlx.DB, boResolver *BOContextResolver, semanticGraphSvc *SemanticGraphService) *PreAggregationService {
	return &PreAggregationService{
		db:               db,
		boResolver:       boResolver,
		semanticGraphSvc: semanticGraphSvc,
		starrocksDB:      newStarRocksDB(),
	}
}

// newStarRocksDB opens a MySQL-protocol connection to the StarRocks FE using
// the same STARROCKS_HOST/PORT/USER/PASSWORD env vars as AnalyticsService and
// AggregateService. No database is selected in the DSN because the DDL this
// service generates always uses fully-qualified `database.table` names
// (e.g. "tenant_<id>.mv_execution_npv", "oms.orm_execution"). Returns nil on
// any failure so callers degrade to the existing stub behavior instead of
// blocking server startup on an optional dependency.
func newStarRocksDB() *sql.DB {
	host := getEnv("STARROCKS_HOST", "127.0.0.1")
	port := getEnvInt("STARROCKS_PORT", 9030)
	user := getEnv("STARROCKS_USER", "root")
	password := getEnv("STARROCKS_PASSWORD", "")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?parseTime=true&multiStatements=true", user, password, host, port)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		fmt.Printf("WARNING: failed to open StarRocks connection for pre-aggregations: %v\n", err)
		return nil
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(time.Hour)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		fmt.Printf("WARNING: StarRocks ping failed for pre-aggregations (materialization apply/refresh will be unavailable): %v\n", err)
		return nil
	}
	return db
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
		ON CONFLICT (tenant_id, qualified_path) DO UPDATE SET
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

// quoteIdent backtick-quotes a StarRocks/MySQL identifier. Needed because
// target database names are derived from tenant UUIDs (e.g.
// "tenant_99e99e99-99e9-...") and unquoted hyphens are a syntax error.
func quoteIdent(ident string) string {
	return "`" + strings.ReplaceAll(ident, "`", "``") + "`"
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

	// 2. Get the real BO (business_objects, not the unrelated legacy
	// catalog_node "business_object" prototype) and its driving table's
	// qualified_path (e.g. "/orm/execution"). business_objects.driver_table_name
	// on this schema is stored in qualified_path form, not "schema.table" -
	// used directly below to scope MAPS_TO edge lookups to this BO's table.
	var bo struct {
		ID              string `db:"id"`
		DriverTableName string `db:"driver_table_name"`
	}
	err = s.db.GetContext(ctx, &bo, `
		SELECT id, COALESCE(driver_table_name, '') AS driver_table_name
		FROM business_objects
		WHERE (bo_key = $1 OR bo_name = $1) AND tenant_id = $2::uuid
		LIMIT 1
	`, props.BOName, props.TenantID)
	if err != nil {
		return "", fmt.Errorf("BO '%s' not found: %w", props.BOName, err)
	}
	if bo.DriverTableName == "" {
		return "", fmt.Errorf("BO '%s' has no driver_table_name set", props.BOName)
	}
	// StarRocks hot-tier mirror tables are named oms.orm_<table>, matching
	// the last qualified_path segment (see starrocks_init.sql).
	tableSuffix := bo.DriverTableName
	if idx := strings.LastIndex(tableSuffix, "/"); idx >= 0 {
		tableSuffix = tableSuffix[idx+1:]
	}
	starrocksTable := fmt.Sprintf("%s.%s", quoteIdent("oms"), quoteIdent("orm_"+tableSuffix))

	// 3. Resolve each requested field to its real physical column via the
	// business_object_fields -> MAPS_TO -> catalog_node(column) chain,
	// scoped to this BO's table by qualified_path prefix (the same fields
	// can be shared across many tables' columns, e.g. "CreatedAt", so an
	// unscoped lookup would be ambiguous).
	resolveColumn := func(fieldName string) (string, error) {
		var col string
		err := s.db.GetContext(ctx, &col, `
			SELECT col.node_name
			FROM business_object_fields bf
			JOIN catalog_edge ce ON ce.source_node_id = bf.term_node_id
			JOIN catalog_edge_type et ON et.id = ce.edge_type_id
			JOIN catalog_node col ON col.id = ce.target_node_id
			WHERE bf.bo_id = $1::uuid AND bf.field_name = $2
			  AND et.edge_type_name = 'MAPS_TO'
			  AND col.qualified_path LIKE $3 || '/%'
			LIMIT 1
		`, bo.ID, fieldName, bo.DriverTableName)
		if err != nil {
			return "", err
		}
		return col, nil
	}

	// 4. Build SELECT list from terms + calculations. Dimensions (GroupBy)
	// resolve to real columns on the StarRocks mirror table. Calculated
	// terms compile to SQL via vm.CompileToSQL (internal/rules/vm) when the
	// term's catalog_node.config carries a "rule_ast" — the same
	// FuncCall/BinaryExpr/FieldRef AST already used by the rules VM for
	// validation/MDM rules, reused here as the SQL-pushdown backend for
	// calculated terms. Most calc terms in the catalog still store only a
	// free-text formula (Excel-formula-syntax strings, ${field} macros,
	// etc.) with no "rule_ast" and no compiler for those formats - those
	// remain an explicit NULL placeholder rather than fabricated SQL.
	selectCols := make([]string, 0, len(cfg.GroupBy)+len(cfg.Calculations))
	for _, dim := range cfg.GroupBy {
		col, err := resolveColumn(dim)
		if err != nil {
			return "", fmt.Errorf("failed to resolve dimension %q: %w", dim, err)
		}
		selectCols = append(selectCols, fmt.Sprintf("%s AS %q", col, dim))
	}
	for _, calc := range cfg.Calculations {
		sqlExpr, compileErr := s.compileCalcTermToSQL(ctx, calc, resolveColumn)
		if compileErr != nil {
			selectCols = append(selectCols, fmt.Sprintf(
				"NULL /* TODO: %q could not be compiled to SQL: %s */ AS %q",
				calc, sanitizeSQLComment(compileErr.Error()), calc,
			))
			continue
		}
		selectCols = append(selectCols, fmt.Sprintf("%s AS %q", sqlExpr, calc))
	}
	boSQL := fmt.Sprintf("SELECT * FROM %s", starrocksTable)

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
	targetQualified := fmt.Sprintf("%s.%s", quoteIdent(props.TargetDatabase), quoteIdent(cfg.Materialization.TargetName))

	var ddl string
	switch cfg.Materialization.Type {
	case "materialized_view":
		ddl = fmt.Sprintf(`CREATE MATERIALIZED VIEW %s
REFRESH ASYNC
AS
SELECT
    %s
FROM (
    %s
) t
%s
%s;`,
			targetQualified,
			strings.Join(selectCols, ",\n    "),
			boSQL,
			whereClause,
			groupByClause,
		)
	case "table":
		ddl = fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s AS
SELECT
    %s
FROM (
    %s
) t
%s
%s;`,
			targetQualified,
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

// compileCalcTermToSQL looks up a calculated term's catalog_node.config for
// a "rule_ast" — a JSON-encoded vm.Expression — and, if present, compiles it
// to a SQL expression via vm.CompileToSQL, resolving FieldRef leaves through
// resolveColumn (the same BO-scoped MAPS_TO lookup used for dimensions).
// Terms with no rule_ast (still the common case - see the comment above the
// calculation loop in GenerateDDL) return an error, which the caller turns
// into an explicit NULL placeholder rather than fabricated SQL.
func (s *PreAggregationService) compileCalcTermToSQL(ctx context.Context, calcName string, resolveColumn func(string) (string, error)) (string, error) {
	var configRaw json.RawMessage
	err := s.db.GetContext(ctx, &configRaw, `
		SELECT COALESCE(config, '{}'::jsonb)
		FROM catalog_node
		WHERE node_name = $1 AND properties->>'term_type' = 'calculated'
		LIMIT 1
	`, calcName)
	if err != nil {
		return "", fmt.Errorf("calculated term %q not found: %w", calcName, err)
	}

	var config struct {
		RuleAST json.RawMessage `json:"rule_ast"`
	}
	if err := json.Unmarshal(configRaw, &config); err != nil {
		return "", fmt.Errorf("parsing config for %q: %w", calcName, err)
	}
	if len(config.RuleAST) == 0 {
		return "", fmt.Errorf("no rule_ast wired for this calculated term (only formula/expression text stored)")
	}

	var expr vm.Expression
	if err := json.Unmarshal(config.RuleAST, &expr); err != nil {
		return "", fmt.Errorf("parsing rule_ast for %q: %w", calcName, err)
	}

	return vm.CompileToSQL(&expr, resolveColumn)
}

// sanitizeSQLComment strips characters that would break out of a `/* ... */`
// SQL comment (or otherwise be surprising inside one) from an error message
// before it's embedded in generated DDL.
func sanitizeSQLComment(s string) string {
	s = strings.ReplaceAll(s, "*/", "* /")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}

// ApplyMaterialization executes the DDL against StarRocks.
func (s *PreAggregationService) ApplyMaterialization(ctx context.Context, preAggID uuid.UUID) error {
	ddl, err := s.GenerateDDL(ctx, preAggID, "starrocks")
	if err != nil {
		return err
	}

	if s.starrocksDB == nil {
		return fmt.Errorf("starrocks connection is not available (check STARROCKS_HOST/PORT/USER/PASSWORD)")
	}

	if _, err := s.ensureTargetDatabase(ctx, preAggID); err != nil {
		return err
	}

	if _, err := s.starrocksDB.ExecContext(ctx, ddl); err != nil {
		return fmt.Errorf("failed to apply materialization DDL: %w", err)
	}

	return nil
}

// ensureTargetDatabase creates the pre-aggregation's target database
// (e.g. "tenant_<id>") in StarRocks if it doesn't already exist, since
// CREATE MATERIALIZED VIEW/TABLE fails against a missing database.
func (s *PreAggregationService) ensureTargetDatabase(ctx context.Context, preAggID uuid.UUID) (string, error) {
	var node struct {
		Properties json.RawMessage `db:"properties"`
	}
	if err := s.db.GetContext(ctx, &node, `SELECT properties FROM catalog_node WHERE id = $1`, preAggID); err != nil {
		return "", fmt.Errorf("pre-aggregation node not found: %w", err)
	}
	props, err := models.ParsePreAggProperties(node.Properties)
	if err != nil {
		return "", err
	}
	if _, err := s.starrocksDB.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", quoteIdent(props.TargetDatabase))); err != nil {
		return "", fmt.Errorf("failed to ensure target database %q: %w", props.TargetDatabase, err)
	}
	return props.TargetDatabase, nil
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

	props, err := models.ParsePreAggProperties(node.Properties)
	if err != nil {
		return err
	}
	cfg, err := models.ParsePreAggConfig(node.Config)
	if err != nil {
		return err
	}

	if s.starrocksDB == nil {
		return fmt.Errorf("starrocks connection is not available (check STARROCKS_HOST/PORT/USER/PASSWORD)")
	}

	if cfg.Materialization.Type != "materialized_view" {
		// Plain tables have no REFRESH concept; ApplyMaterialization would
		// need to be re-run (CREATE TABLE ... AS SELECT) to pick up new data.
		return fmt.Errorf("refresh is only supported for materialized_view targets, got %q", cfg.Materialization.Type)
	}

	refreshSQL := fmt.Sprintf("REFRESH MATERIALIZED VIEW %s.%s;", quoteIdent(props.TargetDatabase), quoteIdent(cfg.Materialization.TargetName))
	if _, err := s.starrocksDB.ExecContext(ctx, refreshSQL); err != nil {
		return fmt.Errorf("failed to refresh materialized view: %w", err)
	}

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
