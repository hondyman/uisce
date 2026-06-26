package mdm

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/analytics"
	"github.com/hondyman/semlayer/backend/internal/goldcopy"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/internal/portfoliomaster"
)

// PortfolioRepository defines the methods needed from the portfolio domain.
type PortfolioRepository interface {
	ListGoldenRecords(ctx context.Context, tenantID uuid.UUID, accountType string, asOf time.Time) ([]*portfoliomaster.PortfolioGolden, error)
}

// SecurityRepository defines the methods needed from the security domain.
type SecurityRepository interface {
	GetCurrentSecurities(ctx context.Context, tenantID uuid.UUID, securityIDs []string) ([]*goldcopy.SecurityMasterRecord, error)
}

// PortfolioSecurityService establishes the semantic spine between portfolio holdings and security master.
type PortfolioSecurityService struct {
	portfolioRepo   PortfolioRepository
	securityRepo    SecurityRepository
	executionEngine *ExecutionEngine
	graphService    *analytics.SemanticGraphService
}

// NewPortfolioSecurityService wires up the bridge service.
func NewPortfolioSecurityService(
	portfolioRepo PortfolioRepository,
	securityRepo SecurityRepository,
	engine *ExecutionEngine,
	graph *analytics.SemanticGraphService,
) *PortfolioSecurityService {
	return &PortfolioSecurityService{
		portfolioRepo:   portfolioRepo,
		securityRepo:    securityRepo,
		executionEngine: engine,
		graphService:    graph,
	}
}

// CalculatePortfolioAnalytics computes advanced metrics for a portfolio using the semantic graph.
func (s *PortfolioSecurityService) CalculatePortfolioAnalytics(ctx context.Context, tenantID uuid.UUID, portfolioID string) (*models.PortfolioAnalytics, error) {
	// 1. Fetch positions
	positions, err := s.portfolioRepo.ListGoldenRecords(ctx, tenantID, "", time.Now())
	if err != nil {
		return nil, err
	}

	// Filter by portfolio
	var portfolioPositions []*portfoliomaster.PortfolioGolden
	var securityIDs []string
	for _, p := range positions {
		if p.PortfolioID == portfolioID {
			portfolioPositions = append(portfolioPositions, p)
			securityIDs = append(securityIDs, p.SecurityID)
		}
	}

	if len(portfolioPositions) == 0 {
		return nil, fmt.Errorf("portfolio %s not found or has no positions", portfolioID)
	}

	// 2. Fetch security master data in batch
	securities, err := s.securityRepo.GetCurrentSecurities(ctx, tenantID, securityIDs)
	if err != nil {
		return nil, err
	}

	secMap := make(map[string]*goldcopy.SecurityMasterRecord)
	for _, sec := range securities {
		secMap[sec.SecurityID] = sec
	}

	// 3. Resolve NAV using ExecutionEngine if available
	var totalNAV float64
	assetBreakdown := make(map[string]float64)
	sectorExposure := make(map[string]float64)

	// Find NAV term ID
	navNode, err := s.graphService.GetNodeByName(analytics.NodeTypeCalculationTerm, "Net Asset Value", uuid.Nil) // Default DS

	for _, pos := range portfolioPositions {
		sec := secMap[pos.SecurityID]

		val := pos.MarketValue
		if navNode != nil {
			// Context for calculation
			calcCtx := map[string]interface{}{
				"Quantity": pos.Quantity,
				"Price":    pos.Price,
				"TenantID": tenantID,
			}
			res, _, err := s.executionEngine.ExecuteCalculation(ctx, navNode.ID, calcCtx)
			if err == nil {
				if f, ok := res.(float64); ok {
					val = f
				}
			}
		}

		totalNAV += val

		if sec != nil {
			assetBreakdown[sec.AssetClass] += val
			sector := sec.Sector
			if sector == "" {
				sector = "Unknown"
			}
			sectorExposure[sector] += val
		}
	}

	// 4. Populate Analytics model
	analytics := &models.PortfolioAnalytics{
		PortfolioID:         uuid.MustParse(portfolioID),
		TotalValue:          totalNAV,
		TotalPositions:      len(portfolioPositions),
		AssetClassBreakdown: assetBreakdown,
		SectorExposure:      sectorExposure,
		AsOfDate:            time.Now(),
	}

	// Normalize breakdowns
	if totalNAV > 0 {
		for k, v := range assetBreakdown {
			analytics.AssetClassBreakdown[k] = (v / totalNAV) * 100
		}
		for k, v := range sectorExposure {
			analytics.SectorExposure[k] = (v / totalNAV) * 100
		}
	}

	return analytics, nil
}
