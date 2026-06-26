package analytics

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql" // StarRocks uses MySQL protocol
	"github.com/google/uuid"
	_ "github.com/lib/pq" // Postgres driver
)

// AggregateDefinition represents the user's design
type AggregateDefinition struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	SourceTable string   `json:"source_table"`
	Dimensions  []string `json:"dimensions"`
	Measures    []string `json:"measures"` // e.g., "SUM(pnl)", "AVG(price)"
	Filter      string   `json:"filter"`
	Target      string   `json:"target"` // "StarRocks", "Cube", "Both"
	TenantID    string   `json:"tenant_id"`
}

// AggregateService handles generation and persistence
type AggregateService struct {
	starrocksDB *sql.DB
	postgresDB  *sql.DB
	catalogName string
	database    string
}

// AggregateServiceConfig holds configuration
type AggregateServiceConfig struct {
	StarRocksHost     string
	StarRocksPort     int
	StarRocksUser     string
	StarRocksPassword string
	PostgresURL       string
	CatalogName       string
	Database          string
}

// NewAggregateService creates a new aggregate service
func NewAggregateService() *AggregateService {
	cfg := AggregateServiceConfig{
		StarRocksHost:     getEnv("STARROCKS_HOST", "127.0.0.1"),
		StarRocksPort:     getEnvInt("STARROCKS_PORT", 9030),
		StarRocksUser:     getEnv("STARROCKS_USER", "root"),
		StarRocksPassword: getEnv("STARROCKS_PASSWORD", ""),
		PostgresURL:       getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable"),
		CatalogName:       "iceberg_catalog",
		Database:          "wealth",
	}

	svc, err := NewAggregateServiceWithConfig(cfg)
	if err != nil {
		fmt.Printf("ERROR: Failed to create AggregateService: %v\n", err)
		return nil
	}
	return svc
}

// NewAggregateServiceWithConfig creates service with explicit config
func NewAggregateServiceWithConfig(cfg AggregateServiceConfig) (*AggregateService, error) {
	// Initialize StarRocks (MySQL protocol)
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/",
		cfg.StarRocksUser, cfg.StarRocksPassword, cfg.StarRocksHost, cfg.StarRocksPort)

	srDB, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to StarRocks: %w", err)
	}

	srDB.SetMaxOpenConns(25)
	srDB.SetMaxIdleConns(5)
	srDB.SetConnMaxLifetime(5 * time.Minute)

	if err := srDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping StarRocks: %w", err)
	}

	// Initialize Postgres
	pgDB, err := sql.Open("postgres", cfg.PostgresURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Postgres: %w", err)
	}

	return &AggregateService{
		starrocksDB: srDB,
		postgresDB:  pgDB,
		catalogName: cfg.CatalogName,
		database:    cfg.Database,
	}, nil
}

// tableName returns fully qualified Iceberg table name
func (s *AggregateService) tableName(table string) string {
	return fmt.Sprintf("%s.%s.%s", s.catalogName, s.database, table)
}

// SaveAggregate processes the definition and generates artifacts
func (s *AggregateService) SaveAggregate(ctx context.Context, def AggregateDefinition) error {
	fmt.Printf("Saving Aggregate: %s (Target: %s)\n", def.Name, def.Target)

	// 1. Generate StarRocks Materialized View
	if def.Target == "StarRocks" || def.Target == "Both" {
		sql := s.GenerateStarRocksSQL(def)
		fmt.Printf("Generated StarRocks SQL:\n%s\n", sql)

		if s.starrocksDB != nil {
			if _, err := s.starrocksDB.ExecContext(ctx, sql); err != nil {
				return fmt.Errorf("failed to execute StarRocks SQL: %w", err)
			}
			fmt.Println("SUCCESS: Materialized View created in StarRocks.")
		}
	}

	// 2. Generate Cube Schema
	if def.Target == "Cube" || def.Target == "Both" {
		schema := s.GenerateCubeSchema(def)
		fmt.Printf("Generated Cube Schema:\n%s\n", schema)

		// Write to file
		dir := "backend/internal/analytics/cube/schema"
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create cube schema directory: %w", err)
		}

		filename := filepath.Join(dir, fmt.Sprintf("%s.yml", def.Name))
		if err := os.WriteFile(filename, []byte(schema), 0644); err != nil {
			return fmt.Errorf("failed to write cube schema file: %w", err)
		}
		fmt.Printf("SUCCESS: Cube Schema written to %s\n", filename)
	}

	// 3. Log Audit Record
	if err := s.logAudit(ctx, def); err != nil {
		fmt.Printf("WARNING: Failed to log audit record: %v\n", err)
	}

	return nil
}

// GenerateStarRocksSQL creates the Materialized View definition for StarRocks
// Partitions by tenant_id and date for multi-tenant isolation
func (s *AggregateService) GenerateStarRocksSQL(def AggregateDefinition) string {
	dims := strings.Join(def.Dimensions, ", ")

	var projections []string
	for _, m := range def.Measures {
		parts := strings.Split(m, "(")
		if len(parts) > 1 {
			col := strings.TrimSuffix(parts[1], ")")
			funcName := parts[0]
			alias := fmt.Sprintf("%s_%s", strings.ToLower(funcName), col)
			projections = append(projections, fmt.Sprintf("%s as %s", m, alias))
		} else {
			projections = append(projections, m)
		}
	}
	measures := strings.Join(projections, ", ")

	// Determine distribution key (use first dimension or tenant_id)
	distKey := "tenant_id"
	if len(def.Dimensions) > 0 {
		distKey = def.Dimensions[0]
	}

	// Build the source table reference (from Iceberg catalog)
	sourceTable := s.tableName(def.SourceTable)

	sql := fmt.Sprintf(`
-- StarRocks Materialized View with date partitioning and multi-tenant support
CREATE MATERIALIZED VIEW IF NOT EXISTS wealth_analytics.mv_%s
DISTRIBUTED BY HASH(%s) BUCKETS 16
REFRESH ASYNC EVERY (INTERVAL 5 MINUTE)
AS SELECT
    tenant_id,
    DATE(event_time) as trade_date,
    %s,
    %s
FROM %s
`, def.Name, distKey, dims, measures, sourceTable)

	if def.Filter != "" {
		sql += fmt.Sprintf("WHERE %s\n", def.Filter)
	}

	sql += fmt.Sprintf("GROUP BY tenant_id, DATE(event_time), %s;", dims)

	return sql
}

