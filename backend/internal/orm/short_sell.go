package orm

import (
	"context"
	"database/sql"
)

// ErrInsufficientLocate is returned when no confirmed locate row has
// enough remaining quantity to cover the requested short-sell quantity.
var ErrInsufficientLocate = errString("insufficient short sell locate")

type errString string

func (e errString) Error() string { return string(e) }

// ReserveShortSellLocate decrements an existing confirmed locate by the
// requested quantity and returns the locate row ID.
func ReserveShortSellLocate(ctx context.Context, tx *sql.Tx, tenantID,
	securityID, accountID string, qty float64) (string, error) {
	var id string
	var remaining sql.NullFloat64

	err := tx.QueryRowContext(ctx, `
		SELECT id, COALESCE(quantity_confirmed,0) - COALESCE(quantity_used,0)
		FROM orm.short_sell_locate
		WHERE tenant_id = $1 AND security_id = $2 AND account_id = $3
		  AND locate_status = 'CONFIRMED'
		  AND (expiry_at IS NULL OR expiry_at > now())
		  AND COALESCE(quantity_confirmed,0) - COALESCE(quantity_used,0) >= $4
		ORDER BY confirmed_at ASC
		FOR UPDATE
		LIMIT 1
	`, tenantID, securityID, accountID, qty).Scan(&id, &remaining)

	if err == sql.ErrNoRows {
		return "", ErrInsufficientLocate
	}
	if err != nil {
		return "", err
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE orm.short_sell_locate
		SET quantity_used = COALESCE(quantity_used,0) + $1
		WHERE id = $2 AND tenant_id = $3
	`, qty, id, tenantID); err != nil {
		return "", err
	}
	return id, nil
}
