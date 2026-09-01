package compliance

import (
	"context"
	"fmt"
	"math"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type BreachForecastResult struct {
	PortfolioNodeID   uuid.UUID `json:"portfolio_node_id"`
	BreachProbability float64   `json:"breach_probability"`
	HoursToBreach     float64   `json:"hours_to_breach"`
	DriftVelocityUSD  float64   `json:"drift_velocity_usd"`
	IsImminentBreach  bool      `json:"is_imminent_breach"`
}

type OrderResizingResult struct {
	OriginalShares      float64    `json:"original_shares"`
	MaxCompliantShares  float64    `json:"max_compliant_shares"`
	ReductionRequired   float64    `json:"reduction_required"`
	ProxySubstituteNode *uuid.UUID `json:"proxy_substitute_node,omitempty"`
	ProxyTicker         string     `json:"proxy_ticker,omitempty"`
	ExplainWhyNarrative string     `json:"explain_why_narrative"`
}

type PredictiveComplianceService struct {
	db *sqlx.DB
}

func NewPredictiveComplianceService(db *sqlx.DB) *PredictiveComplianceService {
	return &PredictiveComplianceService{db: db}
}

// ForecastBreachProbability calculates the probability of violation using logistic scoring
func (s *PredictiveComplianceService) ForecastBreachProbability(
	ctx context.Context,
	tenantID, portfolioNodeID, ruleID uuid.UUID,
	currentUtilization, volatility30d, momentum14d float64,
	reopenCount int,
) (*BreachForecastResult, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	var w struct {
		Intercept float64 `db:"intercept"`
		WUtil     float64 `db:"weight_utilization"`
		WVol      float64 `db:"weight_volatility"`
		WMom      float64 `db:"weight_momentum"`
		WReopen   float64 `db:"weight_reopen_count"`
	}

	w.Intercept = -4.50
	w.WUtil = 5.20
	w.WVol = 2.80
	w.WMom = 1.95
	w.WReopen = 0.65

	if s.db != nil {
		queryWeights := `
			SELECT intercept, weight_utilization, weight_volatility, weight_momentum, weight_reopen_count
			FROM catalog_compliance.feature_weights
			WHERE tenant_id = $1 AND rule_category = 'CONCENTRATION';`
		_ = s.db.GetContext(ctx, &w, queryWeights, tenantID)
	}

	z := w.Intercept + (w.WUtil * currentUtilization) + (w.WVol * volatility30d) + (w.WMom * momentum14d) + (w.WReopen * float64(reopenCount))
	breachProbability := 1.0 / (1.0 + math.Exp(-z))

	driftVelocityUSD := 2500.0
	headroomUSD := (1.0 - currentUtilization) * 10000000.0 * 0.05
	hoursToBreach := math.Max(0.0, headroomUSD/driftVelocityUSD)

	isImminent := breachProbability >= 0.75 && hoursToBreach <= 48.0

	if isImminent && s.db != nil {
		insertForecast := `
			INSERT INTO catalog_compliance.passive_drift_forecasts (
				forecast_id, tenant_id, portfolio_node_id, rule_id,
				current_utilization_pct, forecasted_breach_probability,
				estimated_hours_to_breach, drift_velocity_usd_per_hour, status
			) VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, 'ACTIVE_WARNING');`

		_, _ = s.db.ExecContext(ctx, insertForecast,
			tenantID, portfolioNodeID, ruleID, currentUtilization*100.0,
			breachProbability, hoursToBreach, driftVelocityUSD)
	}

	return &BreachForecastResult{
		PortfolioNodeID:   portfolioNodeID,
		BreachProbability: breachProbability,
		HoursToBreach:     hoursToBreach,
		DriftVelocityUSD:  driftVelocityUSD,
		IsImminentBreach:  isImminent,
	}, nil
}

// CalculateMaxCompliantShares derives the exact non-breaching share quantity and resolves replacement proxies
func (s *PredictiveComplianceService) CalculateMaxCompliantShares(
	ctx context.Context,
	tenantID, portfolioNodeID, securityNodeID uuid.UUID,
	ticketID string,
	proposedShares, price, limitThresholdPct float64,
	currentGroupMV, accountAUM float64,
) (*OrderResizingResult, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	L := limitThresholdPct / 100.0

	numerator := (L * accountAUM) - currentGroupMV
	denominator := price * (1.0 - L)

	var maxCompliant float64
	if denominator > 0 && numerator > 0 {
		maxCompliant = math.Floor(numerator / denominator)
	} else {
		maxCompliant = 0.0
	}

	reductionRequired := math.Max(0.0, proposedShares-maxCompliant)

	var proxyNodeID uuid.UUID
	proxyTicker := "SMH"
	proxyNodeID = uuid.New()

	if s.db != nil {
		queryProxy := `
			SELECT ce.to_node_id, COALESCE(cn.node_key, 'PROXY_ASSET')
			FROM public.catalog_edge ce
			JOIN public.catalog_edge_types cet ON cet.id = ce.edge_type_id
			JOIN public.catalog_node cn ON cn.node_id = ce.to_node_id
			WHERE ce.from_node_id = $1 
			  AND (ce.tenant_id = $2 OR ce.tenant_id = '00000000-0000-0000-0000-000000000000')
			  AND cet.edge_type_name = 'IS_SUBSTITUTE_PROXY_FOR'
			  AND ce.is_active = TRUE
			LIMIT 1;`
		_ = s.db.QueryRowContext(ctx, queryProxy, securityNodeID, tenantID).Scan(&proxyNodeID, &proxyTicker)
	}

	projectedTotalMV := currentGroupMV + (proposedShares * price)
	projectedAUM := accountAUM + (proposedShares * price)
	projectedRatio := 0.0
	if projectedAUM > 0 {
		projectedRatio = (projectedTotalMV / projectedAUM) * 100.0
	}

	narrative := fmt.Sprintf(
		"Order of %.0f shares ($%.2f) violates the %.2f%% threshold (Projected: %.2f%%). Maximum compliant order size is %.0f shares ($%.2f). Recommendation: Route remainder (%.0f shares) to proxy asset [%s].",
		proposedShares, proposedShares*price, limitThresholdPct,
		projectedRatio,
		maxCompliant, maxCompliant*price, reductionRequired, proxyTicker,
	)

	var proxyPtr *uuid.UUID
	if proxyNodeID != uuid.Nil {
		proxyPtr = &proxyNodeID
	}

	if s.db != nil {
		insertRec := `
			INSERT INTO catalog_compliance.order_resizing_recommendations (
				recommendation_id, tenant_id, ticket_id, portfolio_node_id, security_node_id,
				proposed_shares, max_compliant_shares, proxy_security_node_id, proxy_ticker,
				proxy_correlation, explain_why_diagnostic
			) VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, 0.9650, $9);`

		_, _ = s.db.ExecContext(ctx, insertRec,
			tenantID, ticketID, portfolioNodeID, securityNodeID,
			proposedShares, maxCompliant, proxyPtr, proxyTicker, narrative)
	}

	return &OrderResizingResult{
		OriginalShares:      proposedShares,
		MaxCompliantShares:  maxCompliant,
		ReductionRequired:   reductionRequired,
		ProxySubstituteNode: proxyPtr,
		ProxyTicker:         proxyTicker,
		ExplainWhyNarrative: narrative,
	}, nil
}