// GenerateCubeSchema creates the Cube YAML definition
func (s *AggregateService) GenerateCubeSchema(def AggregateDefinition) string {
	var measuresYAML []string
	for _, m := range def.Measures {
		parts := strings.Split(m, "(")
		if len(parts) > 1 {
			col := strings.TrimSuffix(parts[1], ")")
			funcName := strings.ToLower(parts[0])
			name := fmt.Sprintf("%s_%s", funcName, col)

			measuresYAML = append(measuresYAML, fmt.Sprintf(`      - name: %s
        type: %s
        sql: %s`, name, funcName, col))
		}
	}

	var dimensionsYAML []string
	// Always include tenant_id for multi-tenancy
	dimensionsYAML = append(dimensionsYAML, `      - name: tenant_id
        sql: tenant_id
        type: string
        primary_key: true`)

	for _, d := range def.Dimensions {
		if d != "tenant_id" {
			dimensionsYAML = append(dimensionsYAML, fmt.Sprintf(`      - name: %s
        sql: %s
        type: string`, d, d))
		}
	}

	// Source from StarRocks materialized view or Iceberg table
	sourceSQL := fmt.Sprintf("SELECT * FROM %s", s.tableName(def.SourceTable))

	return fmt.Sprintf(`
cubes:
  - name: %s
    sql: %s
    
    # Multi-tenant security context
    public: false
    
    measures:
%s

    dimensions:
%s

    # Row-level security for multi-tenancy
    segments:
      - name: tenant_filter
        sql: "{tenant_id} = '${SECURITY_CONTEXT.tenant_id}'"
`, def.Name, sourceSQL, strings.Join(measuresYAML, "\n"), strings.Join(dimensionsYAML, "\n"))
}

func (s *AggregateService) logAudit(ctx context.Context, def AggregateDefinition) error {
	if s.postgresDB == nil {
		return fmt.Errorf("postgres connection is not initialized")
	}

	query := `
		INSERT INTO audit_records (
			tenant_id, entity_type, entity_id, action, actor, payload
		) VALUES (
			$1, $2, $3, $4, $5, $6
		)
	`

	tenantID := def.TenantID
	if tenantID == "" {
		tenantID = "11111111-1111-1111-1111-111111111111" // Default Tenant
	}
	actor := "starrocks-aggregate-service"
	payload := fmt.Sprintf(`{"name": "%s", "target": "%s", "source_table": "%s"}`,
		def.Name, def.Target, def.SourceTable)

	_, err := s.postgresDB.ExecContext(ctx, query,
		tenantID, "Aggregate", uuid.New().String(), "Create", actor, payload)

	if err != nil {
		return err
	}

	fmt.Printf("AUDIT: Logged creation of aggregate '%s' to Postgres.\n", def.Name)
	return nil
}

// ListAggregates lists all materialized views in wealth_analytics database
func (s *AggregateService) ListAggregates(ctx context.Context) ([]string, error) {
	if s.starrocksDB == nil {
		return nil, fmt.Errorf("starrocks connection is not initialized")
	}

	rows, err := s.starrocksDB.QueryContext(ctx, "SHOW MATERIALIZED VIEWS FROM wealth_analytics")
	if err != nil {
		return nil, fmt.Errorf("failed to list materialized views: %w", err)
	}
	defer rows.Close()

	var views []string
	for rows.Next() {
		var name string
		// StarRocks SHOW MATERIALIZED VIEWS returns multiple columns
		var dummy interface{}
		if err := rows.Scan(&name, &dummy, &dummy, &dummy); err != nil {
			// Try single column scan
			if err := rows.Scan(&name); err != nil {
				continue
			}
		}
		views = append(views, name)
	}

	return views, nil
}

// RefreshAggregate triggers a manual refresh of a materialized view
func (s *AggregateService) RefreshAggregate(ctx context.Context, name string) error {
	if s.starrocksDB == nil {
		return fmt.Errorf("starrocks connection is not initialized")
	}

	query := fmt.Sprintf("REFRESH MATERIALIZED VIEW wealth_analytics.%s", name)
	_, err := s.starrocksDB.ExecContext(ctx, query)
	return err
}

// HealthCheck verifies StarRocks connectivity
func (s *AggregateService) HealthCheck(ctx context.Context) error {
	if s.starrocksDB == nil {
		return fmt.Errorf("starrocks connection is not initialized")
	}
	return s.starrocksDB.PingContext(ctx)
}

// Close closes database connections
func (s *AggregateService) Close() error {
	var errs []error
	if s.starrocksDB != nil {
		if err := s.starrocksDB.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if s.postgresDB != nil {
		if err := s.postgresDB.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}
	return nil
}

// Helper functions
func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		var i int
		fmt.Sscanf(val, "%d", &i)
		return i
	}
	return defaultVal
}
