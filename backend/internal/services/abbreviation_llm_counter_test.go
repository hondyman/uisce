package services

import (
	"context"
	"testing"

	"github.com/hondyman/uisce/backend/internal/testutils"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// TestRecordLLMCall_IncrementsCounter verifies that recordLLMCall increments the
// glossary_llm_calls_total counter with the correct tenant and method labels.
// This is the PR-α counter: prod-level visibility into the exact number of
// LLM attempts per tenant per method, which PR-β's pre-grouped dispatch is
// expected to reduce.
func TestRecordLLMCall_IncrementsCounter(t *testing.T) {
	// Reset the counter to a known state
	glossaryLLMCalls.Reset()

	ctx := context.WithValue(context.Background(), "tenant_id", "test-tenant-abc")

	// Record one expansion call
	recordLLMCall(ctx, "expansion")
	recordLLMCall(ctx, "expansion") // second call to the same method

	// Record one qualify call
	recordLLMCall(ctx, "qualify")

	// Record one definition call
	recordLLMCall(ctx, "definition")

	// Assert exact values
	if got := testutil.ToFloat64(glossaryLLMCalls.WithLabelValues("test-tenant-abc", "expansion")); got != 2 {
		t.Errorf("expansion: expected 2, got %f", got)
	}
	if got := testutil.ToFloat64(glossaryLLMCalls.WithLabelValues("test-tenant-abc", "qualify")); got != 1 {
		t.Errorf("qualify: expected 1, got %f", got)
	}
	if got := testutil.ToFloat64(glossaryLLMCalls.WithLabelValues("test-tenant-abc", "definition")); got != 1 {
		t.Errorf("definition: expected 1, got %f", got)
	}

	// A different tenant should be independent
	recordLLMCall(ctx, "definition")
	if got := testutil.ToFloat64(glossaryLLMCalls.WithLabelValues("test-tenant-abc", "definition")); got != 2 {
		t.Errorf("definition after second call: expected 2, got %f", got)
	}

	glossaryLLMCalls.Reset()
}

// TestExtractTenantFromContext tests the tenant extraction helper.
func TestExtractTenantFromContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), "tenant_id", "my-tenant")
	if got := extractTenantFromContext(ctx); got != "my-tenant" {
		t.Errorf("expected 'my-tenant', got %q", got)
	}

	// Missing key → "unknown"
	if got := extractTenantFromContext(context.Background()); got != "unknown" {
		t.Errorf("expected 'unknown' for missing key, got %q", got)
	}

	// Empty string → "unknown"
	ctx2 := context.WithValue(context.Background(), "tenant_id", "")
	if got := extractTenantFromContext(ctx2); got != "unknown" {
		t.Errorf("expected 'unknown' for empty string, got %q", got)
	}
}

// TestSuggestExpansionsInContext_IncrementsCounter wires the counter through the
// real SuggestExpansionsInContext method to verify end-to-end: the mock LLM
// provider returns a canned response, and the counter increments.
func TestSuggestExpansionsInContext_IncrementsCounter(t *testing.T) {
	glossaryLLMCalls.Reset()

	mockProvider := &testutils.MockLLMProvider{
		GenerateResponseFunc: func(ctx context.Context, prompt string) (string, error) {
			return `{"EX": "EX_DIVIDEND"}`, nil
		},
	}

	svc := &AbbreviationService{
		llmProvider: mockProvider,
	}

	ctx := context.WithValue(context.Background(), "tenant_id", "counter-test-tenant")

	_, err := svc.SuggestExpansionsInContext(ctx, []string{"EX"}, "ex_date", `table "orders" in schema "orm"`, []string{"settlement_date"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := testutil.ToFloat64(glossaryLLMCalls.WithLabelValues("counter-test-tenant", "expansion")); got != 1 {
		t.Errorf("expansion counter: expected 1, got %f", got)
	}

	glossaryLLMCalls.Reset()
}

// TestQualifyGenericWord_IncrementsCounter wires the counter through the real
// QualifyGenericWord method.
func TestQualifyGenericWord_IncrementsCounter(t *testing.T) {
	glossaryLLMCalls.Reset()

	mockProvider := &testutils.MockLLMProvider{
		GenerateResponseFunc: func(ctx context.Context, prompt string) (string, error) {
			return "EmployeeCity", nil
		},
	}

	svc := &AbbreviationService{
		llmProvider: mockProvider,
	}

	ctx := context.WithValue(context.Background(), "tenant_id", "counter-test-tenant")

	_, err := svc.QualifyGenericWord(ctx, "city", "employees", []string{"first_name", "last_name"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := testutil.ToFloat64(glossaryLLMCalls.WithLabelValues("counter-test-tenant", "qualify")); got != 1 {
		t.Errorf("qualify counter: expected 1, got %f", got)
	}

	glossaryLLMCalls.Reset()
}

// TestGenerateStandardDefinition_IncrementsCounter wires the counter through
// the real GenerateStandardDefinition method.
func TestGenerateStandardDefinition_IncrementsCounter(t *testing.T) {
	glossaryLLMCalls.Reset()

	mockProvider := &testutils.MockLLMProvider{
		GenerateResponseFunc: func(ctx context.Context, prompt string) (string, error) {
			return `{"definition": "A financial instrument identifier.", "source": "FINRA"}`, nil
		},
	}

	svc := &AbbreviationService{
		llmProvider: mockProvider,
	}

	ctx := context.WithValue(context.Background(), "tenant_id", "counter-test-tenant")

	_, err := svc.GenerateStandardDefinition(ctx, "SettlementDate", "settlement_date")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := testutil.ToFloat64(glossaryLLMCalls.WithLabelValues("counter-test-tenant", "definition")); got != 1 {
		t.Errorf("definition counter: expected 1, got %f", got)
	}

	glossaryLLMCalls.Reset()
}
