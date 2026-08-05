package analytics

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/hondyman/uisce/backend/internal/telemetry/optimize"
)

type SemanticService struct {
	DB *sqlx.DB
}

func NewSemanticService(db *sqlx.DB) *SemanticService {
	return &SemanticService{DB: db}
}

func (s *SemanticService) ListSemanticObjects(ctx context.Context, tenantID, datasourceID string) (interface{}, error) {
	return []interface{}{}, nil
}

type SemanticModelService struct {
	DB *sqlx.DB
}

func NewSemanticModelService(db *sqlx.DB) *SemanticModelService {
	return &SemanticModelService{DB: db}
}

func (s *SemanticModelService) ListExtensionModels(ctx context.Context, tenantID, datasourceID string) (interface{}, error) {
	return []interface{}{}, nil
}

func (s *SemanticModelService) SaveExtensionModel(ctx context.Context, tenantID, datasourceID string, model interface{}) error {
	return nil
}

type ModelProvider struct{}

func NewModelProvider(db *sqlx.DB) *ModelProvider {
	return &ModelProvider{}
}

func (m *ModelProvider) GetActiveCatalog(ctx context.Context, tenantID, datasourceID string) (interface{}, error) {
	return nil, nil
}

type QueryService struct {
	DB *sqlx.DB
}

func NewQueryService(db *sqlx.DB, optSvc *optimize.Service, modelProvider *ModelProvider) *QueryService {
	return &QueryService{DB: db}
}
