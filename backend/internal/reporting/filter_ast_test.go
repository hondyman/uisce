package reporting

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─────────────────────────────────────────────────────────────────────────────
// Macro Resolver Tests
// ─────────────────────────────────────────────────────────────────────────────

func TestMacroResolver_DateMacros(t *testing.T) {
	now := time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)
	ctx := &ExecutionContext{Now: now, TenantID: "tenant-abc"}
	r := NewMacroResolver(ctx)

	tests := []struct {
		macro    string
		expected string
	}{
		{"TODAY", "2026-08-23"},
		{"YESTERDAY", "2026-08-22"},
		{"TOMORROW", "2026-08-24"},
		{"MTD", "2026-08-01"},
		{"YTD", "2026-01-01"},
		{"QTD", "2026-07-01"},   // Q3 starts July
		{"START_OF_MONTH", "2026-08-01"},
		{"END_OF_MONTH", "2026-08-31"},
		{"START_OF_YEAR", "2026-01-01"},
		{"END_OF_YEAR", "2026-12-31"},
	}
	for _, tc := range tests {
		t.Run(tc.macro, func(t *testing.T) {
			node := MacroNode(tc.macro)
			result := r.Resolve(node)
			require.NotNil(t, result.Node)
			assert.Equal(t, ExprLiteral, result.Node.Kind)
			require.NotNil(t, result.Node.Literal)
			require.NotNil(t, result.Node.Literal.StrVal)
			assert.Equal(t, tc.expected, *result.Node.Literal.StrVal)
		})
	}
}

func TestMacroResolver_LastNDays(t *testing.T) {
	now := time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)
	ctx := &ExecutionContext{Now: now, TenantID: "t1"}
	r := NewMacroResolver(ctx)

	node := MacroNode("LAST_N_DAYS", LiteralNum(30))
	result := r.Resolve(node)
	require.NotNil(t, result.Node.Literal.StrVal)
	assert.Equal(t, "2026-07-24", *result.Node.Literal.StrVal)
}

func TestMacroResolver_TMinusBusinessDays(t *testing.T) {
	now := time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC) // Sunday
	ctx := &ExecutionContext{Now: now, TenantID: "t1"}
	r := NewMacroResolver(ctx)

	node := MacroNode("T_MINUS", LiteralNum(2))
	result := r.Resolve(node)
	require.NotNil(t, result.Node.Literal.StrVal)
	// Sunday - 2 business days = Thursday 2026-08-20
	assert.Equal(t, "2026-08-20", *result.Node.Literal.StrVal)
}

func TestMacroResolver_SessionVariables_ProducesBindVar(t *testing.T) {
	ctx := &ExecutionContext{
		Now:          time.Now().UTC(),
		TenantID:     "tenant-xyz",
		UserID:       "user-42",
		UserRoles:    []string{"analyst", "viewer"},
		AllowedDesks: []string{"EMEA", "APAC"},
	}
	r := NewMacroResolver(ctx)

	// @Session.TenantID → bind variable $1 with value "tenant-xyz"
	node := SessionNode("TenantID")
	result := r.Resolve(node)
	require.NotNil(t, result.Node)
	assert.Equal(t, ExprParam, result.Node.Kind)
	assert.Equal(t, "$1", result.Node.ParamRef.ParamName)
	require.Len(t, result.Bindings, 1)
	assert.Equal(t, "tenant-xyz", result.Bindings[0].Value)
	assert.Equal(t, "@Session.TenantID", result.Bindings[0].Name)
}

func TestMacroResolver_SessionVariable_NeverLiteral(t *testing.T) {
	// SECURITY: session variables must NEVER produce a LiteralExpr node —
	// they must always be bound variables to prevent SQL injection.
	ctx := &ExecutionContext{Now: time.Now().UTC(), TenantID: "evil'; DROP TABLE tenants; --"}
	r := NewMacroResolver(ctx)

	node := SessionNode("TenantID")
	result := r.Resolve(node)
	assert.NotEqual(t, ExprLiteral, result.Node.Kind,
		"session variable must produce a bind var, not a SQL literal")
	assert.Equal(t, ExprParam, result.Node.Kind)
}

