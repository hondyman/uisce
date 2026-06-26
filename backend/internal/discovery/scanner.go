package discovery

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/hondyman/semlayer/backend/internal/models"
)

// SchemaScannerConfig holds configuration for schema scanning
type SchemaScannerConfig struct {
	PostgresDBs []string
	TrinoDBs    []string
	S3Buckets   []string
}

// FieldMetadata represents a field discovered in schema
type FieldMetadata struct {
	DatabaseType        string // "postgres", "trino", "s3"
	DatabaseName        string
	TableName           string
	FieldName           string
	FieldType           string
	IsNullable          bool
	CardinalityEstimate int64
	SampleValues        []interface{} // First 5 non-null values
	LastScannedAt       time.Time
	Frequency           int // Number of records with non-null value
}

// SchemaScanner discovers fields from databases and data stores
type SchemaScanner struct {
	postgresConn *sql.DB
	trinoConn    *sql.DB
	config       SchemaScannerConfig
	logger       *log.Logger
}

// NewSchemaScanner creates a new schema scanner
func NewSchemaScanner(postgresConn *sql.DB, trinoConn *sql.DB, cfg SchemaScannerConfig, logger *log.Logger) *SchemaScanner {
	return &SchemaScanner{
		postgresConn: postgresConn,
		trinoConn:    trinoConn,
		config:       cfg,
		logger:       logger,
	}
}

// ScanPostgresSchemas discovers all fields in specified Postgres databases
func (ss *SchemaScanner) ScanPostgresSchemas(ctx context.Context) ([]FieldMetadata, error) {
	var fields []FieldMetadata

	query := `
	SELECT 
		table_schema,
		table_name,
		column_name,
		data_type,
		is_nullable,
		(SELECT COUNT(DISTINCT "` + "`" + `" || column_name || "` + "`" + `")
		 FROM information_schema.tables t2
		 WHERE t2.table_schema = t.table_schema
		 AND t2.table_name = t.table_name
		 LIMIT 1000) as cardinality
	FROM information_schema.columns t
	WHERE table_schema NOT IN ('pg_catalog', 'information_schema')
	AND table_type = 'BASE TABLE'
	ORDER BY table_schema, table_name, column_name
	`

	rows, err := ss.postgresConn.QueryContext(ctx, query)
	if err != nil {
		ss.logger.Printf("Error querying Postgres schema: %v", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			schema      string
			tableName   string
			columnName  string
			dataType    string
			isNullable  string
			cardinality sql.NullInt64
		)

		if err := rows.Scan(&schema, &tableName, &columnName, &dataType, &isNullable, &cardinality); err != nil {
			ss.logger.Printf("Error scanning row: %v", err)
			continue
		}

		// Filter out obvious id/timestamp fields
		if shouldSkipField(columnName, dataType) {
			continue
		}

		fm := FieldMetadata{
			DatabaseType:        "postgres",
			DatabaseName:        schema,
			TableName:           tableName,
			FieldName:           columnName,
			FieldType:           dataType,
			IsNullable:          isNullable == "YES",
			CardinalityEstimate: cardinality.Int64,
			LastScannedAt:       time.Now(),
		}

		// Sample values for this field
		fm.SampleValues, fm.Frequency = ss.samplePostgresField(ctx, schema, tableName, columnName, dataType)

		fields = append(fields, fm)
	}

	return fields, rows.Err()
}

// ScanTrinoSchemas discovers all fields in Trino data warehouses
func (ss *SchemaScanner) ScanTrinoSchemas(ctx context.Context) ([]FieldMetadata, error) {
	var fields []FieldMetadata

	query := `
	SELECT 
		table_schema,
		table_name,
		column_name,
		data_type
	FROM information_schema.columns
	WHERE table_schema NOT IN ('information_schema', 'sys')
	ORDER BY table_schema, table_name, column_name
	`

	rows, err := ss.trinoConn.QueryContext(ctx, query)
	if err != nil {
		ss.logger.Printf("Error querying Trino schema: %v", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			schema     string
			tableName  string
			columnName string
			dataType   string
		)

		if err := rows.Scan(&schema, &tableName, &columnName, &dataType); err != nil {
			ss.logger.Printf("Error scanning Trino row: %v", err)
			continue
		}

		if shouldSkipField(columnName, dataType) {
			continue
		}

		fm := FieldMetadata{
			DatabaseType:        "trino",
			DatabaseName:        schema,
			TableName:           tableName,
			FieldName:           columnName,
			FieldType:           dataType,
			CardinalityEstimate: -1, // Unknown for Trino
			LastScannedAt:       time.Now(),
		}

		fields = append(fields, fm)
	}

	return fields, rows.Err()
}

