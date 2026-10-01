package analytics

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/hondyman/uisce/backend/internal/rules/vm"
)

func TestSafeColumnAndTable(t *testing.T) {
	// Valid columns
	c, err := safeColumn("target_quantity")
	if err != nil || c != `"target_quantity"` {
		t.Fatalf("safeColumn target_quantity: %v, got %s", err, c)
	}

	// Invalid column (SQL injection attempt)
	_, err = safeColumn("col; DROP TABLE users;--")
	if err == nil {
		t.Fatal("expected error for SQL injection column, got nil")
	}

	// Valid table
	tbl, err := safeTable("public.trade_order")
	if err != nil || tbl != `"public"."trade_order"` {
		t.Fatalf("safeTable public.trade_order: %v, got %s", err, tbl)
	}

	// Invalid table (too many qualifiers)
	_, err = safeTable("a.b.c")
	if err == nil {
		t.Fatal("expected error for too many qualifiers, got nil")
	}
}

func TestCompileRuleForPushdown_TenantIsolation(t *testing.T) {
	ast := vm.RuleNode{
		Type: vm.NodeTypeCondition,
		Condition: &vm.RuleCondition{
			ID:       "cond1",
			Field:    "TargetQuantity",
			Operator: "greater_than",
			Value:    100.0,
		},
	}
	rule := &EvaluableRule{
		RuleID:      "r1",
		RuleKey:     "qty_check",
		RuleName:    "Quantity Check",
		RuleVersion: "1",
		BOName:      "trade_order",
		Severity:    models.ValidationRuleSeverityBlock,
		AST:         &ast,
		FieldRefs:   []string{"TargetQuantity"},
	}

	cols := map[string]string{
		"TargetQuantity": "target_quantity",
	}

	cr, err := CompileRuleForPushdown(rule, "oms.trade_order", "tenant-123", cols)
	if err != nil {
		t.Fatalf("CompileRuleForPushdown error: %v", err)
	}

	if !cr.tenantBound {
		t.Fatal("tenantBound must be true")
	}
	if len(cr.args) < 1 || cr.args[0] != "tenant-123" {
		t.Fatalf("first argument must be tenantID 'tenant-123', got %v", cr.args)
	}
	if cr.table != `"oms"."trade_order"` {
		t.Fatalf("unexpected table: %s", cr.table)
	}
}

type stubConnResolver struct {
	db *sqlmock.Sqlmock
}

func (s *stubConnResolver) DataPlaneConn(ctx context.Context, tenantID string) (*stdSQLDB, error) {
	return nil, nil
}

type stdSQLDB = struct{}

func TestPushdownExecution_CountsAndSampling(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	ast := vm.RuleNode{
		Type: vm.NodeTypeCondition,
		Condition: &vm.RuleCondition{
			ID:       "c1",
			Field:    "TargetQuantity",
			Operator: "greater_than",
			Value:    100.0,
		},
	}
	rule := &EvaluableRule{
		RuleID:      "r1",
		RuleKey:     "qty_check",
		RuleName:    "Quantity Check",
		RuleVersion: "1",
		BOName:      "trade_order",
		Severity:    models.ValidationRuleSeverityBlock,
		AST:         &ast,
		FieldRefs:   []string{"TargetQuantity"},
	}

	cr, err := CompileRuleForPushdown(rule, "trade_order", "tenant-123", map[string]string{
		"TargetQuantity": "target_quantity",
	})
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	// Mock counts query
	mock.ExpectQuery(`SELECT COUNT\(\*\) AS total`).
		WithArgs("tenant-123", 100.0).
		WillReturnRows(sqlmock.NewRows([]string{"total", "violations", "rule_errors"}).AddRow(1000, 5, 0))

	counts, err := runPushdownCounts(context.Background(), db, cr)
	if err != nil {
		t.Fatalf("runPushdownCounts error: %v", err)
	}
	if counts[0] != 1000 || counts[1] != 5 || counts[2] != 0 {
		t.Fatalf("unexpected counts: %v", counts)
	}

	// Mock detail query
	mock.ExpectQuery(`SELECT "id" FROM "trade_order"`).
		WithArgs("tenant-123", 100.0, 10).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("rec-1").AddRow("rec-2"))

	ids, err := runPushdownDetail(context.Background(), db, cr, `"id"`, 10)
	if err != nil {
		t.Fatalf("runPushdownDetail error: %v", err)
	}
	if len(ids) != 2 || ids[0] != "rec-1" || ids[1] != "rec-2" {
		t.Fatalf("unexpected detail ids: %v", ids)
	}
}
