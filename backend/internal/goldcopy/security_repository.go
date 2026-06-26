package goldcopy

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// ─── Security Master Repository ───────────────────────────────────────────────

// GetCurrentSecurity returns the most-recent valid Security master record.
func (r *Repository) GetCurrentSecurity(ctx context.Context, tenantID uuid.UUID, securityID string) (*SecurityMasterRecord, error) {
	query := `
		SELECT id, tenant_id, core_id,
		       security_id, primary_identifier, isin, cusip, sedol, figi,
		       ticker, local_ticker, ric, bbg_id, vendor_ids,
		       security_name, short_name, description,
		       asset_class, sub_asset_class, instrument_type,
		       sector, industry, currency, settlement_currency,
		       country_of_issue, country_of_risk, region,
		       issue_date, maturity_date,
		       issuer_id, listing_exchange, exchange_code,
		       status, liquidity_profile, regulatory_classification,
		       confidence_score, source_systems, created_at, updated_at, valid_from, valid_to
		FROM edm.security_master
		WHERE tenant_id = $1 AND security_id = $2
		  AND (valid_to IS NULL OR valid_to > NOW())
		ORDER BY valid_from DESC
		LIMIT 1`

	row := r.db.QueryRowContext(ctx, query, tenantID, securityID)
	return scanSecurityMaster(row)
}