func TestMacroResolver_BitemporalAsOf(t *testing.T) {
	asOf := time.Date(2026, 6, 30, 23, 59, 59, 0, time.UTC)
	ctx := &ExecutionContext{Now: time.Now().UTC(), TenantID: "t1", AsOfDate: asOf}
	r := NewMacroResolver(ctx)

	node := &ExprNode{Kind: ExprBitemporal, BitemporalMacro: BitemporalAsOf}
	result := r.Resolve(node)
	assert.Equal(t, ExprParam, result.Node.Kind, "as-of date must be a bind var")
	require.Len(t, result.Bindings, 1)
	assert.Contains(t, result.Bindings[0].Value.(string), "2026-06-30")
}

func TestMacroResolver_ReportParameter(t *testing.T) {
	ctx := &ExecutionContext{
		Now:      time.Now().UTC(),
		TenantID: "t1",
		ReportParameterValues: map[string]interface{}{
			"@MinNAV": 1000000.0,
		},
	}
	r := NewMacroResolver(ctx)

	node := ParamNode("p1", "@MinNAV", "number")
	result := r.Resolve(node)
	assert.Equal(t, ExprParam, result.Node.Kind)
	require.Len(t, result.Bindings, 1)
	assert.Equal(t, 1000000.0, result.Bindings[0].Value)
}

// ─────────────────────────────────────────────────────────────────────────────
// Pushdown Optimizer Tests
// ─────────────────────────────────────────────────────────────────────────────

func TestPushdownOptimizer_SimpleWhere(t *testing.T) {
	opt := NewPushdownOptimizer(DialectPostgres)
	model := &ExpressionFilterModel{
		GroupCombinator: "AND",
		Groups: []ExpressionFilterGroup{{
			ID:         "g1",
			Combinator: "AND",
			Category:   FilterCategoryWhere,
			ExprFilters: []ExpressionFilter{{
				ID:      "f1",
				Enabled: true,
				Predicate: BinaryNode("=",
					FuncNode("SUBSTR", FieldNode("isin"), LiteralNum(1), LiteralNum(2)),
					LiteralStr("US"),
				),
			}},
		}},
	}
	plan := opt.Optimize(model, nil)
	require.Len(t, plan.WherePredicates, 1)
	assert.Contains(t, plan.WherePredicates[0], "SUBSTR")
	assert.Contains(t, plan.WherePredicates[0], "'US'")
	assert.Empty(t, plan.HybridFilters)
}

func TestPushdownOptimizer_NonPushableUDF(t *testing.T) {
	opt := NewPushdownOptimizer(DialectPostgres)
	model := &ExpressionFilterModel{
		GroupCombinator: "AND",
		Groups: []ExpressionFilterGroup{{
			ID:         "g1",
			Combinator: "AND",
			Category:   FilterCategoryWhere,
			ExprFilters: []ExpressionFilter{{
				ID:      "f1",
				Enabled: true,
				Predicate: BinaryNode(">",
					FuncNode("XIRR", FieldNode("cashflows")),
					LiteralNum(0.12),
				),
			}},
		}},
	}
	plan := opt.Optimize(model, nil)
	assert.Empty(t, plan.WherePredicates)
	require.Len(t, plan.HybridFilters, 1)
}

func TestPushdownOptimizer_HavingAggregate(t *testing.T) {
	opt := NewPushdownOptimizer(DialectPostgres)
	model := &ExpressionFilterModel{
		GroupCombinator: "AND",
		Groups: []ExpressionFilterGroup{{
			ID:         "g1",
			Combinator: "AND",
			Category:   FilterCategoryHaving,
			ExprFilters: []ExpressionFilter{{
				ID:      "f1",
				Enabled: true,
				Predicate: BinaryNode(">",
					AggNode("SUM", FieldNode("notional_amount")),
					LiteralNum(10_000_000),
				),
			}},
		}},
	}
	plan := opt.Optimize(model, nil)
	assert.Empty(t, plan.WherePredicates)
	require.Len(t, plan.HavingPredicates, 1)
	assert.Contains(t, plan.HavingPredicates[0], "SUM")
	assert.Contains(t, plan.HavingPredicates[0], "10000000")
}

