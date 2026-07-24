package analytics

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// MetricsMigrationService handles migration of metrics_registry → CalculationTerm
type MetricsMigrationService struct {
	db              *sqlx.DB
	dslConverter    *DslConverter
	depExtractor    *DependencyExtractor
	calcNodeTypeID  uuid.UUID
	termNodeTypeID  uuid.UUID
	tableNodeTypeID uuid.UUID
}

// NewMetricsMigrationService creates a new migration service
func NewMetricsMigrationService(db *sqlx.DB) *MetricsMigrationService {
	return &MetricsMigrationService{
		db:           db,
		dslConverter: NewDslConverter(),
		depExtractor: NewDependencyExtractor(),
	}
}

// MetricsRegistryRow represents a row from fixed_income.metrics_registry
type MetricsRegistryRow struct {
	NodeID           string          `db:"node_id"`
	Category         string          `db:"category"`
	Description      sql.NullString  `db:"description"`
	FormulaType      string          `db:"formula_type"`
	Formula          string          `db:"formula"`
	Arguments        json.RawMessage `db:"arguments"`
	Badge            sql.NullString  `db:"badge"`
	FunctionClass    sql.NullString  `db:"function_class"`
	FunctionsUsed    sql.NullString  `db:"functions_used"`
	GovernanceStatus sql.NullString  `db:"governance_status"`
	Audience         sql.NullString  `db:"audience"`
	Tags             sql.NullString  `db:"tags"`
	CreatedAt        sql.NullTime    `db:"created_at"`
	UpdatedAt        sql.NullTime    `db:"updated_at"`
}

// MigrationResult contains the result of the migration
type MigrationResult struct {
	Total         int                `json:"total"`
	Migrated      int                `json:"migrated"`
	Failed        int                `json:"failed"`
	FailedMetrics []MigrationFailure `json:"failed_metrics"`
	EdgesCrated   int                `json:"edges_created"`
}

// MigrationFailure records a failed migration
type MigrationFailure struct {
	NodeID string `json:"node_id"`
	Error  string `json:"error"`
}

// Initialize looks up required node type IDs
func (s *MetricsMigrationService) Initialize(tenantID uuid.UUID) error {
	// Look up or create calculation_term node type
	var calcTypeID uuid.UUID
	err := s.db.Get(&calcTypeID, `
		SELECT id FROM catalog_node_type WHERE node_type = 'calculation_term' LIMIT 1
	`)
	if err == sql.ErrNoRows {
		// Create the node type
		calcTypeID = uuid.New()
		_, err = s.db.Exec(`
			INSERT INTO catalog_node_type (id, node_type, display_name, description)
			VALUES ($1, 'calculation_term', 'Calculation Term', 'A governed calculation metric with DSL expression and dependencies')
		`, calcTypeID)
		if err != nil {
			return fmt.Errorf("failed to create calculation_term node type: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to lookup calculation_term node type: %w", err)
	}
	s.calcNodeTypeID = calcTypeID

	// Look up semantic_term node type
	err = s.db.Get(&s.termNodeTypeID, `
		SELECT id FROM catalog_node_type WHERE node_type = 'semantic_term' LIMIT 1
	`)
	if err != nil {
		return fmt.Errorf("failed to lookup semantic_term node type: %w", err)
	}

	// Look up table node type
	err = s.db.Get(&s.tableNodeTypeID, `
		SELECT id FROM catalog_node_type WHERE node_type IN ('table', 'physical_table') LIMIT 1
	`)
	if err != nil {
		// Non-fatal - table dependencies may not exist
		s.tableNodeTypeID = uuid.Nil
	}

	return nil
}

// Migrate performs the full migration
func (s *MetricsMigrationService) Migrate(
	tenantID uuid.UUID,
	datasourceID uuid.UUID,
	dryRun bool,
) (*MigrationResult, error) {
	// Initialize node type IDs
	if err := s.Initialize(tenantID); err != nil {
		return nil, err
	}

	result := &MigrationResult{
		FailedMetrics: []MigrationFailure{},
	}

	// Read all metrics from registry
	rows, err := s.fetchMetrics()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch metrics: %w", err)
	}

	result.Total = len(rows)

	// Process each metric
	for _, row := range rows {
		err := s.migrateMetric(row, tenantID, datasourceID, dryRun, result)
		if err != nil {
			result.Failed++
			result.FailedMetrics = append(result.FailedMetrics, MigrationFailure{
				NodeID: row.NodeID,
				Error:  err.Error(),
			})
		} else {
			result.Migrated++
		}
	}

	return result, nil
}

