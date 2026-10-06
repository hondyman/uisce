package engine

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/hondyman/uisce/backend/internal/rules"
	vm "github.com/hondyman/uisce/backend/internal/rules/vm"
)

func setupTestBundle(t *testing.T) (*RuleBundle, *vm.FastRecord, *vm.SymbolDict, *vm.EnumDict) {
	tenantID := uuid.New()
	bundle := NewRuleBundle(tenantID, "PRE_TRADE_CORE_BUNDLE", 1)
	compiler := NewBundleCompiler()

	syms := bundle.SymDict
	enums := bundle.EnumDict

	syms.Intern("order.quantity")
	syms.Intern("order.price")
	syms.Intern("position.projected_weight")
	syms.Intern("account.is_restricted")

	symQty, _ := syms.Resolve("order.quantity")
	symPrice, _ := syms.Resolve("order.price")
	symWeight, _ := syms.Resolve("position.projected_weight")
	symRestricted, _ := syms.Resolve("account.is_restricted")

	// Rule 1: HARD_BLOCK if account.is_restricted == true
	// AST: account.is_restricted == false (Rule passes if NOT restricted)
	ast1 := &rules.RuleNode{
		Type: rules.NodeTypeCondition,
		Condition: &rules.RuleCondition{
			FieldPath: "account.is_restricted",
			Operator:  "==",
			Value:     false,
		},
	}
	r1, err := compiler.CompileRule(uuid.New(), "R_RESTRICTED_ACCT", "Restricted Account Check", SeverityHardBlock, ast1, syms, enums, decimal.Zero)
	if err != nil {
		t.Fatalf("Failed to compile rule 1: %v", err)
	}
	bundle.AddCompiledRule(r1)

	// Rule 2: APPROVAL_REQUIRED if projected_weight > 0.05
	// AST: position.projected_weight <= 0.05 (Rule passes if <= 0.05)
	ast2 := &rules.RuleNode{
		Type: vm.NodeTypeExpression,
		Expression: &vm.Expression{
			Root: &vm.BinaryExpr{
				Op:    "<=",
				Left:  &vm.FieldRef{Path: "position.projected_weight"},
				Right: &vm.Literal{Value: 0.05},
			},
		},
	}
	r2, err := compiler.CompileRule(uuid.New(), "R_CONCENTRATION_5PCT", "5% Single Issuer Concentration", SeverityApprovalRequired, ast2, syms, enums, decimal.RequireFromString("0.05"))
	if err != nil {
		t.Fatalf("Failed to compile rule 2: %v", err)
	}
	bundle.AddCompiledRule(r2)

	// Rule 3: SOFT_WARNING if order.quantity > 10,000
	// AST: order.quantity <= 10000
	ast3 := &rules.RuleNode{
		Type: rules.NodeTypeCondition,
		Condition: &rules.RuleCondition{
			FieldPath: "order.quantity",
			Operator:  "<=",
			Value:     int64(10000),
			ValueType: "int",
		},
	}
	r3, err := compiler.CompileRule(uuid.New(), "R_LARGE_ORDER_QTY", "Large Order Warning", SeveritySoftWarning, ast3, syms, enums, decimal.NewFromInt(10000))
	if err != nil {
		t.Fatalf("Failed to compile rule 3: %v", err)
	}
	bundle.AddCompiledRule(r3)

	// Record builder
	rec := vm.GetFastRecord(syms)
	rec.BoolVals[symRestricted] = false
	rec.Present[symRestricted] |= vm.HasBool

	rec.FNumVals[symWeight] = 0.03 // 3%
	rec.Present[symWeight] |= vm.HasFNum

	rec.NumVals[symQty] = 5000
	rec.FNumVals[symQty] = 5000
	rec.Present[symQty] |= vm.HasNum | vm.HasFNum

	rec.NumVals[symPrice] = 100
	rec.FNumVals[symPrice] = 100
	rec.Present[symPrice] |= vm.HasNum | vm.HasFNum

	return bundle, rec, syms, enums
}