func TestPushdownOptimizer_QualifyWindowPostgres(t *testing.T) {
	opt := NewPushdownOptimizer(DialectPostgres)
	rnNode := &ExprNode{
		Kind:       ExprWindow,
		WindowFunc: "ROW_NUMBER",
		WindowSpec: &WindowSpec{
			PartitionBy: []*ExprNode{FieldNode("order_id")},
			OrderBy:     []*OrderExpr{{Expr: FieldNode("event_time"), Desc: true}},
		},
	}
	model := &ExpressionFilterModel{
		GroupCombinator: "AND",
		Groups: []ExpressionFilterGroup{{
			ID:         "g1",
			Combinator: "AND",
			Category:   FilterCategoryQualify,
			ExprFilters: []ExpressionFilter{{
				ID:        "f1",
				Enabled:   true,
				Predicate: BinaryNode("=", rnNode, LiteralNum(1)),
			}},
		}},
	}
	plan := opt.Optimize(model, nil)
	require.Len(t, plan.QualifyPredicates, 1)
	assert.Contains(t, plan.QualifyPredicates[0], "ROW_NUMBER")
	// Postgres: CTE added
	require.NotEmpty(t, plan.RequiredCTEs)
}

func TestPushdownOptimizer_QualifyStarRocksNative(t *testing.T) {
	opt := NewPushdownOptimizer(DialectStarRocks)
	rnNode := &ExprNode{
		Kind:       ExprWindow,
		WindowFunc: "ROW_NUMBER",
		WindowSpec: &WindowSpec{
			PartitionBy: []*ExprNode{FieldNode("order_id")},
			OrderBy:     []*OrderExpr{{Expr: FieldNode("event_time"), Desc: true}},
		},
	}
	model := &ExpressionFilterModel{
		GroupCombinator: "AND",
		Groups: []ExpressionFilterGroup{{
			ID:         "g1",
			Combinator: "AND",
			Category:   FilterCategoryQualify,
			ExprFilters: []ExpressionFilter{{
				ID:        "f1",
				Enabled:   true,
				Predicate: BinaryNode("=", rnNode, LiteralNum(1)),
			}},
		}},
	}
	plan := opt.Optimize(model, nil)
	// StarRocks uses native QUALIFY
	qualifySQL := plan.QualifySQL(DialectStarRocks)
	assert.Contains(t, qualifySQL, "QUALIFY")
	// No CTE required
	assert.Empty(t, plan.RequiredCTEs)
}

// ─────────────────────────────────────────────────────────────────────────────
// Dialect SQL Serializer Tests
// ─────────────────────────────────────────────────────────────────────────────

