// services/risk_service.go
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/audit"
	"github.com/hondyman/semlayer/backend/internal/wasm"
	"github.com/jmoiron/sqlx"
)

// RiskService orchestrates risk computation ETL
type RiskService struct {
	db       *sqlx.DB
	wasm     wasm.Engine
	audit    audit.Logger
	tenantID uuid.UUID
}

// NewRiskService creates a new risk service
func NewRiskService(
	db *sqlx.DB,
	wasm wasm.Engine,
	audit audit.Logger,
	tenantID uuid.UUID,
) *RiskService {
	return &RiskService{
		db:       db,
		wasm:     wasm,
		audit:    audit,
		tenantID: tenantID,
	}
}

// ListPortfolios fetches portfolios for the tenant
func (s *RiskService) ListPortfolios(ctx context.Context, tenantID uuid.UUID) ([]Portfolio, error) {
	var portfolios []Portfolio
	err := s.db.SelectContext(ctx, &portfolios, "SELECT id, aum, strategy FROM edm.portfolio_master WHERE tenant_id = $1", tenantID)
	return portfolios, err
}

// ComputePortfolioRisk runs factor model + VaR for a portfolio/date
func (s *RiskService) ComputePortfolioRisk(
	ctx context.Context,
	portfolioID uuid.UUID,
	valuationDate string,
) error {
	start := time.Now()

	// 1. Build factor model context
	factorCtx, err := s.buildFactorModelContext(ctx, portfolioID, valuationDate)
	if err != nil {
		return fmt.Errorf("build factor context: %w", err)
	}

	// 2. Compute factor exposures + volatility via WASM
	factorResult, err := s.wasm.ComputeFactorModel(ctx, *factorCtx)
	if err != nil {
		return fmt.Errorf("WASM factor model: %w", err)
	}

	// 3. Compute VaR (parametric using factor model results)
	varCtx := wasm.VaRContext{
		Method:           "parametric",
		ConfidenceLevels: []float64{0.95, 0.99},
		FactorExposures:  factorResult.PortfolioFactorExposures,
		FactorCovariance: factorCtx.FactorCovariance,
		TenantID:         s.tenantID.String(),
	}
	varCtx.Portfolio.ID = portfolioID.String()
	varCtx.Portfolio.AUM = factorCtx.Portfolio.AUM

	varResult, err := s.wasm.ComputeVaR(ctx, varCtx)
	if err != nil {
		return fmt.Errorf("WASM VaR: %w", err)
	}

	// 4. Persist portfolio_risk record
	err = s.insertPortfolioRisk(ctx, portfolioID, valuationDate, factorResult, varResult)
	if err != nil {
		return fmt.Errorf("insert portfolio risk: %w", err)
	}

	// 5. Audit
	s.audit.Log(ctx, audit.RiskComputation{
		PortfolioID:   portfolioID,
		ValuationDate: valuationDate,
		Volatility:    factorResult.TotalVolatility,
		VaR95:         varResult.VaR["95"],
		VaR99:         varResult.VaR["99"],
		ExecutionMs:   time.Since(start).Milliseconds(),
	})

	return nil
}

