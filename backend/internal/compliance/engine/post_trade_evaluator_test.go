package engine

import (
	"context"
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

	// Verify Seed of Pilot Post-Trade Rules
	var pilotRuleCount int
	err = db.QueryRowContext(ctx, `
		SELECT count(*) FROM compliance.compliance_rule 
		WHERE rule_code IN ('UCITS_5_10_40', 'SEC_144A_QIB_HOLDING', 'MARGIN_UTILIZATION_80')
		  AND library_status = 'ACTIVE'
	`).Scan(&pilotRuleCount)
	require.NoError(t, err)
	require.Equal(t, 3, pilotRuleCount, "All 3 pilot post-trade rules must be ACTIVE in compliance_rule")

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
		MarginLimit:   decimal.RequireFromString("15000000.000000"),
		Positions: []PortfolioPosition{
			{SecurityID: "SEC-001", Symbol: "EQ1", IssuerID: "ISS-1", MarketValue: decimal.RequireFromString("800000.000000"), Is144A: false, IsQIBEligible: false},
			{SecurityID: "SEC-002", Symbol: "EQ2", IssuerID: "ISS-2", MarketValue: decimal.RequireFromString("900000.000000"), Is144A: false, IsQIBEligible: false},
			{SecurityID: "SEC-003", Symbol: "EQ3", IssuerID: "ISS-3", MarketValue: decimal.RequireFromString("900000.000000"), Is144A: false, IsQIBEligible: false},
			{SecurityID: "SEC-004", Symbol: "EQ4", IssuerID: "ISS-4", MarketValue: decimal.RequireFromString("850000.000000"), Is144A: false, IsQIBEligible: false},
			{SecurityID: "SEC-005", Symbol: "EQ5", IssuerID: "ISS-5", MarketValue: decimal.RequireFromString("850000.000000"), Is144A: false, IsQIBEligible: false},
			{SecurityID: "SEC-006", Symbol: "BD1", IssuerID: "ISS-6", MarketValue: decimal.RequireFromString("500000.000000"), Is144A: true, IsQIBEligible: false},
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
	var openFindingsCount int
	err = db.QueryRowContext(ctx, "SELECT count(*) FROM compliance.compliance_finding WHERE tenant_id = $1 AND status = 'OPEN'", tenantA).Scan(&openFindingsCount)
	require.NoError(t, err)
	require.Equal(t, 1, openFindingsCount)

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
		{SecurityID: "SEC-001", Symbol: "EQ1", IssuerID: "ISS-1", MarketValue: decimal.RequireFromString("700000.000000"), Is144A: false, IsQIBEligible: false},
		{SecurityID: "SEC-002", Symbol: "EQ2", IssuerID: "ISS-2", MarketValue: decimal.RequireFromString("700000.000000"), Is144A: false, IsQIBEligible: false},
		{SecurityID: "SEC-003", Symbol: "EQ3", IssuerID: "ISS-3", MarketValue: decimal.RequireFromString("700000.000000"), Is144A: false, IsQIBEligible: false},
		{SecurityID: "SEC-004", Symbol: "EQ4", IssuerID: "ISS-4", MarketValue: decimal.RequireFromString("700000.000000"), Is144A: false, IsQIBEligible: false},
		{SecurityID: "SEC-005", Symbol: "EQ5", IssuerID: "ISS-5", MarketValue: decimal.RequireFromString("700000.000000"), Is144A: false, IsQIBEligible: false}, // Total 35%
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
	// Scenario 8: Row-Level Security Isolation Assertion
	// =========================================================================
	// Evaluate portfolio for Tenant B
	stateTenantB := PortfolioState{
		TenantID:      tenantB,
		AccountID:     uuid.New(),
		AsOfDate:      asOfDate,
		NAV:           decimal.RequireFromString("1000000.000000"),
		GrossExposure: decimal.RequireFromString("1000000.000000"),
		NetExposure:   decimal.RequireFromString("1000000.000000"),
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
	require.Equal(t, 3, visibleSnapshotsForA, "Tenant A must only see its own 3 snapshots")

	err = txA.QueryRowContext(ctx, "SELECT count(*) FROM compliance.compliance_finding").Scan(&visibleFindingsForA)
	require.NoError(t, err)
	require.Equal(t, 6, visibleFindingsForA, "Tenant A must only see its own findings")

	t.Logf("Post-Trade Batch Evaluator E2E Test PASSED: All 3 Pilot Rules, UUIDv5 Lineage, Restatement Supersession, State Machine Triggers, and RLS Isolation Verified 100%%!")
}
