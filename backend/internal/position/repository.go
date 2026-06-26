package position

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/goldcopy"
	"github.com/shopspring/decimal"
)

// sourceQualityScore maps source systems to quality scores for survivorship.
var sourceQualityScore = map[string]int{
	"Custodian":     98,
	"Accounting":    85,
	"TradingSystem": 75,
	"InternalModel": 60,
}

// defaultPositionRules defines field-level survivorship rules for position data.
var defaultPositionRules = []*goldcopy.SurvivorshipRule{
	{EntityType: "position", FieldName: "position_quantity", Strategy: "prefer_source", PreferredSources: []string{"Custodian", "Accounting", "TradingSystem"}, Priority: 1},
	{EntityType: "position", FieldName: "market_value_local", Strategy: "prefer_source", PreferredSources: []string{"Custodian", "TradingSystem", "Accounting"}, Priority: 1},
	{EntityType: "position", FieldName: "cost_basis_local", Strategy: "prefer_source", PreferredSources: []string{"Accounting", "Custodian", "TradingSystem"}, Priority: 1},
	{EntityType: "position", FieldName: "valuation_fx_rate", Strategy: "prefer_source", PreferredSources: []string{"Custodian", "Accounting"}, Priority: 1},
	{EntityType: "position", FieldName: "is_reconciled", Strategy: "prefer_source", PreferredSources: []string{"Custodian"}, Priority: 1},
	{EntityType: "position", FieldName: "position_currency", Strategy: "prefer_source", PreferredSources: []string{"Custodian", "Accounting"}, Priority: 1},
}

// toRawRecord converts a PositionMasterRecord to a goldcopy.RawPortfolioRecord
// so it can participate in the standard Survivorship.Resolve flow.
func toRawRecord(p *PositionMasterRecord) *goldcopy.RawPortfolioRecord {
	q, _ := sourceQualityScore[p.PositionSource]
	fields := map[string]string{
		"position_quantity": p.PositionQuantity.String(),
		"position_side":     p.PositionSide,
		"position_currency": p.PositionCurrency,
		"is_reconciled":     fmt.Sprintf("%v", p.IsReconciled),
	}
	if p.MarketValueLocal != nil {
		fields["market_value_local"] = p.MarketValueLocal.String()
	}
	if p.MarketValueBase != nil {
		fields["market_value_base"] = p.MarketValueBase.String()
	}
	if p.ValuationFXRate != nil {
		fields["valuation_fx_rate"] = p.ValuationFXRate.String()
	}
	if p.CostBasisLocal != nil {
		fields["cost_basis_local"] = p.CostBasisLocal.String()
	}
	return &goldcopy.RawPortfolioRecord{
		PortfolioID:  p.PortfolioID.String(),
		SourceSystem: p.PositionSource,
		Fields:       fields,
		QualityScore: q,
	}
}

// applyWinningField parses the survivorship result back onto a record.
func applyWinningField(p *PositionMasterRecord, results []goldcopy.SurvivorshipResult) {
	for _, r := range results {
		switch r.FieldName {
		case "position_quantity":
			if v, err := decimal.NewFromString(r.ChosenValue); err == nil {
				p.PositionQuantity = v
			}
		case "market_value_local":
			if v, err := decimal.NewFromString(r.ChosenValue); err == nil {
				p.MarketValueLocal = &v
			}
		case "market_value_base":
			if v, err := decimal.NewFromString(r.ChosenValue); err == nil {
				p.MarketValueBase = &v
			}
		case "valuation_fx_rate":
			if v, err := decimal.NewFromString(r.ChosenValue); err == nil {
				p.ValuationFXRate = &v
			}
		case "cost_basis_local":
			if v, err := decimal.NewFromString(r.ChosenValue); err == nil {
				p.CostBasisLocal = &v
			}
		case "position_currency":
			if r.ChosenValue != "" {
				p.PositionCurrency = r.ChosenValue
			}
		case "is_reconciled":
			p.IsReconciled = (r.ChosenValue == "true" && r.ChosenSource == "Custodian")
		case "position_side":
			if r.ChosenValue != "" {
				p.PositionSide = r.ChosenValue
			}
		}
		// Update source attribution
		p.PositionSource = r.ChosenSource
	}
}

