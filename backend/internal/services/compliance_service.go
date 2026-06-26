// services/compliance_service.go
package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/audit"
	"github.com/hondyman/semlayer/backend/internal/wasm"
	"github.com/jmoiron/sqlx"
)

// ComplianceBreachRecord defines internal struct for breaching inserts
type ComplianceBreachRecord struct {
	BreachID       uuid.UUID
	EvaluationID   uuid.UUID
	RuleID         string
	PortfolioID    uuid.UUID
	ValuationDate  string
	Severity       string
	MetricValue    float64
	ThresholdValue float64
	Deviation      float64
	Message        string
	Status         string
	TenantID       uuid.UUID
}

// Portfolio basic struct
type Portfolio struct {
	ID       uuid.UUID `db:"id"`
	AUM      float64   `db:"aum"`
	Strategy string    `db:"strategy"`
}

// ComplianceService orchestrates compliance evaluation ETL
type ComplianceService struct {
	db       *sqlx.DB
	wasm     wasm.Engine
	audit    audit.Logger
	tenantID uuid.UUID
}

// NewComplianceService creates a new compliance evaluation service
func NewComplianceService(
	db *sqlx.DB,
	wasm wasm.Engine,
	audit audit.Logger,
	tenantID uuid.UUID,
) *ComplianceService {
	return &ComplianceService{
		db:       db,
		wasm:     wasm,
		audit:    audit,
		tenantID: tenantID,
	}
}

// ListPortfolios fetches portfolios for the tenant
func (s *ComplianceService) ListPortfolios(ctx context.Context, tenantID uuid.UUID) ([]Portfolio, error) {
	var portfolios []Portfolio
	err := s.db.SelectContext(ctx, &portfolios, "SELECT id, aum, strategy FROM edm.portfolio_master WHERE tenant_id = $1", tenantID)
	return portfolios, err
}

