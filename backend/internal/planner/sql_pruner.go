package planner

import (
	"fmt"
	"strings"

	"github.com/hondyman/semlayer/backend/internal/domain"
)

// SQLBuilder helps construct SQL queries with pruning hints
type SQLBuilder struct {
	table      string
	selectCols []string
	where      []string
	args       []any
}

// NewSQLBuilder creates a new SQL builder for the given table
func NewSQLBuilder(table string) *SQLBuilder {
	return &SQLBuilder{table: table}
}

// ApplyHints applies pruning hints to the SQL builder
func (b *SQLBuilder) ApplyHints(h domain.PruningHints) *SQLBuilder {
	if len(h.Columns) > 0 {
		b.selectCols = append(b.selectCols, h.Columns...)
	}
	if len(h.RowFilters) > 0 {
		b.where = append(b.where, h.RowFilters...)
		b.args = append(b.args, h.BindArgs...)
	}
	return b
}

// Build constructs the final SQL query and returns it with bind arguments
func (b *SQLBuilder) Build() (string, []any) {
	cols := "*"
	if len(b.selectCols) > 0 {
		// Deduplicate columns
		seen := map[string]struct{}{}
		dedup := []string{}
		for _, c := range b.selectCols {
			if _, ok := seen[c]; !ok {
				seen[c] = struct{}{}
				dedup = append(dedup, c)
			}
		}
		cols = strings.Join(dedup, ", ")
	}

	sql := fmt.Sprintf("SELECT %s FROM %s", cols, b.table)
	if len(b.where) > 0 {
		sql += " WHERE " + strings.Join(b.where, " AND ")
	}

	return sql, b.args
}

// Example usage:
// dec := domain.EvaluationDecision{
//   Decision: "partial",
//   AllowedScopes: []string{"metrics"}, // dimensions are restricted
// }
// schemaProvider := &inMemorySchema{ /* map scopes to columns */ }
// adapter := &domain.PlannerAdapter{Schema: schemaProvider}
// hints, _ := adapter.BuildHints("asset-orders", dec, "acme")
//
// sql := planner.NewSQLBuilder("semantic.orders_view").ApplyHints(hints).Build()
// → SELECT avg_order_value, total_orders FROM semantic.orders_view WHERE tenant_id = $1
// args: ["acme"]
