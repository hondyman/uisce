package engine

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// PortfolioPosition represents a security holding within an evaluated portfolio.
type PortfolioPosition struct {
	SecurityID    string          `json:"security_id"`
	Symbol        string          `json:"symbol"`
	IssuerID      string          `json:"issuer_id"`
	IssuerName    string          `json:"issuer_name"`
	MarketValue   decimal.Decimal `json:"market_value"`
	Weight        decimal.Decimal `json:"weight"` // MarketValue / NAV
	AssetClass    string          `json:"asset_class"`
	Sector        string          `json:"sector"`
	CountryOfRisk string          `json:"country_of_risk"`
	Is144A        bool            `json:"is_144a"`
	IsQIBEligible bool            `json:"is_qib_eligible"`
	CreditRating  string          `json:"credit_rating"`
}

// PortfolioState represents the aggregated point-in-time state of an account's portfolio.
type PortfolioState struct {
	TenantID      uuid.UUID                  `json:"tenant_id"`
	AccountID     uuid.UUID                  `json:"account_id"`
	AsOfDate      time.Time                  `json:"as_of_date"`
	NAV           decimal.Decimal            `json:"nav"`
	GrossExposure decimal.Decimal            `json:"gross_exposure"`
	NetExposure   decimal.Decimal            `json:"net_exposure"`
	CashBalance   decimal.Decimal            `json:"cash_balance"`
	MarginLimit   decimal.Decimal            `json:"margin_limit"`
	Positions     []PortfolioPosition        `json:"positions"`
	Metrics       map[string]decimal.Decimal `json:"metrics,omitempty"`
}

// PostTradeEvaluationResult represents a compliance decision and generated finding.
type PostTradeEvaluationResult struct {
	FindingID           uuid.UUID              `json:"finding_id"`
	TenantID            uuid.UUID              `json:"tenant_id"`
	RuleID              uuid.UUID              `json:"rule_id"`
	RuleVersion         int                    `json:"rule_version"`
	RuleCode            string                 `json:"rule_code"`
	AccountID           uuid.UUID              `json:"account_id"`
	AsOfDate            time.Time              `json:"as_of_date"`
	EvaluationTier      string                 `json:"evaluation_tier"`
	Status              string                 `json:"status"` // OPEN | SUPERSEDED | RESOLVED | DISMISSED
	Action              string                 `json:"action"` // BREACHED | BREACH_CONFIRMED | WITHIN_LIMITS | WARNING
	FindingSeverity     string                 `json:"finding_severity"`
	SupersedesFindingID *uuid.UUID             `json:"supersedes_finding_id,omitempty"`
	SupersededReason    string                 `json:"superseded_reason,omitempty"`
	LineageHash         string                 `json:"lineage_hash"`
	PortfolioSnapshotID uuid.UUID              `json:"portfolio_snapshot_id"`
	Details             map[string]interface{} `json:"details"`
}

// PostTradeEvaluator orchestrates post-trade portfolio metric calculation, rule evaluation, and finding persistence.
type PostTradeEvaluator struct {
	db *sql.DB
}

// NewPostTradeEvaluator constructs an evaluator backed by Postgres.
func NewPostTradeEvaluator(db *sql.DB) *PostTradeEvaluator {
	return &PostTradeEvaluator{db: db}
}