func TestFastBundleEvaluator_DecisionHierarchy(t *testing.T) {
	bundle, rec, syms, _ := setupTestBundle(t)
	defer vm.PutFastRecord(rec)
	evaluator := NewFastBundleEvaluator()

	tenantID := bundle.TenantID
	accountID := uuid.New()
	symRestricted, _ := syms.Resolve("account.is_restricted")
	symWeight, _ := syms.Resolve("position.projected_weight")
	symQty, _ := syms.Resolve("order.quantity")

	// Case 1: Clean order -> PASS
	outcome := evaluator.EvaluateBundle(context.Background(), uuid.New(), tenantID, accountID, bundle, rec)
	if outcome.Decision != StatusPass {
		t.Fatalf("Expected PASS, got %s", outcome.Decision)
	}
	if outcome.PassedRules != 3 || outcome.BreachedRules != 0 {
		t.Errorf("Expected 3 passed 0 breached, got %d passed %d breached", outcome.PassedRules, outcome.BreachedRules)
	}

	// Case 2: Large quantity -> SOFT_WARNING
	rec.NumVals[symQty] = 15000
	rec.FNumVals[symQty] = 15000
	outcome = evaluator.EvaluateBundle(context.Background(), uuid.New(), tenantID, accountID, bundle, rec)
	if outcome.Decision != StatusSoftWarning {
		t.Fatalf("Expected SOFT_WARNING, got %s", outcome.Decision)
	}
	if outcome.PassedRules != 2 || outcome.BreachedRules != 1 {
		t.Errorf("Expected 2 passed 1 breached, got %d passed %d breached", outcome.PassedRules, outcome.BreachedRules)
	}

	// Case 3: High concentration -> APPROVAL_REQUIRED (dominates SOFT_WARNING)
	rec.FNumVals[symWeight] = 0.08 // 8% > 5%
	outcome = evaluator.EvaluateBundle(context.Background(), uuid.New(), tenantID, accountID, bundle, rec)
	if outcome.Decision != StatusApprovalRequired {
		t.Fatalf("Expected APPROVAL_REQUIRED, got %s", outcome.Decision)
	}
	if outcome.PassedRules != 1 || outcome.BreachedRules != 2 {
		t.Errorf("Expected 1 passed 2 breached, got %d passed %d breached", outcome.PassedRules, outcome.BreachedRules)
	}

	// Case 4: Restricted Account -> HARD_BLOCK (dominates all)
	rec.BoolVals[symRestricted] = true
	outcome = evaluator.EvaluateBundle(context.Background(), uuid.New(), tenantID, accountID, bundle, rec)
	if outcome.Decision != StatusHardBlock {
		t.Fatalf("Expected HARD_BLOCK, got %s", outcome.Decision)
	}
	if outcome.PassedRules != 0 || outcome.BreachedRules != 3 {
		t.Errorf("Expected 0 passed 3 breached, got %d passed %d breached", outcome.PassedRules, outcome.BreachedRules)
	}
}

func BenchmarkFastBundleEvaluator_15Rules(b *testing.B) {
	tenantID := uuid.New()
	bundle := NewRuleBundle(tenantID, "PRE_TRADE_BENCH_BUNDLE", 1)
	compiler := NewBundleCompiler()

	syms := bundle.SymDict
	enums := bundle.EnumDict

	// Build a 15-rule bundle covering diverse checks
	for i := 0; i < 15; i++ {
		field := "metric." + string(rune('a'+i))
		syms.Intern(field)

		ast := &rules.RuleNode{
			Type: vm.NodeTypeExpression,
			Expression: &vm.Expression{
				Root: &vm.BinaryExpr{
					Op:    "<=",
					Left:  &vm.FieldRef{Path: field},
					Right: &vm.Literal{Value: 100.0},
				},
			},
		}

		r, err := compiler.CompileRule(uuid.New(), "R_BENCH_"+string(rune('A'+i)), "Bench Rule", SeverityHardBlock, ast, syms, enums, decimal.NewFromInt(100))
		if err != nil {
			b.Fatalf("compile bench rule: %v", err)
		}
		bundle.AddCompiledRule(r)
	}

	rec := vm.GetFastRecord(syms)
	defer vm.PutFastRecord(rec)
	for i := 0; i < 15; i++ {
		field := "metric." + string(rune('a'+i))
		symID, _ := syms.Resolve(field)
		rec.FNumVals[symID] = float64(i) * 1.5
		rec.Present[symID] |= vm.HasFNum
	}

	evaluator := NewFastBundleEvaluator()
	accountID := uuid.New()
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = evaluator.EvaluateBundle(ctx, uuid.Nil, tenantID, accountID, bundle, rec)
	}
}

func BenchmarkFastBundleEvaluator_15Rules_ZeroAlloc(b *testing.B) {
	tenantID := uuid.New()
	bundle := NewRuleBundle(tenantID, "PRE_TRADE_BENCH_BUNDLE", 1)
	compiler := NewBundleCompiler()

	syms := bundle.SymDict
	enums := bundle.EnumDict

	for i := 0; i < 15; i++ {
		field := "metric." + string(rune('a'+i))
		syms.Intern(field)

		ast := &rules.RuleNode{
			Type: vm.NodeTypeExpression,
			Expression: &vm.Expression{
				Root: &vm.BinaryExpr{
					Op:    "<=",
					Left:  &vm.FieldRef{Path: field},
					Right: &vm.Literal{Value: 100.0},
				},
			},
		}

		r, err := compiler.CompileRule(uuid.New(), "R_BENCH_"+string(rune('A'+i)), "Bench Rule", SeverityHardBlock, ast, syms, enums, decimal.NewFromInt(100))
		if err != nil {
			b.Fatalf("compile bench rule: %v", err)
		}
		bundle.AddCompiledRule(r)
	}

	rec := vm.GetFastRecord(syms)
	defer vm.PutFastRecord(rec)
	for i := 0; i < 15; i++ {
		field := "metric." + string(rune('a'+i))
		symID, _ := syms.Resolve(field)
		rec.FNumVals[symID] = float64(i) * 1.5
		rec.Present[symID] |= vm.HasFNum
	}

	evaluator := NewFastBundleEvaluator()
	var detailsBuf [16]RuleEvaluationDetail

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _, _ = evaluator.EvaluateBundleFast(bundle, rec, detailsBuf[:])
	}
}
