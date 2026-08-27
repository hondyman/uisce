package ledger

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type TradeFillEvent struct {
	TenantID        uuid.UUID `json:"tenant_id"`
	PortfolioNodeID uuid.UUID `json:"portfolio_node_id"`
	SecurityNodeID  uuid.UUID `json:"security_node_id"`
	AccountNodeID   uuid.UUID `json:"account_node_id"`
	Side            string    `json:"side"` // BUY or SELL
	Shares          float64   `json:"shares"`
	Price           float64   `json:"price"`
	GrossAmount     float64   `json:"gross_amount"`
	Commission      float64   `json:"commission"`
	TradeDate       time.Time `json:"trade_date"`
	SettleDate      time.Time `json:"settle_date"`
}

type CustodianPositionRecord struct {
	PortfolioNodeID uuid.UUID `json:"portfolio_node_id"`
	SecurityNodeID  uuid.UUID `json:"security_node_id"`
	Shares          float64   `json:"shares"`
	SourceFile      string    `json:"source_file"`
}

type MultiBookSynchronizerService struct {
	db *sqlx.DB
}

func NewMultiBookSynchronizerService(db *sqlx.DB) *MultiBookSynchronizerService {
	return &MultiBookSynchronizerService{db: db}
}

// ProcessTradeFill updates IBOR in real time and stages ABOR pending settlement entries
func (s *MultiBookSynchronizerService) ProcessTradeFill(ctx context.Context, fill *TradeFillEvent) error {
	if fill.TenantID == uuid.Nil {
		return fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	var openBuyDelta, openSellDelta, cashDelta float64
	if fill.Side == "BUY" {
		openBuyDelta = fill.Shares
		cashDelta = -(fill.GrossAmount + fill.Commission)
	} else {
		openSellDelta = fill.Shares
		cashDelta = (fill.GrossAmount - fill.Commission)
	}

	batchID := uuid.New()
	hasher := sha256.New()
	hasher.Write([]byte(fmt.Sprintf("%s:%s:%f:%s", fill.PortfolioNodeID, fill.SecurityNodeID, fill.GrossAmount, fill.TradeDate)))
	merkleHash := hex.EncodeToString(hasher.Sum(nil))

	var debitAcc, creditAcc string
	if fill.Side == "BUY" {
		debitAcc = "1200-EQUITY-INVESTMENTS"
		creditAcc = "2010-TRADES-PAYABLE"
	} else {
		debitAcc = "1020-TRADES-RECEIVABLE"
		creditAcc = "1200-EQUITY-INVESTMENTS"
	}

	if s.db != nil {
		tx, err := s.db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
		if err != nil {
			return fmt.Errorf("failed initiating serializable transaction: %w", err)
		}
		defer tx.Rollback()

		iborUpsert := `
			INSERT INTO ledger_multi.ibor_intraday_positions (
				position_id, tenant_id, portfolio_node_id, security_node_id, account_node_id,
				settled_shares, open_buy_shares, open_sell_shares, projected_cash_usd, knowledge_time, updated_at
			) VALUES (
				gen_random_uuid(), $1, $2, $3, $4, 0, $5, $6, $7, NOW(), NOW()
			)
			ON CONFLICT (tenant_id, portfolio_node_id, security_node_id, account_node_id) DO UPDATE SET
				open_buy_shares = ledger_multi.ibor_intraday_positions.open_buy_shares + $5,
				open_sell_shares = ledger_multi.ibor_intraday_positions.open_sell_shares + $6,
				projected_cash_usd = ledger_multi.ibor_intraday_positions.projected_cash_usd + $7,
				knowledge_time = NOW(),
				updated_at = NOW();`

		if _, err := tx.ExecContext(ctx, iborUpsert,
			fill.TenantID, fill.PortfolioNodeID, fill.SecurityNodeID, fill.AccountNodeID,
			openBuyDelta, openSellDelta, cashDelta); err != nil {
			return fmt.Errorf("failed updating IBOR intraday position: %w", err)
		}

		glInsert := `
			INSERT INTO ledger_multi.abor_general_ledger_entries (
				entry_id, tenant_id, journal_batch_id, portfolio_node_id, gl_account_code,
				debit_amount, credit_amount, currency, effective_date, entry_type, merkle_leaf_hash
			) VALUES 
			(gen_random_uuid(), $1, $2, $3, $4, $5, 0.00, 'USD', $6, 'TRADE_SETTLEMENT', $7),
			(gen_random_uuid(), $1, $2, $3, $8, 0.00, $5, 'USD', $6, 'TRADE_SETTLEMENT', $7);`

		if _, err := tx.ExecContext(ctx, glInsert,
			fill.TenantID, batchID, fill.PortfolioNodeID, debitAcc, fill.GrossAmount,
			fill.TradeDate, merkleHash, creditAcc); err != nil {
			return fmt.Errorf("failed staging ABOR double-entry GL journals: %w", err)
		}

		return tx.Commit()
	}

	return nil
}

// ReconcileCustodianPosition matches custodian files against internal IBOR/ABOR positions
func (s *MultiBookSynchronizerService) ReconcileCustodianPosition(
	ctx context.Context,
	tenantID uuid.UUID,
	custodianRecord *CustodianPositionRecord,
	tolerance float64,
) error {
	if tenantID == uuid.Nil {
		return fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	var internalShares float64 = 25000 // mock comparison value if db is nil
	if s.db != nil {
		query := `
			SELECT settled_shares + open_buy_shares - open_sell_shares 
			FROM ledger_multi.ibor_intraday_positions
			WHERE tenant_id = $1 AND portfolio_node_id = $2 AND security_node_id = $3;`

		err := s.db.QueryRowContext(ctx, query, tenantID, custodianRecord.PortfolioNodeID, custodianRecord.SecurityNodeID).Scan(&internalShares)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
	}

	variance := math.Abs(internalShares - custodianRecord.Shares)
	if variance > tolerance && s.db != nil {
		breakInsert := `
			INSERT INTO ledger_multi.recon_break_instances (
				break_id, tenant_id, portfolio_node_id, security_node_id, break_type,
				internal_ibor_value, custodian_abor_value, tolerance_threshold, status, statement_source
			) VALUES (gen_random_uuid(), $1, $2, $3, 'POSITION_QUANTITY', $4, $5, $6, 'OPEN', $7);`

		_, err := s.db.ExecContext(ctx, breakInsert,
			tenantID, custodianRecord.PortfolioNodeID, custodianRecord.SecurityNodeID,
			internalShares, custodianRecord.Shares, tolerance, custodianRecord.SourceFile)
		return err
	}

	return nil
}
