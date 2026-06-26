package services

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"

	"github.com/hondyman/semlayer/backend/internal/types"

	"github.com/google/uuid"
)

// CryptoTaxService handles tax lot tracking and gain/loss calculations
type CryptoTaxService struct {
	db *sql.DB
}

// NewCryptoTaxService creates a new crypto tax service
func NewCryptoTaxService(db *sql.DB) *CryptoTaxService {
	return &CryptoTaxService{db: db}
}

// RecordAcquisition creates a new tax lot for acquired crypto
func (s *CryptoTaxService) RecordAcquisition(ctx context.Context, acq *types.CryptoTaxLot) error {
	query := `
		INSERT INTO crypto_tax_lots (
			wallet_id, asset_symbol, acquisition_txn_id, acquisition_date,
			acquisition_type, quantity_acquired, cost_basis_per_unit, total_cost_basis
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at
	`

	return s.db.QueryRowContext(ctx, query,
		acq.WalletID, acq.AssetSymbol, acq.AcquisitionTxnID, acq.AcquisitionDate,
		acq.AcquisitionType, acq.QuantityAcquired, acq.CostBasisPerUnit, acq.TotalCostBasis,
	).Scan(&acq.ID, &acq.CreatedAt)
}

// RecordDisposal disposes crypto using specified cost basis method
func (s *CryptoTaxService) RecordDisposal(ctx context.Context, walletID uuid.UUID, assetSymbol string, quantity float64, proceeds float64, disposalDate time.Time, method string) (*types.TaxLotDisposal, error) {
	// Get available tax lots for this asset
	lots, err := s.getAvailableLots(ctx, walletID, assetSymbol)
	if err != nil {
		return nil, err
	}

	if len(lots) == 0 {
		return nil, fmt.Errorf("no tax lots available for disposal")
	}

	// Check if we have enough quantity
	totalAvailable := 0.0
	for _, lot := range lots {
		totalAvailable += lot.QuantityRemaining
	}
	if totalAvailable < quantity {
		return nil, fmt.Errorf("insufficient quantity: need %f, have %f", quantity, totalAvailable)
	}

	// Sort lots based on method
	s.sortLotsByMethod(lots, method)

	// Dispose from lots
	result := &types.TaxLotDisposal{
		DisposedLots: []types.CryptoTaxLot{},
	}

	remainingToDispose := quantity
	pricePerUnit := proceeds / quantity

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	for _, lot := range lots {
		if remainingToDispose <= 0 {
			break
		}

		qtyFromThisLot := remainingToDispose
		if qtyFromThisLot > lot.QuantityRemaining {
			qtyFromThisLot = lot.QuantityRemaining
		}

		proceedsFromThisLot := qtyFromThisLot * pricePerUnit
		costBasisFromThisLot := qtyFromThisLot * lot.CostBasisPerUnit
		gainLoss := proceedsFromThisLot - costBasisFromThisLot

		// Calculate holding period
		holdingPeriodDays := int(disposalDate.Sub(lot.AcquisitionDate).Hours() / 24)
		isLongTerm := holdingPeriodDays >= 365

		// Update lot
		query := `
			UPDATE crypto_tax_lots
			SET 
				disposal_date = $1,
				disposal_type = 'SELL',
				quantity_disposed = quantity_disposed + $2,
				disposal_proceeds = disposal_proceeds + $3,
				holding_period_days = $4,
				realized_gain_loss = COALESCE(realized_gain_loss, 0) + $5
			WHERE id = $6
		`

		_, err = tx.ExecContext(ctx, query,
			disposalDate, qtyFromThisLot, proceedsFromThisLot,
			holdingPeriodDays, gainLoss, lot.ID,
		)
		if err != nil {
			return nil, err
		}

		// Add to result
		disposedLot := lot
		disposedLot.QuantityDisposed = qtyFromThisLot
		disposedLot.DisposalProceeds = proceedsFromThisLot
		disposedLot.DisposalDate = &disposalDate
		disposedLot.HoldingPeriodDays = &holdingPeriodDays
		disposedLot.IsLongTerm = &isLongTerm
		disposedLot.RealizedGainLoss = &gainLoss

		result.DisposedLots = append(result.DisposedLots, disposedLot)
		result.TotalGainLoss += gainLoss

		if isLongTerm {
			result.LongTermGainLoss += gainLoss
		} else {
			result.ShortTermGainLoss += gainLoss
		}

		remainingToDispose -= qtyFromThisLot
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// Check for wash sales
	go s.detectWashSales(context.Background(), walletID, assetSymbol, disposalDate)

	return result, nil
}

// getAvailableLots retrieves tax lots with remaining quantity
func (s *CryptoTaxService) getAvailableLots(ctx context.Context, walletID uuid.UUID, assetSymbol string) ([]types.CryptoTaxLot, error) {
	query := `
		SELECT 
			id, wallet_id, asset_symbol, acquisition_txn_id, acquisition_date,
			acquisition_type, quantity_acquired, cost_basis_per_unit, total_cost_basis,
			disposal_txn_id, disposal_date, disposal_type, quantity_disposed,
			disposal_proceeds, quantity_remaining, is_fully_disposed,
			holding_period_days, is_long_term, realized_gain_loss,
			is_wash_sale, wash_sale_disallowed_loss, linked_wash_sale_lot_id, created_at
		FROM crypto_tax_lots
		WHERE wallet_id = $1 
		  AND asset_symbol = $2
		  AND quantity_remaining > 0
		ORDER BY acquisition_date ASC
	`

	rows, err := s.db.QueryContext(ctx, query, walletID, assetSymbol)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lots []types.CryptoTaxLot
	for rows.Next() {
		var lot types.CryptoTaxLot
		err := rows.Scan(
			&lot.ID, &lot.WalletID, &lot.AssetSymbol, &lot.AcquisitionTxnID, &lot.AcquisitionDate,
			&lot.AcquisitionType, &lot.QuantityAcquired, &lot.CostBasisPerUnit, &lot.TotalCostBasis,
			&lot.DisposalTxnID, &lot.DisposalDate, &lot.DisposalType, &lot.QuantityDisposed,
			&lot.DisposalProceeds, &lot.QuantityRemaining, &lot.IsFullyDisposed,
			&lot.HoldingPeriodDays, &lot.IsLongTerm, &lot.RealizedGainLoss,
			&lot.IsWashSale, &lot.WashSaleDisallowedLoss, &lot.LinkedWashSaleLotID, &lot.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		lots = append(lots, lot)
	}

	return lots, rows.Err()
}

// sortLotsByMethod sorts tax lots based on disposal method
func (s *CryptoTaxService) sortLotsByMethod(lots []types.CryptoTaxLot, method string) {
	switch method {
	case "FIFO": // First In, First Out
		sort.Slice(lots, func(i, j int) bool {
			return lots[i].AcquisitionDate.Before(lots[j].AcquisitionDate)
		})
	case "LIFO": // Last In, First Out
		sort.Slice(lots, func(i, j int) bool {
			return lots[i].AcquisitionDate.After(lots[j].AcquisitionDate)
		})
	case "HIFO": // Highest In, First Out (tax advantaged)
		sort.Slice(lots, func(i, j int) bool {
			return lots[i].CostBasisPerUnit > lots[j].CostBasisPerUnit
		})
	default: // FIFO by default
		sort.Slice(lots, func(i, j int) bool {
			return lots[i].AcquisitionDate.Before(lots[j].AcquisitionDate)
		})
	}
}

// CalculateGainLoss calculates total gains/losses for a tax year
func (s *CryptoTaxService) CalculateGainLoss(ctx context.Context, clientID uuid.UUID, taxYear int) (*types.GainLossReport, error) {
	query := `
		SELECT 
			COUNT(*) as transaction_count,
			COALESCE(SUM(CASE WHEN is_long_term = FALSE THEN realized_gain_loss ELSE 0 END), 0) as short_term,
			COALESCE(SUM(CASE WHEN is_long_term = TRUE THEN realized_gain_loss ELSE 0 END), 0) as long_term
		FROM crypto_tax_lots ctl
		JOIN crypto_wallets cw ON ctl.wallet_id = cw.id
		WHERE cw.client_id = $1
		  AND EXTRACT(YEAR FROM ctl.disposal_date) = $2
		  AND ctl.disposal_date IS NOT NULL
	`

	report := &types.GainLossReport{TaxYear: taxYear}
	var count int
	var shortTerm, longTerm float64

	err := s.db.QueryRowContext(ctx, query, clientID, taxYear).Scan(&count, &shortTerm, &longTerm)
	if err != nil {
		return nil, err
	}

	report.TransactionCount = count
	report.TotalShortTerm = shortTerm
	report.TotalLongTerm = longTerm
	report.TotalNet = shortTerm + longTerm

	return report, nil
}

// GenerateForm8949 generates IRS Form 8949 entries for a tax year
func (s *CryptoTaxService) GenerateForm8949(ctx context.Context, clientID uuid.UUID, taxYear int) ([]types.Form8949Entry, error) {
	query := `
		SELECT 
			ctl.asset_symbol,
			ctl.acquisition_date,
			ctl.disposal_date,
			ctl.quantity_disposed,
			ctl.disposal_proceeds,
			ctl.quantity_disposed * ctl.cost_basis_per_unit as cost_basis,
			ctl.realized_gain_loss,
			ctl.is_long_term,
			ctl.wash_sale_disallowed_loss
		FROM crypto_tax_lots ctl
		JOIN crypto_wallets cw ON ctl.wallet_id = cw.id
		WHERE cw.client_id = $1
		  AND EXTRACT(YEAR FROM ctl.disposal_date) = $2
		  AND ctl.disposal_date IS NOT NULL
		ORDER BY ctl.disposal_date, ctl.asset_symbol
	`

	rows, err := s.db.QueryContext(ctx, query, clientID, taxYear)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []types.Form8949Entry
	for rows.Next() {
		var entry types.Form8949Entry
		var disposalDate sql.NullTime
		var isLongTerm sql.NullBool

		err := rows.Scan(
			&entry.AssetSymbol,
			&entry.DateAcquired,
			&disposalDate,
			&entry.Quantity,
			&entry.Proceeds,
			&entry.CostBasis,
			&entry.GainLoss,
			&isLongTerm,
			&entry.WashSaleAdjustment,
		)
		if err != nil {
			return nil, err
		}

		if disposalDate.Valid {
			entry.DateSold = disposalDate.Time
		}
		if isLongTerm.Valid {
			entry.IsLongTerm = isLongTerm.Bool
		}

		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

// detectWashSales detects wash sales within 30 days
func (s *CryptoTaxService) detectWashSales(ctx context.Context, walletID uuid.UUID, assetSymbol string, disposalDate time.Time) error {
	// IRS wash sale rule: substantially identical security purchased within 30 days before or after a loss sale

	// Find losses sold on this date
	lossLotsQuery := `
		SELECT id, realized_gain_loss, quantity_disposed, cost_basis_per_unit
		FROM crypto_tax_lots
		WHERE wallet_id = $1
		  AND asset_symbol = $2
		  AND disposal_date = $3
		  AND realized_gain_loss < 0
		  AND is_wash_sale = FALSE
	`

	lossRows, err := s.db.QueryContext(ctx, lossLotsQuery, walletID, assetSymbol, disposalDate)
	if err != nil {
		return err
	}
	defer lossRows.Close()

	for lossRows.Next() {
		var lossLotID uuid.UUID
		var lossAmount, qtyDisposed, costBasis float64
		if err := lossRows.Scan(&lossLotID, &lossAmount, &qtyDisposed, &costBasis); err != nil {
			continue
		}

		// Find purchases within 30 days before or after
		purchaseQuery := `
			SELECT id, acquisition_date
			FROM crypto_tax_lots
			WHERE wallet_id = $1
			  AND asset_symbol = $2
			  AND acquisition_date BETWEEN $3 AND $4
			  AND id != $5
			ORDER BY acquisition_date ASC
			LIMIT 1
		`

		thirtyDaysBefore := disposalDate.AddDate(0, 0, -30)
		thirtyDaysAfter := disposalDate.AddDate(0, 0, 30)

		var purchaseLotID uuid.UUID
		var purchaseDate time.Time
		err := s.db.QueryRowContext(ctx, purchaseQuery, walletID, assetSymbol, thirtyDaysBefore, thirtyDaysAfter, lossLotID).Scan(&purchaseLotID, &purchaseDate)

		if err == sql.ErrNoRows {
			continue // No wash sale
		}
		if err != nil {
			continue
		}

		// Mark as wash sale
		_, err = s.db.ExecContext(ctx, `
			UPDATE crypto_tax_lots
			SET 
				is_wash_sale = TRUE,
				wash_sale_disallowed_loss = ABS($1),
				linked_wash_sale_lot_id = $2
			WHERE id = $3
		`, lossAmount, purchaseLotID, lossLotID)

		if err != nil {
			continue
		}

		// Add disallowed loss to new lot's cost basis
		_, err = s.db.ExecContext(ctx, `
			UPDATE crypto_tax_lots
			SET total_cost_basis = total_cost_basis + ABS($1)
			WHERE id = $2
		`, lossAmount, purchaseLotID)
	}

	return nil
}

// GetTaxLotsByWallet retrieves all tax lots for a wallet
func (s *CryptoTaxService) GetTaxLotsByWallet(ctx context.Context, walletID uuid.UUID, assetSymbol *string) ([]types.CryptoTaxLot, error) {
	query := `
		SELECT 
			id, wallet_id, asset_symbol, acquisition_txn_id, acquisition_date,
			acquisition_type, quantity_acquired, cost_basis_per_unit, total_cost_basis,
			disposal_txn_id, disposal_date, disposal_type, quantity_disposed,
			disposal_proceeds, quantity_remaining, is_fully_disposed,
			holding_period_days, is_long_term, realized_gain_loss,
			is_wash_sale, wash_sale_disallowed_loss, linked_wash_sale_lot_id, created_at
		FROM crypto_tax_lots
		WHERE wallet_id = $1
	`

	args := []interface{}{walletID}
	if assetSymbol != nil {
		query += " AND asset_symbol = $2"
		args = append(args, *assetSymbol)
	}
	query += " ORDER BY acquisition_date DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lots []types.CryptoTaxLot
	for rows.Next() {
		var lot types.CryptoTaxLot
		err := rows.Scan(
			&lot.ID, &lot.WalletID, &lot.AssetSymbol, &lot.AcquisitionTxnID, &lot.AcquisitionDate,
			&lot.AcquisitionType, &lot.QuantityAcquired, &lot.CostBasisPerUnit, &lot.TotalCostBasis,
			&lot.DisposalTxnID, &lot.DisposalDate, &lot.DisposalType, &lot.QuantityDisposed,
			&lot.DisposalProceeds, &lot.QuantityRemaining, &lot.IsFullyDisposed,
			&lot.HoldingPeriodDays, &lot.IsLongTerm, &lot.RealizedGainLoss,
			&lot.IsWashSale, &lot.WashSaleDisallowedLoss, &lot.LinkedWashSaleLotID, &lot.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		lots = append(lots, lot)
	}

	return lots, rows.Err()
}
