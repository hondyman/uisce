package cash

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/goldcopy"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
)

var sourceQualityScore = map[string]int{
	"AccountingSystem": 98,
	"Custodian":        85,
	"OMS":              75,
	"TreasurySystem":   60,
}

var defaultCashLedgerRules = []*goldcopy.SurvivorshipRule{
	{EntityType: "cash_ledger", FieldName: "amount", Strategy: "prefer_source", PreferredSources: []string{"AccountingSystem", "Custodian"}, Priority: 1},
	{EntityType: "cash_ledger", FieldName: "value_date", Strategy: "prefer_source", PreferredSources: []string{"Custodian", "AccountingSystem"}, Priority: 1},
	{EntityType: "cash_ledger", FieldName: "status", Strategy: "prefer_source", PreferredSources: []string{"AccountingSystem", "Custodian"}, Priority: 1},
}

func toRawRecord(l *CashLedgerEntryRecord) *goldcopy.RawPortfolioRecord {
	q, _ := sourceQualityScore[l.SourceSystem]
	fields := map[string]string{
		"currency":        l.Currency,
		"cash_event_type": l.CashEventType,
		"status":          l.Status,
	}
	if !l.Amount.IsZero() {
		fields["amount"] = l.Amount.String()
	}
	if !l.ValueDate.IsZero() {
		fields["value_date"] = l.ValueDate.Format(time.RFC3339)
	}

	return &goldcopy.RawPortfolioRecord{
		PortfolioID:  l.PortfolioID.String(),
		SourceSystem: l.SourceSystem,
		Fields:       fields,
		QualityScore: q,
	}
}

func applyWinningField(l *CashLedgerEntryRecord, results []goldcopy.SurvivorshipResult) {
	for _, r := range results {
		switch r.FieldName {
		case "amount":
			if v, err := decimal.NewFromString(r.ChosenValue); err == nil {
				l.Amount = v
			}
		case "value_date":
			if v, err := time.Parse(time.RFC3339, r.ChosenValue); err == nil {
				l.ValueDate = v
			}
		case "status":
			if r.ChosenValue != "" {
				l.Status = r.ChosenValue
			}
		}
	}
}

type CashRepository interface {
	ListCashBalances(ctx context.Context, portfolioID *uuid.UUID, startDate, endDate *string) ([]CashBalanceRecord, error)
	ListCashLedger(ctx context.Context, portfolioID *uuid.UUID, startDate, endDate *string) ([]CashLedgerEntryRecord, error)
	UpsertCashLedger(ctx context.Context, l *CashLedgerEntryRecord, sourcePriority []string) (*CashLedgerEntryRecord, error)
	RunBalanceRollForward(ctx context.Context, portfolioID uuid.UUID, currency string, valuationDate string) (*CashBalanceRecord, error)
	RecordFlowTrace(ctx context.Context, balanceID uuid.UUID, ledgerID uuid.UUID, amount decimal.Decimal, contributionType string) error
	CreateTransactionCashMapping(ctx context.Context, mapping *TransactionCashMapping) error
}

type pgCashRepo struct {
	db           *sqlx.DB
	survivorship *goldcopy.Survivorship
}

func NewCashRepository(db *sqlx.DB) CashRepository {
	return &pgCashRepo{
		db:           db,
		survivorship: goldcopy.NewSurvivorship(defaultCashLedgerRules),
	}
}

func (r *pgCashRepo) ListCashBalances(ctx context.Context, portfolioID *uuid.UUID, startDate, endDate *string) ([]CashBalanceRecord, error) {
	query := `
		SELECT * FROM edm.cash_balance_master 
		WHERE valid_to = 'infinity' AND tenant_id = $1
	`
	args := []interface{}{ctx.Value("tenant_id").(uuid.UUID)}
	argIdx := 2

	if portfolioID != nil {
		query += fmt.Sprintf(" AND portfolio_id = $%d", argIdx)
		args = append(args, *portfolioID)
		argIdx++
	}
	if startDate != nil {
		query += fmt.Sprintf(" AND valuation_date >= $%d", argIdx)
		args = append(args, *startDate)
		argIdx++
	}
	if endDate != nil {
		query += fmt.Sprintf(" AND valuation_date <= $%d", argIdx)
		args = append(args, *endDate)
		argIdx++
	}
	query += " ORDER BY valuation_date DESC"

	var balances []CashBalanceRecord
	err := r.db.SelectContext(ctx, &balances, query, args...)
	return balances, err
}

