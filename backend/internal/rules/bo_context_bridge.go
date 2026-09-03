package rules

import (
	"strings"

	"github.com/hondyman/uisce/backend/internal/boresolver"
)

// BuildContextFromBORow converts a raw DB row (keyed by physical column name,
// bare or table-qualified as "table.column") into the map[string]interface{}
// context ConditionEvaluator/HierarchyResolver expect, keyed by Business
// Object field name — the same semantic vocabulary rulefabric
// (see boresolver/expression_bridge.go) and the calc engine
// (analytics.ExecuteFormulaCalculation) already use.
//
// This lets a compliance/validation rule be authored once against BO field
// names and evaluated correctly regardless of the underlying physical
// schema, instead of hardcoding raw column names into every rule condition.
//
// Columns with no matching BOField are passed through unchanged under their
// original key, so rules already written against raw column names keep
// working during migration.
func BuildContextFromBORow(bo *boresolver.BODefinition, row map[string]interface{}) map[string]interface{} {
	ctx := make(map[string]interface{}, len(row)+len(bo.Fields))
	for k, v := range row {
		ctx[k] = v
	}

	for _, f := range bo.Fields {
		col := f.PhysicalColumn
		if col == "" || f.Name == "" {
			continue
		}
		if v, ok := row[col]; ok {
			ctx[f.Name] = v
			continue
		}
		// PhysicalColumn is fully qualified ("table.column") but the row may
		// be keyed by the bare column name depending on the query that
		// produced it (e.g. a plain SELECT vs. one aliasing every column).
		if idx := strings.LastIndex(col, "."); idx >= 0 {
			if v, ok := row[col[idx+1:]]; ok {
				ctx[f.Name] = v
			}
		}
	}

	return ctx
}
