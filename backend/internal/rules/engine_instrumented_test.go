package rules

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type fakeDriftHealer struct {
	calls atomic.Int64
}

func (f *fakeDriftHealer) HandleCompileFailure(ctx context.Context, tenantID, boID uuid.UUID, ruleID, missingSymbol string) error {
	f.calls.Add(1)
	return nil
}

func TestEvaluate_RecordsLatency(t *testing.T) {
	profiler := NewLatencyProfiler()
	eng := NewRuleEngine(nil)
	eng.SetProfiler(profiler)

	input := map[string]any{"field": float64(100)}
	node := &RuleNode{Type: NodeTypeCondition, Condition: &RuleCondition{
		Field:    "field",
		Operator: "GREATER_THAN",
		Value:    float64(50),
	}}

	passed, _, err := eng.Evaluate(context.Background(), "", "r1", 0, node, input, false)
	assert.NoError(t, err)
	assert.True(t, passed)

	dist := profiler.GetDistribution()
	assert.Greater(t, dist.Count, uint64(0))
}

func TestEvaluate_DispatchesDriftHealer(t *testing.T) {
	healer := &fakeDriftHealer{}
	eng := NewRuleEngine(nil)
	eng.SetDriftHealer(healer)

	input := map[string]any{"field": float64(100)}
	node := &RuleNode{Type: NodeTypeCondition, Condition: &RuleCondition{
		Field:    "field",
		Operator: "GREATER_THAN",
		Value:    float64(50),
	}}

	state := eng.GetState("")
	state.Syms.Freeze()

	node.Condition.Field = "never_ever_registered_field_xyz"
	_, _, err := eng.Evaluate(context.Background(), "tenant-1", "r2", 0, node, input, false)
	assert.NoError(t, err)

	time.Sleep(200 * time.Millisecond)
	assert.Greater(t, healer.calls.Load(), int64(0), "drift healer should have been called for unregistered symbol after syms frozen")
}

func TestEvaluate_NoPanicIfDriftHealerNil(t *testing.T) {
	eng := NewRuleEngine(nil)

	input := map[string]any{"field": float64(100)}
	node := &RuleNode{Type: NodeTypeCondition, Condition: &RuleCondition{
		Field:    "unregistered_symbol_xyz",
		Operator: "GREATER_THAN",
		Value:    float64(50),
	}}

	_, _, err := eng.Evaluate(context.Background(), "tenant-1", "r3", 0, node, input, false)
	assert.NoError(t, err)
}

func TestGetSymsForTenant_ReturnsSyms(t *testing.T) {
	eng := NewRuleEngine(nil)
	syms := eng.GetSymsForTenant("")
	assert.NotNil(t, syms)
}

func TestGetEnumsForTenant_ReturnsEnums(t *testing.T) {
	eng := NewRuleEngine(nil)
	enums := eng.GetEnumsForTenant("")
	assert.NotNil(t, enums)
}

func TestGetState_ReturnsState(t *testing.T) {
	eng := NewRuleEngine(nil)
	state := eng.GetState("")
	assert.NotNil(t, state)
	assert.Equal(t, uint64(0), state.Revision)
}

func TestWithBOID_RoundTrips(t *testing.T) {
	id := uuid.New()
	ctx := WithBOID(context.Background(), id)
	retrieved := BOIDFromContext(ctx)
	assert.Equal(t, id, retrieved)
}

func TestBOIDFromContext_NilContext(t *testing.T) {
	id := BOIDFromContext(context.Background())
	assert.Equal(t, uuid.Nil, id)
}

func TestExtractSymbolFromReason(t *testing.T) {
	tests := []struct {
		reason string
		want   string
	}{
		{`symbol not registered for "foo.bar.baz": some error`, "foo.bar.baz"},
		{`symbol not registered for "a.b": other error`, "a.b"},
		{`unrelated error`, `unrelated error`},
		{`symbol not registered for "x":`, "x"},
		{`no quotes here`, `no quotes here`},
	}
	for _, tt := range tests {
		got := extractSymbolFromReason(tt.reason)
		assert.Equal(t, tt.want, got)
	}
}