func TestDialectSQLSerializer_ScalarFunctions(t *testing.T) {
	pg := NewDialectSQLSerializer(DialectPostgres)
	sr := NewDialectSQLSerializer(DialectStarRocks)
	tri := NewDialectSQLSerializer(DialectTrino)

	tests := []struct {
		name     string
		node     *ExprNode
		pgSQL    string
		srSQL    string
		trinoSQL string
	}{
		{
			name:     "SUBSTR",
			node:     FuncNode("SUBSTR", FieldNode("isin"), LiteralNum(1), LiteralNum(2)),
			pgSQL:    `SUBSTR("isin", 1, 2)`,
			srSQL:    `SUBSTR("isin", 1, 2)`,
			trinoSQL: `SUBSTR("isin", 1, 2)`,
		},
		{
			name:     "UPPER",
			node:     FuncNode("UPPER", FieldNode("city")),
			pgSQL:    `UPPER("city")`,
			srSQL:    `UPPER("city")`,
			trinoSQL: `UPPER("city")`,
		},
		{
			name:     "REGEXP_LIKE Postgres",
			node:     FuncNode("REGEXP_LIKE", FieldNode("cusip"), LiteralStr("^[0-9]")),
			pgSQL:    `("cusip" ~ '^[0-9]')`,
			srSQL:    `REGEXP("cusip", '^[0-9]')`,
			trinoSQL: `REGEXP_LIKE("cusip", '^[0-9]')`,
		},
		{
			name:     "JSON_EXTRACT Postgres",
			node:     FuncNode("JSON_EXTRACT_SCALAR", FieldNode("custom_fields"), LiteralStr("$.tier")),
			pgSQL:    `"custom_fields"->>'tier'`,
			srSQL:    `JSON_EXTRACT_SCALAR("custom_fields", '$.tier')`,
			trinoSQL: `JSON_EXTRACT_SCALAR("custom_fields", '$.tier')`,
		},
		{
			name:     "ARRAY_CONTAINS Postgres",
			node:     FuncNode("ARRAY_CONTAINS", FieldNode("tags"), LiteralStr("VIP")),
			pgSQL:    `"tags" @> ARRAY['VIP']`,
			srSQL:    `ARRAY_CONTAINS("tags", 'VIP')`,
			trinoSQL: `CONTAINS("tags", 'VIP')`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.pgSQL, pg.Serialize(tc.node), "postgres")
			assert.Equal(t, tc.srSQL, sr.Serialize(tc.node), "starrocks")
			assert.Equal(t, tc.trinoSQL, tri.Serialize(tc.node), "trino")
		})
	}
}

func TestDialectSQLSerializer_CrossFieldArithmetic(t *testing.T) {
	pg := NewDialectSQLSerializer(DialectPostgres)
	// WHERE (unrealized_pnl / market_value) > 0.05
	node := BinaryNode(">",
		BinaryNode("/", FieldNode("unrealized_pnl"), FieldNode("market_value")),
		LiteralNum(0.05),
	)
	sql := pg.Serialize(node)
	assert.Contains(t, sql, `"unrealized_pnl"`)
	assert.Contains(t, sql, `"market_value"`)
	assert.Contains(t, sql, "0.05")
	assert.Contains(t, sql, "/")
}

func TestDialectSQLSerializer_CaseWhen(t *testing.T) {
	pg := NewDialectSQLSerializer(DialectPostgres)
	node := &ExprNode{
		Kind: ExprCase,
		CaseWhen: []*WhenClause{
			{When: BinaryNode(">", FieldNode("pnl"), LiteralNum(0)), Then: LiteralStr("GAIN")},
			{When: BinaryNode("<", FieldNode("pnl"), LiteralNum(0)), Then: LiteralStr("LOSS")},
		},
		CaseElse: LiteralStr("FLAT"),
	}
	sql := pg.Serialize(node)
	assert.Contains(t, sql, "CASE WHEN")
	assert.Contains(t, sql, "THEN 'GAIN'")
	assert.Contains(t, sql, "THEN 'LOSS'")
	assert.Contains(t, sql, "ELSE 'FLAT'")
	assert.Contains(t, sql, "END")
}

func TestDialectSQLSerializer_ExistsSubquery(t *testing.T) {
	pg := NewDialectSQLSerializer(DialectPostgres)
	node := &ExprNode{
		Kind: ExprSubquery,
		Subquery: &SubquerySpec{
			Operator:    "EXISTS",
			TargetTable: "oms.trade_order",
			Correlations: []CorrelationPair{
				{OuterField: "portfolio_id", InnerField: "portfolio_id"},
			},
			InnerFilters: []*ExprNode{
				BinaryNode("=", FieldNode("status"), LiteralStr("PENDING")),
			},
		},
	}
	sql := pg.Serialize(node)
	assert.Contains(t, sql, "EXISTS")
	assert.Contains(t, sql, `"oms"."trade_order"`)
	assert.Contains(t, sql, "portfolio_id")
	assert.Contains(t, sql, "'PENDING'")
}

