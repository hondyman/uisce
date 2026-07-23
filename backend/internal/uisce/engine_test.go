package uisce

import (
	"context"
	"errors"
	"testing"
)

// MockFilter implements Filter interface for testing
type MockFilter struct {
	name        string
	shouldFail  bool
	failMessage string
}

func (f *MockFilter) Name() string {
	return f.name
}

func (f *MockFilter) Purify(ctx context.Context, data map[string]interface{}) error {
	if f.shouldFail {
		return errors.New(f.failMessage)
	}
	return nil
}

func TestRunDebug(t *testing.T) {
	// Setup Mock Filters
	filters := []Filter{
		&MockFilter{name: "Check1", shouldFail: false},
		&MockFilter{name: "Check2", shouldFail: true, failMessage: "Validation failed"},
		&MockFilter{name: "Check3", shouldFail: false}, // Should not be reached
	}

	// Create Engine (PolicyManager can be nil for this test as we mocking filters)
	engine := NewEngine(nil, filters)

	// Test Data
	tradeData := map[string]interface{}{
		"id":     "TRADE-001",
		"amount": 1000,
	}

	// Run Debug
	result := engine.RunDebug(context.Background(), tradeData)

	// Assertions
	if result.TradeID != "TRADE-001" {
		t.Errorf("Expected TradeID 'TRADE-001', got %s", result.TradeID)
	}

	if result.Success {
		t.Error("Expected result.Success to be false")
	}

	if len(result.Steps) != 2 {
		t.Errorf("Expected 2 steps (Pass, Fail), got %d", len(result.Steps))
	}

	step1 := result.Steps[0]
	if step1.FilterName != "Check1" || step1.Status != "PASS" {
		t.Errorf("Step 1 incorrect: %+v", step1)
	}

	step2 := result.Steps[1]
	if step2.FilterName != "Check2" || step2.Status != "FAIL" {
		t.Errorf("Step 2 incorrect: %+v", step2)
	}

	if step2.ErrorDetails != "Validation failed" {
		t.Errorf("Expected error details 'Validation failed', got '%s'", step2.ErrorDetails)
	}
}

func TestRunDebugSuccess(t *testing.T) {
	filters := []Filter{
		&MockFilter{name: "Check1", shouldFail: false},
		&MockFilter{name: "Check2", shouldFail: false},
	}

	engine := NewEngine(nil, filters)
	tradeData := map[string]interface{}{"id": "TRADE-OK"}

	result := engine.RunDebug(context.Background(), tradeData)

	if !result.Success {
		t.Error("Expected success")
	}

	if len(result.Steps) != 2 {
		t.Errorf("Expected 2 steps, got %d", len(result.Steps))
	}
}