func (r *pgCashRepo) ListCashLedger(ctx context.Context, portfolioID *uuid.UUID, startDate, endDate *string) ([]CashLedgerEntryRecord, error) {
	query := `
		SELECT * FROM edm.cash_ledger 
		WHERE valid_to = 'infinity' AND tenant_id = $1
	`
	args := []interface{}{ctx.Value("tenant_id").(uuid.UUID)}
	argIdx := 2

	if portfolioID != nil {
		query += fmt.Sprintf(" AND portfolio_id = $%d", argIdx)
		args = append(args, *portfolioID)
		argIdx++
	}
	if startDate != nil {
		query += fmt.Sprintf(" AND value_date >= $%d", argIdx)
		args = append(args, *startDate)
		argIdx++
	}
	if endDate != nil {
		query += fmt.Sprintf(" AND value_date <= $%d", argIdx)
		args = append(args, *endDate)
		argIdx++
	}
	query += " ORDER BY value_date DESC"

	var ledger []CashLedgerEntryRecord
	err := r.db.SelectContext(ctx, &ledger, query, args...)
	return ledger, err
}

func (r *pgCashRepo) UpsertCashLedger(ctx context.Context, l *CashLedgerEntryRecord, sourcePriority []string) (*CashLedgerEntryRecord, error) {
	clusterQuery := `
		SELECT * FROM edm.cash_ledger 
		WHERE portfolio_id = $1 AND value_date = $2 AND external_reference = $3 
		AND valid_to = 'infinity' AND tenant_id = $4
	`
	var existing []CashLedgerEntryRecord
	err := r.db.SelectContext(ctx, &existing, clusterQuery,
		l.PortfolioID, l.ValueDate, l.ExternalReference, l.TenantID)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	rawAll := []*goldcopy.RawPortfolioRecord{toRawRecord(l)}
	for i := range existing {
		rawAll = append(rawAll, toRawRecord(&existing[i]))
	}

	fieldsToResolve := []string{"amount", "value_date", "status"}
	var survivorshipLog []goldcopy.SurvivorshipResult
	gold := *l

	for _, field := range fieldsToResolve {
		result := r.survivorship.Resolve(field, rawAll)
		survivorshipLog = append(survivorshipLog, result)
	}
	applyWinningField(&gold, survivorshipLog)

	if len(existing) > 0 {
		var oldIDs []string
		for _, rec := range existing {
			oldIDs = append(oldIDs, rec.CashLedgerID.String())
		}
		if len(oldIDs) > 0 {
			for _, id := range oldIDs {
				_, _ = r.db.ExecContext(ctx, "UPDATE edm.cash_ledger SET valid_to = NOW() WHERE cash_ledger_id = $1", id)
			}
		}
	}

	insertQuery := `
		INSERT INTO edm.cash_ledger (
			cash_ledger_id, portfolio_id, cash_account_id, currency, value_date, booking_date,
			cash_event_type, cash_event_subtype, amount, amount_sign, transaction_id, security_id,
			counterparty_id, status, source_system, external_reference, valid_from, valid_to, tenant_id, core_id
		) VALUES (
			:cash_ledger_id, :portfolio_id, :cash_account_id, :currency, :value_date, :booking_date,
			:cash_event_type, :cash_event_subtype, :amount, :amount_sign, :transaction_id, :security_id,
			:counterparty_id, :status, :source_system, :external_reference, :valid_from, :valid_to, :tenant_id, :core_id
		) RETURNING *
	`

	if gold.CashLedgerID == uuid.Nil {
		gold.CashLedgerID = uuid.New()
	}
	if gold.ValidTo.IsZero() {
		gold.ValidTo, _ = time.Parse(time.RFC3339, "9999-12-31T23:59:59Z")
	}

	rows, err := r.db.NamedQueryContext(ctx, insertQuery, gold)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var returned CashLedgerEntryRecord
		err = rows.StructScan(&returned)
		if err != nil {
			return nil, err
		}
		return &returned, nil
	}

	return nil, fmt.Errorf("failed to insert and return cash ledger record")
}

