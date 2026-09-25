package boresolver

import (
	"strings"
	"testing"
)

// The cross-field form (valueFieldId) puts another column on the right-hand
// side, so its operator is SQL text: only plain comparisons may reach it.
func TestCompileFilterPredicate_CrossFieldRejectsNonComparison(t *testing.T) {
	for _, op := range []string{"1=1; --", "CONTAINS", "IN", "OR 1=1"} {
		g := &BOSQLGenerator{Dialect: PostgresDialect{}}
		sql, err := CompileFilterPredicate(g, &GenerationContext{}, "t0.a", FilterClause{Operator: op, ValueFieldID: "other"})
		if err == nil || !strings.Contains(err.Error(), "unsupported filter operator") {
			t.Errorf("op %q: got %q, %v; want unsupported operator error", op, sql, err)
		}
	}
}

func TestCompileFilterPredicate_ValuelessIgnoresComparisonField(t *testing.T) {
	g := &BOSQLGenerator{Dialect: PostgresDialect{}}
	sql, err := CompileFilterPredicate(g, &GenerationContext{}, "t0.a", FilterClause{Operator: "IS NULL", ValueFieldID: "other"})
	if err != nil || sql != "t0.a IS NULL" {
		t.Fatalf("got %q, %v", sql, err)
	}
}

func TestCompileFilterPredicate_BindsForDialect(t *testing.T) {
	g := &BOSQLGenerator{Dialect: SQLServerDialect{}}
	ctx := &GenerationContext{}
	sql, err := CompileFilterPredicate(g, ctx, "t0.a", FilterClause{Operator: "NOT_IN", Value: []interface{}{"x", "y"}})
	if err != nil || sql != "t0.a NOT IN (@p1, @p2)" || len(ctx.Args) != 2 {
		t.Fatalf("got %q args=%v err=%v", sql, ctx.Args, err)
	}
}
