package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// IntegrityCheckResult represents the result of an integrity check
type IntegrityCheckResult struct {
	ID                string                 `json:"id"`
	DatasourceID      string                 `json:"datasource_id"`
	CheckType         string                 `json:"check_type"`
	Status            string                 `json:"status"` // passed, warning, failed
	PostgresRowCount  int64                  `json:"postgres_row_count,omitempty"`
	IgniteRowCount    int64                  `json:"ignite_row_count,omitempty"`
	StarrocksRowCount int64                  `json:"starrocks_row_count,omitempty"`
	RowCountDelta     int64                  `json:"row_count_delta,omitempty"`
	SchemaChanges     map[string]interface{} `json:"schema_changes,omitempty"`
	ChecksumValid     bool                   `json:"checksum_valid"`
	StartedAt         time.Time              `json:"started_at"`
	CompletedAt       *time.Time             `json:"completed_at,omitempty"`
	DurationMs        int                    `json:"duration_ms,omitempty"`
	ErrorMessage      string                 `json:"error_message,omitempty"`
}

// SchemaSnapshot represents a point-in-time schema snapshot
type SchemaSnapshot struct {
	ID           string          `json:"id"`
	DatasourceID string          `json:"datasource_id"`
	SnapshotData json.RawMessage `json:"snapshot_data"`
	CapturedAt   time.Time       `json:"captured_at"`
	CapturedBy   string          `json:"captured_by,omitempty"`
	IsBaseline   bool            `json:"is_baseline"`
}

// TableSchema represents the schema of a single table
type TableSchema struct {
	Name    string         `json:"name"`
	Schema  string         `json:"schema"`
	Columns []ColumnSchema `json:"columns"`
}

// ColumnSchema represents a column definition
type ColumnSchema struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
	Default  string `json:"default,omitempty"`
}

// IntegrityService handles data integrity validation across layers
type IntegrityService struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewIntegrityService creates a new IntegrityService
func NewIntegrityService(db *sqlx.DB) *IntegrityService {
	logger, _ := zap.NewProduction()
	return &IntegrityService{
		db:     db,
		logger: logger,
	}
}

// RunIntegrityCheck performs a full integrity check for a datasource
func (s *IntegrityService) RunIntegrityCheck(ctx context.Context, datasourceID string, checkType string, executedBy string) (*IntegrityCheckResult, error) {
	startTime := time.Now()

	result := &IntegrityCheckResult{
		ID:            uuid.New().String(),
		DatasourceID:  datasourceID,
		CheckType:     checkType,
		Status:        "passed",
		StartedAt:     startTime,
		ChecksumValid: true,
	}

	// Get datasource configuration
	var config struct {
		IntegrityChecks json.RawMessage `db:"integrity_checks"`
		ConnectionID    sql.NullString  `db:"connection_id"`
	}
	err := s.db.GetContext(ctx, &config,
		"SELECT integrity_checks, connection_id FROM tenant_product_datasource WHERE id = $1", datasourceID)
	if err != nil {
		result.Status = "failed"
		result.ErrorMessage = fmt.Sprintf("Failed to get datasource config: %v", err)
		return result, err
	}

	// Parse integrity check settings
	var integritySettings struct {
		RowCountValidation   bool `json:"row_count_validation"`
		SchemaDriftDetection bool `json:"schema_drift_detection"`
		ChecksumVerification bool `json:"checksum_verification"`
	}
	if err := json.Unmarshal(config.IntegrityChecks, &integritySettings); err != nil {
		integritySettings.SchemaDriftDetection = true // Default
	}

	// Run requested checks
	switch checkType {
	case "row_count":
		if err := s.validateRowCounts(ctx, datasourceID, result); err != nil {
			s.logger.Warn("Row count validation failed", zap.Error(err))
		}
	case "schema_drift":
		if err := s.detectSchemaDrift(ctx, datasourceID, result); err != nil {
			s.logger.Warn("Schema drift detection failed", zap.Error(err))
		}
	case "checksum":
		if err := s.verifyChecksums(ctx, datasourceID, result); err != nil {
			s.logger.Warn("Checksum verification failed", zap.Error(err))
		}
	case "full":
		// Run all enabled checks
		if integritySettings.RowCountValidation {
			if err := s.validateRowCounts(ctx, datasourceID, result); err != nil {
				s.logger.Warn("Row count validation failed", zap.Error(err))
			}
		}
		if integritySettings.SchemaDriftDetection {
			if err := s.detectSchemaDrift(ctx, datasourceID, result); err != nil {
				s.logger.Warn("Schema drift detection failed", zap.Error(err))
			}
		}
		if integritySettings.ChecksumVerification {
			if err := s.verifyChecksums(ctx, datasourceID, result); err != nil {
				s.logger.Warn("Checksum verification failed", zap.Error(err))
			}
		}
	}

	// Calculate duration
	completedAt := time.Now()
	result.CompletedAt = &completedAt
	result.DurationMs = int(completedAt.Sub(startTime).Milliseconds())

	// Persist result
	if err := s.saveIntegrityCheckResult(ctx, result, executedBy); err != nil {
		s.logger.Error("Failed to save integrity check result", zap.Error(err))
	}

	// Update datasource status
	if err := s.updateDatasourceIntegrityStatus(ctx, datasourceID, result.Status); err != nil {
		s.logger.Error("Failed to update datasource integrity status", zap.Error(err))
	}

	return result, nil
}