// fetchMetrics reads all metrics from the registry
func (s *MetricsMigrationService) fetchMetrics() ([]MetricsRegistryRow, error) {
	var rows []MetricsRegistryRow

	query := `
		SELECT
			node_id,
			category,
			description,
			formula_type,
			formula,
			arguments,
			badge,
			function_class,
			functions_used,
			governance_status,
			audience,
			tags,
			created_at,
			updated_at
		FROM fixed_income.metrics_registry
	`

	err := s.db.Select(&rows, query)
	if err != nil {
		return nil, err
	}

	return rows, nil
}

// migrateMetric migrates a single metric
func (s *MetricsMigrationService) migrateMetric(
	row MetricsRegistryRow,
	tenantID uuid.UUID,
	datasourceID uuid.UUID,
	dryRun bool,
	result *MigrationResult,
) error {
	// 1. Convert formula to DSL
	convResult := s.dslConverter.Convert(row.Formula, row.FormulaType)
	if !convResult.Success {
		return fmt.Errorf("DSL conversion failed: %s", convResult.Error)
	}

	// 2. Extract dependencies
	deps, err := s.depExtractor.ExtractDependencies(row.Arguments)
	if err != nil {
		return fmt.Errorf("dependency extraction failed: %w", err)
	}

	// 3. Build properties JSON
	properties := map[string]interface{}{
		"category":          row.Category,
		"formula_type":      row.FormulaType,
		"governance_status": s.nullStringValue(row.GovernanceStatus, "draft"),
	}

	if row.FunctionClass.Valid {
		properties["function_class"] = row.FunctionClass.String
	}

	if row.Badge.Valid {
		properties["badge"] = row.Badge.String
	}

	// Parse arrays
	if row.FunctionsUsed.Valid {
		properties["functions_used"] = s.parseJsonArray(row.FunctionsUsed.String)
	} else {
		properties["functions_used"] = convResult.FunctionsUsed
	}

	if row.Tags.Valid {
		properties["tags"] = s.parseJsonArray(row.Tags.String)
	}

	if row.Audience.Valid {
		properties["audience"] = s.parseJsonArray(row.Audience.String)
	}

	propertiesJSON, _ := json.Marshal(properties)

	// 4. Build config JSON
	config := map[string]interface{}{
		"expression_dsl": convResult.DSL,
		"dependencies":   deps,
	}
	configJSON, _ := json.Marshal(config)

	// 5. Build qualified path
	qualifiedPath := fmt.Sprintf("calculation_term/%s", row.NodeID)

	// 6. Insert catalog_node (if not dry run)
	if dryRun {
		return nil
	}

	nodeID := uuid.New()
	now := time.Now()

	_, err = s.db.Exec(`
		INSERT INTO catalog_node (
			id, node_type_id, node_name, description, qualified_path,
			properties, config, tenant_id, tenant_datasource_id,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
		ON CONFLICT (tenant_datasource_id, node_type_id, qualified_path)
		DO UPDATE SET
			description = EXCLUDED.description,
			properties = EXCLUDED.properties,
			config = EXCLUDED.config,
			updated_at = EXCLUDED.updated_at
	`,
		nodeID,
		s.calcNodeTypeID,
		row.NodeID,
		s.nullStringValue(row.Description, ""),
		qualifiedPath,
		propertiesJSON,
		configJSON,
		tenantID,
		datasourceID,
		now,
		now,
	)

	if err != nil {
		return fmt.Errorf("failed to insert catalog_node: %w", err)
	}

	// 7. Create dependency edges
	for _, dep := range deps {
		edgeErr := s.createDependencyEdge(nodeID, dep, tenantID, datasourceID)
		if edgeErr == nil {
			result.EdgesCrated++
		}
		// Non-fatal if edge creation fails (target may not exist)
	}

	return nil
}

// createDependencyEdge creates a dependency edge
func (s *MetricsMigrationService) createDependencyEdge(
	sourceNodeID uuid.UUID,
	dep Dependency,
	tenantID uuid.UUID,
	datasourceID uuid.UUID,
) error {
	// Determine edge type
	var edgeType string
	var targetNodeTypeID uuid.UUID

	switch dep.Type {
	case "term":
		edgeType = "CALC_USES_TERM"
		targetNodeTypeID = s.termNodeTypeID
	case "calc":
		edgeType = "CALC_USES_CALC"
		targetNodeTypeID = s.calcNodeTypeID
	case "table":
		edgeType = "CALC_USES_TABLE"
		targetNodeTypeID = s.tableNodeTypeID
	default:
		return fmt.Errorf("unknown dependency type: %s", dep.Type)
	}

	// Find target node
	var targetNodeID uuid.UUID
	err := s.db.Get(&targetNodeID, `
		SELECT id FROM catalog_node
		WHERE node_name = $1
		AND node_type_id = $2
		AND tenant_datasource_id = $3
		LIMIT 1
	`, dep.Ref, targetNodeTypeID, datasourceID)

	if err != nil {
		return fmt.Errorf("target node not found: %s", dep.Ref)
	}

	// Create edge
	_, err = s.db.Exec(`
		INSERT INTO catalog_edge (
			id, source_node_id, target_node_id, edge_type_name,
			tenant_id, tenant_datasource_id, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		)
		ON CONFLICT DO NOTHING
	`,
		uuid.New(),
		sourceNodeID,
		targetNodeID,
		edgeType,
		tenantID,
		datasourceID,
		time.Now(),
	)

	return err
}

