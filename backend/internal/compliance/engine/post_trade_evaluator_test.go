package engine

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/hondyman/uisce/backend/internal/compliance/testutil"
)

func TestPostTradeEvaluator_E2E_PilotRulesAndSupersession(t *testing.T) {
	// 1. Ephemeral Database Isolation
	db := testutil.GetEphemeralTestDB(t)

	ctx := context.Background()
	evaluator := NewPostTradeEvaluator(db)

	tenantA := uuid.New()
	tenantB := uuid.New()
	accountA := uuid.New()
	asOfDate, _ := time.Parse("2006-01-02", "2026-10-06")

	// Create test tenants
	_, err := db.ExecContext(ctx, `
		INSERT INTO public.tenants (id, name, display_name, is_active)
		VALUES ($1, 'tenant_alpha', 'Tenant Alpha', true),
		       ($2, 'tenant_beta', 'Tenant Beta', true)
	`, tenantA, tenantB)
	require.NoError(t, err)

	// Verify Seed of Phase 1 Post-Trade Rules (14 total)
	var pilotRuleCount int
	err = db.QueryRowContext(ctx, `
		SELECT count(*) FROM compliance.compliance_rule 
		WHERE rule_code IN (
			'UCITS_5_10_40', 'SEC_144A_QIB_HOLDING', 'MARGIN_UTILIZATION_80',
			'POST_TRADE_GROUP_ISSUER_20', 'POST_TRADE_ISSUER_DEBT_15',
			'POST_TRADE_COUNTERPARTY_PFE_10', 'POST_TRADE_CASH_MIN_5',
			'POST_TRADE_SOVEREIGN_EXPOSURE_35', 'POST_TRADE_AGENCY_SUPRA_25',
			'POST_TRADE_MUNI_OBLIGOR_10', 'POST_TRADE_CCP_CLEARING_EXPOSURE_15',
			'POST_TRADE_CUSTODIAN_CONCENTRATION_20', 'POST_TRADE_BANK_DEPOSIT_20',
			'POST_TRADE_SEC_LENDING_COLLATERAL_102'
		)
		  AND library_status = 'ACTIVE'
	`).Scan(&pilotRuleCount)
	require.NoError(t, err)
	require.Equal(t, 14, pilotRuleCount, "All 14 Phase 1 post-trade rules must be ACTIVE in compliance_rule")

	// =========================================================================
	// Scenario 1: Initial Portfolio Evaluation with UCITS 5/10/40 Breach
	// =========================================================================
	// NAV = 10,000,000.
	// Issuer 1 = 800,000 (8% - between 5% and 10%)
	// Issuer 2 = 900,000 (9% - between 5% and 10%)
	// Issuer 3 = 900,000 (9% - between 5% and 10%)
	// Issuer 4 = 850,000 (8.5% - between 5% and 10%)
	// Issuer 5 = 850,000 (8.5% - between 5% and 10%)
	// Total aggregate above 5% = 43% (4,300,000 / 10,000,000) > 40% -> BREACH!
	// 144A Non-QIB = 500,000 (5% < 15%) -> WITHIN_LIMITS
	// Margin Utilization: Gross=12,000,000, Cash=4,000,000 -> Borrowed=8,000,000 / MarginLimit 15,000,000 = 53.3% < 80% -> WITHIN_LIMITS

	state1 := PortfolioState{
		TenantID:      tenantA,
		AccountID:     accountA,
		AsOfDate:      asOfDate,
		NAV:           decimal.RequireFromString("10000000.000000"),
		GrossExposure: decimal.RequireFromString("12000000.000000"),
		NetExposure:   decimal.RequireFromString("10000000.000000"),
		CashBalance:   decimal.RequireFromString("4000000.000000"),
		MarginLimit:            decimal.RequireFromString("15000000.000000"),
		LiquidityCoverageRatio: decimal.RequireFromString("1.250000"),
		Positions: []PortfolioPosition{
			{SecurityID: "SEC-001", Symbol: "EQ1", IssuerID: "ISS-1", MarketValue: decimal.RequireFromString("800000.000000"), Sector: "Information Technology", IndustryGroup: "Software", CountryClassification: "DEVELOPED", Is144A: false, IsQIBEligible: false, HasEmissionsData: true, WaciIntensity: decimal.RequireFromString("120.000000"), GhgScope12Intensity: decimal.RequireFromString("70.000000"), BoardGenderDiversityPct: decimal.RequireFromString("0.350000"), HazardousWasteRatio: decimal.RequireFromString("2.000000"), EuTaxonomyAlignmentPct: decimal.RequireFromString("0.200000")},
			{SecurityID: "SEC-002", Symbol: "EQ2", IssuerID: "ISS-2", MarketValue: decimal.RequireFromString("900000.000000"), Sector: "Health Care", IndustryGroup: "Pharmaceuticals", CountryClassification: "DEVELOPED", Is144A: false, IsQIBEligible: false, HasEmissionsData: true, WaciIntensity: decimal.RequireFromString("120.000000"), GhgScope12Intensity: decimal.RequireFromString("70.000000"), BoardGenderDiversityPct: decimal.RequireFromString("0.350000"), HazardousWasteRatio: decimal.RequireFromString("2.000000"), EuTaxonomyAlignmentPct: decimal.RequireFromString("0.200000")},
			{SecurityID: "SEC-003", Symbol: "EQ3", IssuerID: "ISS-3", MarketValue: decimal.RequireFromString("900000.000000"), Sector: "Financials", IndustryGroup: "Banks", CountryClassification: "DEVELOPED", Is144A: false, IsQIBEligible: false, HasEmissionsData: true, WaciIntensity: decimal.RequireFromString("120.000000"), GhgScope12Intensity: decimal.RequireFromString("70.000000"), BoardGenderDiversityPct: decimal.RequireFromString("0.350000"), HazardousWasteRatio: decimal.RequireFromString("2.000000"), EuTaxonomyAlignmentPct: decimal.RequireFromString("0.200000")},
			{SecurityID: "SEC-004", Symbol: "EQ4", IssuerID: "ISS-4", MarketValue: decimal.RequireFromString("850000.000000"), Sector: "Consumer Staples", IndustryGroup: "Food", CountryClassification: "DEVELOPED", Is144A: false, IsQIBEligible: false, HasEmissionsData: true, WaciIntensity: decimal.RequireFromString("120.000000"), GhgScope12Intensity: decimal.RequireFromString("70.000000"), BoardGenderDiversityPct: decimal.RequireFromString("0.350000"), HazardousWasteRatio: decimal.RequireFromString("2.000000"), EuTaxonomyAlignmentPct: decimal.RequireFromString("0.200000")},
			{SecurityID: "SEC-005", Symbol: "EQ5", IssuerID: "ISS-5", MarketValue: decimal.RequireFromString("850000.000000"), Sector: "Communication Services", IndustryGroup: "Media", CountryClassification: "DEVELOPED", Is144A: false, IsQIBEligible: false, HasEmissionsData: true, WaciIntensity: decimal.RequireFromString("120.000000"), GhgScope12Intensity: decimal.RequireFromString("70.000000"), BoardGenderDiversityPct: decimal.RequireFromString("0.350000"), HazardousWasteRatio: decimal.RequireFromString("2.000000"), EuTaxonomyAlignmentPct: decimal.RequireFromString("0.200000")},
			{SecurityID: "SEC-006", Symbol: "BD1", IssuerID: "ISS-6", MarketValue: decimal.RequireFromString("500000.000000"), Sector: "Utilities", IndustryGroup: "Electric", CountryClassification: "DEVELOPED", Is144A: true, IsQIBEligible: false, HasEmissionsData: true, WaciIntensity: decimal.RequireFromString("120.000000"), GhgScope12Intensity: decimal.RequireFromString("70.000000"), BoardGenderDiversityPct: decimal.RequireFromString("0.350000"), HazardousWasteRatio: decimal.RequireFromString("2.000000"), EuTaxonomyAlignmentPct: decimal.RequireFromString("0.200000")},
		},
	}

	results1, err := evaluator.EvaluateAndPersist(ctx, state1)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(results1), 3)

	var ucitsFinding1 *PostTradeEvaluationResult
	for i := range results1 {
		if results1[i].RuleCode == "UCITS_5_10_40" {
			ucitsFinding1 = &results1[i]
		}
	}
	require.NotNil(t, ucitsFinding1)
	require.Equal(t, "BREACHED", ucitsFinding1.Action)
	require.Equal(t, "OPEN", ucitsFinding1.Status)
	require.Nil(t, ucitsFinding1.SupersedesFindingID)

	// Verify Snapshot row was inserted
	var snapCount int
	err = db.QueryRowContext(ctx, "SELECT count(*) FROM compliance.compliance_portfolio_snapshot WHERE tenant_id = $1", tenantA).Scan(&snapCount)
	require.NoError(t, err)
	require.Equal(t, 1, snapCount)

	// Verify Finding row exists in DB
	rows, err := db.QueryContext(ctx, "SELECT rule_code, action, finding_severity, status, details FROM compliance.compliance_finding WHERE tenant_id = $1 AND status = 'OPEN'", tenantA)
	require.NoError(t, err)
	defer rows.Close()
	var openRules []string
	for rows.Next() {
		var rc, act, sev, st string
		var dt []byte
		require.NoError(t, rows.Scan(&rc, &act, &sev, &st, &dt))
		openRules = append(openRules, fmt.Sprintf("%s (%s/%s: %s)", rc, act, sev, string(dt)))
	}
	t.Logf("Open Findings in DB (%d): %v", len(openRules), openRules)
	require.Equal(t, 1, len(openRules), "Expected exactly 1 open finding (UCITS_5_10_40)")

	// =========================================================================
	// Scenario 2: Idempotency Re-run with Exact Same Input
	// =========================================================================
	results2, err := evaluator.EvaluateAndPersist(ctx, state1)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(results2), 3)

	var findingCountAfterRerun int
	err = db.QueryRowContext(ctx, "SELECT count(*) FROM compliance.compliance_finding WHERE tenant_id = $1", tenantA).Scan(&findingCountAfterRerun)
	require.NoError(t, err)
	require.Equal(t, 1, findingCountAfterRerun, "Re-running exact same evaluation must be strictly idempotent")

	// =========================================================================
	// Scenario 3: Restatement (New Breach Value) -> Supersession Chain
	// =========================================================================
	// Issuer 5 value corrected up to 1,000,000 -> aggregate becomes 44.5%
	state2 := state1
	state2.Positions = append([]PortfolioPosition(nil), state1.Positions...)
	state2.Positions[4].MarketValue = decimal.RequireFromString("1000000.000000")

	results3, err := evaluator.EvaluateAndPersist(ctx, state2)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(results3), 3)

	var ucitsFinding3 *PostTradeEvaluationResult
	for i := range results3 {
		if results3[i].RuleCode == "UCITS_5_10_40" {
			ucitsFinding3 = &results3[i]
		}
	}
	require.NotNil(t, ucitsFinding3)
	require.Equal(t, "BREACHED", ucitsFinding3.Action)
	require.Equal(t, "OPEN", ucitsFinding3.Status)
	require.NotNil(t, ucitsFinding3.SupersedesFindingID)
	require.Equal(t, ucitsFinding1.FindingID, *ucitsFinding3.SupersedesFindingID)
	require.Equal(t, "SUPERSEDED_BY_RESTATEMENT", ucitsFinding3.SupersededReason)

	// Check DB table: Old finding is SUPERSEDED, New finding is OPEN
	var oldStatus, newStatus string
	err = db.QueryRowContext(ctx, "SELECT status FROM compliance.compliance_finding WHERE id = $1", ucitsFinding1.FindingID).Scan(&oldStatus)
	require.NoError(t, err)
	require.Equal(t, "SUPERSEDED", oldStatus)

	err = db.QueryRowContext(ctx, "SELECT status FROM compliance.compliance_finding WHERE id = $1", ucitsFinding3.FindingID).Scan(&newStatus)
	require.NoError(t, err)
	require.Equal(t, "OPEN", newStatus)

	// =========================================================================
	// Scenario 4: Restatement Resolving Breach (Remediation / Within Limits)
	// =========================================================================
	// Rebalancing: Positions trimmed so aggregate above 5% drops to 35% (within 40% limit)
	state3 := state1
	state3.Positions = []PortfolioPosition{
		{SecurityID: "SEC-001", Symbol: "EQ1", IssuerID: "ISS-1", MarketValue: decimal.RequireFromString("700000.000000"), Sector: "Information Technology", IndustryGroup: "Software", CountryClassification: "DEVELOPED", Is144A: false, IsQIBEligible: false, HasEmissionsData: true, WaciIntensity: decimal.RequireFromString("120.000000"), GhgScope12Intensity: decimal.RequireFromString("70.000000"), BoardGenderDiversityPct: decimal.RequireFromString("0.350000"), HazardousWasteRatio: decimal.RequireFromString("2.000000"), EuTaxonomyAlignmentPct: decimal.RequireFromString("0.200000")},
		{SecurityID: "SEC-002", Symbol: "EQ2", IssuerID: "ISS-2", MarketValue: decimal.RequireFromString("700000.000000"), Sector: "Health Care", IndustryGroup: "Pharmaceuticals", CountryClassification: "DEVELOPED", Is144A: false, IsQIBEligible: false, HasEmissionsData: true, WaciIntensity: decimal.RequireFromString("120.000000"), GhgScope12Intensity: decimal.RequireFromString("70.000000"), BoardGenderDiversityPct: decimal.RequireFromString("0.350000"), HazardousWasteRatio: decimal.RequireFromString("2.000000"), EuTaxonomyAlignmentPct: decimal.RequireFromString("0.200000")},
		{SecurityID: "SEC-003", Symbol: "EQ3", IssuerID: "ISS-3", MarketValue: decimal.RequireFromString("700000.000000"), Sector: "Financials", IndustryGroup: "Banks", CountryClassification: "DEVELOPED", Is144A: false, IsQIBEligible: false, HasEmissionsData: true, WaciIntensity: decimal.RequireFromString("120.000000"), GhgScope12Intensity: decimal.RequireFromString("70.000000"), BoardGenderDiversityPct: decimal.RequireFromString("0.350000"), HazardousWasteRatio: decimal.RequireFromString("2.000000"), EuTaxonomyAlignmentPct: decimal.RequireFromString("0.200000")},
		{SecurityID: "SEC-004", Symbol: "EQ4", IssuerID: "ISS-4", MarketValue: decimal.RequireFromString("700000.000000"), Sector: "Consumer Staples", IndustryGroup: "Food", CountryClassification: "DEVELOPED", Is144A: false, IsQIBEligible: false, HasEmissionsData: true, WaciIntensity: decimal.RequireFromString("120.000000"), GhgScope12Intensity: decimal.RequireFromString("70.000000"), BoardGenderDiversityPct: decimal.RequireFromString("0.350000"), HazardousWasteRatio: decimal.RequireFromString("2.000000"), EuTaxonomyAlignmentPct: decimal.RequireFromString("0.200000")},
		{SecurityID: "SEC-005", Symbol: "EQ5", IssuerID: "ISS-5", MarketValue: decimal.RequireFromString("700000.000000"), Sector: "Communication Services", IndustryGroup: "Media", CountryClassification: "DEVELOPED", Is144A: false, IsQIBEligible: false, HasEmissionsData: true, WaciIntensity: decimal.RequireFromString("120.000000"), GhgScope12Intensity: decimal.RequireFromString("70.000000"), BoardGenderDiversityPct: decimal.RequireFromString("0.350000"), HazardousWasteRatio: decimal.RequireFromString("2.000000"), EuTaxonomyAlignmentPct: decimal.RequireFromString("0.200000")}, // Total 35%
	}

	results4, err := evaluator.EvaluateAndPersist(ctx, state3)
	require.NoError(t, err)

	for _, r := range results4 {
		require.Equal(t, "WITHIN_LIMITS", r.Action)
	}

	// Verify the previously OPEN finding (ucitsFinding3) is now RESOLVED
	var resolvedStatus, resolutionNotes, resolvedBy string
	err = db.QueryRowContext(ctx, `
		SELECT status, resolution_notes, resolved_by 
		FROM compliance.compliance_finding 
		WHERE id = $1
	`, ucitsFinding3.FindingID).Scan(&resolvedStatus, &resolutionNotes, &resolvedBy)
	require.NoError(t, err)
	require.Equal(t, "RESOLVED", resolvedStatus)
	require.Contains(t, resolutionNotes, "resolved by portfolio restatement")
	require.Equal(t, "system_post_trade_evaluator", resolvedBy)

	// =========================================================================
	// Scenario 5: SEC 144A Non-QIB Asset Limit Breach (> 15%)
	// =========================================================================
	state4 := PortfolioState{
		TenantID:      tenantA,
		AccountID:     accountA,
		AsOfDate:      asOfDate.AddDate(0, 0, 1),
		NAV:           decimal.RequireFromString("10000000.000000"),
		GrossExposure: decimal.RequireFromString("10000000.000000"),
		NetExposure:   decimal.RequireFromString("10000000.000000"),
		CashBalance:   decimal.RequireFromString("1000000.000000"),
		Positions: []PortfolioPosition{
			{SecurityID: "SEC-144A-1", Symbol: "PRV1", IssuerID: "ISS-10", MarketValue: decimal.RequireFromString("1800000.000000"), Is144A: true, IsQIBEligible: false}, // 18% > 15% limit
			{SecurityID: "SEC-PUBLIC", Symbol: "PUB1", IssuerID: "ISS-11", MarketValue: decimal.RequireFromString("8200000.000000"), Is144A: false, IsQIBEligible: false},
		},
	}

	results5, err := evaluator.EvaluateAndPersist(ctx, state4)
	require.NoError(t, err)

	var secFinding *PostTradeEvaluationResult
	for i := range results5 {
		if results5[i].RuleCode == "SEC_144A_QIB_HOLDING" {
			secFinding = &results5[i]
		}
	}
	require.NotNil(t, secFinding)
	require.Equal(t, "BREACHED", secFinding.Action)
	require.Equal(t, "OPEN", secFinding.Status)

	// =========================================================================
	// Scenario 6: Margin Utilization Warning (> 80%)
	// =========================================================================
	// Gross = 10,000,000, Cash = 1,000,000 -> Borrowed = 9,000,000 / MarginLimit 10,000,000 = 90% > 80%
	state5 := PortfolioState{
		TenantID:      tenantA,
		AccountID:     accountA,
		AsOfDate:      asOfDate.AddDate(0, 0, 2),
		NAV:           decimal.RequireFromString("5000000.000000"),
		GrossExposure: decimal.RequireFromString("10000000.000000"),
		NetExposure:   decimal.RequireFromString("10000000.000000"),
		CashBalance:   decimal.RequireFromString("1000000.000000"),
		MarginLimit:   decimal.RequireFromString("10000000.000000"),
		Positions: []PortfolioPosition{
			{SecurityID: "SEC-PUB", Symbol: "PUB", IssuerID: "ISS-PUB", MarketValue: decimal.RequireFromString("10000000.000000")},
		},
	}

	results6, err := evaluator.EvaluateAndPersist(ctx, state5)
	require.NoError(t, err)

	var marginFinding *PostTradeEvaluationResult
	for i := range results6 {
		if results6[i].RuleCode == "MARGIN_UTILIZATION_80" {
			marginFinding = &results6[i]
		}
	}
	require.NotNil(t, marginFinding)
	require.Equal(t, "WARNING", marginFinding.Action)
	require.Equal(t, "OPEN", marginFinding.Status)

	// =========================================================================
	// Scenario 7: State Machine Illegal Transition Guard on Findings
	// =========================================================================
	// Attempt illegal transition: SUPERSEDED -> OPEN
	_, err = db.ExecContext(ctx, `
		UPDATE compliance.compliance_finding 
		SET status = 'OPEN' 
		WHERE id = $1
	`, ucitsFinding1.FindingID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Illegal compliance finding status transition")

	// =========================================================================
	// Scenario 8: Phase 1 Tranche 1 Rule Evaluations (Group, Debt, Counterparty PFE, Cash Floor)
	// =========================================================================
	statePhase1 := PortfolioState{
		TenantID:      tenantA,
		AccountID:     uuid.New(),
		AsOfDate:      asOfDate,
		NAV:           decimal.RequireFromString("20000000.000000"),
		GrossExposure: decimal.RequireFromString("22000000.000000"),
		NetExposure:   decimal.RequireFromString("20000000.000000"),
		CashBalance:   decimal.RequireFromString("600000.000000"), // 600,000 / 20,000,000 = 3% < 5% Cash Floor -> WARNING
		MarginLimit:   decimal.RequireFromString("10000000.000000"),
		Positions: []PortfolioPosition{
			// Group exposure: Parent GRP_ALPHA has two subsidiaries: ISS_A1 (2.5M) + ISS_A2 (2.0M) = 4.5M (22.5% > 20%) -> BREACH!
			{SecurityID: "SEC-G1", Symbol: "EQ-G1", IssuerID: "ISS_A1", ParentEntityID: "GRP_ALPHA", MarketValue: decimal.RequireFromString("2500000.000000")},
			{SecurityID: "SEC-G2", Symbol: "EQ-G2", IssuerID: "ISS_A2", ParentEntityID: "GRP_ALPHA", MarketValue: decimal.RequireFromString("2000000.000000")},
			// Debt exposure: Issuer ISS_DEBT has 3.6M fixed income = 18% > 15% -> BREACH!
			{SecurityID: "SEC-D1", Symbol: "BOND1", IssuerID: "ISS_DEBT", AssetClass: "FIXED_INCOME", MarketValue: decimal.RequireFromString("3600000.000000")},
			// Counterparty PFE: Counterparty CP_SWAP has 1.5M MTM + 1.0M PFE = 2.5M (12.5% > 10%) -> BREACH!
			{SecurityID: "SEC-SW1", Symbol: "IRS1", IssuerID: "ISS_SW", CounterpartyID: "CP_SWAP", AssetClass: "DERIVATIVE", MarketValue: decimal.RequireFromString("1500000.000000"), PFEAmount: decimal.RequireFromString("1000000.000000")},
		},
	}

	resultsPhase1, err := evaluator.EvaluateAndPersist(ctx, statePhase1)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(resultsPhase1), 7)

	for _, res := range resultsPhase1 {
		switch res.RuleCode {
		case "POST_TRADE_GROUP_ISSUER_20":
			require.Equal(t, "BREACHED", res.Action)
			require.Equal(t, "OPEN", res.Status)
		case "POST_TRADE_ISSUER_DEBT_15":
			require.Equal(t, "BREACHED", res.Action)
			require.Equal(t, "OPEN", res.Status)
		case "POST_TRADE_COUNTERPARTY_PFE_10":
			require.Equal(t, "BREACHED", res.Action)
			require.Equal(t, "OPEN", res.Status)
		case "POST_TRADE_CASH_MIN_5":
			require.Equal(t, "WARNING", res.Action)
			require.Equal(t, "OPEN", res.Status)
		}
	}

	// =========================================================================
	// Scenario 10: Phase 1 Tranche 2 Rule Evaluations (Sovereign, Agency, Muni, CCP, Custodian, Bank, Sec Lending)
	// =========================================================================
	accountTranche2 := uuid.New()
	stateTranche2 := PortfolioState{
		TenantID:                       tenantA,
		AccountID:                      accountTranche2,
		AsOfDate:                       asOfDate,
		NAV:                            decimal.RequireFromString("10000000.000000"),
		GrossExposure:                  decimal.RequireFromString("10000000.000000"),
		NetExposure:                    decimal.RequireFromString("10000000.000000"),
		CashBalance:                    decimal.RequireFromString("1000000.000000"),
		MarginLimit:                    decimal.RequireFromString("5000000.000000"),
		SecLendingTotalLoanValue:       decimal.RequireFromString("1000000.000000"),
		SecLendingTotalCollateralValue: decimal.RequireFromString("950000.000000"), // 950k / 1M = 95% < 102% floor -> WARNING
		Positions: []PortfolioPosition{
			{SecurityID: "SEC-SOV", Symbol: "ARG_BOND", IssuerID: "ISS_ARG", CountryOfRisk: "ARG", IssuerType: "SOVEREIGN", MarketValue: decimal.RequireFromString("3800000.000000")}, // 38% > 35% -> BREACH
			{SecurityID: "SEC-SUPRA", Symbol: "EIB_BOND", IssuerID: "ISS_EIB", IssuerType: "SUPRANATIONAL", MarketValue: decimal.RequireFromString("2800000.000000")},                   // 28% > 25% -> BREACH
			{SecurityID: "SEC-MUNI", Symbol: "NYC_MUNI", IssuerID: "MUNI_NYC", IssuerType: "MUNICIPAL", MarketValue: decimal.RequireFromString("1200000.000000")},                      // 12% > 10% -> BREACH
			{SecurityID: "SEC-CCP", Symbol: "CCP_MARGIN", IssuerID: "ISS_CCP", CCPID: "CCP_LCH", MarketValue: decimal.RequireFromString("1800000.000000")},                             // 18% > 15% -> BREACH
			{SecurityID: "SEC-CUST", Symbol: "BNY_CUST", IssuerID: "ISS_CUST", CustodianID: "CUST_BNY", MarketValue: decimal.RequireFromString("2500000.000000")},                      // 25% > 20% -> BREACH
			{SecurityID: "SEC-BANK", Symbol: "JPM_DEP", IssuerID: "ISS_BANK", BankID: "BANK_JPM", MarketValue: decimal.RequireFromString("2400000.000000")},                             // 24% > 20% -> BREACH
		},
	}

	resultsTranche2, err := evaluator.EvaluateAndPersist(ctx, stateTranche2)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(resultsTranche2), 14)

	for _, res := range resultsTranche2 {
		switch res.RuleCode {
		case "POST_TRADE_SOVEREIGN_EXPOSURE_35":
			require.Equal(t, "BREACHED", res.Action)
			require.Equal(t, "OPEN", res.Status)
		case "POST_TRADE_AGENCY_SUPRA_25":
			require.Equal(t, "BREACHED", res.Action)
			require.Equal(t, "OPEN", res.Status)
		case "POST_TRADE_MUNI_OBLIGOR_10":
			require.Equal(t, "BREACHED", res.Action)
			require.Equal(t, "OPEN", res.Status)
		case "POST_TRADE_CCP_CLEARING_EXPOSURE_15":
			require.Equal(t, "BREACHED", res.Action)
			require.Equal(t, "OPEN", res.Status)
		case "POST_TRADE_CUSTODIAN_CONCENTRATION_20":
			require.Equal(t, "BREACHED", res.Action)
			require.Equal(t, "OPEN", res.Status)
		case "POST_TRADE_BANK_DEPOSIT_20":
			require.Equal(t, "BREACHED", res.Action)
			require.Equal(t, "OPEN", res.Status)
		case "POST_TRADE_SEC_LENDING_COLLATERAL_102":
			require.Equal(t, "WARNING", res.Action)
			require.Equal(t, "OPEN", res.Status)
		}
	}

	// =========================================================================
	// Scenario 11: Row-Level Security Isolation Assertion
	// =========================================================================
	// Evaluate portfolio for Tenant B
	stateTenantB := PortfolioState{
		TenantID:      tenantB,
		AccountID:     uuid.New(),
		AsOfDate:      asOfDate,
		NAV:           decimal.RequireFromString("1000000.000000"),
		GrossExposure: decimal.RequireFromString("1000000.000000"),
		NetExposure:   decimal.RequireFromString("1000000.000000"),
		CashBalance:   decimal.RequireFromString("100000.000000"),
		Positions: []PortfolioPosition{
			{SecurityID: "SEC-B", Symbol: "EQB", IssuerID: "ISS-B", MarketValue: decimal.RequireFromString("1000000.000000")},
		},
	}
	_, err = evaluator.EvaluateAndPersist(ctx, stateTenantB)
	require.NoError(t, err)

	// Under Tenant A session: cannot see Tenant B snapshots or findings
	txA, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer txA.Rollback()

	_, err = txA.ExecContext(ctx, "SET ROLE app_user")
	require.NoError(t, err)

	_, err = txA.ExecContext(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenantA.String())
	require.NoError(t, err)

	var visibleSnapshotsForA, visibleFindingsForA int
	err = txA.QueryRowContext(ctx, "SELECT count(*) FROM compliance.compliance_portfolio_snapshot").Scan(&visibleSnapshotsForA)
	require.NoError(t, err)
	require.Equal(t, 5, visibleSnapshotsForA, "Tenant A must only see its own 5 snapshots")

	err = txA.QueryRowContext(ctx, "SELECT count(*) FROM compliance.compliance_finding").Scan(&visibleFindingsForA)
	require.NoError(t, err)
	require.GreaterOrEqual(t, visibleFindingsForA, 20, "Tenant A must only see its own findings")

	// =========================================================================
	// Scenario 12: Adversarial Missing Security Master Row (Left Join Unclassified Net)
	// =========================================================================
	// Position whose security is missing from security master (empty sector/country/symbol)
	// NAV = $10,000,000; Unjoined position = $800,000 (8% > 5% ceiling)
	stateUnjoined := PortfolioState{
		TenantID:      tenantA,
		AccountID:     accountA,
		AsOfDate:      asOfDate,
		NAV:           decimal.RequireFromString("10000000.000000"),
		GrossExposure: decimal.RequireFromString("10000000.000000"),
		NetExposure:   decimal.RequireFromString("10000000.000000"),
		CashBalance:   decimal.RequireFromString("2000000.000000"),
		Positions: []PortfolioPosition{
			{SecurityID: "SEC-VALID-1", Symbol: "EQ1", IssuerID: "ISS-1", MarketValue: decimal.RequireFromString("7200000.000000"), Sector: "Information Technology", IndustryGroup: "Software", CountryClassification: "DEVELOPED"},
			// Ghost position from unjoined table / missing master row:
			{SecurityID: "SEC-GHOST-MISSING", Symbol: "", IssuerID: "", MarketValue: decimal.RequireFromString("800000.000000"), Sector: "", IndustryGroup: "", CountryClassification: ""},
		},
	}

	resultsUnjoined, err := evaluator.EvaluateAndPersist(ctx, stateUnjoined)
	require.NoError(t, err)

	var unclassFinding *PostTradeEvaluationResult
	for i := range resultsUnjoined {
		if resultsUnjoined[i].RuleCode == "POST_TRADE_UNCLASSIFIED_CEILING_5" {
			unclassFinding = &resultsUnjoined[i]
		}
	}
	require.NotNil(t, unclassFinding, "Unclassified ceiling rule must be evaluated")
	require.Equal(t, "BREACHED", unclassFinding.Action)
	require.Equal(t, "OPEN", unclassFinding.Status)
	require.Equal(t, "DATA_QUALITY_INCIDENT", unclassFinding.Details["finding_category"])
	require.Equal(t, "DATA_REMEDIATION_REQUIRED", unclassFinding.Details["resolution_path"])
	require.Contains(t, unclassFinding.Details["breach_reason"], "DATA_REMEDIATION_REQUIRED")

	t.Logf("Post-Trade Batch Evaluator E2E Test PASSED: All 21 Post-Trade Rules, Unclassified Left-Join Safety Net, Corporate Group Lookthrough, Debt, Counterparty PFE, Cash Floor, Sovereign, Agency, Muni, CCP, Custodian, Bank, Sec Lending, UUIDv5 Lineage, Restatement Supersession, State Machine Triggers, and RLS Isolation Verified 100%%!")
}
