package governance

import (
	"context"
	"fmt"
	"math"

	"github.com/hondyman/semlayer/backend/internal/investment/iceberg"
	"github.com/hondyman/semlayer/backend/internal/investment/speed"
)

// ReconciliationService ensures consistency between System of Record and Query Layer
type ReconciliationService struct {
	icebergReader *iceberg.TaxLotWriter // Using Writer as it has access to storage, in real app would be a Reader
	queryClient   speed.StarRocksClient
}

func NewReconciliationService(icebergReader *iceberg.TaxLotWriter, queryClient speed.StarRocksClient) *ReconciliationService {
	return &ReconciliationService{
		icebergReader: icebergReader,
		queryClient:   queryClient,
	}
}

// ReconcilePortfolio compares the total market value of a portfolio in both systems
func (s *ReconciliationService) ReconcilePortfolio(ctx context.Context, portfolioID uint64) error {
	// 1. Fetch Total Market Value from Iceberg (System of Record)
	// In a real implementation, this would query the Iceberg catalog/table via StarRocks
	// For this MVP, we'll simulate it.
	icebergValue, err := s.mockFetchIcebergValue(ctx, portfolioID)
	if err != nil {
		return fmt.Errorf("failed to fetch iceberg value: %w", err)
	}

	// 2. Fetch Total Market Value from StarRocks materialized view (Query Layer)
	// In a real implementation, this would run: SELECT sum(market_value) FROM wealth_analytics.current_positions WHERE portfolio_id = ?
	queryValue, err := s.mockFetchQueryValue(ctx, portfolioID)
	if err != nil {
		return fmt.Errorf("failed to fetch query value: %w", err)
	}

	// 3. Compare with tolerance
	diff := math.Abs(icebergValue - queryValue)
	tolerance := 0.01 // $0.01 tolerance

	if diff > tolerance {
		return fmt.Errorf("reconciliation failed for portfolio %d: iceberg=%.2f, query=%.2f, diff=%.2f",
			portfolioID, icebergValue, queryValue, diff)
	}

	fmt.Printf("✅ Reconciliation passed for portfolio %d (Value: $%.2f)\n", portfolioID, icebergValue)
	return nil
}

func (s *ReconciliationService) mockFetchIcebergValue(ctx context.Context, portfolioID uint64) (float64, error) {
	// Mock logic
	return 1000000.00, nil
}

func (s *ReconciliationService) mockFetchQueryValue(ctx context.Context, portfolioID uint64) (float64, error) {
	// Mock logic - simulate a slight drift or match
	return 1000000.00, nil
}
