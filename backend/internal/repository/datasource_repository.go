package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	gql "github.com/hondyman/semlayer/backend/internal/graphql"
	hasuraclient "github.com/hondyman/semlayer/libs/hasura-client"
)

// DatasourceRepository provides GraphQL-based CRUD for datasources
type DatasourceRepository struct {
	client *hasuraclient.HasuraClient
}

// NewDatasourceRepository creates a new DatasourceRepository
func NewDatasourceRepository(client *hasuraclient.HasuraClient) *DatasourceRepository {
	return &DatasourceRepository{client: client}
}

// Datasource represents the datasource entity
type Datasource struct {
	ID                   string                 `json:"id"`
	TenantProductID      string                 `json:"tenant_product_id"`
	AlphaDatasourceID    string                 `json:"alpha_datasource_id"`
	SourceName           string                 `json:"source_name"`
	IsActive             bool                   `json:"is_active"`
	Config               map[string]interface{} `json:"config,omitempty"`
	Environment          string                 `json:"environment"`
	Tags                 []string               `json:"tags"`
	Description          *string                `json:"description,omitempty"`
	ReadOnly             bool                   `json:"read_only"`
	PoolConfig           map[string]interface{} `json:"pool_config,omitempty"`
	ScanSchedule         map[string]interface{} `json:"scan_schedule,omitempty"`
	HealthConfig         map[string]interface{} `json:"health_config,omitempty"`
	IntegrityChecks      map[string]interface{} `json:"integrity_checks,omitempty"`
	SLAConfig            map[string]interface{} `json:"sla_config,omitempty"`
	DataClassification   map[string]interface{} `json:"data_classification,omitempty"`
	LastHeartbeatAt      *time.Time             `json:"last_heartbeat_at,omitempty"`
	HealthStatus         string                 `json:"health_status"`
	HealthMessage        *string                `json:"health_message,omitempty"`
	LastIntegrityCheckAt *time.Time             `json:"last_integrity_check_at,omitempty"`
	IntegrityStatus      string                 `json:"integrity_status"`
	IntegrityMessage     *string                `json:"integrity_message,omitempty"`
	LastScanAt           *time.Time             `json:"last_scan_at,omitempty"`
	LastScanStatus       string                 `json:"last_scan_status"`
	ConnectionID         *string                `json:"connection_id,omitempty"`
	CreatedAt            time.Time              `json:"created_at"`
	UpdatedAt            time.Time              `json:"updated_at"`
	CreatedBy            *string                `json:"created_by,omitempty"`
	UpdatedBy            *string                `json:"updated_by,omitempty"`
}

// IntegrityCheckResult represents an integrity check result
type IntegrityCheckResult struct {
	ID                   string                 `json:"id"`
	DatasourceID         string                 `json:"datasource_id"`
	CheckType            string                 `json:"check_type"`
	Status               string                 `json:"status"`
	PostgresRowCount     *int64                 `json:"postgres_row_count,omitempty"`
	IgniteRowCount       *int64                 `json:"ignite_row_count,omitempty"`
	StarrocksRowCount    *int64                 `json:"starrocks_row_count,omitempty"`
	RowCountDelta        *int64                 `json:"row_count_delta,omitempty"`
	RowCountDeltaPercent *float64               `json:"row_count_delta_percent,omitempty"`
	SchemaChanges        map[string]interface{} `json:"schema_changes,omitempty"`
	ChecksumValid        *bool                  `json:"checksum_valid,omitempty"`
	ExecutedBy           *string                `json:"executed_by,omitempty"`
	StartedAt            time.Time              `json:"started_at"`
	CompletedAt          *time.Time             `json:"completed_at,omitempty"`
	DurationMs           *int                   `json:"duration_ms,omitempty"`
	ErrorMessage         *string                `json:"error_message,omitempty"`
	Recommendations      map[string]interface{} `json:"recommendations,omitempty"`
}