func (s *RiskService) buildFactorModelContext(
	ctx context.Context,
	portfolioID uuid.UUID,
	valuationDate string,
) (*wasm.FactorModelContext, error) {
	// Load portfolio AUM
	var portfolio struct {
		ID  uuid.UUID `db:"id"`
		AUM float64   `db:"aum"`
	}
	err := s.db.GetContext(ctx, &portfolio, `
		SELECT id, aum FROM edm.portfolio_master WHERE id = $1 AND tenant_id = $2
	`, portfolioID, s.tenantID)
	if err != nil {
		return nil, err
	}

	// Load positions
	var positions []struct {
		SecurityID  uuid.UUID `db:"security_id"`
		MarketValue float64   `db:"market_value"`
	}
	err = s.db.SelectContext(ctx, &positions, `
		SELECT security_id, market_value_base as market_value FROM edm.position_master
		WHERE portfolio_id = $1 AND position_date = $2 AND valid_to = 'infinity' AND tenant_id = $3
	`, portfolioID, valuationDate, s.tenantID)
	if err != nil {
		return nil, err
	}

	// Load factor exposures for positions
	var factorExposures []struct {
		SecurityID uuid.UUID `db:"security_id"`
		FactorID   string    `db:"factor_id"`
		Exposure   float64   `db:"exposure"`
	}
	securityIDs := make([]uuid.UUID, len(positions))
	for i, p := range positions {
		securityIDs[i] = p.SecurityID
	}
	// Avoid empty IN clause
	if len(securityIDs) > 0 {
		query, args, _ := sqlx.In(`
			SELECT security_id, factor_id, exposure FROM edm.security_factor_exposure
			WHERE security_id IN (?) AND as_of_date <= ? AND tenant_id = ?
		`, securityIDs, valuationDate, s.tenantID)
		query = s.db.Rebind(query)
		err = s.db.SelectContext(ctx, &factorExposures, query, args...)
		if err != nil {
			return nil, err
		}
	}

	// Load factor covariance matrix (from catalog or external source)
	covariance, err := s.loadFactorCovariance(ctx, valuationDate)
	if err != nil {
		return nil, err
	}

	// Build context
	ctxObj := &wasm.FactorModelContext{
		TenantID: s.tenantID.String(),
	}
	ctxObj.Portfolio.ID = portfolio.ID.String()
	ctxObj.Portfolio.AUM = portfolio.AUM
	for _, p := range positions {
		ctxObj.Positions = append(ctxObj.Positions, struct {
			SecurityID  string  `json:"security_id" jsonschema:"required,format=uuid"`
			MarketValue float64 `json:"market_value" jsonschema:"required,minimum=0"`
		}{
			SecurityID:  p.SecurityID.String(),
			MarketValue: p.MarketValue,
		})
	}
	for _, fe := range factorExposures {
		ctxObj.FactorExposures = append(ctxObj.FactorExposures, struct {
			SecurityID string  `json:"security_id" jsonschema:"required,format=uuid"`
			FactorID   string  `json:"factor_id" jsonschema:"required"`
			Exposure   float64 `json:"exposure" jsonschema:"required"`
		}{
			SecurityID: fe.SecurityID.String(),
			FactorID:   fe.FactorID,
			Exposure:   fe.Exposure,
		})
	}
	ctxObj.FactorCovariance = covariance

	return ctxObj, nil
}

func (s *RiskService) loadFactorCovariance(ctx context.Context, asOfDate string) (map[string]map[string]float64, error) {
	// Load from catalog_node or external source
	// Simplified: return static matrix for demo
	return map[string]map[string]float64{
		"VALUE": {"VALUE": 0.04, "SIZE": 0.01},
		"SIZE":  {"VALUE": 0.01, "SIZE": 0.03},
	}, nil
}

func (s *RiskService) insertPortfolioRisk(
	ctx context.Context,
	portfolioID uuid.UUID,
	valuationDate string,
	factorResult *wasm.FactorModelResult,
	varResult *wasm.VaRResult,
) error {
	query := `
		INSERT INTO edm.portfolio_risk (
			portfolio_risk_id, portfolio_id, valuation_date,
			total_volatility, var_95, var_99, expected_shortfall_97_5,
			factor_contributions, var_method, confidence_level, tenant_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	factorJSON, _ := json.Marshal(factorResult.FactorContributions)
	_, err := s.db.ExecContext(ctx, query,
		uuid.New(), portfolioID, valuationDate,
		factorResult.TotalVolatility,
		varResult.VaR["95"], varResult.VaR["99"], varResult.ExpectedShortfall["97.5"],
		factorJSON, "parametric", 0.95, s.tenantID,
	)
	return err
}