func TestDialectSQLSerializer_WindowRowNumber(t *testing.T) {
	pg := NewDialectSQLSerializer(DialectPostgres)
	node := &ExprNode{
		Kind:       ExprWindow,
		WindowFunc: "ROW_NUMBER",
		WindowSpec: &WindowSpec{
			PartitionBy: []*ExprNode{FieldNode("order_id")},
			OrderBy:     []*OrderExpr{{Expr: FieldNode("event_time"), Desc: true}},
		},
	}
	sql := pg.Serialize(node)
	assert.Contains(t, sql, "ROW_NUMBER()")
	assert.Contains(t, sql, "PARTITION BY")
	assert.Contains(t, sql, "ORDER BY")
	assert.Contains(t, sql, "DESC")
}

func TestDialectSQLSerializer_Coalesce(t *testing.T) {
	pg := NewDialectSQLSerializer(DialectPostgres)
	node := BinaryNode("!=",
		FuncNode("COALESCE", FieldNode("tax_id"), FieldNode("legal_entity_id"), LiteralStr("UNKNOWN")),
		LiteralStr("UNKNOWN"),
	)
	sql := pg.Serialize(node)
	assert.Contains(t, sql, "COALESCE")
	assert.Contains(t, sql, "'UNKNOWN'")
}

// ─────────────────────────────────────────────────────────────────────────────
// Hybrid CEL Evaluator Tests
// ─────────────────────────────────────────────────────────────────────────────