// SchemaSnapshot represents a schema snapshot
type SchemaSnapshot struct {
	ID                 string                 `json:"id"`
	DatasourceID       string                 `json:"datasource_id"`
	SnapshotData       map[string]interface{} `json:"snapshot_data"`
	TableCount         *int                   `json:"table_count,omitempty"`
	ColumnCount        *int                   `json:"column_count,omitempty"`
	CapturedAt         time.Time              `json:"captured_at"`
	CapturedBy         *string                `json:"captured_by,omitempty"`
	IsBaseline         bool                   `json:"is_baseline"`
	Notes              *string                `json:"notes,omitempty"`
	PreviousSnapshotID *string                `json:"previous_snapshot_id,omitempty"`
	ChangeSummary      map[string]interface{} `json:"change_summary,omitempty"`
}

// GetByID fetches a datasource by ID
func (r *DatasourceRepository) GetByID(ctx context.Context, id string) (*Datasource, error) {
	result, err := r.client.Query(gql.GetDatasourceByID, map[string]interface{}{
		"id": id,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get datasource: %w", err)
	}

	data, ok := result["tenant_product_datasource_by_pk"].(map[string]interface{})
	if !ok || data == nil {
		return nil, fmt.Errorf("datasource not found: %s", id)
	}

	return mapToDatasource(data), nil
}

// GetByTenantProduct fetches all datasources for a tenant product
func (r *DatasourceRepository) GetByTenantProduct(ctx context.Context, tenantProductID string) ([]*Datasource, error) {
	result, err := r.client.Query(gql.GetDatasourcesByTenantProduct, map[string]interface{}{
		"tenant_product_id": tenantProductID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get datasources: %w", err)
	}

	items, ok := result["tenant_product_datasource"].([]interface{})
	if !ok {
		return []*Datasource{}, nil
	}

	datasources := make([]*Datasource, 0, len(items))
	for _, item := range items {
		if data, ok := item.(map[string]interface{}); ok {
			datasources = append(datasources, mapToDatasource(data))
		}
	}

	return datasources, nil
}

// Create inserts a new datasource
func (r *DatasourceRepository) Create(ctx context.Context, ds *Datasource) (*Datasource, error) {
	object := map[string]interface{}{
		"tenant_product_id":   ds.TenantProductID,
		"alpha_datasource_id": ds.AlphaDatasourceID,
		"source_name":         ds.SourceName,
		"is_active":           ds.IsActive,
		"config":              ds.Config,
		"environment":         ds.Environment,
		"tags":                ds.Tags,
		"description":         ds.Description,
		"read_only":           ds.ReadOnly,
		"pool_config":         ds.PoolConfig,
		"scan_schedule":       ds.ScanSchedule,
		"health_config":       ds.HealthConfig,
		"integrity_checks":    ds.IntegrityChecks,
		"sla_config":          ds.SLAConfig,
		"data_classification": ds.DataClassification,
		"created_by":          ds.CreatedBy,
	}

	result, err := r.client.Mutate(gql.InsertDatasource, map[string]interface{}{
		"object": object,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create datasource: %w", err)
	}

	data, ok := result["insert_tenant_product_datasource_one"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response format")
	}

	ds.ID = data["id"].(string)
	return ds, nil
}

// Update modifies an existing datasource
func (r *DatasourceRepository) Update(ctx context.Context, id string, changes map[string]interface{}) error {
	changes["updated_at"] = time.Now()

	_, err := r.client.Mutate(gql.UpdateDatasource, map[string]interface{}{
		"id":      id,
		"changes": changes,
	})
	if err != nil {
		return fmt.Errorf("failed to update datasource: %w", err)
	}

	return nil
}

// UpdateHealthStatus updates the health status
func (r *DatasourceRepository) UpdateHealthStatus(ctx context.Context, id, status, message string) error {
	now := time.Now()
	_, err := r.client.Mutate(gql.UpdateDatasourceHealth, map[string]interface{}{
		"id":           id,
		"status":       status,
		"message":      message,
		"heartbeat_at": now,
	})
	return err
}

// UpdateIntegrityStatus updates the integrity status
func (r *DatasourceRepository) UpdateIntegrityStatus(ctx context.Context, id, status, message string) error {
	now := time.Now()
	_, err := r.client.Mutate(gql.UpdateDatasourceIntegrity, map[string]interface{}{
		"id":       id,
		"status":   status,
		"message":  message,
		"check_at": now,
	})
	return err
}

// Delete removes a datasource
func (r *DatasourceRepository) Delete(ctx context.Context, id string) error {
	_, err := r.client.Mutate(gql.DeleteDatasource, map[string]interface{}{
		"id": id,
	})
	return err
}

// CreateIntegrityCheck records a new integrity check
func (r *DatasourceRepository) CreateIntegrityCheck(ctx context.Context, check *IntegrityCheckResult) (*IntegrityCheckResult, error) {
	check.ID = uuid.New().String()
	check.StartedAt = time.Now()

	object := map[string]interface{}{
		"id":            check.ID,
		"datasource_id": check.DatasourceID,
		"check_type":    check.CheckType,
		"status":        "running",
		"executed_by":   check.ExecutedBy,
		"started_at":    check.StartedAt,
	}

	_, err := r.client.Mutate(gql.InsertIntegrityCheck, map[string]interface{}{
		"object": object,
	})
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

	_, err := r.client.Mutate(gql.UpdateIntegrityCheckComplete, map[string]interface{}{
		"id":                      check.ID,
		"status":                  check.Status,
		"completed_at":            check.CompletedAt,
		"duration_ms":             check.DurationMs,
		"postgres_row_count":      check.PostgresRowCount,
		"ignite_row_count":        check.IgniteRowCount,
		"starrocks_row_count":     check.StarrocksRowCount,
		"row_count_delta":         check.RowCountDelta,
		"row_count_delta_percent": check.RowCountDeltaPercent,
		"schema_changes":          string(schemaChangesJSON),
		"checksum_valid":          check.ChecksumValid,
		"error_message":           check.ErrorMessage,
		"recommendations":         string(recommendationsJSON),
	})
	return err
}

// GetIntegrityHistory fetches recent integrity checks
func (r *DatasourceRepository) GetIntegrityHistory(ctx context.Context, datasourceID string, limit int) ([]*IntegrityCheckResult, error) {
	result, err := r.client.Query(gql.GetIntegrityCheckHistory, map[string]interface{}{
		"datasource_id": datasourceID,
		"limit":         limit,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get integrity history: %w", err)
	}

	items, ok := result["datasource_integrity_checks"].([]interface{})
	if !ok {
		return []*IntegrityCheckResult{}, nil
	}

	checks := make([]*IntegrityCheckResult, 0, len(items))
	for _, item := range items {
		if data, ok := item.(map[string]interface{}); ok {
			checks = append(checks, mapToIntegrityCheck(data))
		}
	}

	return checks, nil
}

// GetLatestBaseline fetches the most recent schema baseline
func (r *DatasourceRepository) GetLatestBaseline(ctx context.Context, datasourceID string) (*SchemaSnapshot, error) {
	result, err := r.client.Query(gql.GetLatestSchemaBaseline, map[string]interface{}{
		"datasource_id": datasourceID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get baseline: %w", err)
	}

	items, ok := result["datasource_schema_snapshots"].([]interface{})
	if !ok || len(items) == 0 {
		return nil, nil
	}

	data := items[0].(map[string]interface{})
	return mapToSchemaSnapshot(data), nil
}

// SaveSchemaSnapshot saves a new schema snapshot
func (r *DatasourceRepository) SaveSchemaSnapshot(ctx context.Context, snapshot *SchemaSnapshot) (*SchemaSnapshot, error) {
	snapshot.ID = uuid.New().String()
	snapshot.CapturedAt = time.Now()

	if snapshot.IsBaseline {
		// Clear previous baselines first
		_, _ = r.client.Mutate(gql.ClearPreviousBaselines, map[string]interface{}{
			"datasource_id": snapshot.DatasourceID,
		})
	}

	snapshotDataJSON, _ := json.Marshal(snapshot.SnapshotData)

	object := map[string]interface{}{
		"id":            snapshot.ID,
		"datasource_id": snapshot.DatasourceID,
		"snapshot_data": string(snapshotDataJSON),
		"table_count":   snapshot.TableCount,
		"column_count":  snapshot.ColumnCount,
		"captured_by":   snapshot.CapturedBy,
		"is_baseline":   snapshot.IsBaseline,
		"notes":         snapshot.Notes,
	}

	_, err := r.client.Mutate(gql.InsertSchemaSnapshot, map[string]interface{}{
		"object": object,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to save snapshot: %w", err)
	}

	return snapshot, nil
}

// GetHealthSummary gets aggregated health status counts
func (r *DatasourceRepository) GetHealthSummary(ctx context.Context, tenantProductID string) (map[string]int, error) {
	result, err := r.client.Query(gql.GetDatasourceHealthSummary, map[string]interface{}{
		"tenant_product_id": tenantProductID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get health summary: %w", err)
	}

	summary := map[string]int{
		"healthy":   extractCount(result, "healthy"),
		"degraded":  extractCount(result, "degraded"),
		"unhealthy": extractCount(result, "unhealthy"),
		"unknown":   extractCount(result, "unknown"),
		"total":     extractCount(result, "total"),
	}

	return summary, nil
}

// Helper functions

func mapToDatasource(data map[string]interface{}) *Datasource {
	ds := &Datasource{
		ID:              getString(data, "id"),
		SourceName:      getString(data, "source_name"),
		Environment:     getString(data, "environment"),
		IsActive:        getBool(data, "is_active"),
		HealthStatus:    getString(data, "health_status"),
		IntegrityStatus: getString(data, "integrity_status"),
		LastScanStatus:  getString(data, "last_scan_status"),
		ReadOnly:        getBool(data, "read_only"),
	}

	if v, ok := data["tenant_product_id"].(string); ok {
		ds.TenantProductID = v
	}
	if v, ok := data["alpha_datasource_id"].(string); ok {
		ds.AlphaDatasourceID = v
	}
	if v, ok := data["tags"].([]interface{}); ok {
		ds.Tags = make([]string, len(v))
		for i, tag := range v {
			if s, ok := tag.(string); ok {
				ds.Tags[i] = s
			}
		}
	}
	if v, ok := data["config"].(map[string]interface{}); ok {
		ds.Config = v
	}
	if v, ok := data["pool_config"].(map[string]interface{}); ok {
		ds.PoolConfig = v
	}
	if v, ok := data["sla_config"].(map[string]interface{}); ok {
		ds.SLAConfig = v
	}

	return ds
}

func mapToIntegrityCheck(data map[string]interface{}) *IntegrityCheckResult {
	check := &IntegrityCheckResult{
		ID:        getString(data, "id"),
		CheckType: getString(data, "check_type"),
		Status:    getString(data, "status"),
	}

	if v, ok := data["postgres_row_count"].(float64); ok {
		val := int64(v)
		check.PostgresRowCount = &val
	}
	if v, ok := data["row_count_delta"].(float64); ok {
		val := int64(v)
		check.RowCountDelta = &val
	}
	if v, ok := data["duration_ms"].(float64); ok {
		val := int(v)
		check.DurationMs = &val
	}
	if v, ok := data["checksum_valid"].(bool); ok {
		check.ChecksumValid = &v
	}
	if v, ok := data["error_message"].(string); ok {
		check.ErrorMessage = &v
	}
	if v, ok := data["schema_changes"].(map[string]interface{}); ok {
		check.SchemaChanges = v
	}

	return check
}

func mapToSchemaSnapshot(data map[string]interface{}) *SchemaSnapshot {
	snapshot := &SchemaSnapshot{
		ID:         getString(data, "id"),
		IsBaseline: getBool(data, "is_baseline"),
	}

	if v, ok := data["snapshot_data"].(map[string]interface{}); ok {
		snapshot.SnapshotData = v
	}
	if v, ok := data["table_count"].(float64); ok {
		val := int(v)
		snapshot.TableCount = &val
	}
	if v, ok := data["column_count"].(float64); ok {
		val := int(v)
		snapshot.ColumnCount = &val
	}

	return snapshot
}

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

func extractCount(data map[string]interface{}, key string) int {
	if agg, ok := data[key].(map[string]interface{}); ok {
		if aggregate, ok := agg["aggregate"].(map[string]interface{}); ok {
			if count, ok := aggregate["count"].(float64); ok {
				return int(count)
			}
		}
	}
	return 0
}