// EvaluatePortfolio runs compliance evaluation for a portfolio/date
// Aligns with Whitepaper §7: Semantic Execution Fabric
func (s *ComplianceService) EvaluatePortfolio(
	ctx context.Context,
	portfolioID uuid.UUID,
	valuationDate string,
) error {
	start := time.Now()

	// 1. Load active rules for portfolio/date (Semantic Design §3.4)
	rules, err := s.loadActiveRules(ctx, portfolioID, valuationDate)
	if err != nil {
		return fmt.Errorf("load active rules: %w", err)
	}
	if len(rules) == 0 {
		s.audit.Log(ctx, audit.ComplianceEvaluation{
			PortfolioID:    portfolioID,
			ValuationDate:  valuationDate,
			RulesEvaluated: 0,
			Status:         "NO_RULES",
		})
		return nil
	}

	// 2. Build compliance context from Position/Cash/Security masters
	complianceCtx, err := s.buildComplianceContext(ctx, portfolioID, valuationDate)
	if err != nil {
		return fmt.Errorf("build compliance context: %w", err)
	}

	// 3. Evaluate each rule via WASM (deterministic, tenant-isolated)
	var evaluations []wasm.ComplianceEvaluationResult
	var breaches []ComplianceBreachRecord
	for _, rule := range rules {
		result, err := s.wasm.EvaluateComplianceRule(ctx, rule, *complianceCtx)
		if err != nil {
			s.audit.Log(ctx, audit.WASMError{
				Function: "EvaluateComplianceRule",
				Error:    err.Error(),
				RuleID:   rule.RuleID,
			})
			continue
		}

		evaluations = append(evaluations, *result)

		// 4. Record breach if FAIL (Whitepaper §9: Operational Intelligence)
		if result.Status == "FAIL" {
			breaches = append(breaches, ComplianceBreachRecord{
				BreachID:       uuid.New(),
				EvaluationID:   uuid.New(), // Will be set after insert
				RuleID:         rule.RuleID,
				PortfolioID:    portfolioID,
				ValuationDate:  valuationDate,
				Severity:       rule.Severity,
				MetricValue:    result.MetricValue,
				ThresholdValue: result.ThresholdValue,
				Deviation:      result.MetricValue - result.ThresholdValue,
				Message: fmt.Sprintf("Rule %s failed: metric=%.4f, threshold=%.4f",
					rule.RuleCode, result.MetricValue, result.ThresholdValue),
				Status:   "OPEN",
				TenantID: s.tenantID,
			})
		}
	}

	// 5. Persist evaluations and breaches (atomic transaction)
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	for _, eval := range evaluations {
		evalID := uuid.New()
		if err := s.insertEvaluation(tx, evalID, eval, portfolioID, valuationDate); err != nil {
			return fmt.Errorf("insert evaluation: %w", err)
		}
		// Match Breach evaluationIDs
		for i := range breaches {
			if breaches[i].RuleID == eval.RuleID {
				breaches[i].EvaluationID = evalID
			}
		}
	}
	for _, breach := range breaches {
		if err := s.insertBreach(tx, breach); err != nil {
			return fmt.Errorf("insert breach: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	// 6. Audit completion (Whitepaper §9)
	s.audit.Log(ctx, audit.ComplianceEvaluation{
		PortfolioID:    portfolioID,
		ValuationDate:  valuationDate,
		RulesEvaluated: len(rules),
		BreachesFound:  len(breaches),
		ExecutionMs:    time.Since(start).Milliseconds(),
		Status:         "SUCCESS",
	})

	return nil
}

func (s *ComplianceService) loadActiveRules(
	ctx context.Context,
	portfolioID uuid.UUID,
	valuationDate string,
) ([]wasm.RuleConfig, error) {
	query := `
		SELECT rule_id, rule_code, severity, expression
		FROM edm.compliance_rule
		WHERE status = 'ACTIVE'
		  AND effective_from <= $1
		  AND (effective_to IS NULL OR effective_to >= $1)
		  AND (
			scope_type = 'PORTFOLIO' AND scope_value = $2
			OR scope_type = 'STRATEGY' AND scope_value = (SELECT strategy FROM edm.portfolio_master WHERE id = $2)
			OR scope_type = 'GLOBAL'
		  )
		  AND tenant_id = $3
	`
	var rules []struct {
		RuleID     string `db:"rule_id"`
		RuleCode   string `db:"rule_code"`
		Severity   string `db:"severity"`
		Expression string `db:"expression"`
	}
	err := s.db.SelectContext(ctx, &rules, query, valuationDate, portfolioID, s.tenantID)
	if err != nil {
		return nil, err
	}

	// Parse DSL expressions to RuleConfig (simplified)
	var configs []wasm.RuleConfig
	for _, r := range rules {
		config, err := parseRuleExpression(r.Expression)
		if err != nil {
			s.audit.Log(ctx, audit.RuleParseError{
				RuleID: r.RuleID,
				Error:  err.Error(),
			})
			continue
		}
		config.RuleID = r.RuleID
		config.RuleCode = r.RuleCode
		config.Severity = r.Severity
		configs = append(configs, config)
	}
	return configs, nil
}

func (s *ComplianceService) buildComplianceContext(
	ctx context.Context,
	portfolioID uuid.UUID,
	valuationDate string,
) (*wasm.ComplianceContext, error) {
	// Load portfolio metadata
	var portfolio struct {
		ID       uuid.UUID `db:"id"`
		AUM      float64   `db:"aum"`
		Strategy string    `db:"strategy"`
	}
	err := s.db.GetContext(ctx, &portfolio, `
		SELECT id, aum, strategy FROM edm.portfolio_master 
		WHERE id = $1 AND tenant_id = $2
	`, portfolioID, s.tenantID)
	if err != nil {
		return nil, err
	}

	// Load positions with security attributes
	var positions []struct {
		SecurityID  uuid.UUID `db:"security_id"`
		MarketValue float64   `db:"market_value"`
		IssuerID    string    `db:"issuer_id"`
		Sector      string    `db:"sector"`
		Country     string    `db:"country"`
		Rating      string    `db:"rating"`
		ESGScore    float64   `db:"esg_score"`
	}
	err = s.db.SelectContext(ctx, &positions, `
		SELECT p.security_id, p.market_value_base as market_value, s.issuer_id, s.sector, s.country, s.rating, s.esg_score
		FROM edm.position_master p
		JOIN edm.security_master s ON p.security_id = s.id
		WHERE p.portfolio_id = $1 AND p.position_date = $2 AND p.valid_to = 'infinity' AND p.tenant_id = $3
	`, portfolioID, valuationDate, s.tenantID)
	if err != nil {
		return nil, err
	}

	// Load cash balance
	var cash struct {
		ClosingBalance float64 `db:"closing_balance"`
		Currency       string  `db:"currency"`
	}
	err = s.db.GetContext(ctx, &cash, `
		SELECT closing_balance, currency FROM edm.cash_balance_master
		WHERE portfolio_id = $1 AND valuation_date = $2 AND valid_to = 'infinity' AND tenant_id = $3
	`, portfolioID, valuationDate, s.tenantID)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if err == sql.ErrNoRows {
		// Mock 0 cash balance
		cash.ClosingBalance = 0
		cash.Currency = "USD"
	}

	// Build context
	ctxObj := &wasm.ComplianceContext{
		TenantID: s.tenantID.String(),
	}
	ctxObj.Portfolio.ID = portfolio.ID.String()
	ctxObj.Portfolio.AUM = portfolio.AUM
	ctxObj.Portfolio.Strategy = portfolio.Strategy
	for _, p := range positions {
		ctxObj.Positions = append(ctxObj.Positions, struct {
			SecurityID  string  `json:"security_id" jsonschema:"required,format=uuid"`
			Quantity    float64 `json:"quantity"`
			MarketValue float64 `json:"market_value" jsonschema:"required,minimum=0"`
			IssuerID    string  `json:"issuer_id"`
			Sector      string  `json:"sector"`
			Country     string  `json:"country"`
			Rating      string  `json:"rating"`
			ESGScore    float64 `json:"esg_score" jsonschema:"minimum=0,maximum=10"`
		}{
			SecurityID:  p.SecurityID.String(),
			MarketValue: p.MarketValue,
			IssuerID:    p.IssuerID,
			Sector:      p.Sector,
			Country:     p.Country,
			Rating:      p.Rating,
			ESGScore:    p.ESGScore,
		})
	}
	ctxObj.Cash.ClosingBalance = cash.ClosingBalance
	ctxObj.Cash.Currency = cash.Currency

	return ctxObj, nil
}

func parseRuleExpression(expr string) (wasm.RuleConfig, error) {
	// Simplified DSL parser - in production, use your full DSL compiler
	// Example: "METRIC = Exposure.IssuerWeight('AAPL'); CONDITION METRIC <= 0.05"
	var config wasm.RuleConfig
	// Parse logic here...
	return config, nil
}

func (s *ComplianceService) insertEvaluation(
	tx *sqlx.Tx,
	evalID uuid.UUID,
	eval wasm.ComplianceEvaluationResult,
	portfolioID uuid.UUID,
	valuationDate string,
) error {
	query := `
		INSERT INTO edm.compliance_evaluation (
			evaluation_id, rule_id, portfolio_id, valuation_date,
			metric_value, threshold_value, result, details, evaluation_time_ms, tenant_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := tx.ExecContext(context.Background(), query,
		evalID, eval.RuleID, portfolioID, valuationDate,
		eval.MetricValue, eval.ThresholdValue, eval.Status,
		fmt.Sprintf("%v", eval.Details), eval.Lineage.ExecutionTimeMs, s.tenantID,
	)
	return err
}

func (s *ComplianceService) insertBreach(tx *sqlx.Tx, breach ComplianceBreachRecord) error {
	query := `
		INSERT INTO edm.compliance_breach (
			breach_id, evaluation_id, rule_id, portfolio_id, valuation_date,
			severity, metric_value, threshold_value, deviation, message, status, tenant_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err := tx.ExecContext(context.Background(), query,
		breach.BreachID, breach.EvaluationID, breach.RuleID,
		breach.PortfolioID, breach.ValuationDate,
		breach.Severity, breach.MetricValue, breach.ThresholdValue,
		breach.Deviation, breach.Message, breach.Status, s.tenantID,
	)
	return err
}