// Repository provides data access and survivorship logic for the Position Master.
type Repository struct {
	db           *sql.DB
	survivorship *goldcopy.Survivorship
}

// NewRepository creates a new Position Repository with default survivorship rules.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{
		db:           db,
		survivorship: goldcopy.NewSurvivorship(defaultPositionRules),
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// LIST QUERIES
// ─────────────────────────────────────────────────────────────────────────────

// ListPositions returns gold-copy positions for a tenant (optional portfolio filter).
func (r *Repository) ListPositions(ctx context.Context, tenantID uuid.UUID, portfolioID *uuid.UUID, limit, offset int) ([]PositionMasterRecord, error) {
	query := `
		SELECT id, tenant_id, core_id, portfolio_id, security_id, position_date,
		       position_quantity, position_side, position_currency, price_id,
		       market_value_local, market_value_base, valuation_fx_rate,
		       cost_basis_local, unrealized_pl_local, unrealized_pl_pct,
		       position_weight_pct, position_source, position_confidence,
		       is_reconciled, reconciliation_diff, source_systems,
		       created_at, updated_at, valid_from, valid_to
		FROM edm.position_master
		WHERE tenant_id = $1
		  AND (valid_to IS NULL OR valid_to > NOW())
		  AND ($2::uuid IS NULL OR portfolio_id = $2)
		ORDER BY position_date DESC, market_value_local DESC NULLS LAST
		LIMIT $3 OFFSET $4`

	var pidArg interface{}
	if portfolioID != nil {
		pidArg = *portfolioID
	}
	rows, err := r.db.QueryContext(ctx, query, tenantID, pidArg, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list positions: %w", err)
	}
	defer rows.Close()

	var records []PositionMasterRecord
	for rows.Next() {
		var rec PositionMasterRecord
		var sourceSystems []byte
		if err := rows.Scan(
			&rec.ID, &rec.TenantID, &rec.CoreID, &rec.PortfolioID, &rec.SecurityID,
			&rec.PositionDate, &rec.PositionQuantity, &rec.PositionSide,
			&rec.PositionCurrency, &rec.PriceID, &rec.MarketValueLocal,
			&rec.MarketValueBase, &rec.ValuationFXRate, &rec.CostBasisLocal,
			&rec.UnrealizedPLLocal, &rec.UnrealizedPLPct, &rec.PositionWeightPct,
			&rec.PositionSource, &rec.PositionConfidence, &rec.IsReconciled,
			&rec.ReconciliationDiff, &sourceSystems,
			&rec.CreatedAt, &rec.UpdatedAt, &rec.ValidFrom, &rec.ValidTo,
		); err != nil {
			return nil, fmt.Errorf("scan position: %w", err)
		}
		rec.SourceSystems = sourceSystems
		records = append(records, rec)
	}
	return records, rows.Err()
}