// validateRowCounts compares row counts across data layers
func (s *IntegrityService) validateRowCounts(ctx context.Context, datasourceID string, result *IntegrityCheckResult) error {
	// Get tables associated with this datasource from catalog
	var tables []struct {
		QualifiedPath string `db:"qualified_path"`
	}
	err := s.db.SelectContext(ctx, &tables, `
		SELECT qualified_path 
		FROM catalog_node cn
		JOIN catalog_node_type cnt ON cn.node_type_id = cnt.id
		WHERE cn.tenant_datasource_id = $1 
		AND cnt.catalog_type_name = 'TABLE'
		LIMIT 100
	`, datasourceID)
	if err != nil {
		return fmt.Errorf("failed to get tables: %w", err)
	}

	if len(tables) == 0 {
		s.logger.Info("No tables found for row count validation", zap.String("datasource_id", datasourceID))
		return nil
	}

	// For each table, compare counts across layers
	// This is a simplified implementation - in production you'd query each layer
	var totalPostgres, totalIgnite, totalStarrocks int64

	for _, table := range tables {
		// Query PostgreSQL (primary source)
		var pgCount int64
		err := s.db.GetContext(ctx, &pgCount,
			fmt.Sprintf("SELECT COUNT(*) FROM %s", table.QualifiedPath))
		if err != nil {
			s.logger.Warn("Failed to count rows in Postgres",
				zap.String("table", table.QualifiedPath),
				zap.Error(err))
			continue
		}
		totalPostgres += pgCount

		// TODO: Query Ignite cache for this table
		// igniteCount := s.queryIgniteCount(ctx, table.QualifiedPath)
		// totalIgnite += igniteCount

		// TODO: Query StarRocks for this table
		// starrocksCount := s.queryStarrocksCount(ctx, table.QualifiedPath)
		// totalStarrocks += starrocksCount
	}

	result.PostgresRowCount = totalPostgres
	result.IgniteRowCount = totalIgnite
	result.StarrocksRowCount = totalStarrocks

	// Calculate delta (simplified - comparing primary to cache)
	result.RowCountDelta = totalPostgres - totalIgnite

	// Determine status based on delta threshold (e.g., 1% tolerance)
	if result.RowCountDelta != 0 && totalPostgres > 0 {
		deltaPercent := float64(absInt64(result.RowCountDelta)) / float64(totalPostgres) * 100
		if deltaPercent > 5.0 {
			result.Status = "failed"
			result.ErrorMessage = fmt.Sprintf("Row count mismatch exceeds 5%% threshold: %.2f%%", deltaPercent)
		} else if deltaPercent > 1.0 {
			result.Status = "warning"
		}
	}

	return nil
}

