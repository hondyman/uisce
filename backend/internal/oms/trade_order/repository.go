package trade_order

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

func (r *Repository) List(ctx context.Context, tenantID uuid.UUID, subtypeCode string) ([]TradeOrderRecord, error) {
	query := `
		SELECT id, tenant_id, account_id, security_id, order_side, ordered_quantity, execution_price,
		       order_status, subtype_code,
		       allocation_profile_id, total_requested_quantity, average_price,
		       execution_algo_id, venue_id, liquidity_flag, route_time_micros,
		       counterparty_dealer_id, confirmation_status, isda_schedule_version,
		       base_currency, quote_currency, fx_rate, value_date,
		       syndicate_manager_id, concession_amount, allotment_shares,
		       created_at, updated_at, valid_from, valid_to
		FROM oms.trade_order
		WHERE tenant_id = $1 AND (valid_to IS NULL OR valid_to > NOW())`
	args := []interface{}{tenantID}

	if subtypeCode != "" {
		query += " AND subtype_code = $2"
		args = append(args, subtypeCode)
	}
	query += " ORDER BY created_at DESC LIMIT 100"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list trade orders: %w", err)
	}
	defer rows.Close()

	var records []TradeOrderRecord
	for rows.Next() {
		var rec TradeOrderRecord
		if err := rows.Scan(
			&rec.ID, &rec.TenantID, &rec.AccountID, &rec.SecurityID,
			&rec.OrderSide, &rec.OrderedQuantity, &rec.ExecutionPrice,
			&rec.OrderStatus, &rec.SubtypeCode,
			&rec.AllocationProfileID, &rec.TotalRequestedQuantity, &rec.AveragePrice,
			&rec.ExecutionAlgoID, &rec.VenueID, &rec.LiquidityFlag, &rec.RouteTimeMicros,
			&rec.CounterpartyDealerID, &rec.ConfirmationStatus, &rec.ISDAScheduleVersion,
			&rec.BaseCurrency, &rec.QuoteCurrency, &rec.FXRate, &rec.ValueDate,
			&rec.SyndicateManagerID, &rec.ConcessionAmount, &rec.AllotmentShares,
			&rec.CreatedAt, &rec.UpdatedAt, &rec.ValidFrom, &rec.ValidTo,
		); err != nil {
			return nil, fmt.Errorf("failed to scan trade order: %w", err)
		}
		records = append(records, rec)
	}

	return records, nil
}

func (r *Repository) Get(ctx context.Context, tenantID, id uuid.UUID) (*TradeOrderRecord, error) {
	var rec TradeOrderRecord
	err := r.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, account_id, security_id, order_side, ordered_quantity, execution_price,
		       order_status, subtype_code,
		       allocation_profile_id, total_requested_quantity, average_price,
		       execution_algo_id, venue_id, liquidity_flag, route_time_micros,
		       counterparty_dealer_id, confirmation_status, isda_schedule_version,
		       base_currency, quote_currency, fx_rate, value_date,
		       syndicate_manager_id, concession_amount, allotment_shares,
		       created_at, updated_at, valid_from, valid_to
		FROM oms.trade_order
		WHERE id = $1 AND tenant_id = $2 AND (valid_to IS NULL OR valid_to > NOW())`,
		id, tenantID,
	).Scan(
		&rec.ID, &rec.TenantID, &rec.AccountID, &rec.SecurityID,
		&rec.OrderSide, &rec.OrderedQuantity, &rec.ExecutionPrice,
		&rec.OrderStatus, &rec.SubtypeCode,
		&rec.AllocationProfileID, &rec.TotalRequestedQuantity, &rec.AveragePrice,
		&rec.ExecutionAlgoID, &rec.VenueID, &rec.LiquidityFlag, &rec.RouteTimeMicros,
		&rec.CounterpartyDealerID, &rec.ConfirmationStatus, &rec.ISDAScheduleVersion,
		&rec.BaseCurrency, &rec.QuoteCurrency, &rec.FXRate, &rec.ValueDate,
		&rec.SyndicateManagerID, &rec.ConcessionAmount, &rec.AllotmentShares,
		&rec.CreatedAt, &rec.UpdatedAt, &rec.ValidFrom, &rec.ValidTo,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get trade order: %w", err)
	}
	return &rec, nil
}

func (r *Repository) Create(ctx context.Context, rec *TradeOrderRecord) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO oms.trade_order (
			id, tenant_id, account_id, security_id, order_side, ordered_quantity, execution_price,
			order_status, subtype_code,
			allocation_profile_id, total_requested_quantity, average_price,
			execution_algo_id, venue_id, liquidity_flag, route_time_micros,
			counterparty_dealer_id, confirmation_status, isda_schedule_version,
			base_currency, quote_currency, fx_rate, value_date,
			syndicate_manager_id, concession_amount, allotment_shares,
			created_at, updated_at, valid_from
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,NOW(),NOW(),NOW())`,
		rec.ID, rec.TenantID, rec.AccountID, rec.SecurityID,
		rec.OrderSide, rec.OrderedQuantity, rec.ExecutionPrice,
		rec.OrderStatus, rec.SubtypeCode,
		rec.AllocationProfileID, rec.TotalRequestedQuantity, rec.AveragePrice,
		rec.ExecutionAlgoID, rec.VenueID, rec.LiquidityFlag, rec.RouteTimeMicros,
		rec.CounterpartyDealerID, rec.ConfirmationStatus, rec.ISDAScheduleVersion,
		rec.BaseCurrency, rec.QuoteCurrency, rec.FXRate, rec.ValueDate,
		rec.SyndicateManagerID, rec.ConcessionAmount, rec.AllotmentShares,
	)
	if err != nil {
		return fmt.Errorf("failed to create trade order: %w", err)
	}
	return nil
}

func (r *Repository) SoftDelete(ctx context.Context, tenantID, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE oms.trade_order SET valid_to = NOW(), updated_at = NOW() WHERE id = $1 AND tenant_id = $2 AND valid_to IS NULL`,
		id, tenantID,
	)
	if err != nil {
		return fmt.Errorf("failed to soft delete trade order: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}