// ListPositionLots returns the tax lots for a specific position.
func (r *Repository) ListPositionLots(ctx context.Context, positionID uuid.UUID, limit, offset int) ([]PositionLotRecord, error) {
	query := `
		SELECT id, tenant_id, position_id, lot_reference, acquisition_date, settlement_date,
		       lot_quantity, cost_per_unit, total_cost_basis, lot_method,
		       is_closed, closed_date, realized_pl, created_at
		FROM edm.position_lot_master
		WHERE position_id = $1
		ORDER BY acquisition_date ASC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, positionID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list lots: %w", err)
	}
	defer rows.Close()

	var records []PositionLotRecord
	for rows.Next() {
		var rec PositionLotRecord
		if err := rows.Scan(
			&rec.ID, &rec.TenantID, &rec.PositionID, &rec.LotReference,
			&rec.AcquisitionDate, &rec.SettlementDate, &rec.LotQuantity,
			&rec.CostPerUnit, &rec.TotalCostBasis, &rec.LotMethod,
			&rec.IsClosed, &rec.ClosedDate, &rec.RealizedPL, &rec.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan lot: %w", err)
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}

// ListCashPositions returns cash balances for a tenant (optional portfolio filter).
func (r *Repository) ListCashPositions(ctx context.Context, tenantID uuid.UUID, portfolioID *uuid.UUID, limit, offset int) ([]CashPositionRecord, error) {
	query := `
		SELECT id, tenant_id, portfolio_id, cash_currency, account_id, value_date,
		       balance_amount, available_balance, pending_settlements,
		       interest_accrued, cash_source, source_systems,
		       created_at, updated_at, valid_from, valid_to
		FROM edm.cash_position_master
		WHERE tenant_id = $1
		  AND (valid_to IS NULL OR valid_to > NOW())
		  AND ($2::uuid IS NULL OR portfolio_id = $2)
		ORDER BY value_date DESC, balance_amount DESC
		LIMIT $3 OFFSET $4`

	var pidArg interface{}
	if portfolioID != nil {
		pidArg = *portfolioID
	}
	rows, err := r.db.QueryContext(ctx, query, tenantID, pidArg, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list cash positions: %w", err)
	}
	defer rows.Close()

	var records []CashPositionRecord
	for rows.Next() {
		var rec CashPositionRecord
		var pending, sourceSystems []byte
		if err := rows.Scan(
			&rec.ID, &rec.TenantID, &rec.PortfolioID, &rec.CashCurrency, &rec.AccountID,
			&rec.ValueDate, &rec.BalanceAmount, &rec.AvailableBalance,
			&pending, &rec.InterestAccrued, &rec.CashSource,
			&sourceSystems, &rec.CreatedAt, &rec.UpdatedAt, &rec.ValidFrom, &rec.ValidTo,
		); err != nil {
			return nil, fmt.Errorf("scan cash position: %w", err)
		}
		rec.PendingSettlements = pending
		rec.SourceSystems = sourceSystems
		records = append(records, rec)
	}
	return records, rows.Err()
}

// ListSnapshots returns historical snapshots for a position.
func (r *Repository) ListSnapshots(ctx context.Context, positionID uuid.UUID, limit, offset int) ([]PositionSnapshotRecord, error) {
	query := `
		SELECT id, tenant_id, position_id, snapshot_date, snapshot_quantity,
		       snapshot_market_value, snapshot_price_used, snapshot_fx_rate,
		       portfolio_composition, snapshot_source, created_at
		FROM edm.position_snapshot_master
		WHERE position_id = $1
		ORDER BY snapshot_date DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, positionID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list snapshots: %w", err)
	}
	defer rows.Close()

	var records []PositionSnapshotRecord
	for rows.Next() {
		var rec PositionSnapshotRecord
		var comp []byte
		if err := rows.Scan(
			&rec.ID, &rec.TenantID, &rec.PositionID, &rec.SnapshotDate,
			&rec.SnapshotQuantity, &rec.SnapshotMarketValue, &rec.SnapshotPriceUsed,
			&rec.SnapshotFXRate, &comp, &rec.SnapshotSource, &rec.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan snapshot: %w", err)
		}
		rec.PortfolioComposition = comp
		records = append(records, rec)
	}
	return records, rows.Err()
}

// ─────────────────────────────────────────────────────────────────────────────
// UPSERT WITH SURVIVORSHIP
// Strategy: prefer_source("Custodian") for quantity/MV; prefer_source("Accounting")
// for cost_basis. Uses goldcopy.Survivorship for field-level resolution.
// Cluster key: (portfolio_id, security_id, position_date, tenant_id)
// ─────────────────────────────────────────────────────────────────────────────

