package orm

import (
	"context"
	"database/sql"
)

// RouteOrder picks the highest-priority active routing rule matching the
// order attributes. Returns nil if no rule matches.
func RouteOrder(ctx context.Context, tx *sql.Tx, tenantID,
	assetClass, orderType, side string, qty, notional float64) (*RoutingDecision, error) {
	var d RoutingDecision
	err := tx.QueryRowContext(ctx, `
		SELECT venue_mic, broker_id, routing_strategy, algo_cd
		FROM orm.routing_rule
		WHERE tenant_id = $1
		  AND is_active = true
		  AND effective_from <= CURRENT_DATE
		  AND (effective_to IS NULL OR effective_to >= CURRENT_DATE)
		  AND (asset_class_cd IS NULL OR asset_class_cd = $2)
		  AND (order_type IS NULL OR order_type = $3)
		  AND (side IS NULL OR side = $4)
		  AND (quantity_min IS NULL OR $5 >= quantity_min)
		  AND (quantity_max IS NULL OR $5 <= quantity_max)
		  AND (notional_min IS NULL OR $6 >= notional_min)
		  AND (notional_max IS NULL OR $6 <= notional_max)
		ORDER BY priority ASC
		LIMIT 1
	`, tenantID, assetClass, orderType, side, qty, notional).
		Scan(&d.VenueMIC, &d.BrokerID, &d.Strategy, &d.AlgoCd)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &d, err
}

type RoutingDecision struct {
	VenueMIC sql.NullString
	BrokerID sql.NullString
	Strategy sql.NullString
	AlgoCd   sql.NullString
}
