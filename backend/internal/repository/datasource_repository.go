package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// DatasourceRepository provides direct SQL CRUD for datasources
type DatasourceRepository struct {
	db *sqlx.DB
}

// NewDatasourceRepository creates a new DatasourceRepository
func NewDatasourceRepository(db *sqlx.DB) *DatasourceRepository {
	return &DatasourceRepository{db: db}
}

// Datasource represents the datasource entity
type Datasource struct {
	ID                   string                 `json:"id" db:"id"`
	TenantProductID      string                 `json:"tenant_product_id" db:"tenant_product_id"`
	AlphaDatasourceID    string                 `json:"alpha_datasource_id" db:"alpha_datasource_id"`
	SourceName           string                 `json:"source_name" db:"source_name"`
	IsActive             bool                   `json:"is_active" db:"is_active"`
	Config               map[string]interface{} `json:"config,omitempty"`
	Environment          string                 `json:"environment" db:"environment"`
	Tags                 []string               `json:"tags"`
	Description          *string                `json:"description,omitempty" db:"description"`
	ReadOnly             bool                   `json:"read_only" db:"read_only"`
	PoolConfig           map[string]interface{} `json:"pool_config,omitempty"`
	ScanSchedule         map[string]interface{} `json:"scan_schedule,omitempty"`
	HealthConfig         map[string]interface{} `json:"health_config,omitempty"`
	IntegrityChecks      map[string]interface{} `json:"integrity_checks,omitempty"`
	SLAConfig            map[string]interface{} `json:"sla_config,omitempty"`
	DataClassification   map[string]interface{} `json:"data_classification,omitempty"`
	LastHeartbeatAt      *time.Time             `json:"last_heartbeat_at,omitempty" db:"last_heartbeat_at"`
	HealthStatus         string                 `json:"health_status" db:"health_status"`
	HealthMessage        *string                `json:"health_message,omitempty" db:"health_message"`
	LastIntegrityCheckAt *time.Time             `json:"last_integrity_check_at,omitempty" db:"last_integrity_check_at"`
	IntegrityStatus      string                 `json:"integrity_status" db:"integrity_status"`
	IntegrityMessage     *string                `json:"integrity_message,omitempty" db:"integrity_message"`
	LastScanAt           *time.Time             `json:"last_scan_at,omitempty" db:"last_scan_at"`
	LastScanStatus       string                 `json:"last_scan_status" db:"last_scan_status"`
	ConnectionID         *string                `json:"connection_id,omitempty" db:"connection_id"`
	CreatedAt            time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time              `json:"updated_at" db:"updated_at"`
	CreatedBy            *string                `json:"created_by,omitempty" db:"created_by"`
	UpdatedBy            *string                `json:"updated_by,omitempty" db:"updated_by"`
}

// IntegrityCheckResult represents an integrity check result
type IntegrityCheckResult struct {
	ID                   string                 `json:"id" db:"id"`
	DatasourceID         string                 `json:"datasource_id" db:"datasource_id"`
	CheckType            string                 `json:"check_type" db:"check_type"`
	Status               string                 `json:"status" db:"status"`
	PostgresRowCount     *int64                 `json:"postgres_row_count,omitempty" db:"postgres_row_count"`
	IgniteRowCount       *int64                 `json:"ignite_row_count,omitempty" db:"ignite_row_count"`
	StarrocksRowCount    *int64                 `json:"starrocks_row_count,omitempty" db:"starrocks_row_count"`
	RowCountDelta        *int64                 `json:"row_count_delta,omitempty" db:"row_count_delta"`
	RowCountDeltaPercent *float64               `json:"row_count_delta_percent,omitempty" db:"row_count_delta_percent"`
	SchemaChanges        map[string]interface{} `json:"schema_changes,omitempty"`
	ChecksumValid        *bool                  `json:"checksum_valid,omitempty" db:"checksum_valid"`
	ExecutedBy           *string                `json:"executed_by,omitempty" db:"executed_by"`
	StartedAt            time.Time              `json:"started_at" db:"started_at"`
	CompletedAt          *time.Time             `json:"completed_at,omitempty" db:"completed_at"`
	DurationMs           *int                   `json:"duration_ms,omitempty" db:"duration_ms"`
	ErrorMessage         *string                `json:"error_message,omitempty" db:"error_message"`
	Recommendations      map[string]interface{} `json:"recommendations,omitempty"`
}

