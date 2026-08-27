package reporting

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCompileEmptyModel(t *testing.T) {
	model := &FilterModel{Groups: []FilterGroup{}, GroupCombinator: "AND"}
	result := CompileFilterModel(model, nil, &TenantDefaults{DefaultCalendarCode: "US"})
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestCompileDisabledFilter(t *testing.T) {
	model := &FilterModel{
		Groups: []FilterGroup{
			{
				ID:         "g1",
				Combinator: "AND",
				Filters: []Filter{
					{ID: "f1", Field: "status", Operator: OpEquals, Enabled: false, ValueSource: ValueSource{Kind: ValueSourceConstant, Value: "Active"}},
				},
			},
		},
		GroupCombinator: "AND",
	}
	result := CompileFilterModel(model, nil, &TenantDefaults{})
	if result != "" {
		t.Errorf("expected empty string for disabled filter, got %q", result)
	}
}

func TestCompileEquals(t *testing.T) {
	model := &FilterModel{
		Groups: []FilterGroup{
			{ID: "g1", Combinator: "AND", Filters: []Filter{
				{ID: "f1", Field: "status", Operator: OpEquals, Enabled: true, ValueSource: ValueSource{Kind: ValueSourceConstant, Value: "Active"}},
			}},
		},
	}
	result := CompileFilterModel(model, nil, &TenantDefaults{})
	expected := `"status" = 'Active'`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestCompileBetween(t *testing.T) {
	model := &FilterModel{
		Groups: []FilterGroup{
			{ID: "g1", Combinator: "AND", Filters: []Filter{
				{ID: "f1", Field: "trade_date", Operator: OpBetween, Enabled: true, Values: []string{"2026-01-01", "2026-12-31"}, ValueSource: ValueSource{Kind: ValueSourceConstant}},
			}},
		},
	}
	result := CompileFilterModel(model, nil, &TenantDefaults{})
	expected := `"trade_date" BETWEEN '2026-01-01' AND '2026-12-31'`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestCompileIn(t *testing.T) {
	model := &FilterModel{
		Groups: []FilterGroup{
			{ID: "g1", Combinator: "AND", Filters: []Filter{
				{ID: "f1", Field: "region", Operator: OpIn, Enabled: true, Values: []string{"APAC", "EMEA", "Americas"}, ValueSource: ValueSource{Kind: ValueSourceConstant}},
			}},
		},
	}
	result := CompileFilterModel(model, nil, &TenantDefaults{})
	expected := `"region" IN ('APAC', 'EMEA', 'Americas')`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestCompileIsNull(t *testing.T) {
	model := &FilterModel{
		Groups: []FilterGroup{
			{ID: "g1", Combinator: "AND", Filters: []Filter{
				{ID: "f1", Field: "closed_at", Operator: OpIsNull, Enabled: true, ValueSource: ValueSource{Kind: ValueSourceConstant}},
			}},
		},
	}
	result := CompileFilterModel(model, nil, &TenantDefaults{})
	expected := `"closed_at" IS NULL`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestCompileContains(t *testing.T) {
	model := &FilterModel{
		Groups: []FilterGroup{
			{ID: "g1", Combinator: "AND", Filters: []Filter{
				{ID: "f1", Field: "name", Operator: OpContains, Enabled: true, ValueSource: ValueSource{Kind: ValueSourceConstant, Value: "Smith"}},
			}},
		},
	}
	result := CompileFilterModel(model, nil, &TenantDefaults{})
	expected := `"name" ILIKE '%Smith%'`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestCompileNextBusinessDay(t *testing.T) {
	model := &FilterModel{
		Groups: []FilterGroup{
			{ID: "g1", Combinator: "AND", Filters: []Filter{
				{ID: "f1", Field: "settle_date", Operator: OpNextBusinessDay, Enabled: true, ValueSource: ValueSource{Kind: ValueSourceTenantDefault}},
			}},
		},
	}
	defaults := &TenantDefaults{DefaultCalendarCode: "NYSE"}
	result := CompileFilterModel(model, nil, defaults)
	expected := `"settle_date" = calendar_next_business_day("settle_date", 'NYSE')`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestCompilePreviousBusinessDay(t *testing.T) {
	model := &FilterModel{
		Groups: []FilterGroup{
			{ID: "g1", Combinator: "AND", Filters: []Filter{
				{ID: "f1", Field: "trade_date", Operator: OpPreviousBusinessDay, Enabled: true, ValueSource: ValueSource{Kind: ValueSourceTenantDefault}},
			}},
		},
	}
	defaults := &TenantDefaults{DefaultCalendarCode: "LSE"}
	result := CompileFilterModel(model, nil, defaults)
	expected := `"trade_date" = calendar_previous_business_day("trade_date", 'LSE')`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestCompileAddBusinessDays(t *testing.T) {
	model := &FilterModel{
		Groups: []FilterGroup{
			{ID: "g1", Combinator: "AND", Filters: []Filter{
				{ID: "f1", Field: "value_date", Operator: OpAddBusinessDays, Enabled: true, Values: []string{"5"}, ValueSource: ValueSource{Kind: ValueSourceTenantDefault}},
			}},
		},
	}
	defaults := &TenantDefaults{DefaultCalendarCode: "TARGET2"}
	result := CompileFilterModel(model, nil, defaults)
	expected := `"value_date" = calendar_add_business_days("value_date", 'TARGET2', 5)`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestCompileIsBusinessDay(t *testing.T) {
	model := &FilterModel{
		Groups: []FilterGroup{
			{ID: "g1", Combinator: "AND", Filters: []Filter{
				{ID: "f1", Field: "processing_date", Operator: OpIsBusinessDay, Enabled: true, ValueSource: ValueSource{Kind: ValueSourceTenantDefault}},
			}},
		},
	}
	defaults := &TenantDefaults{DefaultCalendarCode: "NYSE"}
	result := CompileFilterModel(model, nil, defaults)
	expected := `calendar_is_business_day("processing_date", 'NYSE')`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestCompileLastNBusinessDays(t *testing.T) {
	model := &FilterModel{
		Groups: []FilterGroup{
			{ID: "g1", Combinator: "AND", Filters: []Filter{
				{ID: "f1", Field: "trade_date", Operator: OpLastNBusinessDays, Enabled: true, Values: []string{"10"}, ValueSource: ValueSource{Kind: ValueSourceTenantDefault}},
			}},
		},
	}
	defaults := &TenantDefaults{DefaultCalendarCode: "NYSE"}
	result := CompileFilterModel(model, nil, defaults)
	expected := `"trade_date" >= calendar_add_business_days(CURRENT_DATE, 'NYSE', -10) AND "trade_date" <= CURRENT_DATE`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestCompileToday(t *testing.T) {
	model := &FilterModel{
		Groups: []FilterGroup{
			{ID: "g1", Combinator: "AND", Filters: []Filter{
				{ID: "f1", Field: "trade_date", Operator: OpToday, Enabled: true, ValueSource: ValueSource{Kind: ValueSourceConstant}},
			}},
		},
	}
	result := CompileFilterModel(model, nil, &TenantDefaults{})
	expected := `"trade_date" = CURRENT_DATE`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestCompileStartOfMonth(t *testing.T) {
	model := &FilterModel{
		Groups: []FilterGroup{
			{ID: "g1", Combinator: "AND", Filters: []Filter{
				{ID: "f1", Field: "period_end", Operator: OpStartOfMonth, Enabled: true, ValueSource: ValueSource{Kind: ValueSourceConstant}},
			}},
		},
	}
	result := CompileFilterModel(model, nil, &TenantDefaults{})
	expected := `"period_end" >= date_trunc('month', CURRENT_DATE)`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestCompileLastNDays(t *testing.T) {
	model := &FilterModel{
		Groups: []FilterGroup{
			{ID: "g1", Combinator: "AND", Filters: []Filter{
				{ID: "f1", Field: "created_at", Operator: OpLastNDays, Enabled: true, Values: []string{"30"}, ValueSource: ValueSource{Kind: ValueSourceConstant}},
			}},
		},
	}
	result := CompileFilterModel(model, nil, &TenantDefaults{})
	expected := `"created_at" >= CURRENT_DATE - INTERVAL '30 days' AND "created_at" <= CURRENT_DATE`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestCompileParameterRef(t *testing.T) {
	model := &FilterModel{
		Groups: []FilterGroup{
			{ID: "g1", Combinator: "AND", Filters: []Filter{
				{ID: "f1", Field: "region", Operator: OpEquals, Enabled: true, ValueSource: ValueSource{Kind: ValueSourceParameter, ParameterName: "RegionFilter"}},
			}},
		},
	}
	params := map[string]interface{}{"RegionFilter": "APAC"}
	result := CompileFilterModel(model, params, &TenantDefaults{})
	expected := `"region" = @RegionFilter`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestCompileExplicitCalendar(t *testing.T) {
	model := &FilterModel{
		Groups: []FilterGroup{
			{ID: "g1", Combinator: "AND", Filters: []Filter{
				{ID: "f1", Field: "settle_date", Operator: OpNextBusinessDay, Enabled: true, ValueSource: ValueSource{Kind: ValueSourceCalendar, CalendarCode: "LSE"}},
			}},
		},
	}
	result := CompileFilterModel(model, nil, &TenantDefaults{})
	expected := `"settle_date" = calendar_next_business_day("settle_date", 'LSE')`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestCompileAndGroups(t *testing.T) {
	model := &FilterModel{
		Groups: []FilterGroup{
			{
				ID:         "g1",
				Combinator: "AND",
				Filters: []Filter{
					{ID: "f1", Field: "region", Operator: OpEquals, Enabled: true, ValueSource: ValueSource{Kind: ValueSourceConstant, Value: "APAC"}},
					{ID: "f2", Field: "amount", Operator: OpGreaterThan, Enabled: true, ValueSource: ValueSource{Kind: ValueSourceConstant, Value: "1000"}},
				},
			},
		},
		GroupCombinator: "AND",
	}
	result := CompileFilterModel(model, nil, &TenantDefaults{})
	expected := `"region" = 'APAC' AND "amount" > '1000'`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestCompileOrGroups(t *testing.T) {
	model := &FilterModel{
		Groups: []FilterGroup{
			{
				ID:         "g1",
				Combinator: "OR",
				Filters: []Filter{
					{ID: "f1", Field: "status", Operator: OpEquals, Enabled: true, ValueSource: ValueSource{Kind: ValueSourceConstant, Value: "Active"}},
					{ID: "f2", Field: "status", Operator: OpEquals, Enabled: true, ValueSource: ValueSource{Kind: ValueSourceConstant, Value: "Pending"}},
				},
			},
		},
		GroupCombinator: "OR",
	}
	result := CompileFilterModel(model, nil, &TenantDefaults{})
	expected := `"status" = 'Active' OR "status" = 'Pending'`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestCompileMultipleGroupsWithAndBetween(t *testing.T) {
	model := &FilterModel{
		Groups: []FilterGroup{
			{
				ID:         "g1",
				Combinator: "OR",
				Filters: []Filter{
					{ID: "f1", Field: "region", Operator: OpEquals, Enabled: true, ValueSource: ValueSource{Kind: ValueSourceConstant, Value: "APAC"}},
					{ID: "f2", Field: "region", Operator: OpEquals, Enabled: true, ValueSource: ValueSource{Kind: ValueSourceConstant, Value: "EMEA"}},
				},
			},
			{
				ID:         "g2",
				Combinator: "AND",
				Filters: []Filter{
					{ID: "f3", Field: "amount", Operator: OpGreaterThan, Enabled: true, ValueSource: ValueSource{Kind: ValueSourceConstant, Value: "5000"}},
					{ID: "f4", Field: "trade_date", Operator: OpToday, Enabled: true, ValueSource: ValueSource{Kind: ValueSourceConstant}},
				},
			},
		},
		GroupCombinator: "AND",
	}
	result := CompileFilterModel(model, nil, &TenantDefaults{})
	// Order depends on iteration of map; just check both groups are present
	expectedG1 := `("region" = 'APAC' OR "region" = 'EMEA')`
	expectedG2 := `("amount" > '5000' AND "trade_date" = CURRENT_DATE)`
	if !containsString(result, expectedG1) {
		t.Errorf("expected group 1 clause %q in result %q", expectedG1, result)
	}
	if !containsString(result, expectedG2) {
		t.Errorf("expected group 2 clause %q in result %q", expectedG2, result)
	}
}

func TestCompileFunctionExpression(t *testing.T) {
	model := &FilterModel{
		Groups: []FilterGroup{
			{ID: "g1", Combinator: "AND", Filters: []Filter{
				{ID: "f1", Field: "settle_date", Operator: OpEquals, Enabled: true, ValueSource: ValueSource{Kind: ValueSourceFunction, Expression: "DATESINPERIOD('trade_date', NOW(), -30, DAY)"}},
			}},
		},
	}
	result := CompileFilterModel(model, nil, &TenantDefaults{})
	expected := `"settle_date" = DATESINPERIOD('trade_date', NOW(), -30, DAY)`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestCompileFallbackCalendar(t *testing.T) {
	model := &FilterModel{
		Groups: []FilterGroup{
			{ID: "g1", Combinator: "AND", Filters: []Filter{
				{ID: "f1", Field: "trade_date", Operator: OpNextBusinessDay, Enabled: true, ValueSource: ValueSource{Kind: ValueSourceTenantDefault}}, // no defaults set
			}},
		},
	}
	// Empty defaults should fallback to "US"
	result := CompileFilterModel(model, nil, &TenantDefaults{})
	expected := `"trade_date" = calendar_next_business_day("trade_date", 'US')`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestCompileDottedFieldName(t *testing.T) {
	model := &FilterModel{
		Groups: []FilterGroup{
			{ID: "g1", Combinator: "AND", Filters: []Filter{
				{ID: "f1", Field: "t0.trade_date", Operator: OpEquals, Enabled: true, ValueSource: ValueSource{Kind: ValueSourceConstant, Value: "2026-08-01"}},
			}},
		},
	}
	result := CompileFilterModel(model, nil, &TenantDefaults{})
	expected := `"t0"."trade_date" = '2026-08-01'`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestCompilePreviousOffset(t *testing.T) {
	model := &FilterModel{
		Groups: []FilterGroup{
			{ID: "g1", Combinator: "AND", Filters: []Filter{
				{ID: "f1", Field: "period_date", Operator: OpPrevious, Enabled: true, Values: []string{"quarter"}, ValueSource: ValueSource{Kind: ValueSourceConstant}},
			}},
		},
	}
	result := CompileFilterModel(model, nil, &TenantDefaults{})
	expected := `"period_date" = date_trunc('quarter', CURRENT_DATE) - INTERVAL '3 months'`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestCompileNextOffset(t *testing.T) {
	model := &FilterModel{
		Groups: []FilterGroup{
			{ID: "g1", Combinator: "AND", Filters: []Filter{
				{ID: "f1", Field: "period_date", Operator: OpNext, Enabled: true, Values: []string{"year"}, ValueSource: ValueSource{Kind: ValueSourceConstant}},
			}},
		},
	}
	result := CompileFilterModel(model, nil, &TenantDefaults{})
	expected := `"period_date" = date_trunc('year', CURRENT_DATE) + INTERVAL '1 year'`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func containsString(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (haystack == needle || len(haystack) > 0 && (haystack[:len(needle)] == needle || containsString(haystack[1:], needle)))
}

func TestFilterSQLCompiler_FunctionWrapRelativeDatesAndParameters(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	macroResolver := NewCalendarMacroResolver(nil)
	compiler := NewFilterSQLCompiler(macroResolver)

	execCtx := FilterExecutionContext{
		TenantID: tenantID,
		Parameters: map[string]interface{}{
			"TargetPrefix": "101",
			"MinThreshold": 1000000.0,
		},
		SessionVariables: map[string]interface{}{
			"TenantID": tenantID.String(),
			"UserID":   "usr_compliance_officer_01",
		},
		ExecutionTime:    time.Date(2026, 8, 23, 10, 0, 0, 0, time.UTC),
		ExchangeCalendar: "NYSE",
	}

	// Construct AST:
	// GROUP (AND):
	//   1. SUBSTR(account_bk, 1, 3) = @TargetPrefix
	//   2. trade_date >= PREVIOUS_BUSINESS_DAY('NYSE')
	//   3. total_nav > @MinThreshold
	astRoot := FilterExpressionNode{
		Type:       NodeGroup,
		Combinator: CombinatorAnd,
		IsEnabled:  true,
		Children: []FilterExpressionNode{
			{
				Type:      NodeComparison,
				Operator:  CmpOpEquals,
				IsEnabled: true,
				LeftExpression: &FilterExpressionNode{
					Type:         NodeFunction,
					FunctionName: "SUBSTR",
					FunctionArgs: []FilterExpressionNode{
						{Type: NodeField, FieldKey: "account_bk"},
						{Type: NodeLiteral, LiteralValue: 1},
						{Type: NodeLiteral, LiteralValue: 3},
					},
				},
				RightExpression: &FilterExpressionNode{
					Type:     NodeParameter,
					ParamKey: "@TargetPrefix",
				},
			},
			{
				Type:      NodeComparison,
				Operator:  CmpOpGreaterThanOrEquals,
				IsEnabled: true,
				LeftExpression: &FilterExpressionNode{
					Type:     NodeField,
					FieldKey: "trade_date",
				},
				RightExpression: &FilterExpressionNode{
					Type:         NodeFunction,
					FunctionName: "PREVIOUS_BUSINESS_DAY",
				},
			},
			{
				Type:      NodeComparison,
				Operator:  CmpOpGreaterThan,
				IsEnabled: true,
				LeftExpression: &FilterExpressionNode{
					Type:     NodeField,
					FieldKey: "total_nav",
				},
				RightExpression: &FilterExpressionNode{
					Type:     NodeParameter,
					ParamKey: "@MinThreshold",
				},
			},
		},
	}

	result, err := compiler.CompileFilterAST(ctx, astRoot, execCtx)
	if err != nil {
		t.Fatalf("filter compilation failed: %v", err)
	}

	// Verify Function Pushdown
	if !strings.Contains(result.SQLWhereClause, "SUBSTR(account_bk, $1, $2) = $3") {
		t.Errorf("expected SUBSTR function pushdown in SQL, got: %s", result.SQLWhereClause)
	}

	// Verify Bind Arguments (1, 3, "101", "2026-08-21", 1000000.0)
	if len(result.BindArguments) != 5 {
		t.Fatalf("expected 5 bind arguments, got %d: %v", len(result.BindArguments), result.BindArguments)
	}

	if result.BindArguments[2] != "101" {
		t.Errorf("expected parameter @TargetPrefix resolved to '101', got: %v", result.BindArguments[2])
	}

	// 2026-08-23 was a Sunday; Previous business day was Friday 2026-08-21
	expectedDateStr := "2026-08-21"
	if result.BindArguments[3] != expectedDateStr {
		t.Errorf("expected relative date resolved to %s, got: %v", expectedDateStr, result.BindArguments[3])
	}
}