// ComputePortfolioMetrics calculates aggregate compliance metrics across positions.
func (e *PostTradeEvaluator) ComputePortfolioMetrics(state *PortfolioState) map[string]decimal.Decimal {
	metrics := make(map[string]decimal.Decimal)
	if state.NAV.IsZero() {
		return metrics
	}

	// 1. Issuer aggregations for UCITS 5/10/40
	issuerExposureMap := make(map[string]decimal.Decimal)
	for i := range state.Positions {
		pos := &state.Positions[i]
		if state.NAV.GreaterThan(decimal.Zero) {
			pos.Weight = pos.MarketValue.Div(state.NAV)
		}
		issuerExposureMap[pos.IssuerID] = issuerExposureMap[pos.IssuerID].Add(pos.Weight)
	}

	var maxSingleIssuer decimal.Decimal
	var ucitsAggregateAbove5Pct decimal.Decimal
	fivePct := decimal.RequireFromString("0.050000")

	for _, exp := range issuerExposureMap {
		if exp.GreaterThan(maxSingleIssuer) {
			maxSingleIssuer = exp
		}
		if exp.GreaterThan(fivePct) {
			ucitsAggregateAbove5Pct = ucitsAggregateAbove5Pct.Add(exp)
		}
	}

	metrics["portfolio.max_single_issuer_exposure"] = maxSingleIssuer
	metrics["portfolio.ucits_aggregate_above_5pct_exposure"] = ucitsAggregateAbove5Pct

	// 2. SEC 144A / QIB Illiquid Asset Exposure
	var restricted144aExposure decimal.Decimal
	for _, pos := range state.Positions {
		if pos.Is144A && !pos.IsQIBEligible {
			restricted144aExposure = restricted144aExposure.Add(pos.Weight)
		}
	}
	metrics["portfolio.restricted_144a_exposure_pct"] = restricted144aExposure

	// 3. Margin Utilization
	if state.MarginLimit.GreaterThan(decimal.Zero) {
		borrowed := state.GrossExposure.Sub(state.CashBalance)
		if borrowed.IsNegative() {
			borrowed = decimal.Zero
		}
		marginUtilization := borrowed.Div(state.MarginLimit)
		metrics["portfolio.margin_utilization_pct"] = marginUtilization
	} else {
		metrics["portfolio.margin_utilization_pct"] = decimal.Zero
	}

	state.Metrics = metrics
	return metrics
}

