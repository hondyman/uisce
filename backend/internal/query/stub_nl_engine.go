package query

import (
	"context"

	"github.com/hondyman/uisce/backend/internal/domain"
)

// StubSchemaProvider is a minimal schema provider for cases where
// full schema information is not needed
type StubSchemaProvider struct{}

// GetAssetSchema returns an empty asset schema
func (s *StubSchemaProvider) GetAssetSchema(assetID string) (domain.AssetSchema, error) {
	return domain.AssetSchema{
		ColumnsByScope: map[string][]string{
			"metrics":    {"value", "amount", "count", "total"},
			"dimensions": {"date", "category", "region", "status"},
		},
		DefaultFilters: []string{},
	}, nil
}

// GetTableSchema returns an empty table schema
func (s *StubSchemaProvider) GetTableSchema(assetID, tableName string) (domain.TableSchema, error) {
	return domain.TableSchema{
		Name:    tableName,
		Columns: []domain.ColumnSchema{},
	}, nil
}

// StubGovernanceProvider provides an allow-all governance context
type StubGovernanceProvider struct{}

// NewStubGovernanceProvider creates a new stub governance provider
func NewStubGovernanceProvider() *StubGovernanceProvider {
	return &StubGovernanceProvider{}
}

// GetContext returns an allow-all governance context
func (s *StubGovernanceProvider) GetContext(ctx context.Context, userID, tenantID, datasource string) (*GovernanceContext, error) {
	return &GovernanceContext{
		UserID:            userID,
		TenantID:          tenantID,
		Datasource:        datasource,
		AllowedMetrics:    []string{},
		AllowedDimensions: []string{},
		RequiredFilters:   []QueryFilter{},
		AppliedPolicies:   []AppliedGovernancePolicy{},
		AssetMappings:     make(map[string]string),
	}, nil
}

// StubNLQueryEngine is a minimal NL engine for dashboard conversations
// that doesn't require full AI/NL processing
type StubNLQueryEngine struct {
	governanceProvider *StubGovernanceProvider
}

// NewStubNLQueryEngine creates a new stub NL query engine
func NewStubNLQueryEngine() *StubNLQueryEngine {
	return &StubNLQueryEngine{
		governanceProvider: NewStubGovernanceProvider(),
	}
}

// GovernanceProvider getter for the dashboard conversation manager
func (s *StubNLQueryEngine) GetGovernanceProvider() *StubGovernanceProvider {
	return s.governanceProvider
}