// GetCurrentSecurities returns current valid master records for multiple security IDs.
func (r *Repository) GetCurrentSecurities(ctx context.Context, tenantID uuid.UUID, securityIDs []string) ([]*SecurityMasterRecord, error) {
	if len(securityIDs) == 0 {
		return nil, nil
	}

	query := `
		SELECT id, tenant_id, core_id,
		       security_id, primary_identifier, isin, cusip, sedol, figi,
		       ticker, local_ticker, ric, bbg_id, vendor_ids,
		       security_name, short_name, description,
		       asset_class, sub_asset_class, instrument_type,
		       sector, industry, currency, settlement_currency,
		       country_of_issue, country_of_risk, region,
		       issue_date, maturity_date,
		       issuer_id, listing_exchange, exchange_code,
		       status, liquidity_profile, regulatory_classification,
		       confidence_score, source_systems, created_at, updated_at, valid_from, valid_to
		FROM edm.security_master
		WHERE tenant_id = $1 AND security_id = ANY($2)
		  AND (valid_to IS NULL OR valid_to > NOW())`

	rows, err := r.db.QueryContext(ctx, query, tenantID, pq.Array(securityIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*SecurityMasterRecord
	for rows.Next() {
		rec, err := scanSecurityMaster(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, rec)
	}

	return results, nil
}

// UpsertSecurityMaster inserts or updates a security master record (bi-temporal close-and-reopen).
func (r *Repository) UpsertSecurityMaster(ctx context.Context, rec *SecurityMasterRecord) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("goldcopy security: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	// 1. Close any open record for this security_id
	_, err = tx.ExecContext(ctx, `
		UPDATE edm.security_master
		SET valid_to = $1, updated_at = NOW()
		WHERE tenant_id = $2 AND security_id = $3
		  AND (valid_to IS NULL OR valid_to > NOW())`,
		rec.ValidFrom, rec.TenantID, rec.SecurityID)
	if err != nil {
		return fmt.Errorf("goldcopy security: close existing: %w", err)
	}

	// 2. Marshal JSONB columns
	vendorIDsJSON, _ := json.Marshal(rec.VendorIDs)
	sourceSystemsJSON, _ := json.Marshal(rec.SourceSystems)

	// 3. Insert the new gold copy record
	_, err = tx.ExecContext(ctx, `
		INSERT INTO edm.security_master (
			id, tenant_id, core_id,
			security_id, primary_identifier, isin, cusip, sedol, figi,
			ticker, local_ticker, ric, bbg_id, vendor_ids,
			security_name, short_name, description,
			asset_class, sub_asset_class, instrument_type,
			sector, industry, currency, settlement_currency,
			country_of_issue, country_of_risk, region,
			issue_date, maturity_date,
			issuer_id, listing_exchange, exchange_code,
			status, liquidity_profile, regulatory_classification,
			confidence_score, source_systems,
			created_at, updated_at, valid_from, valid_to
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,
			$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,
			$28,$29,$30,$31,$32,$33,$34,$35,$36,$37,
			NOW(),NOW(),$38,$39
		)`,
		rec.ID, rec.TenantID, rec.CoreID,
		rec.SecurityID, rec.PrimaryIdentifier, nilStr(rec.ISIN), nilStr(rec.CUSIP),
		nilStr(rec.SEDOL), nilStr(rec.FIGI),
		nilStr(rec.Ticker), nilStr(rec.LocalTicker), nilStr(rec.RIC), nilStr(rec.BBGID),
		vendorIDsJSON,
		rec.SecurityName, nilStr(rec.ShortName), nilStr(rec.Description),
		rec.AssetClass, nilStr(rec.SubAssetClass), nilStr(rec.InstrumentType),
		nilStr(rec.Sector), nilStr(rec.Industry),
		rec.Currency, nilStr(rec.SettlementCurrency),
		nilStr(rec.CountryOfIssue), nilStr(rec.CountryOfRisk), nilStr(rec.Region),
		rec.IssueDate, rec.MaturityDate,
		rec.IssuerID, nilStr(rec.ListingExchange), nilStr(rec.ExchangeCode),
		rec.Status, nilStr(rec.LiquidityProfile), nilStr(rec.RegulatoryClassification),
		rec.ConfidenceScore, sourceSystemsJSON,
		rec.ValidFrom, rec.ValidTo,
	)
	if err != nil {
		return fmt.Errorf("goldcopy security: insert: %w", err)
	}

	// 4. Upsert subtype attributes
	switch rec.AssetClass {
	case "FixedIncome":
		if rec.FixedIncome != nil {
			if err := upsertFixedIncome(ctx, tx, rec.ID, rec.TenantID, rec.FixedIncome); err != nil {
				return err
			}
		}
	case "Equity":
		if rec.Equity != nil {
			if err := upsertEquity(ctx, tx, rec.ID, rec.TenantID, rec.Equity); err != nil {
				return err
			}
		}
	case "Fund":
		if rec.Fund != nil {
			if err := upsertFund(ctx, tx, rec.ID, rec.TenantID, rec.Fund); err != nil {
				return err
			}
		}
	case "Derivative":
		if rec.Derivative != nil {
			if err := upsertDerivative(ctx, tx, rec.ID, rec.TenantID, rec.Derivative); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

// ─── Subtype upserts ──────────────────────────────────────────────────────────

func upsertFixedIncome(ctx context.Context, tx *sql.Tx, secID, tenantID uuid.UUID, fi *FixedIncomeAttributes) error {
	ratingsJSON, _ := json.Marshal(fi.RatingAgencyRatings)
	_, err := tx.ExecContext(ctx, `
		INSERT INTO edm.fixed_income_attributes (
			security_id, tenant_id,
			coupon_type, coupon_rate, coupon_frequency, day_count_convention,
			issue_price, issue_size, par_value,
			yield_to_maturity, yield_to_worst,
			seniority, secured, collateral_type,
			rating_agency_ratings, rating_composite,
			updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,NOW())
		ON CONFLICT (security_id) DO UPDATE SET
			coupon_type          = EXCLUDED.coupon_type,
			coupon_rate          = EXCLUDED.coupon_rate,
			coupon_frequency     = EXCLUDED.coupon_frequency,
			day_count_convention = EXCLUDED.day_count_convention,
			issue_price          = EXCLUDED.issue_price,
			issue_size           = EXCLUDED.issue_size,
			par_value            = EXCLUDED.par_value,
			yield_to_maturity    = EXCLUDED.yield_to_maturity,
			yield_to_worst       = EXCLUDED.yield_to_worst,
			seniority            = EXCLUDED.seniority,
			secured              = EXCLUDED.secured,
			collateral_type      = EXCLUDED.collateral_type,
			rating_agency_ratings= EXCLUDED.rating_agency_ratings,
			rating_composite     = EXCLUDED.rating_composite,
			updated_at           = NOW()`,
		secID, tenantID,
		fi.CouponType, fi.CouponRate, nilStr(fi.CouponFrequency), nilStr(fi.DayCountConvention),
		fi.IssuePrice, fi.IssueSize, fi.ParValue,
		fi.YieldToMaturity, fi.YieldToWorst,
		nilStr(fi.Seniority), fi.Secured, nilStr(fi.CollateralType),
		ratingsJSON, nilStr(fi.RatingComposite),
	)
	if err != nil {
		return fmt.Errorf("goldcopy security: upsert fixed_income: %w", err)
	}
	return nil
}

func upsertEquity(ctx context.Context, tx *sql.Tx, secID, tenantID uuid.UUID, eq *EquityAttributes) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO edm.equity_attributes (
			security_id, tenant_id,
			share_class, shares_outstanding, free_float, dividend_yield, dividend_frequency,
			updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,NOW())
		ON CONFLICT (security_id) DO UPDATE SET
			share_class        = EXCLUDED.share_class,
			shares_outstanding = EXCLUDED.shares_outstanding,
			free_float         = EXCLUDED.free_float,
			dividend_yield     = EXCLUDED.dividend_yield,
			dividend_frequency = EXCLUDED.dividend_frequency,
			updated_at         = NOW()`,
		secID, tenantID,
		nilStr(eq.ShareClass), eq.SharesOutstanding, eq.FreeFloat, eq.DividendYield, nilStr(eq.DividendFrequency),
	)
	if err != nil {
		return fmt.Errorf("goldcopy security: upsert equity: %w", err)
	}
	return nil
}

func upsertFund(ctx context.Context, tx *sql.Tx, secID, tenantID uuid.UUID, f *FundAttributes) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO edm.fund_attributes (
			security_id, tenant_id,
			fund_type, domicile, management_company, administrator, custodian,
			total_expense_ratio, management_fee, performance_fee,
			distribution_policy, prospectus_link,
			updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,NOW())
		ON CONFLICT (security_id) DO UPDATE SET
			fund_type            = EXCLUDED.fund_type,
			domicile             = EXCLUDED.domicile,
			management_company   = EXCLUDED.management_company,
			administrator        = EXCLUDED.administrator,
			custodian            = EXCLUDED.custodian,
			total_expense_ratio  = EXCLUDED.total_expense_ratio,
			management_fee       = EXCLUDED.management_fee,
			performance_fee      = EXCLUDED.performance_fee,
			distribution_policy  = EXCLUDED.distribution_policy,
			prospectus_link      = EXCLUDED.prospectus_link,
			updated_at           = NOW()`,
		secID, tenantID,
		f.FundType, nilStr(f.Domicile), nilStr(f.ManagementCompany), nilStr(f.Administrator), nilStr(f.Custodian),
		f.TotalExpenseRatio, f.ManagementFee, f.PerformanceFee,
		nilStr(f.DistributionPolicy), nilStr(f.ProspectusLink),
	)
	if err != nil {
		return fmt.Errorf("goldcopy security: upsert fund: %w", err)
	}
	return nil
}

func upsertDerivative(ctx context.Context, tx *sql.Tx, secID, tenantID uuid.UUID, d *DerivativeAttributes) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO edm.derivative_attributes (
			security_id, tenant_id,
			underlier_security_id, underlier_type,
			contract_size, contract_month, strike_price,
			option_type, exercise_style, settlement_type, expiry_date,
			updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,NOW())
		ON CONFLICT (security_id) DO UPDATE SET
			underlier_security_id = EXCLUDED.underlier_security_id,
			underlier_type        = EXCLUDED.underlier_type,
			contract_size         = EXCLUDED.contract_size,
			contract_month        = EXCLUDED.contract_month,
			strike_price          = EXCLUDED.strike_price,
			option_type           = EXCLUDED.option_type,
			exercise_style        = EXCLUDED.exercise_style,
			settlement_type       = EXCLUDED.settlement_type,
			expiry_date           = EXCLUDED.expiry_date,
			updated_at            = NOW()`,
		secID, tenantID,
		d.UnderlierSecurityID, nilStr(d.UnderlierType),
		d.ContractSize, nilStr(d.ContractMonth), d.StrikePrice,
		nilStr(d.OptionType), nilStr(d.ExerciseStyle), nilStr(d.SettlementType), d.ExpiryDate,
	)
	if err != nil {
		return fmt.Errorf("goldcopy security: upsert derivative: %w", err)
	}
	return nil
}

// ─── Gold Trace ───────────────────────────────────────────────────────────────

// InsertSecurityGoldTrace writes per-field lineage for one Security gold copy run.
func (r *Repository) InsertSecurityGoldTrace(ctx context.Context, secID uuid.UUID, tenantID uuid.UUID, runID uuid.UUID, results []SurvivorshipResult) error {
	for _, res := range results {
		rejectedJSON, _ := json.Marshal(res.RejectedSources)
		_, err := r.db.ExecContext(ctx, `
			INSERT INTO edm.security_gold_trace (
				id, tenant_id, security_id, run_id,
				field_name, chosen_value, chosen_source,
				survivorship_rule, rejected_sources, created_at
			) VALUES (gen_random_uuid(),$1,$2,$3,$4,$5,$6,$7,$8,NOW())`,
			tenantID, secID, runID,
			res.FieldName, res.ChosenValue, res.ChosenSource,
			res.Strategy, rejectedJSON,
		)
		if err != nil {
			return fmt.Errorf("goldcopy security: insert trace: %w", err)
		}
	}
	return nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// nilStr converts an empty string to nil for nullable VARCHAR columns.
func nilStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// scanner is an interface that matches both *sql.Row and *sql.Rows
type scanner interface {
	Scan(dest ...interface{}) error
}

// scanSecurityMaster reads one row from security_master into a SecurityMasterRecord.
func scanSecurityMaster(s scanner) (*SecurityMasterRecord, error) {
	var (
		rec                               SecurityMasterRecord
		coreID                            sql.NullString
		isin, cusip                       sql.NullString
		sedol, figi                       sql.NullString
		ticker, lTicker                   sql.NullString
		ric, bbgID                        sql.NullString
		vendorIDs                         []byte
		shortName, desc                   sql.NullString
		subAsset, instrType               sql.NullString
		sector, industry                  sql.NullString
		settleCurr                        sql.NullString
		countryIssue, countryRisk, region sql.NullString
		issueDate                         sql.NullTime
		matDate                           sql.NullTime
		issuerID                          sql.NullString
		listExch, exchCode                sql.NullString
		liqProfile, regClass              sql.NullString
		sourceSystems                     []byte
		validTo                           sql.NullTime
	)
	err := s.Scan(
		&rec.ID, &rec.TenantID, &coreID,
		&rec.SecurityID, &rec.PrimaryIdentifier,
		&isin, &cusip, &sedol, &figi,
		&ticker, &lTicker, &ric, &bbgID, &vendorIDs,
		&rec.SecurityName, &shortName, &desc,
		&rec.AssetClass, &subAsset, &instrType,
		&sector, &industry,
		&rec.Currency, &settleCurr,
		&countryIssue, &countryRisk, &region,
		&issueDate, &matDate,
		&issuerID, &listExch, &exchCode,
		&rec.Status, &liqProfile, &regClass,
		&rec.ConfidenceScore, &sourceSystems,
		&rec.CreatedAt, &rec.UpdatedAt, &rec.ValidFrom, &validTo,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("goldcopy security: scan: %w", err)
	}

	// Nullable strings
	if coreID.Valid {
		id := uuid.MustParse(coreID.String)
		rec.CoreID = &id
	}
	if issuerID.Valid {
		id := uuid.MustParse(issuerID.String)
		rec.IssuerID = &id
	}
	rec.ISIN = isin.String
	rec.CUSIP = cusip.String
	rec.SEDOL = sedol.String
	rec.FIGI = figi.String
	rec.Ticker = ticker.String
	rec.LocalTicker = lTicker.String
	rec.RIC = ric.String
	rec.BBGID = bbgID.String
	rec.ShortName = shortName.String
	rec.Description = desc.String
	rec.SubAssetClass = subAsset.String
	rec.InstrumentType = instrType.String
	rec.Sector = sector.String
	rec.Industry = industry.String
	rec.SettlementCurrency = settleCurr.String
	rec.CountryOfIssue = countryIssue.String
	rec.CountryOfRisk = countryRisk.String
	rec.Region = region.String
	rec.ListingExchange = listExch.String
	rec.ExchangeCode = exchCode.String
	rec.LiquidityProfile = liqProfile.String
	rec.RegulatoryClassification = regClass.String
	if issueDate.Valid {
		rec.IssueDate = &issueDate.Time
	}
	if matDate.Valid {
		rec.MaturityDate = &matDate.Time
	}
	if validTo.Valid {
		t := validTo.Time
		rec.ValidTo = &t
	}

	// JSONB
	if len(vendorIDs) > 0 {
		_ = json.Unmarshal(vendorIDs, &rec.VendorIDs)
	}
	if len(sourceSystems) > 0 {
		_ = json.Unmarshal(sourceSystems, &rec.SourceSystems)
	}

	return &rec, nil
}

// ─── Security Gold Copy Engine ────────────────────────────────────────────────

// BuildSecurityGoldCopy runs the full match→DQ→survivorship→persist pipeline
// for a cluster of raw source records representing the same real-world security.
//
// Matching is assumed to have already occurred upstream; this function receives
// records that have been grouped into a single cluster (same ISIN/FIGI/CUSIP).
func BuildSecurityGoldCopy(
	ctx context.Context,
	repo *Repository,
	tenantID uuid.UUID,
	rawRecords []RawSecurityRecord,
) (*SecurityGoldCopyRunResult, error) {
	if len(rawRecords) == 0 {
		return nil, fmt.Errorf("goldcopy security: no source records provided")
	}

	runID := uuid.New()
	startedAt := time.Now()

	result := &SecurityGoldCopyRunResult{
		RunID:      runID,
		TenantID:   tenantID,
		EntityType: "security",
		ClusterKey: clusterKey(rawRecords),
		StartedAt:  startedAt,
	}

	// 1. Convert to RawPortfolioRecord so we can reuse the Survivorship engine.
	portfolioRecs := securityToPortfolioRecs(rawRecords)

	// 2. Load survivorship rules for the "security" entity type.
	rules, err := repo.ListSurvivorshipRules(ctx, tenantID, "security")
	if err != nil {
		return nil, fmt.Errorf("goldcopy security: load surv rules: %w", err)
	}
	survivorship := NewSurvivorship(rules)

	// 3. Apply survivorship to all mastered fields.
	fieldsToMaster := []string{
		"isin", "cusip", "sedol", "figi", "ticker", "local_ticker", "ric", "bbg_id",
		"security_name", "short_name", "asset_class", "sub_asset_class", "instrument_type",
		"sector", "industry", "currency", "settlement_currency",
		"country_of_issue", "country_of_risk", "region",
		"issue_date", "maturity_date",
		"issuer_id", "listing_exchange", "exchange_code",
		"status", "liquidity_profile", "regulatory_classification",
		// Fixed income
		"coupon_type", "coupon_rate", "coupon_frequency", "day_count_convention",
		"par_value", "issue_size", "seniority", "rating_composite",
		// Equity
		"share_class", "shares_outstanding", "dividend_yield",
		// Fund
		"fund_type", "management_company", "total_expense_ratio", "distribution_policy",
		// Derivative
		"underlier_security_id", "strike_price", "option_type", "exercise_style", "expiry_date",
	}

	var survResults []SurvivorshipResult
	for _, field := range fieldsToMaster {
		survResults = append(survResults, survivorship.Resolve(field, portfolioRecs))
	}
	result.SurvivorshipLog = survResults

	// 4. Build the gold record from survivorship results.
	gold := buildSecurityFromSurv(uuid.New(), tenantID, survResults, rawRecords)
	result.ConfidenceScore = gold.ConfidenceScore

	// 5. Run core DQ rules.
	violations := runSecurityCoreDQ(gold)
	result.DQViolations = violations
	if hasFatal(violations) {
		result.Success = false
		result.ErrorMessage = "hard DQ violations prevented gold copy creation"
		result.CompletedAt = time.Now()
		return result, nil
	}

	// 6. Persist the gold copy (with bi-temporal close-and-reopen).
	gold.ValidFrom = time.Now()
	if err := repo.UpsertSecurityMaster(ctx, gold); err != nil {
		return nil, fmt.Errorf("goldcopy security: upsert: %w", err)
	}

	// 7. Persist lineage trace.
	_ = repo.InsertSecurityGoldTrace(ctx, gold.ID, tenantID, runID, survResults)

	// 8. Publish gold copy event to Redpanda
	if pub, ok := repo.publisher.(GoldCopyEventPublisher); ok && pub != nil {
		hash := "" // TODO: Use actual hash from data if available, or publisher will compute
		err = pub.PublishSecurityAsGoldCopy(
			ctx,
			gold,
			tenantID.String(),
			gold.ID.String(),
			gold.PrimaryIdentifier,
			"update",
			"Security data ingestion and survivorship",
			"system",
			hash,
		)
		if err != nil {
			// Non-blocking error logging
			log.Printf("Warning: Failed to publish security %s to gold copy: %v", gold.ID, err)
		}
	}

	result.GoldenRecord = gold
	result.Success = true
	result.CompletedAt = time.Now()
	return result, nil
}

// ─── Internal helpers ─────────────────────────────────────────────────────────

// clusterKey returns the best available identifier from the first record (for logging).
func clusterKey(records []RawSecurityRecord) string {
	for _, r := range records {
		if r.ISIN != "" {
			return r.ISIN
		}
		if r.FIGI != "" {
			return r.FIGI
		}
		if r.CUSIP != "" {
			return r.CUSIP
		}
	}
	return records[0].SecurityID
}

// securityToPortfolioRecs adapts []RawSecurityRecord → []*RawPortfolioRecord so the
// shared Survivorship engine (which operates on RawPortfolioRecord) can be reused.
func securityToPortfolioRecs(recs []RawSecurityRecord) []*RawPortfolioRecord {
	out := make([]*RawPortfolioRecord, 0, len(recs))
	for _, r := range recs {
		fields := make(map[string]string, len(r.Fields)+4)
		for k, v := range r.Fields {
			fields[k] = v
		}
		// Promote top-level identifier fields into the Fields map so survivorship can use them.
		if r.ISIN != "" {
			fields["isin"] = r.ISIN
		}
		if r.FIGI != "" {
			fields["figi"] = r.FIGI
		}
		if r.CUSIP != "" {
			fields["cusip"] = r.CUSIP
		}
		out = append(out, &RawPortfolioRecord{
			PortfolioID:   r.SecurityID, // reuse PortfolioID as the cluster identifier
			SourceSystem:  r.SourceSystem,
			EffectiveDate: r.EffectiveDate,
			QualityScore:  r.QualityScore,
			Fields:        fields,
		})
	}
	return out
}

// buildSecurityFromSurv assembles a SecurityMasterRecord from survivorship results.
func buildSecurityFromSurv(id, tenantID uuid.UUID, results []SurvivorshipResult, raw []RawSecurityRecord) *SecurityMasterRecord {
	vals := survResultsToMap(results)
	sourceSystems := make(map[string]string, len(results))
	for _, r := range results {
		sourceSystems[r.FieldName] = r.ChosenSource
	}

	// Simple confidence heuristic: start at 80, add 1 per field that has a value.
	confidence := 80
	populated := 0
	for _, r := range results {
		if r.ChosenValue != "" {
			populated++
		}
	}
	if populated > 15 {
		confidence = 95
	} else if populated > 8 {
		confidence = 87
	}

	gold := &SecurityMasterRecord{
		ID:                       id,
		TenantID:                 tenantID,
		SecurityID:               firstNonEmpty(vals["security_id"], raw[0].SecurityID),
		PrimaryIdentifier:        firstNonEmpty(vals["isin"], vals["figi"], vals["cusip"]),
		ISIN:                     vals["isin"],
		CUSIP:                    vals["cusip"],
		SEDOL:                    vals["sedol"],
		FIGI:                     vals["figi"],
		Ticker:                   vals["ticker"],
		LocalTicker:              vals["local_ticker"],
		RIC:                      vals["ric"],
		BBGID:                    vals["bbg_id"],
		SecurityName:             vals["security_name"],
		ShortName:                vals["short_name"],
		AssetClass:               vals["asset_class"],
		SubAssetClass:            vals["sub_asset_class"],
		InstrumentType:           vals["instrument_type"],
		Sector:                   vals["sector"],
		Industry:                 vals["industry"],
		Currency:                 vals["currency"],
		SettlementCurrency:       vals["settlement_currency"],
		CountryOfIssue:           vals["country_of_issue"],
		CountryOfRisk:            vals["country_of_risk"],
		Region:                   vals["region"],
		ListingExchange:          vals["listing_exchange"],
		ExchangeCode:             vals["exchange_code"],
		Status:                   firstNonEmpty(vals["status"], "Active"),
		LiquidityProfile:         vals["liquidity_profile"],
		RegulatoryClassification: vals["regulatory_classification"],
		ConfidenceScore:          confidence,
		SourceSystems:            sourceSystems,
		ValidFrom:                time.Now(),
	}

	// Populate subtype attributes.
	switch gold.AssetClass {
	case "FixedIncome":
		gold.FixedIncome = &FixedIncomeAttributes{
			SecurityID:         id,
			TenantID:           tenantID,
			CouponType:         firstNonEmpty(vals["coupon_type"], "Fixed"),
			CouponFrequency:    vals["coupon_frequency"],
			DayCountConvention: vals["day_count_convention"],
			Seniority:          vals["seniority"],
			RatingComposite:    vals["rating_composite"],
		}
		if v := vals["coupon_rate"]; v != "" {
			f := parseFloat(v)
			gold.FixedIncome.CouponRate = &f
		}
		if v := vals["par_value"]; v != "" {
			f := parseFloat(v)
			gold.FixedIncome.ParValue = &f
		}
	case "Equity":
		gold.Equity = &EquityAttributes{
			SecurityID:        id,
			TenantID:          tenantID,
			ShareClass:        vals["share_class"],
			DividendFrequency: vals["dividend_frequency"],
		}
		if v := vals["shares_outstanding"]; v != "" {
			f := parseFloat(v)
			gold.Equity.SharesOutstanding = &f
		}
		if v := vals["dividend_yield"]; v != "" {
			f := parseFloat(v)
			gold.Equity.DividendYield = &f
		}
	case "Fund":
		gold.Fund = &FundAttributes{
			SecurityID:         id,
			TenantID:           tenantID,
			FundType:           firstNonEmpty(vals["fund_type"], "ETF"),
			ManagementCompany:  vals["management_company"],
			DistributionPolicy: vals["distribution_policy"],
		}
		if v := vals["total_expense_ratio"]; v != "" {
			f := parseFloat(v)
			gold.Fund.TotalExpenseRatio = &f
		}
	case "Derivative":
		gold.Derivative = &DerivativeAttributes{
			SecurityID:     id,
			TenantID:       tenantID,
			OptionType:     vals["option_type"],
			ExerciseStyle:  vals["exercise_style"],
			SettlementType: vals["settlement_type"],
		}
		if v := vals["strike_price"]; v != "" {
			f := parseFloat(v)
			gold.Derivative.StrikePrice = &f
		}
		if v := vals["underlier_security_id"]; v != "" {
			uid, err := uuid.Parse(v)
			if err == nil {
				gold.Derivative.UnderlierSecurityID = &uid
			}
		}
	}

	return gold
}

// survResultsToMap converts []SurvivorshipResult into a simple field→value map.
func survResultsToMap(results []SurvivorshipResult) map[string]string {
	m := make(map[string]string, len(results))
	for _, r := range results {
		m[r.FieldName] = r.ChosenValue
	}
	return m
}

// firstNonEmpty returns the first non-empty string.
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// parseFloat safely converts a string to float64, returning 0 on error.
func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

// runSecurityCoreDQ validates the gold record against hard DQ rules.
func runSecurityCoreDQ(gold *SecurityMasterRecord) []DQViolation {
	var violations []DQViolation
	if gold.SecurityName == "" {
		violations = append(violations, DQViolation{RuleName: "Security_RequiredIdentifiers", Field: "security_name", Severity: "Hard", Message: "security_name is required"})
	}
	if gold.ISIN == "" && gold.FIGI == "" && gold.CUSIP == "" {
		violations = append(violations, DQViolation{RuleName: "Security_RequiredIdentifiers", Field: "isin/figi/cusip", Severity: "Hard", Message: "at least one of ISIN, FIGI, or CUSIP is required"})
	}
	if gold.AssetClass == "" {
		violations = append(violations, DQViolation{RuleName: "Security_AssetClassValidity", Field: "asset_class", Severity: "Hard", Message: "asset_class is required"})
	}
	if gold.Currency == "" {
		violations = append(violations, DQViolation{RuleName: "Security_CurrencyRequired", Field: "currency", Severity: "Hard", Message: "currency is required"})
	}
	return violations
}

// hasFatal returns true if any DQ violation is severity "Hard".
func hasFatal(violations []DQViolation) bool {
	for _, v := range violations {
		if v.Severity == "Hard" {
			return true
		}
	}
	return false
}
