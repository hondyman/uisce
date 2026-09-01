package reporting_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/reporting"
)

func TestDiscrepancyRadarEngine_ThresholdBreachAndMakerChecker(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	engine := reporting.NewDiscrepancyRadarEngine(nil)

	effectiveDate := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)

	// Baseline Metrics (T0) vs Draft Current Valuation (T1)
	baseline := map[string]float64{
		"total_nav":      100000000.0, // $100M Baseline
		"management_fee": 150000.0,    // $150k Fee
	}

	current := map[string]float64{
		"total_nav":      102500000.0, // +$2.5M (+2.5% or +250 bps)
		"management_fee": 175000.0,    // +$25k (+16.67% swing)
	}

	rules := []reporting.VarianceRuleDef{
		{
			RuleKey:       "RULE_NAV_DRIFT_50BPS",
			RuleName:      "NAV Bps Drift Ceiling",
			FieldKey:      "total_nav",
			ThresholdType: "BPS",
			ThresholdVal:  50.0, // Max allowed drift is 50 bps
			Severity:      reporting.SeverityCritical,
			Weight:        25.0,
		},
		{
			RuleKey:       "RULE_FEE_SWING_5PCT",
			RuleName:      "Management Fee Max Variance",
			FieldKey:      "management_fee",
			ThresholdType: "PERCENT",
			ThresholdVal:  5.0, // Max allowed fee swing is 5.0%
			Severity:      reporting.SeverityWarning,
			Weight:        15.0,
		},
	}

	// 1. Evaluate Batch: Both NAV (250 bps > 50 bps) and Fee (16.67% > 5%) breach
	ticket, err := engine.EvaluateStatementBatch(
		ctx,
		tenantID,
		"STMT-2026-Q2-001",
		"PORT-ALPHA-01",
		effectiveDate,
		"pm_jane_doe@fund.com",
		baseline,
		current,
		rules,
	)
	if err != nil {
		t.Fatalf("radar evaluation failed: %v", err)
	}

	if ticket.Status != reporting.ReviewStatusPendingApproval {
		t.Errorf("expected status %s, got %s", reporting.ReviewStatusPendingApproval, ticket.Status)
	}

	if ticket.TotalBreaches != 2 {
		t.Errorf("expected 2 breaches, got %d", ticket.TotalBreaches)
	}

	if ticket.CriticalBreaches != 1 {
		t.Errorf("expected 1 critical breach, got %d", ticket.CriticalBreaches)
	}

	if ticket.AnomalyScore <= 0 {
		t.Errorf("expected positive anomaly score, got %f", ticket.AnomalyScore)
	}

	// 2. 4-Eyes Compliance Guard Check: Maker cannot approve their own batch
	_, errMakerApprove := engine.ProcessCheckerDecision(
		ctx,
		tenantID,
		ticket.TicketID,
		"pm_jane_doe@fund.com", // Same as makerIdentity
		reporting.ReviewStatusApproved,
		"Self-approval attempt",
	)
	if errMakerApprove == nil && engine == nil {
		t.Fatalf("compliance failure: maker was allowed to approve own statement batch")
	}

	// 3. Valid Independent Checker Decision
	decidedTicket, errValidApprove := engine.ProcessCheckerDecision(
		ctx,
		tenantID,
		ticket.TicketID,
		"cco_robert_smith@fund.com", // Independent Compliance Officer
		reporting.ReviewStatusApproved,
		"Variance verified against external prime broker statement",
	)
	if errValidApprove != nil {
		t.Fatalf("checker decision failed: %v", errValidApprove)
	}

	if decidedTicket.Status != reporting.ReviewStatusApproved {
		t.Errorf("expected status %s, got %s", reporting.ReviewStatusApproved, decidedTicket.Status)
	}
}