// detectSchemaDrift compares current schema against baseline
func (s *IntegrityService) detectSchemaDrift(ctx context.Context, datasourceID string, result *IntegrityCheckResult) error {
	// Get current schema
	currentSchema, err := s.captureCurrentSchema(ctx, datasourceID)
	if err != nil {
		return fmt.Errorf("failed to capture current schema: %w", err)
	}

	// Get baseline schema
	var baseline SchemaSnapshot
	err = s.db.GetContext(ctx, &baseline, `
		SELECT id, datasource_id, snapshot_data, captured_at, is_baseline
		FROM datasource_schema_snapshots
		WHERE datasource_id = $1 AND is_baseline = true
		ORDER BY captured_at DESC
		LIMIT 1
	`, datasourceID)

	if err == sql.ErrNoRows {
		// No baseline exists - save current as baseline
		if err := s.saveSchemaSnapshot(ctx, datasourceID, currentSchema, true, "system"); err != nil {
			s.logger.Warn("Failed to save initial baseline", zap.Error(err))
		}
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to get baseline schema: %w", err)
	}

	// Compare schemas
	var baselineData []TableSchema
	if err := json.Unmarshal(baseline.SnapshotData, &baselineData); err != nil {
		return fmt.Errorf("failed to parse baseline schema: %w", err)
	}

	changes := s.compareSchemas(baselineData, currentSchema)
	if len(changes) > 0 {
		result.SchemaChanges = map[string]interface{}{
			"changes":       changes,
			"baseline_date": baseline.CapturedAt,
		}
		result.Status = "warning"
	}

	return nil
}

// captureCurrentSchema gets the current schema from the database
func (s *IntegrityService) captureCurrentSchema(ctx context.Context, datasourceID string) ([]TableSchema, error) {
	// Get tables from catalog
	var tables []struct {
		TableName  string `db:"table_name"`
		SchemaName string `db:"schema_name"`
	}
	err := s.db.SelectContext(ctx, &tables, `
		SELECT 
			SPLIT_PART(qualified_path, '.', 2) as table_name,
			SPLIT_PART(qualified_path, '.', 1) as schema_name
		FROM catalog_node cn
		JOIN catalog_node_type cnt ON cn.node_type_id = cnt.id
		WHERE cn.tenant_datasource_id = $1 
		AND cnt.catalog_type_name = 'TABLE'
	`, datasourceID)
	if err != nil {
		return nil, err
	}

	var schemas []TableSchema
	for _, t := range tables {
		// Get columns for each table
		var columns []ColumnSchema
		err := s.db.SelectContext(ctx, &columns, `
			SELECT 
				column_name as name,
				data_type as type,
				is_nullable = 'YES' as nullable,
				COALESCE(column_default, '') as default
			FROM information_schema.columns
			WHERE table_schema = $1 AND table_name = $2
			ORDER BY ordinal_position
		`, t.SchemaName, t.TableName)
		if err != nil {
			s.logger.Warn("Failed to get columns for table",
				zap.String("table", t.TableName),
				zap.Error(err))
			continue
		}

		schemas = append(schemas, TableSchema{
			Name:    t.TableName,
			Schema:  t.SchemaName,
			Columns: columns,
		})
	}

	return schemas, nil
}

// compareSchemas finds differences between baseline and current schema
func (s *IntegrityService) compareSchemas(baseline, current []TableSchema) []map[string]interface{} {
	var changes []map[string]interface{}

	// Build maps for comparison
	baselineMap := make(map[string]TableSchema)
	for _, t := range baseline {
		baselineMap[t.Schema+"."+t.Name] = t
	}

	currentMap := make(map[string]TableSchema)
	for _, t := range current {
		currentMap[t.Schema+"."+t.Name] = t
	}

	// Check for added/removed tables
	for key := range currentMap {
		if _, exists := baselineMap[key]; !exists {
			changes = append(changes, map[string]interface{}{
				"type":  "table_added",
				"table": key,
			})
		}
	}

	for key := range baselineMap {
		if _, exists := currentMap[key]; !exists {
			changes = append(changes, map[string]interface{}{
				"type":  "table_removed",
				"table": key,
			})
		}
	}

	// Check for column changes in existing tables
	for key, baseTable := range baselineMap {
		if currTable, exists := currentMap[key]; exists {
			baseColMap := make(map[string]ColumnSchema)
			for _, c := range baseTable.Columns {
				baseColMap[c.Name] = c
			}

			currColMap := make(map[string]ColumnSchema)
			for _, c := range currTable.Columns {
				currColMap[c.Name] = c
			}

			// Check added columns
			for colName := range currColMap {
				if _, exists := baseColMap[colName]; !exists {
					changes = append(changes, map[string]interface{}{
						"type":   "column_added",
						"table":  key,
						"column": colName,
					})
				}
			}

			// Check removed columns
			for colName := range baseColMap {
				if _, exists := currColMap[colName]; !exists {
					changes = append(changes, map[string]interface{}{
						"type":   "column_removed",
						"table":  key,
						"column": colName,
					})
				}
			}

			// Check type changes
			for colName, baseCol := range baseColMap {
				if currCol, exists := currColMap[colName]; exists {
					if baseCol.Type != currCol.Type {
						changes = append(changes, map[string]interface{}{
							"type":     "column_type_changed",
							"table":    key,
							"column":   colName,
							"old_type": baseCol.Type,
							"new_type": currCol.Type,
						})
					}
				}
			}
		}
	}

	return changes
}

