package goldcopy

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Engine orchestrates the full gold copy build process for portfolio_master.
type Engine struct {
	repo *Repository
}

func NewEngine(repo *Repository) *Engine {
	return &Engine{repo: repo}
}

// BuildPortfolioGoldCopy runs the full MDM pipeline for one portfolio cluster:
//  1. Validate raw records (DQ, hard-fail if no records)
//  2. Load survivorship rules from edm.survivorship_rules
//  3. Apply survivorship to each mastered field
//  4. Compute overall confidence score
//  5. Upsert gold copy into edm.portfolio_master
//  6. Write lineage traces to edm.gold_copy_lineage
func (e *Engine) BuildPortfolioGoldCopy(
	ctx context.Context,
	tenantID uuid.UUID,
	rawRecords []*RawPortfolioRecord,
) (*GoldCopyRunResult, error) {
	if len(rawRecords) == 0 {
		return nil, fmt.Errorf("goldcopy: no raw records provided")
	}

	runID := uuid.New()
	startedAt := time.Now()
	portfolioID := rawRecords[0].PortfolioID

	result := &GoldCopyRunResult{
		RunID:       runID,
		TenantID:    tenantID,
		EntityType:  "portfolio_master",
		PortfolioID: portfolioID,
		StartedAt:   startedAt,
	}

	// ── 1. Basic DQ validation
	violations := e.validateRaw(rawRecords)
	result.DQViolations = violations

	// Hard failures block gold copy creation
	for _, v := range violations {
		if v.Severity == "Hard" {
			result.Success = false
			result.ErrorMessage = fmt.Sprintf("hard DQ violation on field %q: %s", v.Field, v.Message)
			result.CompletedAt = time.Now()
			return result, nil
		}
	}

	// ── 2. Load survivorship rules
	rules, err := e.repo.ListSurvivorshipRules(ctx, tenantID, "portfolio_master")
	if err != nil {
		return nil, fmt.Errorf("goldcopy: load survivorship rules: %w", err)
	}
	survivorship := NewSurvivorship(rules)

	// ── 3. Apply survivorship to all mastered fields
	fieldsToMaster := []string{
		"portfolio_name", "portfolio_code", "portfolio_type", "portfolio_category",
		"base_currency", "inception_date", "domicile", "legal_structure",
		"regulatory_classification", "liquidity_profile", "risk_profile",
		"investment_objective", "investment_guidelines",
		"valuation_frequency", "pricing_source",
		"is_model_portfolio", "is_composite_member",
		"benchmark_id", "strategy_id", "mandate_id",
		"portfolio_manager_id", "custodian_id",
	}

	sourceSystems := make(map[string]string)
	var survivorshipLog []SurvivorshipResult
	masteredFields := make(map[string]string)

	for _, field := range fieldsToMaster {
		sr := survivorship.Resolve(field, rawRecords)
		sourceSystems[field] = sr.ChosenSource
		masteredFields[field] = sr.ChosenValue
		survivorshipLog = append(survivorshipLog, sr)
	}

	// ── 4. Compute confidence score
	//    Base = average quality score of sources, adjusted by number of sources
	//    and any DQ soft failures.
	confidence := e.computeConfidence(rawRecords, violations)

	// ── 5. Assemble golden record
	golden := e.assembleMasterRecord(tenantID, runID, portfolioID, masteredFields, sourceSystems, confidence)
	result.GoldenRecord = golden
	result.SurvivorshipLog = survivorshipLog
	result.ConfidenceScore = confidence

	// ── 6. Persist
	if err := e.repo.UpsertPortfolioMaster(ctx, golden); err != nil {
		return nil, fmt.Errorf("goldcopy: upsert gold copy: %w", err)
	}

	// ── 7. Build Performance Settings (if provided)
	perfFields := []string{
		"valuation_method", "fee_treatment", "cash_flow_method",
		"currency_hedging_policy", "lookthrough_policy", "treatment_of_derivatives",
	}

	hasPerf := false
	for _, f := range perfFields {
		if anyHasField(rawRecords, f) {
			hasPerf = true
			break
		}
	}

	if hasPerf {
		masteredPerf := make(map[string]string)
		sourceSystemsPerf := make(map[string]string)
		var psSurvivorshipLog []SurvivorshipResult

		for _, field := range perfFields {
			sr := survivorship.Resolve(field, rawRecords) // Note: needs rules for these fields
			sourceSystemsPerf[field] = sr.ChosenSource
			masteredPerf[field] = sr.ChosenValue
			psSurvivorshipLog = append(psSurvivorshipLog, sr)
		}

		perfRec := e.assemblePerformanceRecord(tenantID, portfolioID, masteredPerf, sourceSystemsPerf)
		if err := e.repo.UpsertPerformanceSettings(ctx, perfRec); err != nil {
			return nil, fmt.Errorf("goldcopy: upsert perf settings: %w", err)
		}

		// Link it back to the golden record and re-persist the golden record with the link
		golden.PerformanceSettingsID = &perfRec.ID
		if err := e.repo.UpsertPortfolioMaster(ctx, golden); err != nil {
			return nil, fmt.Errorf("goldcopy: update portfolio master with link: %w", err)
		}

		// Write lineage for performance settings
		psLineage := buildLineageEntries(tenantID, perfRec.ID, runID, psSurvivorshipLog, violations)
		for i := range psLineage {
			psLineage[i].EntityType = "performance_settings"
		}
		if err := e.repo.InsertLineageEntries(ctx, psLineage); err != nil {
			_ = err
		}
	}

	// ── 8. Write lineage for portfolio master
	entries := buildLineageEntries(tenantID, golden.ID, runID, survivorshipLog, violations)
	if err := e.repo.InsertLineageEntries(ctx, entries); err != nil {
		// Non-fatal: log but don't fail the run
		_ = err
	}

	result.Success = true
	result.CompletedAt = time.Now()
	return result, nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// validateRaw runs lightweight in-process DQ rules on the raw input.
// Full SQL-expression rules (from catalog_node) are run by the rules engine;
// this handles the critical guards that must pass before survivorship.
func (e *Engine) validateRaw(recs []*RawPortfolioRecord) []DQViolation {
	var violations []DQViolation
	portfolioID := recs[0].PortfolioID

	// Check that at least one record provides a portfolio_name
	if !anyHasField(recs, "portfolio_name") {
		violations = append(violations, DQViolation{
			RuleName: "Portfolio_RequiredPortfolioName",
			Field:    "portfolio_name",
			Severity: "Hard",
			Message:  fmt.Sprintf("portfolio %q: no source provides portfolio_name", portfolioID),
		})
	}

	// Check that at least one record provides a base_currency with exactly 3 chars
	for _, r := range recs {
		if v := r.Fields["base_currency"]; v != "" && len(v) != 3 {
			violations = append(violations, DQViolation{
				RuleName: "Portfolio_ValidBaseCurrency",
				Field:    "base_currency",
				Severity: "Hard",
				Message:  fmt.Sprintf("source %q provided invalid currency code %q", r.SourceSystem, v),
			})
		}
	}

	// Inception date must not be blank across all sources (Soft)
	if !anyHasField(recs, "inception_date") {
		violations = append(violations, DQViolation{
			RuleName: "Portfolio_RequiredInceptionDate",
			Field:    "inception_date",
			Severity: "Soft",
			Message:  fmt.Sprintf("portfolio %q: no source provides inception_date", portfolioID),
		})
	}

	// Inception date must not be in the future
	for _, r := range recs {
		if v := r.Fields["inception_date"]; v != "" {
			t, err := time.Parse("2006-01-02", v)
			if err == nil && t.After(time.Now()) {
				violations = append(violations, DQViolation{
					RuleName: "Portfolio_InceptionDateNotFuture",
					Field:    "inception_date",
					Severity: "Hard",
					Message:  fmt.Sprintf("source %q provided future inception_date %s", r.SourceSystem, v),
				})
			}
		}
	}

	return violations
}

func anyHasField(recs []*RawPortfolioRecord, field string) bool {
	for _, r := range recs {
		if r.Fields[field] != "" {
			return true
		}
	}
	return false
}

func (e *Engine) computeConfidence(recs []*RawPortfolioRecord, violations []DQViolation) int {
	if len(recs) == 0 {
		return 0
	}
	var sum int
	for _, r := range recs {
		sum += r.QualityScore
	}
	base := sum / len(recs)

	// Bonus for multiple corroborating sources
	if len(recs) >= 3 {
		base += 5
	} else if len(recs) == 2 {
		base += 2
	}

	// Penalty for soft violations
	for _, v := range violations {
		if v.Severity == "Soft" {
			base -= 3
		} else if v.Severity == "Warning" {
			base -= 1
		}
	}
	if base < 0 {
		base = 0
	}
	if base > 100 {
		base = 100
	}
	return base
}

func (e *Engine) assembleMasterRecord(
	tenantID uuid.UUID,
	_ uuid.UUID, // runID not stored on the record itself
	portfolioID string,
	fields map[string]string,
	sourceSystems map[string]string,
	confidence int,
) *PortfolioMasterRecord {
	rec := &PortfolioMasterRecord{
		ID:              uuid.New(),
		TenantID:        tenantID,
		PortfolioID:     portfolioID,
		SourceSystems:   sourceSystems,
		ConfidenceScore: confidence,
	}

	rec.PortfolioName = fields["portfolio_name"]
	rec.PortfolioCode = fields["portfolio_code"]
	rec.PortfolioType = coalesce(fields["portfolio_type"], "Fund")
	rec.PortfolioCategory = fields["portfolio_category"]
	rec.BaseCurrency = coalesce(fields["base_currency"], "USD")
	rec.Domicile = fields["domicile"]
	rec.LegalStructure = fields["legal_structure"]
	rec.RegulatoryClassification = fields["regulatory_classification"]
	rec.LiquidityProfile = fields["liquidity_profile"]
	rec.RiskProfile = fields["risk_profile"]
	rec.InvestmentObjective = fields["investment_objective"]
	rec.InvestmentGuidelines = fields["investment_guidelines"]
	rec.ValuationFrequency = fields["valuation_frequency"]
	rec.PricingSource = fields["pricing_source"]
	rec.IsModelPortfolio = strings.EqualFold(fields["is_model_portfolio"], "true")
	rec.IsCompositeMember = strings.EqualFold(fields["is_composite_member"], "true")
	rec.PortfolioManagerID = fields["portfolio_manager_id"]
	rec.CustodianID = fields["custodian_id"]

	if v := fields["inception_date"]; v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			rec.InceptionDate = t
		}
	}

	return rec
}