// ComputeStateContentHash produces a deterministic SHA-256 fingerprint of the portfolio input.
func (e *PostTradeEvaluator) ComputeStateContentHash(state PortfolioState) string {
	raw, _ := json.Marshal(struct {
		TenantID      string              `json:"tenant_id"`
		AccountID     string              `json:"account_id"`
		AsOfDate      string              `json:"as_of_date"`
		NAV           string              `json:"nav"`
		GrossExposure string              `json:"gross_exposure"`
		NetExposure   string              `json:"net_exposure"`
		CashBalance   string              `json:"cash_balance"`
		Positions     []PortfolioPosition `json:"positions"`
	}{
		TenantID:      state.TenantID.String(),
		AccountID:     state.AccountID.String(),
		AsOfDate:      state.AsOfDate.Format("2006-01-02"),
		NAV:           state.NAV.StringFixed(6),
		GrossExposure: state.GrossExposure.StringFixed(6),
		NetExposure:   state.NetExposure.StringFixed(6),
		CashBalance:   state.CashBalance.StringFixed(6),
		Positions:     state.Positions,
	})

	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// ComputeDeterministicFindingID generates an RFC 4122 UUIDv5 lineage identifier.
func (e *PostTradeEvaluator) ComputeDeterministicFindingID(
	tenantID uuid.UUID,
	ruleCode string,
	accountID uuid.UUID,
	asOfDate time.Time,
	inputHash string,
) (uuid.UUID, string) {
	lineageSeed := fmt.Sprintf(
		"tenant:%s:rule:%s:account:%s:asOfDate:%s:inputHash:%s",
		tenantID.String(),
		ruleCode,
		accountID.String(),
		asOfDate.Format("2006-01-02"),
		inputHash,
	)
	findingID := uuid.NewSHA1(uuid.NameSpaceURL, []byte(lineageSeed))
	return findingID, lineageSeed
}

// EvaluateAndPersist executes post-trade evaluation against an account's portfolio state.
func (e *PostTradeEvaluator) EvaluateAndPersist(
	ctx context.Context,
	state PortfolioState,
) ([]PostTradeEvaluationResult, error) {
	metrics := e.ComputePortfolioMetrics(&state)
	contentHash := e.ComputeStateContentHash(state)

	tx, err := e.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, fmt.Errorf("begin tx failed: %w", err)
	}
	defer tx.Rollback()

	// Fence tenant context for RLS
	_, err = tx.ExecContext(ctx, "SELECT set_config('app.current_tenant', $1, true)", state.TenantID.String())
	if err != nil {
		return nil, fmt.Errorf("set tenant context failed: %w", err)
	}

	// 1. Upsert Portfolio Snapshot
	var snapshotID uuid.UUID
	positionsJSON, _ := json.Marshal(state.Positions)
	metricsJSON, _ := json.Marshal(metrics)

	err = tx.QueryRowContext(ctx, `
		INSERT INTO compliance.compliance_portfolio_snapshot (
			tenant_id, account_id, as_of_date, nav, gross_exposure, net_exposure,
			cash_balance, positions, metrics, content_hash
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9::jsonb, $10
		)
		ON CONFLICT (tenant_id, account_id, as_of_date) DO UPDATE SET
			nav = EXCLUDED.nav,
			gross_exposure = EXCLUDED.gross_exposure,
			net_exposure = EXCLUDED.net_exposure,
			cash_balance = EXCLUDED.cash_balance,
			positions = EXCLUDED.positions,
			metrics = EXCLUDED.metrics,
			content_hash = EXCLUDED.content_hash
		RETURNING id
	`, state.TenantID, state.AccountID, state.AsOfDate.Format("2006-01-02"),
		state.NAV.StringFixed(6), state.GrossExposure.StringFixed(6), state.NetExposure.StringFixed(6),
		state.CashBalance.StringFixed(6), positionsJSON, metricsJSON, contentHash,
	).Scan(&snapshotID)
	if err != nil {
		return nil, fmt.Errorf("upsert portfolio snapshot failed: %w", err)
	}

	// 2. Fetch Active Post-Trade Rules for Tenant
	rows, err := tx.QueryContext(ctx, `
		SELECT r.id, r.rule_code, r.current_version, r.severity, r.parameter_thresholds
		FROM compliance.compliance_rule r
		WHERE (r.tenant_id = $1 OR r.tenant_id = (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1))
		  AND r.rule_phase = 'POST_TRADE'
		  AND r.is_active = true
		  AND r.valid_to IS NULL
		ORDER BY r.priority DESC
	`, state.TenantID)
	if err != nil {
		return nil, fmt.Errorf("query post-trade rules failed: %w", err)
	}
	defer rows.Close()

	type RuleEntry struct {
		ID         uuid.UUID
		Code       string
		Version    int
		Severity   string
		Thresholds map[string]interface{}
	}
	var rules []RuleEntry
	for rows.Next() {
		var re RuleEntry
		var threshBytes []byte
		if err := rows.Scan(&re.ID, &re.Code, &re.Version, &re.Severity, &threshBytes); err != nil {
			return nil, fmt.Errorf("scan rule entry failed: %w", err)
		}
		_ = json.Unmarshal(threshBytes, &re.Thresholds)
		rules = append(rules, re)
	}

	var results []PostTradeEvaluationResult

	// 3. Evaluate Each Post-Trade Rule
	for _, rule := range rules {
		res, evalErr := e.evaluateRule(rule.ID, rule.Code, rule.Version, rule.Severity, rule.Thresholds, state, metrics, snapshotID, contentHash)
		if evalErr != nil {
			return nil, evalErr
		}

		// 4. Handle Finding State Persistence with Deterministic Lineage and Supersession
		err = e.persistFindingLifecycle(ctx, tx, res)
		if err != nil {
			return nil, fmt.Errorf("persist finding lifecycle for %s failed: %w", rule.Code, err)
		}

		results = append(results, *res)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit post-trade evaluation tx failed: %w", err)
	}

	return results, nil
}