// UpsertPosition upserts a position record using goldcopy Survivorship resolution.
func (r *Repository) UpsertPosition(ctx context.Context, tenantID uuid.UUID, incoming *PositionMasterRecord) error {
	// 1. Fetch all competing records for the cluster
	clusterRows, err := r.db.QueryContext(ctx, `
		SELECT id, tenant_id, core_id, portfolio_id, security_id, position_date,
		       position_quantity, position_side, position_currency, price_id,
		       market_value_local, market_value_base, valuation_fx_rate,
		       cost_basis_local, unrealized_pl_local, position_source, position_confidence,
		       is_reconciled, reconciliation_diff, source_systems,
		       created_at, updated_at, valid_from, valid_to
		FROM edm.position_master
		WHERE portfolio_id = $1 AND security_id = $2 AND position_date = $3
		  AND tenant_id = $4 AND (valid_to IS NULL OR valid_to > NOW())`,
		incoming.PortfolioID, incoming.SecurityID, incoming.PositionDate, tenantID,
	)
	if err != nil {
		return fmt.Errorf("fetch cluster: %w", err)
	}
	defer clusterRows.Close()

	var competitors []PositionMasterRecord
	for clusterRows.Next() {
		var rec PositionMasterRecord
		var sourceSystems []byte
		if err := clusterRows.Scan(
			&rec.ID, &rec.TenantID, &rec.CoreID, &rec.PortfolioID, &rec.SecurityID,
			&rec.PositionDate, &rec.PositionQuantity, &rec.PositionSide, &rec.PositionCurrency,
			&rec.PriceID, &rec.MarketValueLocal, &rec.MarketValueBase,
			&rec.ValuationFXRate, &rec.CostBasisLocal, &rec.UnrealizedPLLocal,
			&rec.PositionSource, &rec.PositionConfidence, &rec.IsReconciled,
			&rec.ReconciliationDiff, &sourceSystems,
			&rec.CreatedAt, &rec.UpdatedAt, &rec.ValidFrom, &rec.ValidTo,
		); err != nil {
			return fmt.Errorf("scan cluster record: %w", err)
		}
		rec.SourceSystems = sourceSystems
		competitors = append(competitors, rec)
	}
	if err := clusterRows.Err(); err != nil {
		return err
	}

	// 2. Build raw records for survivorship (including the incoming record)
	rawAll := []*goldcopy.RawPortfolioRecord{toRawRecord(incoming)}
	for i := range competitors {
		rawAll = append(rawAll, toRawRecord(&competitors[i]))
	}

	// 3. Run field-level survivorship for all mastered fields
	fieldsToResolve := []string{
		"position_quantity", "market_value_local", "market_value_base",
		"cost_basis_local", "valuation_fx_rate", "position_currency",
		"is_reconciled", "position_side",
	}
	var survivorshipLog []goldcopy.SurvivorshipResult
	gold := *incoming // start with incoming as base
	gold.TenantID = tenantID

	for _, field := range fieldsToResolve {
		result := r.survivorship.Resolve(field, rawAll)
		survivorshipLog = append(survivorshipLog, result)
	}
	applyWinningField(&gold, survivorshipLog)

	// 4. Compute derived fields
	if gold.MarketValueLocal != nil && gold.CostBasisLocal != nil {
		pl := gold.MarketValueLocal.Sub(*gold.CostBasisLocal)
		gold.UnrealizedPLLocal = &pl
		if gold.CostBasisLocal.IsPositive() {
			pct := pl.Div(*gold.CostBasisLocal)
			gold.UnrealizedPLPct = &pct
		}
	}
	gold.PositionConfidence = sourceQualityScore[gold.PositionSource]

	// 5. Build merged source_systems JSONB
	sourceMap := map[string]interface{}{}
	for _, c := range competitors {
		sourceMap[c.PositionSource] = map[string]interface{}{
			"quantity":   c.PositionQuantity.String(),
			"confidence": sourceQualityScore[c.PositionSource],
		}
	}
	sourceMap[incoming.PositionSource] = map[string]interface{}{
		"quantity":    incoming.PositionQuantity.String(),
		"confidence":  sourceQualityScore[incoming.PositionSource],
		"ingested_at": time.Now(),
	}
	sourceSystems, _ := json.Marshal(sourceMap)
	gold.SourceSystems = sourceSystems

	// 6. Upsert the gold copy record
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO edm.position_master (
			tenant_id, portfolio_id, security_id, position_date,
			position_quantity, position_side, position_currency, price_id,
			market_value_local, market_value_base, valuation_fx_rate,
			cost_basis_local, unrealized_pl_local, unrealized_pl_pct,
			position_weight_pct, position_source, position_confidence,
			is_reconciled, reconciliation_diff, source_systems, updated_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,NOW()
		)
		ON CONFLICT (portfolio_id, security_id, position_date, position_source, tenant_id)
		DO UPDATE SET
			position_quantity   = EXCLUDED.position_quantity,
			market_value_local  = EXCLUDED.market_value_local,
			market_value_base   = EXCLUDED.market_value_base,
			valuation_fx_rate   = EXCLUDED.valuation_fx_rate,
			cost_basis_local    = EXCLUDED.cost_basis_local,
			unrealized_pl_local = EXCLUDED.unrealized_pl_local,
			unrealized_pl_pct   = EXCLUDED.unrealized_pl_pct,
			position_confidence = EXCLUDED.position_confidence,
			is_reconciled       = EXCLUDED.is_reconciled,
			source_systems      = edm.position_master.source_systems || EXCLUDED.source_systems,
			updated_at          = NOW()`,
		tenantID, gold.PortfolioID, gold.SecurityID, gold.PositionDate,
		gold.PositionQuantity, gold.PositionSide, gold.PositionCurrency, gold.PriceID,
		gold.MarketValueLocal, gold.MarketValueBase, gold.ValuationFXRate,
		gold.CostBasisLocal, gold.UnrealizedPLLocal, gold.UnrealizedPLPct,
		gold.PositionWeightPct, gold.PositionSource, gold.PositionConfidence,
		gold.IsReconciled, gold.ReconciliationDiff, sourceSystems,
	)
	if err != nil {
		return fmt.Errorf("upsert position: %w", err)
	}

	// 7. Record snapshot for custodian-sourced updates
	if gold.IsReconciled {
		if snapErr := r.recordSnapshotForPosition(ctx, tenantID, &gold); snapErr != nil {
			log.Printf("Warning: failed to record position snapshot: %v", snapErr)
		}
	}
	return nil
}

// recordSnapshotForPosition writes an append-only snapshot after each reconciled upsert.
func (r *Repository) recordSnapshotForPosition(ctx context.Context, tenantID uuid.UUID, rec *PositionMasterRecord) error {
	var posID uuid.UUID
	err := r.db.QueryRowContext(ctx, `
		SELECT id FROM edm.position_master
		WHERE portfolio_id = $1 AND security_id = $2 AND position_date = $3
		  AND position_source = $4 AND tenant_id = $5 LIMIT 1`,
		rec.PortfolioID, rec.SecurityID, rec.PositionDate, rec.PositionSource, tenantID,
	).Scan(&posID)
	if err != nil {
		return fmt.Errorf("lookup position id for snapshot: %w", err)
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO edm.position_snapshot_master (
			tenant_id, position_id, snapshot_date, snapshot_quantity,
			snapshot_market_value, snapshot_price_used, snapshot_fx_rate, snapshot_source
		) VALUES ($1,$2,$3,$4,$5,$6,$7,'GoldCopyRun')
		ON CONFLICT (position_id, snapshot_date) DO UPDATE
		SET snapshot_quantity     = EXCLUDED.snapshot_quantity,
		    snapshot_market_value = EXCLUDED.snapshot_market_value`,
		tenantID, posID, rec.PositionDate, rec.PositionQuantity,
		rec.MarketValueLocal, nil, rec.ValuationFXRate,
	)
	return err
}

// UpsertCashPosition upserts a cash balance with source merging.
func (r *Repository) UpsertCashPosition(ctx context.Context, tenantID uuid.UUID, rec *CashPositionRecord) error {
	sourceSystems, _ := json.Marshal(map[string]interface{}{
		rec.CashSource: map[string]interface{}{
			"balance":     rec.BalanceAmount.String(),
			"ingested_at": time.Now(),
		},
	})

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO edm.cash_position_master (
			tenant_id, portfolio_id, cash_currency, value_date,
			balance_amount, available_balance, interest_accrued,
			cash_source, source_systems, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NOW())
		ON CONFLICT (portfolio_id, cash_currency, value_date, cash_source, tenant_id)
		DO UPDATE SET
			balance_amount    = EXCLUDED.balance_amount,
			available_balance = EXCLUDED.available_balance,
			interest_accrued  = EXCLUDED.interest_accrued,
			source_systems    = edm.cash_position_master.source_systems || EXCLUDED.source_systems,
			updated_at        = NOW()`,
		tenantID, rec.PortfolioID, rec.CashCurrency, rec.ValueDate,
		rec.BalanceAmount, rec.AvailableBalance, rec.InterestAccrued,
		rec.CashSource, sourceSystems,
	)
	return err
}
