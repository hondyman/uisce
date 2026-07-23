package wealth

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// TaxCalculationService handles estate and gift tax calculations
type TaxCalculationService struct {
	db *pgxpool.Pool
}

// NewTaxCalculationService creates a new tax calculation service
func NewTaxCalculationService(db *pgxpool.Pool) *TaxCalculationService {
	return &TaxCalculationService{
		db: db,
	}
}

// TaxResult represents the result of a tax calculation
type TaxResult struct {
	TaxableAmount      decimal.Decimal `json:"taxable_amount"`
	TotalTax           decimal.Decimal `json:"total_tax"`
	EffectiveTaxRate   decimal.Decimal `json:"effective_tax_rate"`
	MarginalTaxRate    decimal.Decimal `json:"marginal_tax_rate"`
	ExemptionUsed      decimal.Decimal `json:"exemption_used"`
	ExemptionRemaining decimal.Decimal `json:"exemption_remaining"`
	ByBracket          []BracketTax    `json:"by_bracket"`
	Jurisdiction       string          `json:"jurisdiction"`
}

// BracketTax represents tax calculated in a specific bracket
type BracketTax struct {
	BracketMin       decimal.Decimal `json:"bracket_min"`
	BracketMax       decimal.Decimal `json:"bracket_max"`
	Rate             decimal.Decimal `json:"rate"`
	TaxableInBracket decimal.Decimal `json:"taxable_in_bracket"`
	TaxInBracket     decimal.Decimal `json:"tax_in_bracket"`
}

// CalculateFederalEstateTax calculates federal estate tax
func (s *TaxCalculationService) CalculateFederalEstateTax(
	ctx context.Context,
	grossEstate decimal.Decimal,
	priorExemptionUsed decimal.Decimal,
	year *int,
) (*TaxResult, error) {
	// Get current federal tax jurisdiction
	juris, err := s.getTaxJurisdiction(ctx, "US-FEDERAL", year)
	if err != nil {
		return nil, fmt.Errorf("failed to get federal jurisdiction: %w", err)
	}

	// Calculate taxable estate (after exemption)
	exemptionAvailable := juris.EstateTaxExemption.Sub(priorExemptionUsed)
	taxableAmount := grossEstate.Sub(exemptionAvailable)
	if taxableAmount.LessThan(decimal.Zero) {
		taxableAmount = decimal.Zero
	}

	// Calculate tax using progressive brackets
	totalTax, byBracket := s.calculateProgressiveTax(taxableAmount, juris.EstateTaxRateSchedule)

	// Calculate effective rate
	effectiveRate := decimal.Zero
	if grossEstate.GreaterThan(decimal.Zero) {
		effectiveRate = totalTax.Div(grossEstate).Mul(decimal.NewFromInt(100))
	}

	// Marginal rate (highest bracket reached)
	marginalRate := decimal.Zero
	if len(byBracket) > 0 {
		marginalRate = byBracket[len(byBracket)-1].Rate.Mul(decimal.NewFromInt(100))
	}

	exemptionUsed := grossEstate
	if exemptionUsed.GreaterThan(*juris.EstateTaxExemption) {
		exemptionUsed = *juris.EstateTaxExemption
	}

	return &TaxResult{
		TaxableAmount:      taxableAmount,
		TotalTax:           totalTax,
		EffectiveTaxRate:   effectiveRate,
		MarginalTaxRate:    marginalRate,
		ExemptionUsed:      exemptionUsed,
		ExemptionRemaining: exemptionAvailable.Sub(exemptionUsed),
		ByBracket:          byBracket,
		Jurisdiction:       "US-FEDERAL",
	}, nil
}

