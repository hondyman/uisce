package analytics

import (
	"strings"
	"testing"
)

func TestQualitySentinel_RequiredNullsBlock(t *testing.T) {
	// Sample: 500 rows, 25 nulls => NullRatio = 0.05
	status, reasons := EvaluateGatekeeperRules("DIMENSION", "REQUIRED", 0.05, 1.0, 500, 25, 475)
	if status != QualityBlockedRequiredNulls {
		t.Fatalf("expected BLOCKED_REQUIRED_NULLS, got %s", status)
	}
	if len(reasons) == 0 || !strings.Contains(reasons[0], "REQUIRED but contains 5.00% NULL values") {
		t.Fatalf("expected blocking reason mentioning required nulls, got %v", reasons)
	}
}

func TestQualitySentinel_DuplicateBusinessKeyBlock(t *testing.T) {
	// Sample: 500 rows, 0 nulls, 450 distinct keys => UniquenessRatio = 0.90 (50 duplicates)
	status, reasons := EvaluateGatekeeperRules("BUSINESS_KEY", "REQUIRED", 0.0, 0.90, 500, 0, 450)
	if status != QualityBlockedIdentityCollision {
		t.Fatalf("expected BLOCKED_IDENTITY_COLLISION, got %s", status)
	}
	if len(reasons) == 0 || !strings.Contains(reasons[0], "Identity duplicate detected") {
		t.Fatalf("expected blocking reason mentioning identity duplicates, got %v", reasons)
	}
}

func TestQualitySentinel_OptionalNullsWarning(t *testing.T) {
	// Sample: 500 rows, 10 nulls on OPTIONAL field
	status, reasons := EvaluateGatekeeperRules("ATTRIBUTE", "OPTIONAL", 0.02, 1.0, 500, 10, 490)
	if status != QualityWarnNulls {
		t.Fatalf("expected WARN_NULLS, got %s", status)
	}
	if len(reasons) != 0 {
		t.Fatalf("optional nulls should warn but not block, got reasons: %v", reasons)
	}
}

func TestQualitySentinel_HealthyField(t *testing.T) {
	// Clean field: 0 nulls, 100% unique
	status, reasons := EvaluateGatekeeperRules("KEY", "REQUIRED", 0.0, 1.0, 500, 0, 500)
	if status != QualityHealthy {
		t.Fatalf("expected HEALTHY, got %s", status)
	}
	if len(reasons) != 0 {
		t.Fatalf("expected no reasons for healthy field, got %v", reasons)
	}
}

func TestQualitySentinel_DefensiveASTRewrite(t *testing.T) {
	// 1. Division: Margin = gross_profit / revenue -> (gross_profit / NULLIF(revenue, 0))
	divExpr := RewriteDefensiveASTProjection("gross_profit / revenue", "NUMERIC", "", false, true)
	expectedDiv := "(gross_profit / NULLIF(revenue, 0))"
	if divExpr != expectedDiv {
		t.Errorf("expected %s, got %s", expectedDiv, divExpr)
	}

	// 2. Numeric Nullable with Fallback: COALESCE(discount_rate, 0)
	numExpr := RewriteDefensiveASTProjection("discount_rate", "NUMERIC(10,4)", "0", true, false)
	expectedNum := "COALESCE(discount_rate, 0)"
	if numExpr != expectedNum {
		t.Errorf("expected %s, got %s", expectedNum, numExpr)
	}

	// 3. String Nullable with Fallback: COALESCE(middle_name, 'N/A')
	strExpr := RewriteDefensiveASTProjection("middle_name", "VARCHAR(50)", "N/A", true, false)
	expectedStr := "COALESCE(middle_name, 'N/A')"
	if strExpr != expectedStr {
		t.Errorf("expected %s, got %s", expectedStr, strExpr)
	}

	// 4. Combined Division and Fallback
	comboExpr := RewriteDefensiveASTProjection("total_spend / visit_count", "NUMERIC", "0.0", true, true)
	expectedCombo := "COALESCE((total_spend / NULLIF(visit_count, 0)), 0.0)"
	if comboExpr != expectedCombo {
		t.Errorf("expected %s, got %s", expectedCombo, comboExpr)
	}
}
