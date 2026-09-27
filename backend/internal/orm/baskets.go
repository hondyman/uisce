package orm

import (
	"context"
	"database/sql"
)

// CreateBasket inserts a basket header, returns the new ID.
func CreateBasket(ctx context.Context, tx *sql.Tx, tenantID,
	cd, name, basketType string, createdBy string) (string, error) {
	var id string
	err := tx.QueryRowContext(ctx, `
		INSERT INTO orm.basket (
			basket_cd, name, basket_type, status, created_by, tenant_id
		) VALUES ($1,$2,$3,'DRAFT',$4,$5)
		RETURNING id
	`, cd, name, basketType, createdBy, tenantID).Scan(&id)
	return id, err
}

// AddBasketItem appends one security to a basket.
func AddBasketItem(ctx context.Context, tx *sql.Tx, tenantID,
	basketID, securityID, side string, qty sql.NullFloat64, seq int) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO orm.basket_item (
			basket_id, security_id, side, target_qty, sequence_number,
			status, tenant_id
		) VALUES ($1,$2,$3,$4,$5,'PENDING',$6)
	`, basketID, securityID, side, qty, seq, tenantID)
	return err
}

// LinkBasketItemToOrder sets order_id on a basket item once the child
// order has been created.
func LinkBasketItemToOrder(ctx context.Context, tx *sql.Tx, tenantID,
	itemID, orderID string) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE orm.basket_item
		SET order_id = $1, status = 'ROUTED'
		WHERE id = $2 AND tenant_id = $3
	`, orderID, itemID, tenantID)
	return err
}
