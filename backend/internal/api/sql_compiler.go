package api

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type SQLCompiler struct {
	DB *sql.DB
}

func NewSQLCompiler(db *sql.DB) *SQLCompiler {
	return &SQLCompiler{DB: db}
}

// SemanticDictionary holds the active tenant's approved aliases and formulas
type SemanticDictionary struct {
	Formulas map[string]string // e.g. "net_yield" -> "(revenue - broker_fees) / avg_aum"
	Aliases  map[string]string // e.g. "aum" -> "total_valuation"
}

// LoadTenantDictionary fetches all approved custom logic for the active tenant
func (c *SQLCompiler) LoadTenantDictionary(ctx context.Context, tenantID string) (*SemanticDictionary, error) {
	dict := &SemanticDictionary{
		Formulas: make(map[string]string),
		Aliases:  make(map[string]string),
	}

	if c.DB == nil {
		// Built-in defaults
		dict.Formulas["net_yield"] = "(revenue - broker_fees) / NULLIF(avg_aum, 0)"
		dict.Formulas["avg_trade_size"] = "total_valuation / NULLIF(trade_count, 0)"
		dict.Aliases["aum"] = "total_valuation"
		dict.Aliases["nii"] = "net_interest_income"
		return dict, nil
	}

	query := `
		SELECT type, term, COALESCE(target_field_id, ''), COALESCE(expression, '') 
		FROM ai_knowledge_candidates 
		WHERE (tenant_id = $1 OR tenant_id = 'default' OR tenant_id = 'global') AND status = 'approved'
	`
	rows, err := c.DB.QueryContext(ctx, query, tenantID)
	if err != nil {
		return dict, nil
	}
	defer rows.Close()

	for rows.Next() {
		var kType, term, targetField, expression string
		if err := rows.Scan(&kType, &term, &targetField, &expression); err == nil {
			normalizedTerm := strings.ToLower(strings.ReplaceAll(term, " ", "_"))
			if kType == "calculated_measure" && expression != "" {
				dict.Formulas[normalizedTerm] = expression
			} else if kType == "alias" && targetField != "" {
				dict.Aliases[normalizedTerm] = targetField
			}
		}
	}

	return dict, nil
}

func (dict *SemanticDictionary) resolveField(field string) string {
	normalized := strings.ToLower(field)
	if target, exists := dict.Aliases[normalized]; exists {
		return target
	}
	return field
}

// Compile generates the final SQL string and parameterized arguments
func (c *SQLCompiler) Compile(ctx context.Context, tenantID string, query AIExplorerQueryDefinition, baseTable string) (string, []interface{}, error) {
	dict, err := c.LoadTenantDictionary(ctx, tenantID)
	if err != nil {
		return "", nil, fmt.Errorf("failed to load semantic dictionary: %w", err)
	}

	var sb strings.Builder
	var args []interface{}
	argCount := 1

	var selectCols []string
	var groupByIndices []string
	colIndex := 1

	// 1. Process Dimensions
	for _, dim := range query.Dimensions {
		resolvedDim := dict.resolveField(dim)
		selectCols = append(selectCols, fmt.Sprintf("%s AS \"%s\"", resolvedDim, dim))
		groupByIndices = append(groupByIndices, fmt.Sprintf("%d", colIndex))
		colIndex++
	}

	// 2. Process Time Dimensions
	for _, tDim := range query.TimeDimensions {
		resolvedTime := dict.resolveField(tDim.FieldID)
		grain := tDim.Granularity
		if grain == "" || grain == "raw" {
			grain = "month"
		}
		timeExpr := fmt.Sprintf("DATE_TRUNC('%s', %s)", grain, resolvedTime)
		selectCols = append(selectCols, fmt.Sprintf("%s AS \"%s\"", timeExpr, tDim.FieldID))
		groupByIndices = append(groupByIndices, fmt.Sprintf("%d", colIndex))
		colIndex++
	}

	// 3. Process Measures (and inject formulas)
	for _, meas := range query.Measures {
		normalized := strings.ToLower(strings.ReplaceAll(meas.FieldID, " ", "_"))
		var expr string
		if formula, exists := dict.Formulas[normalized]; exists {
			expr = formula
		} else {
			expr = dict.resolveField(meas.FieldID)
		}

		agg := meas.Agg
		if agg == "" {
			agg = "SUM"
		}
		aggExpr := fmt.Sprintf("%s(%s)", agg, expr)
		selectCols = append(selectCols, fmt.Sprintf("%s AS \"%s\"", aggExpr, meas.FieldID))
	}

	if len(selectCols) == 0 {
		selectCols = append(selectCols, "*")
	}

	// 4. Build core query
	sb.WriteString("SELECT \n  ")
	sb.WriteString(strings.Join(selectCols, ",\n  "))
	sb.WriteString(fmt.Sprintf("\nFROM %s\n", baseTable))

	// 5. Process Filters safely with parameters
	if len(query.Filters) > 0 {
		sb.WriteString("WHERE ")
		var filterClauses []string

		for _, f := range query.Filters {
			resolvedField := dict.resolveField(f.FieldID)
			op := strings.ToUpper(f.Operator)
			if op == "" {
				op = "="
			}

			switch op {
			case "IN", "NOT IN":
				filterClauses = append(filterClauses, fmt.Sprintf("%s %s (%v)", resolvedField, op, f.Value))
			default:
				filterClauses = append(filterClauses, fmt.Sprintf("%s %s $%d", resolvedField, op, argCount))
				args = append(args, f.Value)
				argCount++
			}
		}
		sb.WriteString(strings.Join(filterClauses, " AND "))
		sb.WriteString("\n")
	}

	// 6. Apply GROUP BY
	if len(groupByIndices) > 0 && len(query.Measures) > 0 {
		sb.WriteString(fmt.Sprintf("GROUP BY %s\n", strings.Join(groupByIndices, ", ")))
	}

	// 7. Apply Limit
	limit := query.Limit
	if limit <= 0 || limit > 10000 {
		limit = 500
	}
	sb.WriteString(fmt.Sprintf("LIMIT %d;", limit))

	return sb.String(), args, nil
}
