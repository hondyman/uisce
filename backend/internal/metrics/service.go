package metrics

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

// MetricService manages metric definitions
type MetricService struct {
	repo MetricRepository
}

// NewMetricService creates a new metric service
func NewMetricService(db *sql.DB) *MetricService {
	return &MetricService{
		repo: NewSQLMetricRepository(db),
	}
}

// ListDefinitions returns all metric definitions
func (s *MetricService) ListDefinitions(ctx context.Context) ([]MetricDefinition, error) {
	return s.repo.List(ctx)
}

// GetDefinition returns a single metric definition by ID
func (s *MetricService) GetDefinition(ctx context.Context, id string) (*MetricDefinition, error) {
	return s.repo.Get(ctx, id)
}

// CreateDefinition creates a new metric definition
func (s *MetricService) CreateDefinition(ctx context.Context, def *MetricDefinition) error {
	if def.ID == uuid.Nil {
		def.ID = uuid.New()
	}
	return s.repo.Create(ctx, def)
}

// UpdateDefinition updates an existing metric definition
func (s *MetricService) UpdateDefinition(ctx context.Context, def *MetricDefinition) error {
	return s.repo.Update(ctx, def)
}
