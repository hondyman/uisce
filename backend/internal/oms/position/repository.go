package position

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) List(ctx context.Context, tenantID uuid.UUID, subtypeCode string) ([]PositionRecord, error) {
	query := `
		SELECT id, tenant_id, account_id, security_id, quantity, market_value, currency, subtype_code,
		       custody_account_id, settled_shares, cost_basis_method, held_to_maturity_flag,
		       prime_broker_id, borrow_rate_bps, locate_id, hard_to_borrow_flag,
		       underlying_security_id, notional_amount, unrealized_pnl, expiration_date,
		       pledged_to_party, haircut_pct, rehypothecation_allowed_flag,
		       trade_date_shares, pending_settlement_cash, fails_to_deliver_flag,
		       created_at, updated_at, valid_from, valid_to
		FROM oms.position
		WHERE tenant_id = $1 AND (valid_to IS NULL OR valid_to > NOW())`
	args := []interface{}{tenantID}

	if subtypeCode != "" {
		query += " AND subtype_code = $2"
		args = append(args, subtypeCode)
	}
	query += " ORDER BY created_at DESC LIMIT 100"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list positions: %w", err)
	}
	defer rows.Close()

	var records []PositionRecord
	for rows.Next() {
		var rec PositionRecord
		if err := rows.Scan(
			&rec.ID, &rec.TenantID, &rec.AccountID, &rec.SecurityID, &rec.Quantity,
			&rec.MarketValue, &rec.Currency, &rec.SubtypeCode,
			&rec.CustodyAccountID, &rec.SettledShares, &rec.CostBasisMethod, &rec.HeldToMaturityFlag,
			&rec.PrimeBrokerID, &rec.BorrowRateBPS, &rec.LocateID, &rec.HardToBorrowFlag,
			&rec.UnderlyingSecurityID, &rec.NotionalAmount, &rec.UnrealizedPnL, &rec.ExpirationDate,
			&rec.PledgedToParty, &rec.HaircutPct, &rec.RehypothecationAllowed,
			&rec.TradeDateShares, &rec.PendingSettlementCash, &rec.FailsToDeliverFlag,
			&rec.CreatedAt, &rec.UpdatedAt, &rec.ValidFrom, &rec.ValidTo,
		); err != nil {
			return nil, fmt.Errorf("failed to scan position: %w", err)
		}
		records = append(records, rec)
	}

	return records, nil
}

func (r *Repository) Get(ctx context.Context, tenantID, id uuid.UUID) (*PositionRecord, error) {
	var rec PositionRecord
	err := r.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, account_id, security_id, quantity, market_value, currency, subtype_code,
		       custody_account_id, settled_shares, cost_basis_method, held_to_maturity_flag,
		       prime_broker_id, borrow_rate_bps, locate_id, hard_to_borrow_flag,
		       underlying_security_id, notional_amount, unrealized_pnl, expiration_date,
		       pledged_to_party, haircut_pct, rehypothecation_allowed_flag,
		       trade_date_shares, pending_settlement_cash, fails_to_deliver_flag,
		       created_at, updated_at, valid_from, valid_to
		FROM oms.position
		WHERE id = $1 AND tenant_id = $2 AND (valid_to IS NULL OR valid_to > NOW())`,
		id, tenantID,
	).Scan(
		&rec.ID, &rec.TenantID, &rec.AccountID, &rec.SecurityID, &rec.Quantity,
		&rec.MarketValue, &rec.Currency, &rec.SubtypeCode,
		&rec.CustodyAccountID, &rec.SettledShares, &rec.CostBasisMethod, &rec.HeldToMaturityFlag,
		&rec.PrimeBrokerID, &rec.BorrowRateBPS, &rec.LocateID, &rec.HardToBorrowFlag,
		&rec.UnderlyingSecurityID, &rec.NotionalAmount, &rec.UnrealizedPnL, &rec.ExpirationDate,
		&rec.PledgedToParty, &rec.HaircutPct, &rec.RehypothecationAllowed,
		&rec.TradeDateShares, &rec.PendingSettlementCash, &rec.FailsToDeliverFlag,
		&rec.CreatedAt, &rec.UpdatedAt, &rec.ValidFrom, &rec.ValidTo,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get position: %w", err)
	}
	return &rec, nil
}

func (r *Repository) Create(ctx context.Context, rec *PositionRecord) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO oms.position (
			id, tenant_id, account_id, security_id, quantity, market_value, currency, subtype_code,
			custody_account_id, settled_shares, cost_basis_method, held_to_maturity_flag,
			prime_broker_id, borrow_rate_bps, locate_id, hard_to_borrow_flag,
			underlying_security_id, notional_amount, unrealized_pnl, expiration_date,
			pledged_to_party, haircut_pct, rehypothecation_allowed_flag,
			trade_date_shares, pending_settlement_cash, fails_to_deliver_flag,
			created_at, updated_at, valid_from
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,NOW(),NOW(),NOW())`,
		rec.ID, rec.TenantID, rec.AccountID, rec.SecurityID, rec.Quantity, rec.MarketValue,
		rec.Currency, rec.SubtypeCode,
		rec.CustodyAccountID, rec.SettledShares, rec.CostBasisMethod, rec.HeldToMaturityFlag,
		rec.PrimeBrokerID, rec.BorrowRateBPS, rec.LocateID, rec.HardToBorrowFlag,
		rec.UnderlyingSecurityID, rec.NotionalAmount, rec.UnrealizedPnL, rec.ExpirationDate,
		rec.PledgedToParty, rec.HaircutPct, rec.RehypothecationAllowed,
		rec.TradeDateShares, rec.PendingSettlementCash, rec.FailsToDeliverFlag,
	)
	if err != nil {
		return fmt.Errorf("failed to create position: %w", err)
	}
	return nil
}

func (r *Repository) SoftDelete(ctx context.Context, tenantID, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE oms.position SET valid_to = NOW(), updated_at = NOW() WHERE id = $1 AND tenant_id = $2 AND valid_to IS NULL`,
		id, tenantID,
	)
	if err != nil {
		return fmt.Errorf("failed to soft delete position: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}
