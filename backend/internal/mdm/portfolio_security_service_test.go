package mdm

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/analytics"
	"github.com/hondyman/semlayer/backend/internal/goldcopy"
	"github.com/hondyman/semlayer/backend/internal/portfoliomaster"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockPortfolioRepo implements PortfolioRepository
type MockPortfolioRepo struct {
	mock.Mock
}

func (m *MockPortfolioRepo) ListGoldenRecords(ctx context.Context, tenantID uuid.UUID, accountType string, asOf time.Time) ([]*portfoliomaster.PortfolioGolden, error) {
	args := m.Called(ctx, tenantID, accountType, asOf)
	return args.Get(0).([]*portfoliomaster.PortfolioGolden), args.Error(1)
}

// MockSecurityRepo implements SecurityRepository
type MockSecurityRepo struct {
	mock.Mock
}

func (m *MockSecurityRepo) GetCurrentSecurities(ctx context.Context, tenantID uuid.UUID, securityIDs []string) ([]*goldcopy.SecurityMasterRecord, error) {
	args := m.Called(ctx, tenantID, securityIDs)
	return args.Get(0).([]*goldcopy.SecurityMasterRecord), args.Error(1)
}

func TestPortfolioSecurityService_CalculatePortfolioAnalytics(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	portfolioID := uuid.New().String()

	// Setup in-memory DB for SemanticGraphService (needed by ExecutionEngine)
	db, err := sqlx.Connect("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Setup graph
	setupGraphTables(db)
	graphService := analytics.NewSemanticGraphService(db)
	engine, err := NewExecutionEngine(ctx, graphService, nil)
	assert.NoError(t, err)
	defer engine.Close(ctx)

	// Mock repositories
	pmRepo := new(MockPortfolioRepo)
	gcRepo := new(MockSecurityRepo)

	service := NewPortfolioSecurityService(pmRepo, gcRepo, engine, graphService)

	// Seed data for mocks
	secID := "AAPL"
	positions := []*portfoliomaster.PortfolioGolden{
		{
			PortfolioID: portfolioID,
			SecurityID:  secID,
			Quantity:    100.0,
			Price:       150.0,
			MarketValue: 15000.0,
		},
	}
	securities := []*goldcopy.SecurityMasterRecord{
		{
			SecurityID: secID,
			AssetClass: "Equity",
			Sector:     "Technology",
		},
	}

	pmRepo.On("ListGoldenRecords", mock.Anything, tenantID, "", mock.Anything).Return(positions, nil)
	gcRepo.On("GetCurrentSecurities", mock.Anything, tenantID, []string{secID}).Return(securities, nil)

	// Execute
	analyticsResult, err := service.CalculatePortfolioAnalytics(ctx, tenantID, portfolioID)

	assert.NoError(t, err)
	assert.NotNil(t, analyticsResult)
	assert.Equal(t, 15000.0, analyticsResult.TotalValue)
	assert.Equal(t, 1, analyticsResult.TotalPositions)
	assert.Equal(t, 100.0, analyticsResult.AssetClassBreakdown["Equity"])
	assert.Equal(t, 100.0, analyticsResult.SectorExposure["Technology"])
}

func setupGraphTables(db *sqlx.DB) {
	db.MustExec(`CREATE TABLE catalog_node_type (id UUID PRIMARY KEY, catalog_type_name TEXT, node_type TEXT, tenant_id UUID)`)
	db.MustExec(`CREATE TABLE catalog_node (id UUID PRIMARY KEY, node_name TEXT, properties TEXT, config TEXT, node_type_id UUID, tenant_id UUID, qualified_path TEXT)`)
	db.MustExec(`CREATE TABLE catalog_edge (id UUID PRIMARY KEY, source_node_id UUID, target_node_id UUID, edge_type_name TEXT, tenant_id UUID)`)
}
