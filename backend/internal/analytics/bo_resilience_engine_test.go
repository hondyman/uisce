package analytics

import (
	"testing"
)

func TestBOResilienceEngine_CircularDependencyDetection(t *testing.T) {
	engine := NewBOResilienceEngine(nil)

	// Test 1: Acyclic Calculation DAG
	dagAcyclic := map[string][]string{
		"order_total": {"line_amount", "tax_amount"},
		"line_amount": {"unit_price", "quantity", "discount_rate"},
		"tax_amount":  {"line_amount", "tax_rate"},
		"unit_price":  {},
		"quantity":    {},
		"discount_rate": {},
		"tax_rate":    {},
	}
	cycle, err := engine.DetectCircularCalculations(dagAcyclic)
	if err != nil || cycle != nil {
		t.Fatalf("expected acyclic graph to pass, got err=%v, cycle=%v", err, cycle)
	}

	// Test 2: Cyclic Calculation Loop: line_amount -> discount_rate -> customer_discount -> line_amount
	dagCyclic := map[string][]string{
		"order_total":       {"line_amount"},
		"line_amount":       {"discount_rate"},
		"discount_rate":     {"customer_discount"},
		"customer_discount": {"line_amount"},
	}
	cycle, err = engine.DetectCircularCalculations(dagCyclic)
	if err == nil {
		t.Fatalf("expected circular dependency error, got nil")
	}
	if len(cycle) < 3 {
		t.Fatalf("expected cycle path with at least 3 nodes, got %v", cycle)
	}
	t.Logf("Detected cycle successfully: %v (error: %v)", cycle, err)
}

func TestBOResilienceEngine_MakerCheckerLifecycle(t *testing.T) {
	engine := NewBOResilienceEngine(nil)

	// 1. Maker submits DRAFT
	status, ver, err := engine.TransitionLifecycle(StatusDraft, "SUBMIT_FOR_APPROVAL", false)
	if err != nil || status != StatusPendingApproval {
		t.Fatalf("expected PENDING_APPROVAL, got status=%s, err=%v", status, err)
	}

	// 2. Maker attempts to approve own submission (Violation of Maker-Checker)
	_, _, err = engine.TransitionLifecycle(StatusPendingApproval, "APPROVE", false)
	if err == nil {
		t.Fatalf("expected maker-checker rejection when maker tries to approve own submission")
	}

	// 3. Checker approves submission
	status, ver, err = engine.TransitionLifecycle(StatusPendingApproval, "APPROVE", true)
	if err != nil || status != StatusPublished || ver != 1 {
		t.Fatalf("expected PUBLISHED with version 1, got status=%s, ver=%d, err=%v", status, ver, err)
	}

	// 4. Draft new version from Published
	status, ver, err = engine.TransitionLifecycle(StatusPublished, "DRAFT_NEW_VERSION", false)
	if err != nil || status != StatusDraft || ver != 2 {
		t.Fatalf("expected DRAFT with version 2, got status=%s, ver=%d, err=%v", status, ver, err)
	}
}
