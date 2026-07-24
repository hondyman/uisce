package metrics

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

// HasuraClient defines the interface for Hasura GraphQL operations
type HasuraClient interface {
	Query(ctx context.Context, query string, variables map[string]interface{}) (map[string]interface{}, error)
	Mutate(ctx context.Context, mutation string, variables map[string]interface{}) (map[string]interface{}, error)
}

// MetricRepository defines the interface for metric storage
type MetricRepository interface {
	List(ctx context.Context) ([]MetricDefinition, error)
	Get(ctx context.Context, id string) (*MetricDefinition, error)
	Create(ctx context.Context, metric *MetricDefinition) error
	Update(ctx context.Context, metric *MetricDefinition) error
}

// SQLMetricRepository implements MetricRepository using database/sql
type SQLMetricRepository struct {
	DB           *sql.DB
	hasuraClient HasuraClient
}

func NewSQLMetricRepository(db *sql.DB) *SQLMetricRepository {
	return &SQLMetricRepository{DB: db}
}

func NewSQLMetricRepositoryWithHasura(db *sql.DB, hasura HasuraClient) *SQLMetricRepository {
	return &SQLMetricRepository{DB: db, hasuraClient: hasura}
}

func (r *SQLMetricRepository) List(ctx context.Context) ([]MetricDefinition, error) {
	return r.listMetricsRecords(ctx)
}

func (r *SQLMetricRepository) Get(ctx context.Context, id string) (*MetricDefinition, error) {
	return r.getMetricRecord(ctx, id)
}

func (r *SQLMetricRepository) Create(ctx context.Context, metric *MetricDefinition) error {
	return r.createMetricRecord(ctx, metric)
}

func (r *SQLMetricRepository) Update(ctx context.Context, metric *MetricDefinition) error {
	return r.updateMetricRecord(ctx, metric)
}

// listMetricsRecords retrieves all metric definitions from the database
// TODO: Implement Hasura GraphQL query
// SQL fallback: SELECT all fields with ORDER BY name
func (r *SQLMetricRepository) listMetricsRecords(ctx context.Context) ([]MetricDefinition, error) {
	query := `
		SELECT id, name, display_name, description, domain, granularity, aggregation_function, base_query, dimensions, sla_config, owner, created_at, updated_at
		FROM metric_definitions
		ORDER BY name
	`
	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list metrics: %w", err)
	}
	defer rows.Close()

	var metrics []MetricDefinition
	for rows.Next() {
		var m MetricDefinition
		var dims, sla []byte
		if err := rows.Scan(&m.ID, &m.Name, &m.DisplayName, &m.Description, &m.Domain, &m.Granularity, &m.AggregationFunction, &m.BaseQuery, &dims, &sla, &m.Owner, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan metric: %w", err)
		}
		m.Dimensions = json.RawMessage(dims)
		m.SLAConfig = json.RawMessage(sla)
		metrics = append(metrics, m)
	}
	return metrics, rows.Err()
}

// getMetricRecord retrieves a single metric definition by ID
// TODO: Implement Hasura GraphQL query
// SQL fallback: SELECT by id with QueryRowContext + Scan
func (r *SQLMetricRepository) getMetricRecord(ctx context.Context, id string) (*MetricDefinition, error) {
	query := `
		SELECT id, name, display_name, description, domain, granularity, aggregation_function, base_query, dimensions, sla_config, owner, created_at, updated_at
		FROM metric_definitions
		WHERE id = $1
	`
	var m MetricDefinition
	var dims, sla []byte
	err := r.DB.QueryRowContext(ctx, query, id).Scan(&m.ID, &m.Name, &m.DisplayName, &m.Description, &m.Domain, &m.Granularity, &m.AggregationFunction, &m.BaseQuery, &dims, &sla, &m.Owner, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get metric: %w", err)
	}
	m.Dimensions = json.RawMessage(dims)
	m.SLAConfig = json.RawMessage(sla)
	return &m, nil
}

// createMetricRecord inserts a new metric definition
// TODO: Implement Hasura GraphQL mutation
// SQL fallback: INSERT with 11 fields including JSON marshaling
func (r *SQLMetricRepository) createMetricRecord(ctx context.Context, metric *MetricDefinition) error {
	query := `
		INSERT INTO metric_definitions (id, name, display_name, description, domain, granularity, aggregation_function, base_query, dimensions, sla_config, owner)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	dimsJSON, err := json.Marshal(metric.Dimensions)
	if err != nil {
		return fmt.Errorf("failed to marshal dimensions: %w", err)
	}
	slaJSON, err := json.Marshal(metric.SLAConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal SLA config: %w", err)
	}

	_, err = r.DB.ExecContext(ctx, query, metric.ID, metric.Name, metric.DisplayName, metric.Description, metric.Domain, metric.Granularity, metric.AggregationFunction, metric.BaseQuery, dimsJSON, slaJSON, metric.Owner)
	if err != nil {
		return fmt.Errorf("failed to create metric: %w", err)
	}
	return nil
}

// updateMetricRecord updates an existing metric definition
// TODO: Implement Hasura GraphQL mutation
// SQL fallback: UPDATE 11 fields with now() for updated_at
func (r *SQLMetricRepository) updateMetricRecord(ctx context.Context, metric *MetricDefinition) error {
	query := `
		UPDATE metric_definitions
		SET name=$2, display_name=$3, description=$4, domain=$5, granularity=$6, aggregation_function=$7, base_query=$8, dimensions=$9, sla_config=$10, owner=$11, updated_at=now()
		WHERE id=$1
	`
	dimsJSON, err := json.Marshal(metric.Dimensions)
	if err != nil {
		return fmt.Errorf("failed to marshal dimensions: %w", err)
	}
	slaJSON, err := json.Marshal(metric.SLAConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal SLA config: %w", err)
	}

	result, err := r.DB.ExecContext(ctx, query, metric.ID, metric.Name, metric.DisplayName, metric.Description, metric.Domain, metric.Granularity, metric.AggregationFunction, metric.BaseQuery, dimsJSON, slaJSON, metric.Owner)
	if err != nil {
		return fmt.Errorf("failed to update metric: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("metric not found: %s", metric.ID)
	}
	return nil
}
