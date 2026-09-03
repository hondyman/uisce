package boresolver

import (
	"fmt"
	"testing"
)

// testUDFDialect is a minimal Dialect stand-in representing an engine that
// has (hypothetically) shipped a native XIRR UDF, e.g. a future StarRocks
// Java UDF. Embedding PostgresDialect keeps this focused on the one thing
// under test: dialect-aware function capability.
type testUDFDialect struct {
	PostgresDialect
}

func (testUDFDialect) Name() string { return "test_udf_engine" }

// TestResolveTier_BestEngineSelection proves the core "best engine" claim:
// the same function call resolves to TierHostRuntime on a dialect with no
// native implementation, and TierPushdown on a dialect that registers one —
// without CalcNode, CalcGraph, or CompileDeepCalculations changing at all.
func TestResolveTier_BestEngineSelection(t *testing.T) {
	const fnName = "test_quantile"
	RegisterFunction(fnName, &FunctionSpec{
		HostRuntime: func(rows []CalcRow, argNames []string) (float64, error) { return 0, nil },
		Pushdown: &PushdownBuilder{
			Supports: func(d Dialect) bool { return d.Name() == "test_udf_engine" },
			Render:   func(d Dialect, args []string) string { return fmt.Sprintf("TEST_QUANTILE(%s)", args[0]) },
		},
	})
	t.Cleanup(func() { delete(functionRegistry, fnName) })

	expr, err := ParseCalcFormula(fmt.Sprintf("%s(${x})", fnName))
	if err != nil {
		t.Fatalf("ParseCalcFormula: %v", err)
	}

	pgTier, err := ResolveTier(expr, PostgresDialect{}, PreferAuto)
	if err != nil {
		t.Fatalf("ResolveTier(postgres): %v", err)
	}
	if pgTier != TierHostRuntime {
		t.Errorf("expected TierHostRuntime on a dialect without the UDF, got %v", pgTier)
	}

	udfTier, err := ResolveTier(expr, testUDFDialect{}, PreferAuto)
	if err != nil {
		t.Fatalf("ResolveTier(test_udf_engine): %v", err)
	}
	if udfTier != TierPushdown {
		t.Errorf("expected TierPushdown once the dialect registers a UDF, got %v", udfTier)
	}
}

// TestResolveTier_PreferHostRuntimeOverridesPushdown proves the explicit
// override: even when a dialect CAN push a function down, PreferHostRuntime
// forces the Go executor — the knob a published/"official" calc needs to
// guarantee one consistent, auditable code path regardless of which engine
// served the query.
func TestResolveTier_PreferHostRuntimeOverridesPushdown(t *testing.T) {
	const fnName = "test_official_metric"
	RegisterFunction(fnName, &FunctionSpec{
		HostRuntime: func(rows []CalcRow, argNames []string) (float64, error) { return 0, nil },
		Pushdown: &PushdownBuilder{
			Supports: func(d Dialect) bool { return true },
			Render:   func(d Dialect, args []string) string { return fmt.Sprintf("OFFICIAL(%s)", args[0]) },
		},
	})
	t.Cleanup(func() { delete(functionRegistry, fnName) })

	expr, err := ParseCalcFormula(fmt.Sprintf("%s(${x})", fnName))
	if err != nil {
		t.Fatalf("ParseCalcFormula: %v", err)
	}

	tier, err := ResolveTier(expr, PostgresDialect{}, PreferHostRuntime)
	if err != nil {
		t.Fatalf("ResolveTier: %v", err)
	}
	if tier != TierHostRuntime {
		t.Errorf("PreferHostRuntime must win even though pushdown is available, got %v", tier)
	}
}

// TestResolveTier_PreferPushdownFailsLoudly proves PreferPushdown does not
// silently downgrade to host-runtime when the dialect can't actually
// render the formula — a caller that demanded pushdown (e.g. for a
// bulk/high-volume path) needs a compile error, not a silent fallback that
// changes the query's performance characteristics.
func TestResolveTier_PreferPushdownFailsLoudly(t *testing.T) {
	expr, err := ParseCalcFormula("xirr(${amount}, ${date})")
	if err != nil {
		t.Fatalf("ParseCalcFormula: %v", err)
	}

	_, err = ResolveTier(expr, PostgresDialect{}, PreferPushdown)
	if err == nil {
		t.Fatal("expected an error: xirr has no pushdown implementation on postgres")
	}
}