func (e *Engine) assemblePerformanceRecord(
	tenantID uuid.UUID,
	portfolioID string,
	fields map[string]string,
	sourceSystems map[string]string,
) *PerformanceSettingsRecord {
	rec := &PerformanceSettingsRecord{
		ID:                uuid.New(),
		TenantID:          tenantID,
		PortfolioID:       portfolioID,
		ValuationMethod:   fields["valuation_method"],
		FeeTreatment:      fields["fee_treatment"],
		CashFlowMethod:    fields["cash_flow_method"],
		CurrencyHedging:   fields["currency_hedging_policy"],
		LookthroughPolicy: fields["lookthrough_policy"],
		DerivativesPolicy: fields["treatment_of_derivatives"],
		ConfidenceScore:   90, // Default for performance settings
		SourceSystems:     sourceSystems,
	}
	return rec
}

func buildLineageEntries(
	tenantID uuid.UUID,
	entityID uuid.UUID,
	runID uuid.UUID,
	survivorshipLog []SurvivorshipResult,
	violations []DQViolation,
) []*GoldCopyLineageEntry {
	// Build violation index by field
	failedByField := make(map[string][]string)
	for _, v := range violations {
		failedByField[v.Field] = append(failedByField[v.Field], v.RuleName)
	}

	var entries []*GoldCopyLineageEntry
	for _, sr := range survivorshipLog {
		e := &GoldCopyLineageEntry{
			ID:               uuid.New(),
			TenantID:         tenantID,
			EntityType:       "portfolio_master",
			EntityID:         entityID,
			FieldName:        sr.FieldName,
			ChosenValue:      sr.ChosenValue,
			ChosenSource:     sr.ChosenSource,
			RejectedSources:  sr.RejectedSources,
			SurvivorshipRule: sr.Strategy,
			DQRulesPassed:    []string{},
			DQRulesFailed:    failedByField[sr.FieldName],
			RunID:            runID,
		}
		entries = append(entries, e)
	}
	return entries
}

func coalesce(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