// evaluateRule evaluates a single post-trade rule against the computed portfolio metrics.
func (e *PostTradeEvaluator) evaluateRule(
	ruleID uuid.UUID,
	ruleCode string,
	ruleVersion int,
	severity string,
	thresholds map[string]interface{},
	state PortfolioState,
	metrics map[string]decimal.Decimal,
	snapshotID uuid.UUID,
	contentHash string,
) (*PostTradeEvaluationResult, error) {
	findingID, lineageHash := e.ComputeDeterministicFindingID(state.TenantID, ruleCode, state.AccountID, state.AsOfDate, contentHash)

	res := &PostTradeEvaluationResult{
		FindingID:           findingID,
		TenantID:            state.TenantID,
		RuleID:              ruleID,
		RuleVersion:         ruleVersion,
		RuleCode:            ruleCode,
		AccountID:           state.AccountID,
		AsOfDate:            state.AsOfDate,
		EvaluationTier:      "POST_TRADE",
		FindingSeverity:     severity,
		LineageHash:         lineageHash,
		PortfolioSnapshotID: snapshotID,
		Details:             make(map[string]interface{}),
	}

	switch ruleCode {
	case "UCITS_5_10_40":
		maxAggLimit := decimal.RequireFromString("0.400000")
		if v, ok := thresholds["max_aggregate_above_5pct_pct"].(string); ok {
			if d, err := decimal.NewFromString(v); err == nil {
				maxAggLimit = d
			}
		}
		maxSingleLimit := decimal.RequireFromString("0.100000")
		if v, ok := thresholds["max_single_issuer_pct"].(string); ok {
			if d, err := decimal.NewFromString(v); err == nil {
				maxSingleLimit = d
			}
		}

		aggExposure := metrics["portfolio.ucits_aggregate_above_5pct_exposure"]
		maxSingle := metrics["portfolio.max_single_issuer_exposure"]

		res.Details["ucits_aggregate_above_5pct_exposure"] = aggExposure.StringFixed(6)
		res.Details["max_single_issuer_exposure"] = maxSingle.StringFixed(6)
		res.Details["max_aggregate_above_5pct_pct"] = maxAggLimit.StringFixed(6)
		res.Details["max_single_issuer_pct"] = maxSingleLimit.StringFixed(6)

		if maxSingle.GreaterThan(maxSingleLimit) {
			res.Action = "BREACHED"
			res.Status = "OPEN"
			res.Details["breach_reason"] = fmt.Sprintf("Single issuer exposure %s exceeds max limit %s", maxSingle.StringFixed(4), maxSingleLimit.StringFixed(4))
		} else if aggExposure.GreaterThan(maxAggLimit) {
			res.Action = "BREACHED"
			res.Status = "OPEN"
			res.Details["breach_reason"] = fmt.Sprintf("Aggregate exposure of >5%% issuers (%s) exceeds 40%% limit (%s)", aggExposure.StringFixed(4), maxAggLimit.StringFixed(4))
		} else {
			res.Action = "WITHIN_LIMITS"
			res.Status = "RESOLVED"
		}

	case "SEC_144A_QIB_HOLDING":
		maxLimit := decimal.RequireFromString("0.150000")
		if v, ok := thresholds["max_144a_non_qib_pct"].(string); ok {
			if d, err := decimal.NewFromString(v); err == nil {
				maxLimit = d
			}
		}

		exp := metrics["portfolio.restricted_144a_exposure_pct"]
		res.Details["restricted_144a_exposure_pct"] = exp.StringFixed(6)
		res.Details["max_144a_non_qib_pct"] = maxLimit.StringFixed(6)

		if exp.GreaterThan(maxLimit) {
			res.Action = "BREACHED"
			res.Status = "OPEN"
			res.Details["breach_reason"] = fmt.Sprintf("Restricted 144A non-QIB exposure %s exceeds 15%% limit %s", exp.StringFixed(4), maxLimit.StringFixed(4))
		} else {
			res.Action = "WITHIN_LIMITS"
			res.Status = "RESOLVED"
		}

	case "MARGIN_UTILIZATION_80":
		maxLimit := decimal.RequireFromString("0.800000")
		if v, ok := thresholds["max_margin_utilization_pct"].(string); ok {
			if d, err := decimal.NewFromString(v); err == nil {
				maxLimit = d
			}
		}

		utilization := metrics["portfolio.margin_utilization_pct"]
		res.Details["margin_utilization_pct"] = utilization.StringFixed(6)
		res.Details["max_margin_utilization_pct"] = maxLimit.StringFixed(6)

		if utilization.GreaterThan(maxLimit) {
			res.Action = "WARNING"
			res.Status = "OPEN"
			res.Details["breach_reason"] = fmt.Sprintf("Margin utilization %s exceeds 80%% warning threshold %s", utilization.StringFixed(4), maxLimit.StringFixed(4))
		} else {
			res.Action = "WITHIN_LIMITS"
			res.Status = "RESOLVED"
		}

	default:
		res.Action = "WITHIN_LIMITS"
		res.Status = "RESOLVED"
	}

	return res, nil
}