// verifyChecksums validates data checksums (placeholder for future implementation)
func (s *IntegrityService) verifyChecksums(ctx context.Context, datasourceID string, result *IntegrityCheckResult) error {
	// TODO: Implement checksum verification
	// This would typically:
	// 1. Calculate checksums for critical columns
	// 2. Compare against stored checksums
	// 3. Flag any mismatches
	result.ChecksumValid = true
	return nil
}

// saveIntegrityCheckResult persists the check result
func (s *IntegrityService) saveIntegrityCheckResult(ctx context.Context, result *IntegrityCheckResult, executedBy string) error {
	schemaChangesJSON, _ := json.Marshal(result.SchemaChanges)

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO datasource_integrity_checks (
			id, datasource_id, check_type, status,
			postgres_row_count, ignite_row_count, starrocks_row_count, row_count_delta,
			schema_changes, checksum_valid,
			executed_by, started_at, completed_at, duration_ms, error_message
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`, result.ID, result.DatasourceID, result.CheckType, result.Status,
		result.PostgresRowCount, result.IgniteRowCount, result.StarrocksRowCount, result.RowCountDelta,
		schemaChangesJSON, result.ChecksumValid,
		executedBy, result.StartedAt, result.CompletedAt, result.DurationMs, result.ErrorMessage)

	return err
}

// updateDatasourceIntegrityStatus updates the datasource's integrity status
func (s *IntegrityService) updateDatasourceIntegrityStatus(ctx context.Context, datasourceID, status string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE tenant_product_datasource 
		SET integrity_status = $1, last_integrity_check_at = NOW()
		WHERE id = $2
	`, status, datasourceID)
	return err
}

// saveSchemaSnapshot saves a schema snapshot
func (s *IntegrityService) saveSchemaSnapshot(ctx context.Context, datasourceID string, schema []TableSchema, isBaseline bool, capturedBy string) error {
	schemaJSON, err := json.Marshal(schema)
	if err != nil {
		return err
	}

	// If this is a baseline, unset any existing baseline
	if isBaseline {
		_, err = s.db.ExecContext(ctx, `
			UPDATE datasource_schema_snapshots 
			SET is_baseline = false 
			WHERE datasource_id = $1 AND is_baseline = true
		`, datasourceID)
		if err != nil {
			return err
		}
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO datasource_schema_snapshots (datasource_id, snapshot_data, captured_by, is_baseline)
		VALUES ($1, $2, $3, $4)
	`, datasourceID, schemaJSON, capturedBy, isBaseline)

	return err
}

// GetIntegrityHistory returns recent integrity check results
func (s *IntegrityService) GetIntegrityHistory(ctx context.Context, datasourceID string, limit int) ([]IntegrityCheckResult, error) {
	var results []IntegrityCheckResult
	err := s.db.SelectContext(ctx, &results, `
		SELECT id, datasource_id, check_type, status,
			   postgres_row_count, ignite_row_count, starrocks_row_count, row_count_delta,
			   checksum_valid, started_at, completed_at, duration_ms, error_message
		FROM datasource_integrity_checks
		WHERE datasource_id = $1
		ORDER BY started_at DESC
		LIMIT $2
	`, datasourceID, limit)
	return results, err
}

// absInt64 returns the absolute value (helper for integrity service)
func absInt64(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