func TestHybridEvaluator_SimpleEquality(t *testing.T) {
	eval, err := NewHybridEvaluator([]string{"status", "amount"})
	require.NoError(t, err)

	pred, err := eval.CompileExpression(BinaryNode("=", FieldNode("status"), LiteralStr("ACTIVE")))
	require.NoError(t, err)

	ctx := &ExecutionContext{Now: time.Now().UTC(), TenantID: "t1"}
	passRow := RowMap{"status": "ACTIVE", "amount": 100.0}
	failRow := RowMap{"status": "INACTIVE", "amount": 100.0}

	ok, err := eval.Evaluate(pred, passRow, ctx)
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = eval.Evaluate(pred, failRow, ctx)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestHybridEvaluator_ArithmeticComparison(t *testing.T) {
	eval, err := NewHybridEvaluator([]string{"unrealized_pnl", "market_value"})
	require.NoError(t, err)

	// (unrealized_pnl / market_value) > 0.05
	pred, err := eval.CompileExpression(BinaryNode(">",
		BinaryNode("/", FieldNode("unrealized_pnl"), FieldNode("market_value")),
		LiteralNum(0.05),
	))
	require.NoError(t, err)

	ctx := &ExecutionContext{Now: time.Now().UTC(), TenantID: "t1"}
	above := RowMap{"unrealized_pnl": 60.0, "market_value": 1000.0}  // 6% > 5%
	below := RowMap{"unrealized_pnl": 40.0, "market_value": 1000.0}  // 4% < 5%

	ok, err := eval.Evaluate(pred, above, ctx)
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = eval.Evaluate(pred, below, ctx)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestHybridEvaluator_StringContains(t *testing.T) {
	eval, err := NewHybridEvaluator([]string{"account_name"})
	require.NoError(t, err)

	// LOWER(TRIM(account_name)) contains 'boston'
	pred, err := eval.CompileExpression(
		BinaryNode("=",
			FuncNode("LOWER", FuncNode("TRIM", FieldNode("account_name"))),
			LiteralStr("boston fund"),
		),
	)
	require.NoError(t, err)
	ctx := &ExecutionContext{Now: time.Now().UTC(), TenantID: "t1"}

	match := RowMap{"account_name": "  Boston Fund  "}
	ok, err := eval.Evaluate(pred, match, ctx)
	// CEL lower/trim normalisation
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestHybridEvaluator_FilterRows(t *testing.T) {
	eval, err := NewHybridEvaluator([]string{"region", "amount"})
	require.NoError(t, err)

	pred, err := eval.CompileExpression(BinaryNode("=", FieldNode("region"), LiteralStr("EMEA")))
	require.NoError(t, err)

	rows := []RowMap{
		{"region": "EMEA", "amount": 1000.0},
		{"region": "APAC", "amount": 2000.0},
		{"region": "EMEA", "amount": 3000.0},
	}
	ctx := &ExecutionContext{Now: time.Now().UTC(), TenantID: "t1"}
	result, err := eval.FilterRows([]*CompiledPredicate{pred}, rows, ctx)
	require.NoError(t, err)
	assert.Len(t, result, 2)
}

// ─────────────────────────────────────────────────────────────────────────────
// ABAC Injector Tests
// ─────────────────────────────────────────────────────────────────────────────

func TestABACInjector_PrependsTenantIsolation(t *testing.T) {
	ctx := &ExecutionContext{Now: time.Now().UTC(), TenantID: "tenant-secure"}
	r := NewMacroResolver(ctx)

	model := &ExpressionFilterModel{GroupCombinator: "AND"}
	injected := InjectABACPredicates(model, ctx, r)

	require.NotEmpty(t, injected.Groups)
	abacGroup := injected.Groups[0]
	assert.Equal(t, "abac_mandatory", abacGroup.ID)
	require.NotEmpty(t, abacGroup.ExprFilters)
	tenantFilter := abacGroup.ExprFilters[0]
	assert.Equal(t, "abac_tenant_isolation", tenantFilter.ID)
	// Predicate must be a binary = on tenant_id
	require.NotNil(t, tenantFilter.Predicate)
	assert.Equal(t, ExprBinaryOp, tenantFilter.Predicate.Kind)
	assert.Equal(t, "=", tenantFilter.Predicate.Op)
	// Left side must reference tenant_id field
	require.NotNil(t, tenantFilter.Predicate.Left)
	assert.Equal(t, ExprField, tenantFilter.Predicate.Left.Kind)
	assert.Equal(t, "tenant_id", tenantFilter.Predicate.Left.FieldRef.Column)
	// Right side must be a bind var (NOT a literal containing the tenant ID string)
	require.NotNil(t, tenantFilter.Predicate.Right)
	assert.Equal(t, ExprParam, tenantFilter.Predicate.Right.Kind,
		"tenant_id must be a bind variable, not a literal — prevents SQL injection")
}

func TestABACInjector_DeskFilterWhenPresent(t *testing.T) {
	ctx := &ExecutionContext{
		Now:          time.Now().UTC(),
		TenantID:     "t1",
		AllowedDesks: []string{"EMEA", "APAC"},
	}
	r := NewMacroResolver(ctx)
	model := &ExpressionFilterModel{GroupCombinator: "AND"}
	injected := InjectABACPredicates(model, ctx, r)

	abacGroup := injected.Groups[0]
	require.Len(t, abacGroup.ExprFilters, 2)
	deskFilter := abacGroup.ExprFilters[1]
	assert.Equal(t, "abac_desk_filter", deskFilter.ID)
	assert.Equal(t, ExprBinaryOp, deskFilter.Predicate.Kind)
	assert.Equal(t, "IN", deskFilter.Predicate.Op)
}

// ─────────────────────────────────────────────────────────────────────────────
// Bitemporal Predicate Builder Tests
// ─────────────────────────────────────────────────────────────────────────────

func TestBitemporalPredicateBuilder_Postgres(t *testing.T) {
	b := &BitemporalPredicateBuilder{Dialect: DialectPostgres}
	pred := b.BuildSystemPeriodPredicate("$1", "system_valid_from", "system_valid_to")
	assert.Equal(t, "system_valid_from <= $1 AND system_valid_to > $1", pred)
}

func TestBitemporalPredicateBuilder_Trino_IsHint(t *testing.T) {
	b := &BitemporalPredicateBuilder{Dialect: DialectTrino}
	pred := b.BuildSystemPeriodPredicate("$1", "system_valid_from", "system_valid_to")
	assert.Contains(t, pred, "FOR SYSTEM_TIME AS OF")
}

// ─────────────────────────────────────────────────────────────────────────────
// End-to-end pipeline test: Expression → MacroResolver → PushdownOptimizer → SQL
// ─────────────────────────────────────────────────────────────────────────────

func TestPipeline_SubstrParamFilter(t *testing.T) {
	// SUBSTR(account_number, 1, @PrefixLength) = @TargetPrefix
	ctx := &ExecutionContext{
		Now:      time.Date(2026, 8, 23, 0, 0, 0, 0, time.UTC),
		TenantID: "t1",
		ReportParameterValues: map[string]interface{}{
			"@PrefixLength": 3.0,
			"@TargetPrefix": "US-",
		},
	}
	r := NewMacroResolver(ctx)

	rawPred := BinaryNode("=",
		FuncNode("SUBSTR",
			FieldNode("account_number"),
			LiteralNum(1),
			ParamNode("p_len", "@PrefixLength", "number"),
		),
		ParamNode("p_pfx", "@TargetPrefix", "string"),
	)

	resolved := r.Resolve(rawPred)
	require.NotNil(t, resolved.Node)
	assert.Len(t, resolved.Bindings, 2)

	// Bindings: $1 = @PrefixLength = 3, $2 = @TargetPrefix = "US-"
	assert.Equal(t, 3.0, resolved.Bindings[0].Value)
	assert.Equal(t, "US-", resolved.Bindings[1].Value)

	opt := NewPushdownOptimizer(DialectPostgres)
	model := &ExpressionFilterModel{
		GroupCombinator: "AND",
		Groups: []ExpressionFilterGroup{{
			ID:         "g1",
			Combinator: "AND",
			Category:   FilterCategoryWhere,
			ExprFilters: []ExpressionFilter{{
				ID: "f1", Enabled: true, Predicate: resolved.Node,
			}},
		}},
	}
	plan := opt.Optimize(model, []interface{}{3.0, "US-"})
	require.Len(t, plan.WherePredicates, 1)
	assert.Contains(t, plan.WherePredicates[0], "SUBSTR")
	assert.Contains(t, plan.WherePredicates[0], "$1")
	assert.Contains(t, plan.WherePredicates[0], "$2")
	assert.Empty(t, plan.HybridFilters)
}

func TestPipeline_HybridSplitExecution(t *testing.T) {
	// Pushable: trade_date >= LAST_N_DAYS(30)   → WHERE predicate
	// Non-pushable: XIRR(cashflows) > 0.12     → hybrid evaluator
	ctx := &ExecutionContext{Now: time.Date(2026, 8, 23, 0, 0, 0, 0, time.UTC), TenantID: "t1"}
	r := NewMacroResolver(ctx)

	dateNode := r.Resolve(BinaryNode(">=", FieldNode("trade_date"), MacroNode("LAST_N_DAYS", LiteralNum(30))))
	xirrNode := BinaryNode(">", FuncNode("XIRR", FieldNode("cashflows")), LiteralNum(0.12))

	opt := NewPushdownOptimizer(DialectPostgres)
	model := &ExpressionFilterModel{
		GroupCombinator: "AND",
		Groups: []ExpressionFilterGroup{
			{
				ID:         "g_date",
				Combinator: "AND",
				Category:   FilterCategoryWhere,
				ExprFilters: []ExpressionFilter{{ID: "f_date", Enabled: true, Predicate: dateNode.Node}},
			},
			{
				ID:         "g_xirr",
				Combinator: "AND",
				Category:   FilterCategoryWhere,
				ExprFilters: []ExpressionFilter{{ID: "f_xirr", Enabled: true, Predicate: xirrNode}},
			},
		},
	}
	plan := opt.Optimize(model, nil)

	// date filter → pushed to WHERE
	require.Len(t, plan.WherePredicates, 1)
	assert.Contains(t, plan.WherePredicates[0], "trade_date")

	// XIRR → hybrid (non-pushable UDF)
	require.Len(t, plan.HybridFilters, 1)
	assert.Equal(t, ExprBinaryOp, plan.HybridFilters[0].Kind)
	require.NotNil(t, plan.HybridFilters[0].Left)
	assert.Equal(t, "XIRR", plan.HybridFilters[0].Left.FuncName)
}