// persistFindingLifecycle manages the deterministic insertion, supersession, or resolution of compliance findings.
func (e *PostTradeEvaluator) persistFindingLifecycle(
	ctx context.Context,
	tx *sql.Tx,
	res *PostTradeEvaluationResult,
) error {
	// Look for existing OPEN finding for this rule and account on as_of_date
	var existingID uuid.UUID
	var existingLineage string
	var existingSeverity, existingAction string

	err := tx.QueryRowContext(ctx, `
		SELECT id, lineage_hash, finding_severity, action
		FROM compliance.compliance_finding
		WHERE tenant_id = $1 AND rule_code = $2 AND account_id = $3 AND as_of_date = $4 AND status = 'OPEN'
		FOR UPDATE
	`, res.TenantID, res.RuleCode, res.AccountID, res.AsOfDate.Format("2006-01-02")).Scan(
		&existingID, &existingLineage, &existingSeverity, &existingAction,
	)

	hasExistingOpen := (err == nil)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("query existing open finding failed: %w", err)
	}

	detailsJSON, _ := json.Marshal(res.Details)

	if res.Action == "BREACHED" || res.Action == "WARNING" {
		// Rule is currently in breach / warning
		if hasExistingOpen {
			if existingID == res.FindingID {
				// Exact same input and lineage -> idempotent no-op
				return nil
			}

			// Restatement detected: Old finding is superseded by new calculation
			_, err = tx.ExecContext(ctx, `
				UPDATE compliance.compliance_finding
				SET status = 'SUPERSEDED',
					superseded_reason = 'SUPERSEDED_BY_RESTATEMENT',
					updated_at = now()
				WHERE id = $1
			`, existingID)
			if err != nil {
				return fmt.Errorf("supersede existing finding failed: %w", err)
			}

			res.SupersedesFindingID = &existingID
			res.SupersededReason = "SUPERSEDED_BY_RESTATEMENT"
		}

		// Insert the new finding
		_, err = tx.ExecContext(ctx, `
			INSERT INTO compliance.compliance_finding (
				id, tenant_id, rule_id, rule_version, rule_code, account_id,
				as_of_date, evaluation_tier, status, action, finding_severity,
				supersedes_finding_id, superseded_reason, lineage_hash,
				portfolio_snapshot_id, details
			) VALUES (
				$1, $2, $3, $4, $5, $6,
				$7, $8, 'OPEN', $9, $10,
				$11, $12, $13,
				$14, $15::jsonb
			)
			ON CONFLICT (id) DO UPDATE SET
				details = EXCLUDED.details,
				updated_at = now()
		`, res.FindingID, res.TenantID, res.RuleID, res.RuleVersion, res.RuleCode, res.AccountID,
			res.AsOfDate.Format("2006-01-02"), res.EvaluationTier, res.Action, res.FindingSeverity,
			res.SupersedesFindingID, res.SupersededReason, res.LineageHash,
			res.PortfolioSnapshotID, detailsJSON,
		)
		if err != nil {
			return fmt.Errorf("insert new compliance finding failed: %w", err)
		}
	} else {
		// Rule is WITHIN_LIMITS
		if hasExistingOpen {
			// Portfolio restatement resolved prior breach
			_, err = tx.ExecContext(ctx, `
				UPDATE compliance.compliance_finding
				SET status = 'RESOLVED',
					resolution_notes = 'Remediated/resolved by portfolio restatement',
					resolved_by = 'system_post_trade_evaluator',
					resolved_at = now(),
					updated_at = now()
				WHERE id = $1
			`, existingID)
			if err != nil {
				return fmt.Errorf("resolve existing finding failed: %w", err)
			}
		}
	}

	return nil
}
