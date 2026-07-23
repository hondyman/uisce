package goldcopy

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// Repository is the data access layer for the gold copy engine.
type Repository struct {
	db        *sql.DB
	publisher interface{} // Use interface{} to avoid circular dependency (or define a local interface)
}

// GoldCopyEventPublisher defines the local interface for publishing gold copy events
type GoldCopyEventPublisher interface {
	PublishSecurityAsGoldCopy(
		ctx context.Context,
		security interface{},
		tenantID string,
		securityID string,
		primaryIdentifier string,
		changeType string,
		changeReason string,
		publishedByUserID string,
		dataHash string,
	) error
}

func NewRepository(db *sql.DB, publisher GoldCopyEventPublisher) *Repository {
	return &Repository{
		db:        db,
		publisher: publisher,
	}
}

// ─── Survivorship Rules ───────────────────────────────────────────────────────

// ListSurvivorshipRules returns all active survivorship rules for the given
// entity type, ordered by priority ascending.
func (r *Repository) ListSurvivorshipRules(ctx context.Context, tenantID uuid.UUID, entityType string) ([]*SurvivorshipRule, error) {
	query := `
		SELECT entity_type, field_name, strategy, preferred_sources, time_field, priority
		FROM edm.survivorship_rules
		WHERE tenant_id = $1 AND entity_type = $2 AND is_active = true
		ORDER BY field_name, priority ASC`
	rows, err := r.db.QueryContext(ctx, query, tenantID, entityType)
	if err != nil {
		return nil, fmt.Errorf("goldcopy: list survivorship rules: %w", err)
	}
	defer rows.Close()

	var rules []*SurvivorshipRule
	for rows.Next() {
		var (
			s         SurvivorshipRule
			timeField sql.NullString
		)
		if err := rows.Scan(
			&s.EntityType,
			&s.FieldName,
			&s.Strategy,
			pq.Array(&s.PreferredSources),
			&timeField,
			&s.Priority,
		); err != nil {
			return nil, fmt.Errorf("goldcopy: scan survivorship rule: %w", err)
		}
		if timeField.Valid {
			s.TimeField = timeField.String
		}
		rules = append(rules, &s)
	}
	return rules, rows.Err()
}

// ─── Portfolio Master ─────────────────────────────────────────────────────────

