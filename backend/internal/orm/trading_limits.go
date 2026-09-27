package orm

import (
	"context"
	"database/sql"
)

// LimitBreach describes a limit that would be breached by a hypothetical order.
type LimitBreach struct {
	LimitID    string
	LimitType  string
	ScopeType  string
	LimitValue float64
	Proposed   float64
	BreachPct  float64
	Hard       bool
}

// CheckTradingLimits evaluates the active trading_limit rows applicable to
// the proposed order. Returns all breaches; empty slice means clear.
func CheckTradingLimits(ctx context.Context, tx *sql.Tx, tenantID,
	scopeType, scopeID, limitType string, proposedValue float64) ([]LimitBreach, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT id, limit_type, scope_type, limit_value, hard_breach
		FROM orm.trading_limit
		WHERE tenant_id = $1
		  AND scope_type = $2
		  AND (scope_id = $3 OR scope_id IS NULL)
		  AND limit_type = $4
		  AND is_current = true
		  AND effective_from <= CURRENT_DATE
		  AND (effective_to IS NULL OR effective_to >= CURRENT_DATE)
	`, tenantID, scopeType, scopeID, limitType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var breaches []LimitBreach
	for rows.Next() {
		var b LimitBreach
		if err := rows.Scan(&b.LimitID, &b.LimitType, &b.ScopeType, &b.LimitValue, &b.Hard); err != nil {
			return nil, err
		}
		if proposedValue > b.LimitValue {
			b.Proposed = proposedValue
			b.BreachPct = (proposedValue - b.LimitValue) / b.LimitValue * 100
			breaches = append(breaches, b)
		}
	}
	return breaches, rows.Err()
}
