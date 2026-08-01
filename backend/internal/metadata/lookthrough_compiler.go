package metadata

import (
	"fmt"

	"github.com/google/uuid"
)

// LookThroughQueryConfig carries the parameters for compiling a federated
// SQL query that explodes fund/ETF constituents into direct issuer exposure
// and aggregates total effective_exposure_pct for compliance evaluation.
//
// The output SQL is consumed by the StarRocks engine. The final aggregated
// column (effective_exposure_pct) is then fed directly into the Go VM as
// a flat field on a FastRecord, allowing 8-nanosecond rule checks like
// "effective_exposure_pct > 0.25" without re-joining anything at rule time.
type LookThroughQueryConfig struct {
	TenantID       uuid.UUID `json:"tenant_id"`
	PortfolioID    string    `json:"portfolio_id"`
	TargetIssuerID string    `json:"target_issuer_id"`
	WatermarkDate  string    `json:"watermark_date"`
}

// BuildLookThroughExposureSQL constructs a federated CTE that produces one
// row per (portfolio, target_issuer) tuple containing direct exposure,
// look-through (indirect via fund constituents) exposure, and the combined
// effective_exposure_pct against portfolio total AUM.
//
// Table names match the production schema (ibor_positions plural, the
// fund_constituents reference table). The function returns the SQL string
// and the positional argument list to bind.
//
// Args (positional, $1..$4):
//
//	$1: tenant_id (uuid)
//	$2: portfolio_id (text)
//	$3: watermark date (text, ISO-8601, e.g. "2026-07-31")
//	$4: target_issuer_id (text)
func BuildLookThroughExposureSQL(cfg LookThroughQueryConfig) (string, []any, error) {
	if cfg.TenantID == uuid.Nil {
		return "", nil, fmt.Errorf("tenant_id is required for look-through compilation")
	}
	if cfg.PortfolioID == "" {
		return "", nil, fmt.Errorf("portfolio_id is required")
	}
	if cfg.TargetIssuerID == "" {
		return "", nil, fmt.Errorf("target_issuer_id is required")
	}
	if cfg.WatermarkDate == "" {
		return "", nil, fmt.Errorf("watermark_date is required")
	}

	sql := `
WITH direct_exposure AS (
    SELECT
        p.tenant_id,
        p.portfolio_id,
        p.issuer_id,
        SUM(p.market_value) AS direct_val
    FROM public.ibor_positions p
    WHERE p.tenant_id = $1
      AND p.portfolio_id = $2
      AND p.as_of_date >= $3
    GROUP BY p.tenant_id, p.portfolio_id, p.issuer_id
),
indirect_exposure AS (
    SELECT
        p.tenant_id,
        p.portfolio_id,
        c.constituent_issuer_id AS issuer_id,
        SUM(p.market_value * c.position_weight_pct) AS indirect_val
    FROM public.ibor_positions p
    JOIN public.fund_constituents c
      ON p.instrument_id = c.fund_instrument_id
     AND p.tenant_id = c.tenant_id
    WHERE p.tenant_id = $1
      AND p.portfolio_id = $2
      AND p.as_of_date >= $3
    GROUP BY p.tenant_id, p.portfolio_id, c.constituent_issuer_id
),
portfolio_aum AS (
    SELECT
        tenant_id,
        portfolio_id,
        SUM(market_value) AS total_aum
    FROM public.ibor_positions
    WHERE tenant_id = $1 AND portfolio_id = $2 AND as_of_date >= $3
    GROUP BY tenant_id, portfolio_id
)
SELECT
    aum.portfolio_id,
    $4 AS target_issuer_id,
    COALESCE(d.direct_val, 0) AS direct_market_value,
    COALESCE(i.indirect_val, 0) AS indirect_market_value,
    (COALESCE(d.direct_val, 0) + COALESCE(i.indirect_val, 0)) AS total_effective_exposure,
    aum.total_aum,
    ((COALESCE(d.direct_val, 0) + COALESCE(i.indirect_val, 0)) / NULLIF(aum.total_aum, 0)) AS effective_exposure_pct
FROM portfolio_aum aum
LEFT JOIN direct_exposure d
  ON aum.tenant_id = d.tenant_id AND aum.portfolio_id = d.portfolio_id AND d.issuer_id = $4
LEFT JOIN indirect_exposure i
  ON aum.tenant_id = i.tenant_id AND aum.portfolio_id = i.portfolio_id AND i.issuer_id = $4
WHERE aum.tenant_id = $1;
`

	args := []any{cfg.TenantID, cfg.PortfolioID, cfg.WatermarkDate, cfg.TargetIssuerID}
	return sql, args, nil
}