// CalculateStateTax calculates state estate tax
func (s *TaxCalculationService) CalculateStateTax(
	ctx context.Context,
	stateCode string,
	grossEstate decimal.Decimal,
	year *int,
) (*TaxResult, error) {
	jurisdictionCode := fmt.Sprintf("US-%s", stateCode)

	// Get state tax jurisdiction
	juris, err := s.getTaxJurisdiction(ctx, jurisdictionCode, year)
	if err != nil {
		return nil, fmt.Errorf("failed to get state jurisdiction: %w", err)
	}

	// Check if state has estate tax
	if !juris.EstateTaxApplies || juris.EstateTaxExemption == nil {
		return &TaxResult{
			TaxableAmount:    decimal.Zero,
			TotalTax:         decimal.Zero,
			EffectiveTaxRate: decimal.Zero,
			Jurisdiction:     jurisdictionCode,
		}, nil
	}

	// Calculate taxable amount
	taxableAmount := grossEstate.Sub(*juris.EstateTaxExemption)
	if taxableAmount.LessThan(decimal.Zero) {
		taxableAmount = decimal.Zero
	}

	// Calculate tax
	totalTax, byBracket := s.calculateProgressiveTax(taxableAmount, juris.EstateTaxRateSchedule)

	// Calculate effective rate
	effectiveRate := decimal.Zero
	if grossEstate.GreaterThan(decimal.Zero) {
		effectiveRate = totalTax.Div(grossEstate).Mul(decimal.NewFromInt(100))
	}

	// Marginal rate
	marginalRate := decimal.Zero
	if len(byBracket) > 0 {
		marginalRate = byBracket[len(byBracket)-1].Rate.Mul(decimal.NewFromInt(100))
	}

	exemptionUsed := grossEstate
	if exemptionUsed.GreaterThan(*juris.EstateTaxExemption) {
		exemptionUsed = *juris.EstateTaxExemption
	}

	return &TaxResult{
		TaxableAmount:      taxableAmount,
		TotalTax:           totalTax,
		EffectiveTaxRate:   effectiveRate,
		MarginalTaxRate:    marginalRate,
		ExemptionUsed:      exemptionUsed,
		ExemptionRemaining: juris.EstateTaxExemption.Sub(exemptionUsed),
		ByBracket:          byBracket,
		Jurisdiction:       jurisdictionCode,
	}, nil
}

// CalculateCombinedEstateTax calculates federal + state estate tax
func (s *TaxCalculationService) CalculateCombinedEstateTax(
	ctx context.Context,
	stateCode string,
	grossEstate decimal.Decimal,
	priorFederalExemptionUsed decimal.Decimal,
	year *int,
) (*CombinedTaxResult, error) {
	// Calculate federal tax
	federalTax, err := s.CalculateFederalEstateTax(ctx, grossEstate, priorFederalExemptionUsed, year)
	if err != nil {
		return nil, err
	}

	// Calculate state tax
	stateTax, err := s.CalculateStateTax(ctx, stateCode, grossEstate, year)
	if err != nil {
		return nil, err
	}

	totalTax := federalTax.TotalTax.Add(stateTax.TotalTax)
	effectiveRate := decimal.Zero
	if grossEstate.GreaterThan(decimal.Zero) {
		effectiveRate = totalTax.Div(grossEstate).Mul(decimal.NewFromInt(100))
	}

	return &CombinedTaxResult{
		FederalTax:       federalTax,
		StateTax:         stateTax,
		TotalTax:         totalTax,
		EffectiveTaxRate: effectiveRate,
	}, nil
}

// CombinedTaxResult represents federal + state tax
type CombinedTaxResult struct {
	FederalTax       *TaxResult      `json:"federal_tax"`
	StateTax         *TaxResult      `json:"state_tax"`
	TotalTax         decimal.Decimal `json:"total_tax"`
	EffectiveTaxRate decimal.Decimal `json:"effective_tax_rate"`
}

// CalculateGiftTax calculates gift tax for a specific gift
func (s *TaxCalculationService) CalculateGiftTax(
	ctx context.Context,
	giftValue decimal.Decimal,
	annualExclusionUsedThisYear decimal.Decimal,
	lifetimeExemptionUsedPrior decimal.Decimal,
	spousalSplit bool,
	year *int,
) (*GiftTaxResult, error) {
	// Get federal jurisdiction for current year
	juris, err := s.getTaxJurisdiction(ctx, "US-FEDERAL", year)
	if err != nil {
		return nil, err
	}

	// Annual exclusion
	annualExclusion := *juris.AnnualGiftExclusion
	if spousalSplit {
		annualExclusion = annualExclusion.Mul(decimal.NewFromInt(2))
	}

	// Determine what portion uses annual exclusion
	annualExclusionAvailable := annualExclusion.Sub(annualExclusionUsedThisYear)
	annualExclusionApplied := giftValue
	if annualExclusionApplied.GreaterThan(annualExclusionAvailable) {
		annualExclusionApplied = annualExclusionAvailable
	}

	// Remaining gift value after annual exclusion
	taxableGift := giftValue.Sub(annualExclusionApplied)

	// Apply lifetime exemption
	lifetimeExemptionAvailable := juris.LifetimeGiftExemption.Sub(lifetimeExemptionUsedPrior)
	lifetimeExemptionApplied := taxableGift
	if lifetimeExemptionApplied.GreaterThan(lifetimeExemptionAvailable) {
		lifetimeExemptionApplied = lifetimeExemptionAvailable
	}

	// Calculate tax on amount exceeding all exemptions
	amountSubjectToTax := taxableGift.Sub(lifetimeExemptionApplied)
	if amountSubjectToTax.LessThan(decimal.Zero) {
		amountSubjectToTax = decimal.Zero
	}

	giftTax, _ := s.calculateProgressiveTax(amountSubjectToTax, juris.EstateTaxRateSchedule)

	// Determine if Form 709 required
	requiresForm709 := annualExclusionApplied.LessThan(giftValue) || spousalSplit

	return &GiftTaxResult{
		GiftValue:                 giftValue,
		AnnualExclusionUtilized:   annualExclusionApplied,
		LifetimeExemptionUtilized: lifetimeExemptionApplied,
		TaxableGift:               amountSubjectToTax,
		GiftTaxOwed:               giftTax,
		RequiresForm709:           requiresForm709,
		CalculationMethod:         "direct",
	}, nil
}

