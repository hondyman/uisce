package risk

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// RiskRepository defines persistence operations for the Risk domain
type RiskRepository interface {
	ListRiskFactors(ctx context.Context, category string) ([]RiskFactor, error)
	GetSecurityFactorExposures(ctx context.Context, securityID uuid.UUID, asOfDate time.Time) ([]SecurityFactorExposure, error)
	GetPortfolioRisk(ctx context.Context, portfolioID uuid.UUID, asOfDate time.Time) (*PortfolioRisk, error)
	ListRiskScenarios(ctx context.Context, activeOnly bool) ([]RiskScenario, error)
	GetScenarioResult(ctx context.Context, scenarioID, portfolioID uuid.UUID, asOfDate time.Time) (*RiskScenarioResult, error)
}

type pgRiskRepo struct {
	db *sqlx.DB
}

// NewRiskRepository creates a new Postgres-backed RiskRepository
func NewRiskRepository(db *sqlx.DB) RiskRepository {
	return &pgRiskRepo{db: db}
}

func (r *pgRiskRepo) ListRiskFactors(ctx context.Context, category string) ([]RiskFactor, error) {
	tenantID, ok := ctx.Value("tenant_id").(uuid.UUID)
	if !ok {
		return nil, fmt.Errorf("tenant_id not found in context")
	}

	query := `SELECT * FROM edm.risk_factor WHERE tenant_id = $1`
	args := []interface{}{tenantID}

	if category != "" {
		query += ` AND category = $2`
		args = append(args, category)
	}
	query += ` ORDER BY factor_name ASC`

	var factors []RiskFactor
	err := r.db.SelectContext(ctx, &factors, query, args...)
	return factors, err
}

func (r *pgRiskRepo) GetSecurityFactorExposures(ctx context.Context, securityID uuid.UUID, asOfDate time.Time) ([]SecurityFactorExposure, error) {
	tenantID, ok := ctx.Value("tenant_id").(uuid.UUID)
	if !ok {
		return nil, fmt.Errorf("tenant_id not found in context")
	}

	query := `
		SELECT * FROM edm.security_factor_exposure
		WHERE tenant_id = $1 AND security_id = $2 AND as_of_date = $3
	`

	var exposures []SecurityFactorExposure
	err := r.db.SelectContext(ctx, &exposures, query, tenantID, securityID, asOfDate)
	return exposures, err
}

func (r *pgRiskRepo) GetPortfolioRisk(ctx context.Context, portfolioID uuid.UUID, asOfDate time.Time) (*PortfolioRisk, error) {
	tenantID, ok := ctx.Value("tenant_id").(uuid.UUID)
	if !ok {
		return nil, fmt.Errorf("tenant_id not found in context")
	}

	query := `
		SELECT * FROM edm.portfolio_risk
		WHERE tenant_id = $1 AND portfolio_id = $2 AND valuation_date = $3
	`

	var risk PortfolioRisk
	err := r.db.GetContext(ctx, &risk, query, tenantID, portfolioID, asOfDate)
	if err != nil {
		return nil, err
	}
	return &risk, nil
}

func (r *pgRiskRepo) ListRiskScenarios(ctx context.Context, activeOnly bool) ([]RiskScenario, error) {
	tenantID, ok := ctx.Value("tenant_id").(uuid.UUID)
	if !ok {
		return nil, fmt.Errorf("tenant_id not found in context")
	}

	query := `SELECT * FROM edm.risk_scenario WHERE tenant_id = $1`
	if activeOnly {
		query += ` AND status = 'ACTIVE'`
	}
	query += ` ORDER BY scenario_name ASC`

	var scenarios []RiskScenario
	err := r.db.SelectContext(ctx, &scenarios, query, tenantID)
	return scenarios, err
}

func (r *pgRiskRepo) GetScenarioResult(ctx context.Context, scenarioID, portfolioID uuid.UUID, asOfDate time.Time) (*RiskScenarioResult, error) {
	tenantID, ok := ctx.Value("tenant_id").(uuid.UUID)
	if !ok {
		return nil, fmt.Errorf("tenant_id not found in context")
	}

	query := `
		SELECT * FROM edm.risk_scenario_result
		WHERE tenant_id = $1 AND scenario_id = $2 AND portfolio_id = $3 AND valuation_date = $4
	`

	var result RiskScenarioResult
	err := r.db.GetContext(ctx, &result, query, tenantID, scenarioID, portfolioID, asOfDate)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