// SchemaSnapshot represents a schema snapshot
type SchemaSnapshot struct {
	ID                 string                 `json:"id" db:"id"`
	DatasourceID       string                 `json:"datasource_id" db:"datasource_id"`
	SnapshotData       map[string]interface{} `json:"snapshot_data"`
	TableCount         *int                   `json:"table_count,omitempty" db:"table_count"`
	ColumnCount        *int                   `json:"column_count,omitempty" db:"column_count"`
	CapturedAt         time.Time              `json:"captured_at" db:"captured_at"`
	CapturedBy         *string                `json:"captured_by,omitempty" db:"captured_by"`
	IsBaseline         bool                   `json:"is_baseline" db:"is_baseline"`
	Notes              *string                `json:"notes,omitempty" db:"notes"`
	PreviousSnapshotID *string                `json:"previous_snapshot_id,omitempty" db:"previous_snapshot_id"`
	ChangeSummary      map[string]interface{} `json:"change_summary,omitempty"`
}

// GetByID fetches a datasource by ID
func (r *DatasourceRepository) GetByID(ctx context.Context, id string) (*Datasource, error) {
	var ds Datasource
	err := r.db.GetContext(ctx, &ds, `
		SELECT id, COALESCE(tenant_product_id::text,'') as tenant_product_id,
		       COALESCE(alpha_datasource_id::text,'') as alpha_datasource_id,
		       source_name, COALESCE(is_active, false) as is_active,
		       COALESCE(environment,'') as environment, description,
		       COALESCE(read_only, false) as read_only,
		       last_heartbeat_at, COALESCE(health_status,'unknown') as health_status, health_message,
		       last_integrity_check_at, COALESCE(integrity_status,'unknown') as integrity_status, integrity_message,
		       last_scan_at, COALESCE(last_scan_status,'') as last_scan_status,
		       connection_id, created_at, updated_at, created_by, updated_by
		FROM tenant_product_datasource WHERE id = $1
	`, id)

	if err != nil {
		return nil, fmt.Errorf("datasource not found: %s", id)
	}
	return &ds, nil
}

// GetByTenantProduct fetches all datasources for a tenant product
func (r *DatasourceRepository) GetByTenantProduct(ctx context.Context, tenantProductID string) ([]*Datasource, error) {
	var rows []Datasource
	err := r.db.SelectContext(ctx, &rows, `
		SELECT id, COALESCE(tenant_product_id::text,'') as tenant_product_id,
		       COALESCE(alpha_datasource_id::text,'') as alpha_datasource_id,
		       source_name, COALESCE(is_active, false) as is_active,
		       COALESCE(environment,'') as environment, description,
		       COALESCE(read_only, false) as read_only,
		       last_heartbeat_at, COALESCE(health_status,'unknown') as health_status, health_message,
		       last_integrity_check_at, COALESCE(integrity_status,'unknown') as integrity_status, integrity_message,
		       last_scan_at, COALESCE(last_scan_status,'') as last_scan_status,
		       connection_id, created_at, updated_at, created_by, updated_by
		FROM tenant_product_datasource WHERE tenant_product_id = $1
	`, tenantProductID)

	if err != nil {
		return []*Datasource{}, nil
	}

	result := make([]*Datasource, 0, len(rows))
	for i := range rows {
		result = append(result, &rows[i])
	}
	return result, nil
}