func (r *pgCashRepo) RunBalanceRollForward(ctx context.Context, portfolioID uuid.UUID, currency string, valuationDate string) (*CashBalanceRecord, error) {
	priorQuery := `
		SELECT closing_balance FROM edm.cash_balance_master 
		WHERE portfolio_id = $1 AND currency = $2 AND valuation_date < $3 
		AND valid_to = 'infinity' AND tenant_id = $4
		ORDER BY valuation_date DESC LIMIT 1
	`
	var openingBalance decimal.Decimal
	err := r.db.QueryRowxContext(ctx, priorQuery,
		portfolioID, currency, valuationDate, ctx.Value("tenant_id").(uuid.UUID),
	).Scan(&openingBalance)
	if err == sql.ErrNoRows {
		openingBalance = decimal.Zero
	} else if err != nil {
		return nil, err
	}

	ledgerQuery := `
		SELECT 
			COALESCE(SUM(CASE WHEN amount > 0 AND cash_event_type != 'INTEREST' AND cash_event_type NOT LIKE 'FX%' THEN amount ELSE 0 END), 0) as inflows,
			COALESCE(SUM(CASE WHEN amount < 0 AND cash_event_type != 'INTEREST' AND cash_event_type NOT LIKE 'FX%' THEN ABS(amount) ELSE 0 END), 0) as outflows,
			COALESCE(SUM(CASE WHEN cash_event_type = 'INTEREST' THEN amount ELSE 0 END), 0) as interest,
			COALESCE(SUM(CASE WHEN cash_event_type LIKE 'FX%' THEN amount ELSE 0 END), 0) as fx
		FROM edm.cash_ledger 
		WHERE portfolio_id = $1 AND currency = $2 AND value_date = $3 
		AND valid_to = 'infinity' AND tenant_id = $4
	`
	var inflows, outflows, interest, fx decimal.Decimal
	err = r.db.QueryRowxContext(ctx, ledgerQuery,
		portfolioID, currency, valuationDate, ctx.Value("tenant_id").(uuid.UUID),
	).Scan(&inflows, &outflows, &interest, &fx)
	if err != nil {
		return nil, err
	}

	closing := openingBalance.Add(inflows).Sub(outflows).Add(interest).Add(fx)

	// Invalidate previous for this date if exists
	invalidateQuery := `
		UPDATE edm.cash_balance_master 
		SET valid_to = NOW() 
		WHERE portfolio_id = $1 AND currency = $2 AND valuation_date = $3
		AND valid_to = 'infinity' AND tenant_id = $4
		RETURNING cash_balance_id
	`
	var oldID uuid.UUID
	err = r.db.QueryRowxContext(ctx, invalidateQuery, portfolioID, currency, valuationDate, ctx.Value("tenant_id").(uuid.UUID)).Scan(&oldID)
	// We ignore ErrNoRows, it's fine if there wasn't one

	vDate, _ := time.Parse("2006-01-02", valuationDate)
	validTo, _ := time.Parse(time.RFC3339, "9999-12-31T23:59:59Z")

	record := CashBalanceRecord{
		CashBalanceID:   uuid.New(),
		PortfolioID:     portfolioID,
		Currency:        currency,
		ValuationDate:   vDate,
		OpeningBalance:  &openingBalance,
		CashInflows:     &inflows,
		CashOutflows:    &outflows,
		InterestAccrual: &interest,
		FXEffect:        &fx,
		ClosingBalance:  &closing,
		SourceSystem:    "SemlayerRollForward",
		ValidFrom:       time.Now(),
		ValidTo:         validTo,
		TenantID:        ctx.Value("tenant_id").(uuid.UUID),
	}

	insertQuery := `
		INSERT INTO edm.cash_balance_master (
			cash_balance_id, portfolio_id, currency, valuation_date,
			opening_balance, cash_inflows, cash_outflows, interest_accrual, fx_effect, closing_balance,
			source_system, is_closed, valid_from, valid_to, tenant_id
		) VALUES (
			:cash_balance_id, :portfolio_id, :currency, :valuation_date,
			:opening_balance, :cash_inflows, :cash_outflows, :interest_accrual, :fx_effect, :closing_balance,
			:source_system, :is_closed, :valid_from, :valid_to, :tenant_id
		) RETURNING *
	`

	rows, err := r.db.NamedQueryContext(ctx, insertQuery, record)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var returned CashBalanceRecord
		err = rows.StructScan(&returned)
		if err != nil {
			return nil, err
		}
		return &returned, nil
	}

	return nil, fmt.Errorf("failed to insert cash balance record")
}

func (r *pgCashRepo) RecordFlowTrace(ctx context.Context, balanceID uuid.UUID, ledgerID uuid.UUID, amount decimal.Decimal, contributionType string) error {
	query := `
		INSERT INTO edm.cash_flow_trace (trace_id, cash_balance_id, cash_ledger_id, contribution_amount, contribution_type, tenant_id)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.ExecContext(ctx, query,
		uuid.New(), balanceID, ledgerID, amount, contributionType, ctx.Value("tenant_id").(uuid.UUID),
	)
	return err
}

func (r *pgCashRepo) CreateTransactionCashMapping(ctx context.Context, mapping *TransactionCashMapping) error {
	query := `
		INSERT INTO edm.transaction_cash_mapping (mapping_id, transaction_id, cash_ledger_id, mapping_type, amount, currency, value_date, tenant_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.ExecContext(ctx, query,
		uuid.New(), mapping.TransactionID, mapping.CashLedgerID, mapping.MappingType,
		mapping.Amount, mapping.Currency, mapping.ValueDate, mapping.TenantID,
	)
	return err
}