// CalculateGSTTax calculates generation-skipping transfer tax
func (s *TaxCalculationService) CalculateGSTTax(
	ctx context.Context,
	transferValue decimal.Decimal,
	gstExemptionUsedPrior decimal.Decimal,
	generationsSkipped int,
	year *int,
) (*GSTTaxResult, error) {
	if generationsSkipped < 2 {
		return &GSTTaxResult{
			TransferValue: transferValue,
			GSTTax:        decimal.Zero,
			IsGSTTransfer: false,
		}, nil
	}

	juris, err := s.getTaxJurisdiction(ctx, "US-FEDERAL", year)
	if err != nil {
		return nil, err
	}

	// Apply GST exemption
	gstExemptionAvailable := juris.GSTTaxExemption.Sub(gstExemptionUsedPrior)
	gstExemptionApplied := transferValue
	if gstExemptionApplied.GreaterThan(gstExemptionAvailable) {
		gstExemptionApplied = gstExemptionAvailable
	}

	// Tax on amount exceeding exemption
	taxableAmount := transferValue.Sub(gstExemptionApplied)
	if taxableAmount.LessThan(decimal.Zero) {
		taxableAmount = decimal.Zero
	}

	gstTax := taxableAmount.Mul(*juris.GSTTaxRate)

	return &GSTTaxResult{
		TransferValue:         transferValue,
		GSTExemptionApplied:   gstExemptionApplied,
		TaxableAmount:         taxableAmount,
		GSTTax:                gstTax,
		IsGSTTransfer:         true,
		GenerationsSkipped:    generationsSkipped,
		GSTExemptionRemaining: gstExemptionAvailable.Sub(gstExemptionApplied),
	}, nil
}

// GSTTaxResult represents GST tax calculation
type GSTTaxResult struct {
	TransferValue         decimal.Decimal `json:"transfer_value"`
	GSTExemptionApplied   decimal.Decimal `json:"gst_exemption_applied"`
	TaxableAmount         decimal.Decimal `json:"taxable_amount"`
	GSTTax                decimal.Decimal `json:"gst_tax"`
	IsGSTTransfer         bool            `json:"is_gst_transfer"`
	GenerationsSkipped    int             `json:"generations_skipped"`
	GSTExemptionRemaining decimal.Decimal `json:"gst_exemption_remaining"`
}

// ProjectTaxLawChanges projects tax changes for estate planning scenarios
func (s *TaxCalculationService) ProjectTaxLawChanges(
	ctx context.Context,
	years int,
) ([]TaxProjection, error) {
	// Query known tax law changes
	query := `
		SELECT jurisdiction_code, change_title, effective_date, 
			estimated_families_affected, average_tax_impact
		FROM tax_law_changes
		WHERE effective_date <= NOW() + INTERVAL '1 year' * $1
			AND status = 'ENACTED'
		ORDER BY effective_date
	`

	rows, err := s.db.Query(ctx, query, years)
	if err != nil {
		return nil, fmt.Errorf("failed to query tax law changes: %w", err)
	}
	defer rows.Close()

	projections := []TaxProjection{}
	for rows.Next() {
		var p TaxProjection
		err := rows.Scan(
			&p.JurisdictionCode,
			&p.ChangeDescription,
			&p.EffectiveDate,
			&p.EstimatedFamiliesAffected,
			&p.AverageTaxImpact,
		)
		if err != nil {
			return nil, err
		}
		projections = append(projections, p)
	}

	return projections, nil
}