// Create inserts a new datasource
func (r *DatasourceRepository) Create(ctx context.Context, ds *Datasource) (*Datasource, error) {
	ds.ID = uuid.New().String()

	configJSON, _ := json.Marshal(ds.Config)
	tagsJSON, _ := json.Marshal(ds.Tags)

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO tenant_product_datasource (
			id, tenant_product_id, alpha_datasource_id, source_name,
			is_active, config, environment, tags, description, read_only,
			health_status, integrity_status, last_scan_status,
			created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8, $9, $10,
			'unknown', 'unknown', '',
			$11, NOW(), NOW()
		)
	`, ds.ID, ds.TenantProductID, ds.AlphaDatasourceID, ds.SourceName,
		ds.IsActive, string(configJSON), ds.Environment, string(tagsJSON),
		ds.Description, ds.ReadOnly, ds.CreatedBy)

	if err != nil {
		return nil, fmt.Errorf("failed to create datasource: %w", err)
	}
	return ds, nil
}

// Update modifies an existing datasource
func (r *DatasourceRepository) Update(ctx context.Context, id string, changes map[string]interface{}) error {
	changes["updated_at"] = time.Now()
	changes["id"] = id

	// Build SET clause dynamically
	setClauses := make([]string, 0, len(changes))
	args := make([]interface{}, 0, len(changes)+1)
	i := 1
	for k, v := range changes {
		if k == "id" {
			continue
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", k, i))
		args = append(args, v)
		i++
	}
	args = append(args, id)

	query := fmt.Sprintf("UPDATE tenant_product_datasource SET %s WHERE id = $%d",
		joinStrings(setClauses, ", "), i)

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update datasource: %w", err)
	}
	return nil
}

// UpdateHealthStatus updates the health status
func (r *DatasourceRepository) UpdateHealthStatus(ctx context.Context, id, status, message string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE tenant_product_datasource
		SET health_status = $2, health_message = $3, last_heartbeat_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`, id, status, message)
	return err
}

// UpdateIntegrityStatus updates the integrity status
func (r *DatasourceRepository) UpdateIntegrityStatus(ctx context.Context, id, status, message string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE tenant_product_datasource
		SET integrity_status = $2, integrity_message = $3,
		    last_integrity_check_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`, id, status, message)
	return err
}

// Delete removes a datasource
func (r *DatasourceRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM tenant_product_datasource WHERE id = $1`, id)
	return err
}