// GetCurrentPortfolioMaster returns the most-recent valid portfolio master record.
func (r *Repository) GetCurrentPortfolioMaster(ctx context.Context, tenantID uuid.UUID, portfolioID string) (*PortfolioMasterRecord, error) {
	query := `
		SELECT id, tenant_id, core_id, portfolio_id, portfolio_code, portfolio_name,
		       portfolio_type, portfolio_category, inception_date, termination_date,
		       base_currency, domicile, legal_structure, regulatory_classification,
		       liquidity_profile, risk_profile, investment_objective, investment_guidelines,
		       valuation_frequency, pricing_source, is_model_portfolio, is_composite_member,
		       benchmark_id, strategy_id, mandate_id, composite_id, performance_settings_id,
		       portfolio_manager_id, custodian_id,
		       confidence_score, source_systems, created_at, updated_at, valid_from, valid_to
		FROM edm.portfolio_master
		WHERE tenant_id = $1 AND portfolio_id = $2
		  AND (valid_to IS NULL OR valid_to > NOW())
		ORDER BY valid_from DESC
		LIMIT 1`
	row := r.db.QueryRowContext(ctx, query, tenantID, portfolioID)
	rec, err := scanPortfolioMaster(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return rec, err
}

// ListCurrentPortfolioMasters returns all current (valid_to IS NULL) portfolio master records.
func (r *Repository) ListCurrentPortfolioMasters(ctx context.Context, tenantID uuid.UUID, portfolioType string) ([]*PortfolioMasterRecord, error) {
	query := `
		SELECT id, tenant_id, core_id, portfolio_id, portfolio_code, portfolio_name,
		       portfolio_type, portfolio_category, inception_date, termination_date,
		       base_currency, domicile, legal_structure, regulatory_classification,
		       liquidity_profile, risk_profile, investment_objective, investment_guidelines,
		       valuation_frequency, pricing_source, is_model_portfolio, is_composite_member,
		       benchmark_id, strategy_id, mandate_id, composite_id, performance_settings_id,
		       portfolio_manager_id, custodian_id,
		       confidence_score, source_systems, created_at, updated_at, valid_from, valid_to
		FROM edm.portfolio_master
		WHERE tenant_id = $1 AND valid_to IS NULL`
	args := []interface{}{tenantID}
	if portfolioType != "" {
		query += " AND portfolio_type = $2"
		args = append(args, portfolioType)
	}
	query += " ORDER BY portfolio_name"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("goldcopy: list portfolio masters: %w", err)
	}
	defer rows.Close()

	var records []*PortfolioMasterRecord
	for rows.Next() {
		rec, err := scanPortfolioMasterRow(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}

// ─── Performance Settings ─────────────────────────────────────────────────────

// GetCurrentPerformanceSettings returns the most-recent valid performance settings record.
func (r *Repository) GetCurrentPerformanceSettings(ctx context.Context, tenantID uuid.UUID, portfolioID string) (*PerformanceSettingsRecord, error) {
	query := `
		SELECT id, tenant_id, core_id, portfolio_id, valuation_method, fee_treatment,
		       cash_flow_method, currency_hedging_policy, lookthrough_policy,
		       treatment_of_derivatives, confidence_score, source_systems,
		       created_at, updated_at, valid_from, valid_to
		FROM edm.performance_settings
		WHERE tenant_id = $1 AND portfolio_id = $2
		  AND (valid_to IS NULL OR valid_to > NOW())
		ORDER BY valid_from DESC
		LIMIT 1`
	row := r.db.QueryRowContext(ctx, query, tenantID, portfolioID)
	rec, err := scanPerformanceSettings(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return rec, err
}

// UpsertPerformanceSettings inserts a NEW version of a performance settings record.
func (r *Repository) UpsertPerformanceSettings(ctx context.Context, rec *PerformanceSettingsRecord) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("goldcopy: upsert ps begin tx: %w", err)
	}
	defer tx.Rollback()

	now := time.Now()

	// 1. Close old record
	_, err = tx.ExecContext(ctx, `
		UPDATE edm.performance_settings
		SET valid_to = $1, updated_at = $2
		WHERE tenant_id = $3 AND portfolio_id = $4 AND valid_to IS NULL`,
		now, now, rec.TenantID, rec.PortfolioID,
	)
	if err != nil {
		return fmt.Errorf("goldcopy: close old perf settings: %w", err)
	}

	// 2. Insert new record
	sysJSON, _ := json.Marshal(rec.SourceSystems)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO edm.performance_settings (
			id, tenant_id, core_id, portfolio_id, valuation_method, fee_treatment,
			cash_flow_method, currency_hedging_policy, lookthrough_policy,
			treatment_of_derivatives, confidence_score, source_systems,
			created_at, updated_at, created_by, updated_by, valid_from)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`,
		rec.ID, rec.TenantID, rec.CoreID, rec.PortfolioID, rec.ValuationMethod, rec.FeeTreatment,
		rec.CashFlowMethod, rec.CurrencyHedging, rec.LookthroughPolicy,
		rec.DerivativesPolicy, rec.ConfidenceScore, sysJSON,
		now, now, rec.TenantID, rec.TenantID, now,
	)
	if err != nil {
		return fmt.Errorf("goldcopy: insert perf settings: %w", err)
	}
	return tx.Commit()
}

// UpsertPortfolioMaster inserts a NEW version of a portfolio master record
// (bi-temporal: it does not overwrite — it closes the old record and inserts a new one).
func (r *Repository) UpsertPortfolioMaster(ctx context.Context, rec *PortfolioMasterRecord) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("goldcopy: upsert begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	now := time.Now()

	// 1. Close the previous current record (if any)
	_, err = tx.ExecContext(ctx, `
		UPDATE edm.portfolio_master
		SET valid_to = $1, updated_at = $2, updated_by = $3
		WHERE tenant_id = $4 AND portfolio_id = $5 AND valid_to IS NULL`,
		now, now, rec.UpdatedAt /* updated_by */, rec.TenantID, rec.PortfolioID,
	)
	if err != nil {
		return fmt.Errorf("goldcopy: close old portfolio master: %w", err)
	}

	// 2. Insert the new current record
	sysJSON, _ := json.Marshal(rec.SourceSystems)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO edm.portfolio_master (
			id, tenant_id, core_id, portfolio_id, portfolio_code, portfolio_name,
			portfolio_type, portfolio_category, inception_date, termination_date,
			base_currency, domicile, legal_structure, regulatory_classification,
			liquidity_profile, risk_profile, investment_objective, investment_guidelines,
			valuation_frequency, pricing_source, is_model_portfolio, is_composite_member,
			benchmark_id, strategy_id, mandate_id, composite_id, performance_settings_id,
			portfolio_manager_id, custodian_id,
			confidence_score, source_systems,
			created_at, updated_at, created_by, updated_by, valid_from)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,
		        $23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,$35,$36)`,
		rec.ID, rec.TenantID, rec.CoreID, rec.PortfolioID, rec.PortfolioCode, rec.PortfolioName,
		rec.PortfolioType, rec.PortfolioCategory, rec.InceptionDate, rec.TerminationDate,
		rec.BaseCurrency, rec.Domicile, rec.LegalStructure, rec.RegulatoryClassification,
		rec.LiquidityProfile, rec.RiskProfile, rec.InvestmentObjective, rec.InvestmentGuidelines,
		rec.ValuationFrequency, rec.PricingSource, rec.IsModelPortfolio, rec.IsCompositeMember,
		rec.BenchmarkID, rec.StrategyID, rec.MandateID, rec.CompositeID, rec.PerformanceSettingsID,
		rec.PortfolioManagerID, rec.CustodianID,
		rec.ConfidenceScore, sysJSON,
		now, now, rec.TenantID /* created_by */, rec.TenantID /* updated_by */, now,
	)
	if err != nil {
		return fmt.Errorf("goldcopy: insert portfolio master: %w", err)
	}
	return tx.Commit()
}