// Helper functions
func (s *MetricsMigrationService) nullStringValue(ns sql.NullString, defaultVal string) string {
	if ns.Valid {
		return ns.String
	}
	return defaultVal
}

func (s *MetricsMigrationService) parseJsonArray(str string) []string {
	// Try JSON array first
	var arr []string
	if err := json.Unmarshal([]byte(str), &arr); err == nil {
		return arr
	}

	// Fall back to comma-separated
	if strings.Contains(str, ",") {
		parts := strings.Split(str, ",")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		return parts
	}

	// Single value
	if str != "" {
		return []string{str}
	}

	return []string{}
}

// AssignMetricToBO creates a BO_HAS_CALC edge between a BO and a CalculationTerm
func (s *MetricsMigrationService) AssignMetricToBO(
	boName string,
	calcName string,
	tenantID uuid.UUID,
	datasourceID uuid.UUID,
) error {
	// Find BO node
	var boNodeID uuid.UUID
	err := s.db.Get(&boNodeID, `
		SELECT id FROM catalog_node
		WHERE node_name = $1
		AND node_type_id = (SELECT id FROM catalog_node_type WHERE node_type = 'business_object' LIMIT 1)
		AND tenant_datasource_id = $2
		LIMIT 1
	`, boName, datasourceID)
	if err != nil {
		return fmt.Errorf("BO not found: %s", boName)
	}

	// Find calculation node
	var calcNodeID uuid.UUID
	err = s.db.Get(&calcNodeID, `
		SELECT id FROM catalog_node
		WHERE node_name = $1
		AND node_type_id = $2
		AND tenant_datasource_id = $3
		LIMIT 1
	`, calcName, s.calcNodeTypeID, datasourceID)
	if err != nil {
		return fmt.Errorf("calculation not found: %s", calcName)
	}

	// Create BO_HAS_CALC edge
	_, err = s.db.Exec(`
		INSERT INTO catalog_edge (
			id, source_node_id, target_node_id, edge_type_name,
			tenant_id, tenant_datasource_id, created_at
		) VALUES (
			$1, $2, $3, 'BO_HAS_CALC', $4, $5, $6
		)
		ON CONFLICT DO NOTHING
	`,
		uuid.New(),
		boNodeID,
		calcNodeID,
		tenantID,
		datasourceID,
		time.Now(),
	)

	return err
}

// BulkAssignMetricsToBO assigns multiple calculations to a BO
func (s *MetricsMigrationService) BulkAssignMetricsToBO(
	boName string,
	calcNames []string,
	tenantID uuid.UUID,
	datasourceID uuid.UUID,
) (int, int) {
	success, failed := 0, 0
	for _, calcName := range calcNames {
		if err := s.AssignMetricToBO(boName, calcName, tenantID, datasourceID); err != nil {
			failed++
		} else {
			success++
		}
	}
	return success, failed
}

// GetAvailableCalculations returns all calculation terms for a datasource
func (s *MetricsMigrationService) GetAvailableCalculations(datasourceID uuid.UUID) ([]string, error) {
	var calculations []string
	err := s.db.Select(&calculations, `
		SELECT node_name FROM catalog_node
		WHERE node_type_id = $1
		AND tenant_datasource_id = $2
		ORDER BY node_name
	`, s.calcNodeTypeID, datasourceID)
	return calculations, err
}

// GetBOCalculations returns calculations assigned to a BO
func (s *MetricsMigrationService) GetBOCalculations(boName string, datasourceID uuid.UUID) ([]string, error) {
	var calculations []string
	err := s.db.Select(&calculations, `
		SELECT cn.node_name
		FROM catalog_edge ce
		JOIN catalog_node bo ON bo.id = ce.source_node_id
		JOIN catalog_node cn ON cn.id = ce.target_node_id
		WHERE bo.node_name = $1
		AND ce.edge_type_name = 'BO_HAS_CALC'
		AND ce.tenant_datasource_id = $2
		ORDER BY cn.node_name
	`, boName, datasourceID)
	return calculations, err
}