// CreateIntegrityCheck records a new integrity check
func (r *DatasourceRepository) CreateIntegrityCheck(ctx context.Context, check *IntegrityCheckResult) (*IntegrityCheckResult, error) {
	check.ID = uuid.New().String()
	check.StartedAt = time.Now()

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO datasource_integrity_checks (
			id, datasource_id, check_type, status, executed_by, started_at
		) VALUES ($1, $2, $3, 'running', $4, $5)
	`, check.ID, check.DatasourceID, check.CheckType, check.ExecutedBy, check.StartedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create integrity check: %w", err)
	}
	return check, nil
}

// UpdateIntegrityCheck updates an integrity check with results
func (r *DatasourceRepository) UpdateIntegrityCheck(ctx context.Context, check *IntegrityCheckResult) error {
	now := time.Now()
	check.CompletedAt = &now

	schemaChangesJSON, _ := json.Marshal(check.SchemaChanges)
	recommendationsJSON, _ := json.Marshal(check.Recommendations)

	_, err := r.db.ExecContext(ctx, `
		UPDATE datasource_integrity_checks SET
			status = $2, completed_at = $3, duration_ms = $4,
			postgres_row_count = $5, ignite_row_count = $6, starrocks_row_count = $7,
			row_count_delta = $8, row_count_delta_percent = $9,
			schema_changes = $10, checksum_valid = $11,
			error_message = $12, recommendations = $13
		WHERE id = $1
	`, check.ID, check.Status, check.CompletedAt, check.DurationMs,
		check.PostgresRowCount, check.IgniteRowCount, check.StarrocksRowCount,
		check.RowCountDelta, check.RowCountDeltaPercent,
		string(schemaChangesJSON), check.ChecksumValid,
		check.ErrorMessage, string(recommendationsJSON))

	return err
}

// GetIntegrityHistory fetches recent integrity checks
func (r *DatasourceRepository) GetIntegrityHistory(ctx context.Context, datasourceID string, limit int) ([]*IntegrityCheckResult, error) {
	type row struct {
		ID           string     `db:"id"`
		CheckType    string     `db:"check_type"`
		Status       string     `db:"status"`
		DurationMs   *int       `db:"duration_ms"`
		ErrorMessage *string    `db:"error_message"`
		CompletedAt  *time.Time `db:"completed_at"`
	}

	var rows []row
	err := r.db.SelectContext(ctx, &rows, `
		SELECT id, check_type, status, duration_ms, error_message, completed_at
		FROM datasource_integrity_checks
		WHERE datasource_id = $1
		ORDER BY started_at DESC
		LIMIT $2
	`, datasourceID, limit)

	if err != nil {
		return []*IntegrityCheckResult{}, nil
	}

	checks := make([]*IntegrityCheckResult, 0, len(rows))
	for _, r := range rows {
		checks = append(checks, &IntegrityCheckResult{
			ID:           r.ID,
			CheckType:    r.CheckType,
			Status:       r.Status,
			DurationMs:   r.DurationMs,
			ErrorMessage: r.ErrorMessage,
			CompletedAt:  r.CompletedAt,
		})
	}
	return checks, nil
}

// GetLatestBaseline fetches the most recent schema baseline
func (r *DatasourceRepository) GetLatestBaseline(ctx context.Context, datasourceID string) (*SchemaSnapshot, error) {
	type row struct {
		ID          string    `db:"id"`
		SnapshotStr string    `db:"snapshot_data"`
		TableCount  *int      `db:"table_count"`
		ColumnCount *int      `db:"column_count"`
		IsBaseline  bool      `db:"is_baseline"`
		CapturedAt  time.Time `db:"captured_at"`
	}

	var ss row
	err := r.db.GetContext(ctx, &ss, `
		SELECT id, snapshot_data::text as snapshot_data, table_count, column_count,
		       is_baseline, captured_at
		FROM datasource_schema_snapshots
		WHERE datasource_id = $1 AND is_baseline = true
		ORDER BY captured_at DESC
		LIMIT 1
	`, datasourceID)

	if err != nil {
		return nil, nil
	}

	var snapshotData map[string]interface{}
	_ = json.Unmarshal([]byte(ss.SnapshotStr), &snapshotData)

	return &SchemaSnapshot{
		ID:           ss.ID,
		DatasourceID: datasourceID,
		SnapshotData: snapshotData,
		TableCount:   ss.TableCount,
		ColumnCount:  ss.ColumnCount,
		IsBaseline:   ss.IsBaseline,
		CapturedAt:   ss.CapturedAt,
	}, nil
}

// SaveSchemaSnapshot saves a new schema snapshot
func (r *DatasourceRepository) SaveSchemaSnapshot(ctx context.Context, snapshot *SchemaSnapshot) (*SchemaSnapshot, error) {
	snapshot.ID = uuid.New().String()
	snapshot.CapturedAt = time.Now()

	if snapshot.IsBaseline {
		// Clear previous baselines first
		_, _ = r.db.ExecContext(ctx, `
			UPDATE datasource_schema_snapshots SET is_baseline = false
			WHERE datasource_id = $1
		`, snapshot.DatasourceID)
	}

	snapshotDataJSON, _ := json.Marshal(snapshot.SnapshotData)

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO datasource_schema_snapshots (
			id, datasource_id, snapshot_data, table_count, column_count,
			captured_by, is_baseline, notes, captured_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, snapshot.ID, snapshot.DatasourceID, string(snapshotDataJSON),
		snapshot.TableCount, snapshot.ColumnCount, snapshot.CapturedBy,
		snapshot.IsBaseline, snapshot.Notes, snapshot.CapturedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to save snapshot: %w", err)
	}
	return snapshot, nil
}

// GetHealthSummary gets aggregated health status counts
func (r *DatasourceRepository) GetHealthSummary(ctx context.Context, tenantProductID string) (map[string]int, error) {
	type row struct {
		HealthStatus string `db:"health_status"`
		Count        int    `db:"count"`
	}

	var rows []row
	err := r.db.SelectContext(ctx, &rows, `
		SELECT COALESCE(health_status,'unknown') as health_status, COUNT(*) as count
		FROM tenant_product_datasource
		WHERE tenant_product_id = $1
		GROUP BY health_status
	`, tenantProductID)

	summary := map[string]int{
		"healthy": 0, "degraded": 0, "unhealthy": 0, "unknown": 0, "total": 0,
	}

	if err != nil {
		return summary, nil
	}

	for _, r := range rows {
		summary[r.HealthStatus] += r.Count
		summary["total"] += r.Count
	}
	return summary, nil
}

// Helper functions
func getString(data map[string]interface{}, key string) string {
	if v, ok := data[key].(string); ok {
		return v
	}
	return ""
}

func getBool(data map[string]interface{}, key string) bool {
	if v, ok := data[key].(bool); ok {
		return v
	}
	return false
}

func joinStrings(parts []string, sep string) string {
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += sep
		}
		result += p
	}
	return result
}