// ─── Lineage ──────────────────────────────────────────────────────────────────

// InsertLineageEntries bulk-inserts lineage traces for a gold copy run.
func (r *Repository) InsertLineageEntries(ctx context.Context, entries []*GoldCopyLineageEntry) error {
	if len(entries) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("goldcopy: lineage begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO edm.gold_copy_lineage
		  (id, tenant_id, entity_type, entity_id, field_name,
		   chosen_value, chosen_source, rejected_sources, survivorship_rule,
		   dq_rules_passed, dq_rules_failed, confidence_contribution, run_id, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`)
	if err != nil {
		return fmt.Errorf("goldcopy: prepare lineage stmt: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for _, e := range entries {
		rejJSON, _ := json.Marshal(e.RejectedSources)
		if _, err := stmt.ExecContext(ctx,
			e.ID, e.TenantID, e.EntityType, e.EntityID, e.FieldName,
			e.ChosenValue, e.ChosenSource, rejJSON, e.SurvivorshipRule,
			pq.Array(e.DQRulesPassed), pq.Array(e.DQRulesFailed),
			e.ConfidenceContribution, e.RunID, now,
		); err != nil {
			return fmt.Errorf("goldcopy: insert lineage entry: %w", err)
		}
	}
	return tx.Commit()
}

// GetLineageForEntity returns all lineage entries for a specific entity record.
func (r *Repository) GetLineageForEntity(ctx context.Context, tenantID, entityID uuid.UUID) ([]*GoldCopyLineageEntry, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, tenant_id, entity_type, entity_id, field_name,
		       chosen_value, chosen_source, rejected_sources, survivorship_rule,
		       dq_rules_passed, dq_rules_failed, confidence_contribution, run_id, created_at
		FROM edm.gold_copy_lineage
		WHERE tenant_id = $1 AND entity_id = $2
		ORDER BY created_at DESC`, tenantID, entityID)
	if err != nil {
		return nil, fmt.Errorf("goldcopy: get lineage: %w", err)
	}
	defer rows.Close()

	var entries []*GoldCopyLineageEntry
	for rows.Next() {
		e := &GoldCopyLineageEntry{}
		var rejJSON []byte
		if err := rows.Scan(
			&e.ID, &e.TenantID, &e.EntityType, &e.EntityID, &e.FieldName,
			&e.ChosenValue, &e.ChosenSource, &rejJSON, &e.SurvivorshipRule,
			pq.Array(&e.DQRulesPassed), pq.Array(&e.DQRulesFailed),
			&e.ConfidenceContribution, &e.RunID, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(rejJSON, &e.RejectedSources)
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// ─── Scan helpers ─────────────────────────────────────────────────────────────

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanPortfolioMaster(row rowScanner) (*PortfolioMasterRecord, error) {
	return scanPortfolioMasterRow(row)
}

func scanPortfolioMasterRow(row rowScanner) (*PortfolioMasterRecord, error) {
	rec := &PortfolioMasterRecord{}
	var (
		coreID            pq.NullTime
		termDate          pq.NullTime
		benchmarkID       uuid.NullUUID
		strategyID        uuid.NullUUID
		mandateID         uuid.NullUUID
		compositeID       uuid.NullUUID
		perfSettingsID    uuid.NullUUID
		coreUUID          uuid.NullUUID
		portfolioCategory sql.NullString
		domicile          sql.NullString
		legalStructure    sql.NullString
		regClass          sql.NullString
		liquidity         sql.NullString
		riskProfile       sql.NullString
		investObj         sql.NullString
		investGdl         sql.NullString
		valFreq           sql.NullString
		priceSrc          sql.NullString
		pmID              sql.NullString
		custodian         sql.NullString
		validTo           pq.NullTime
		sysJSON           []byte
	)
	_ = coreID // unused, handled via coreUUID below
	err := row.Scan(
		&rec.ID, &rec.TenantID, &coreUUID, &rec.PortfolioID, &rec.PortfolioCode, &rec.PortfolioName,
		&rec.PortfolioType, &portfolioCategory, &rec.InceptionDate, &termDate,
		&rec.BaseCurrency, &domicile, &legalStructure, &regClass,
		&liquidity, &riskProfile, &investObj, &investGdl,
		&valFreq, &priceSrc, &rec.IsModelPortfolio, &rec.IsCompositeMember,
		&benchmarkID, &strategyID, &mandateID, &compositeID, &perfSettingsID,
		&pmID, &custodian,
		&rec.ConfidenceScore, &sysJSON, &rec.CreatedAt, &rec.UpdatedAt, &rec.ValidFrom, &validTo,
	)
	if err != nil {
		return nil, err
	}
	if coreUUID.Valid {
		rec.CoreID = &coreUUID.UUID
	}
	if termDate.Valid {
		rec.TerminationDate = &termDate.Time
	}
	if benchmarkID.Valid {
		rec.BenchmarkID = &benchmarkID.UUID
	}
	if strategyID.Valid {
		rec.StrategyID = &strategyID.UUID
	}
	if mandateID.Valid {
		rec.MandateID = &mandateID.UUID
	}
	if compositeID.Valid {
		rec.CompositeID = &compositeID.UUID
	}
	if perfSettingsID.Valid {
		rec.PerformanceSettingsID = &perfSettingsID.UUID
	}
	if validTo.Valid {
		rec.ValidTo = &validTo.Time
	}
	rec.PortfolioCategory = portfolioCategory.String
	rec.Domicile = domicile.String
	rec.LegalStructure = legalStructure.String
	rec.RegulatoryClassification = regClass.String
	rec.LiquidityProfile = liquidity.String
	rec.RiskProfile = riskProfile.String
	rec.InvestmentObjective = investObj.String
	rec.InvestmentGuidelines = investGdl.String
	rec.ValuationFrequency = valFreq.String
	rec.PricingSource = priceSrc.String
	rec.PortfolioManagerID = pmID.String
	rec.CustodianID = custodian.String
	_ = json.Unmarshal(sysJSON, &rec.SourceSystems)
	return rec, nil
}

func scanPerformanceSettings(row rowScanner) (*PerformanceSettingsRecord, error) {
	return scanPerformanceSettingsRow(row)
}

func scanPerformanceSettingsRow(row rowScanner) (*PerformanceSettingsRecord, error) {
	rec := &PerformanceSettingsRecord{}
	var (
		coreUUID    uuid.NullUUID
		cashFlow    sql.NullString
		hedging     sql.NullString
		lookthrough sql.NullString
		derivatives sql.NullString
		validTo     pq.NullTime
		sysJSON     []byte
	)
	err := row.Scan(
		&rec.ID, &rec.TenantID, &coreUUID, &rec.PortfolioID, &rec.ValuationMethod, &rec.FeeTreatment,
		&cashFlow, &hedging, &lookthrough, &derivatives,
		&rec.ConfidenceScore, &sysJSON, &rec.CreatedAt, &rec.UpdatedAt, &rec.ValidFrom, &validTo,
	)
	if err != nil {
		return nil, err
	}
	if coreUUID.Valid {
		rec.CoreID = &coreUUID.UUID
	}
	if validTo.Valid {
		rec.ValidTo = &validTo.Time
	}
	rec.CashFlowMethod = cashFlow.String
	rec.CurrencyHedging = hedging.String
	rec.LookthroughPolicy = lookthrough.String
	rec.DerivativesPolicy = derivatives.String
	_ = json.Unmarshal(sysJSON, &rec.SourceSystems)
	return rec, nil
}