// TaxProjection represents a projected tax law change
type TaxProjection struct {
	JurisdictionCode          string          `json:"jurisdiction_code"`
	ChangeDescription         string          `json:"change_description"`
	EffectiveDate             string          `json:"effective_date"`
	EstimatedFamiliesAffected int             `json:"estimated_families_affected"`
	AverageTaxImpact          decimal.Decimal `json:"average_tax_impact"`
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// getTaxJurisdiction retrieves tax jurisdiction data for calculations
func (s *TaxCalculationService) getTaxJurisdiction(
	ctx context.Context,
	jurisdictionCode string,
	year *int,
) (*TaxJurisdiction, error) {
	query := `
		SELECT jurisdiction_id, jurisdiction_code, jurisdiction_name, jurisdiction_type,
			estate_tax_applies, estate_tax_exemption, estate_tax_rate_schedule,
			gift_tax_applies, annual_gift_exclusion, lifetime_gift_exemption,
			gst_tax_applies, gst_tax_exemption, gst_tax_rate,
			effective_date, expiration_date
		FROM tax_jurisdictions
		WHERE jurisdiction_code = $1
			AND effective_date <= COALESCE(make_date($2, 1, 1), CURRENT_DATE)
			AND (expiration_date IS NULL OR expiration_date > COALESCE(make_date($2, 1, 1), CURRENT_DATE))
		ORDER BY effective_date DESC
		LIMIT 1
	`

	var yearVal interface{}
	if year != nil {
		yearVal = *year
	}

	var juris TaxJurisdiction
	var scheduleJSON []byte

	err := s.db.QueryRow(ctx, query, jurisdictionCode, yearVal).Scan(
		&juris.JurisdictionID,
		&juris.JurisdictionCode,
		&juris.JurisdictionName,
		&juris.JurisdictionType,
		&juris.EstateTaxApplies,
		&juris.EstateTaxExemption,
		&scheduleJSON,
		&juris.GiftTaxApplies,
		&juris.AnnualGiftExclusion,
		&juris.LifetimeGiftExemption,
		&juris.GSTTaxApplies,
		&juris.GSTTaxExemption,
		&juris.GSTTaxRate,
		&juris.EffectiveDate,
		&juris.ExpirationDate,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get tax jurisdiction %s: %w", jurisdictionCode, err)
	}

	// Parse rate schedule from JSONB
	if err := s.parseRateSchedule(scheduleJSON, &juris); err != nil {
		return nil, err
	}

	return &juris, nil
}

// parseRateSchedule parses JSONB tax rate schedule into TaxBracket array
func (s *TaxCalculationService) parseRateSchedule(scheduleJSON []byte, juris *TaxJurisdiction) error {
	var raw []map[string]interface{}
	if err := json.Unmarshal(scheduleJSON, &raw); err != nil {
		return fmt.Errorf("failed to parse rate schedule: %w", err)
	}

	juris.EstateTaxRateSchedule = make([]TaxBracket, len(raw))
	for i, bracket := range raw {
		threshold, _ := decimal.NewFromString(fmt.Sprintf("%v", bracket["threshold"]))
		rate, _ := decimal.NewFromString(fmt.Sprintf("%v", bracket["rate"]))

		juris.EstateTaxRateSchedule[i] = TaxBracket{
			Threshold: threshold,
			Rate:      rate,
		}
	}

	return nil
}

// calculateProgressiveTax calculates tax using progressive brackets
func (s *TaxCalculationService) calculateProgressiveTax(
	taxableAmount decimal.Decimal,
	schedule []TaxBracket,
) (decimal.Decimal, []BracketTax) {
	if taxableAmount.LessThanOrEqual(decimal.Zero) || len(schedule) == 0 {
		return decimal.Zero, []BracketTax{}
	}

	totalTax := decimal.Zero
	byBracket := []BracketTax{}
	remaining := taxableAmount

	for i := 0; i < len(schedule); i++ {
		bracket := schedule[i]

		// Determine bracket max (next bracket's threshold or infinity)
		bracketMax := decimal.Zero
		if i < len(schedule)-1 {
			bracketMax = schedule[i+1].Threshold
		} else {
			// Last bracket extends to infinity
			bracketMax = taxableAmount.Add(decimal.NewFromInt(1))
		}

		// Amount taxable in this bracket
		bracketMin := bracket.Threshold
		if remaining.LessThanOrEqual(decimal.Zero) {
			break
		}

		taxableInBracket := remaining
		if taxableAmount.GreaterThan(bracketMax) {
			taxableInBracket = bracketMax.Sub(bracketMin)
		}

		taxInBracket := taxableInBracket.Mul(bracket.Rate)
		totalTax = totalTax.Add(taxInBracket)
		remaining = remaining.Sub(taxableInBracket)

		byBracket = append(byBracket, BracketTax{
			BracketMin:       bracketMin,
			BracketMax:       bracketMax,
			Rate:             bracket.Rate,
			TaxableInBracket: taxableInBracket,
			TaxInBracket:     taxInBracket,
		})

		if remaining.LessThanOrEqual(decimal.Zero) {
			break
		}
	}

	return totalTax, byBracket
}