// samplePostgresField gets sample values and frequency for a field
func (ss *SchemaScanner) samplePostgresField(ctx context.Context, schema, table, column, dataType string) ([]interface{}, int) {
	samples := []interface{}{}
	frequency := 0

	query := fmt.Sprintf(`
	SELECT %s, COUNT(*)
	FROM %s.%s
	WHERE %s IS NOT NULL
	GROUP BY %s
	ORDER BY COUNT(*) DESC
	LIMIT 5
	`, column, schema, table, column, column)

	rows, err := ss.postgresConn.QueryContext(ctx, query)
	if err != nil {
		return samples, frequency
	}
	defer rows.Close()

	for rows.Next() {
		var val interface{}
		var count int
		if err := rows.Scan(&val, &count); err != nil {
			continue
		}
		samples = append(samples, val)
		frequency += count
	}

	return samples, frequency
}

// shouldSkipField returns true if field should be excluded from feature discovery
func shouldSkipField(fieldName, dataType string) bool {
	// Skip IDs, timestamps, technical fields
	skipPatterns := []string{
		"id$", "_id$", "pk$",
		"created_at$", "updated_at$", "deleted_at$", "timestamp$",
		"_at$", "_time$",
		"uuid$", "guid$",
		"_hash$", "_md5$", "_sha1$",
		"^system_", "^_",
		"password", "secret", "token", "api_key",
		"row_number", "rank", "dense_rank",
	}

	lowerName := strings.ToLower(fieldName)
	for _, pattern := range skipPatterns {
		if matched, _ := regexp.MatchString(pattern, lowerName); matched {
			return true
		}
	}

	// Skip binary/blob types
	skipTypes := []string{"bytea", "blob", "binary", "varbinary"}
	lowerType := strings.ToLower(dataType)
	for _, t := range skipTypes {
		if strings.Contains(lowerType, t) {
			return true
		}
	}

	return false
}

// GetFieldStats returns aggregated statistics for a discovered field
func (ss *SchemaScanner) GetFieldStats(ctx context.Context, fm FieldMetadata) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	switch fm.DatabaseType {
	case "postgres":
		statsQuery := fmt.Sprintf(`
		SELECT 
			COUNT(*) as total_rows,
			COUNT(CASE WHEN %s IS NOT NULL THEN 1 END) as non_null_count,
			COUNT(DISTINCT %s) as distinct_count
		FROM %s.%s
		`, fm.FieldName, fm.FieldName, fm.DatabaseName, fm.TableName)

		row := ss.postgresConn.QueryRowContext(ctx, statsQuery)
		var totalRows, nonNullCount, distinctCount int64
		if err := row.Scan(&totalRows, &nonNullCount, &distinctCount); err == nil {
			stats["total_rows"] = totalRows
			stats["non_null_count"] = nonNullCount
			stats["distinct_count"] = distinctCount
			stats["completeness"] = float64(nonNullCount) / float64(totalRows)
			stats["uniqueness"] = float64(distinctCount) / float64(nonNullCount)
		}
	}

	return stats, nil
}

// ExportDiscoveredFields exports fields to feature catalog
func (ss *SchemaScanner) ExportDiscoveredFields(ctx context.Context, fields []FieldMetadata) ([]models.FeatureCandidate, error) {
	candidates := make([]models.FeatureCandidate, len(fields))

	for i, fm := range fields {
		stats, _ := ss.GetFieldStats(ctx, fm)

		candidates[i] = models.FeatureCandidate{
			Name:           fmt.Sprintf("%s_%s_%s", fm.DatabaseName, fm.TableName, fm.FieldName),
			SourceDatabase: fm.DatabaseType,
			SourceSchema:   fm.DatabaseName,
			SourceTable:    fm.TableName,
			SourceField:    fm.FieldName,
			DataType:       fm.FieldType,
			Completeness:   stats["completeness"].(float64),
			Cardinality:    fm.CardinalityEstimate,
			BusinessValue:  0, // To be scored later
			TechnicalScore: 0, // To be scored later
			DiscoveredAt:   fm.LastScannedAt,
			Status:         "candidate",
		}
	}

	return candidates, nil
}
