package analytics

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/lib/pq"
)

var safeIdentRegex = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func validateIdent(s string) (string, error) {
	if !safeIdentRegex.MatchString(s) {
		return "", fmt.Errorf("invalid identifier %q", s)
	}
	return s, nil
}

type WindowSQLResult struct {
	Query      string
	GroupCols  []string // physical column names in SELECT/GROUP BY
	MetricCol  string   // physical column aggregated
	TimeCol    string   // physical timestamp column
}

// CompileWindowRuleSQL translates an AggregateRuleSpec into a pushdown aggregation query.
// Strictly enforces SQL injection safety, identifier quoting, and tenant parameter binding ($1).
func CompileWindowRuleSQL(spec models.AggregateRuleSpec, table string, columnMap map[string]string) (*WindowSQLResult, error) {
	if err := spec.Validate(); err != nil {
		return nil, err
	}

	// 1. Validate table identifier (can be schema.table or table)
	parts := strings.Split(table, ".")
	var quotedTable string
	if len(parts) == 1 {
		tbl, err := validateIdent(parts[0])
		if err != nil {
			return nil, fmt.Errorf("table: %w", err)
		}
		quotedTable = pq.QuoteIdentifier(tbl)
	} else if len(parts) == 2 {
		sch, err := validateIdent(parts[0])
		if err != nil {
			return nil, fmt.Errorf("schema: %w", err)
		}
		tbl, err := validateIdent(parts[1])
		if err != nil {
			return nil, fmt.Errorf("table: %w", err)
		}
		quotedTable = pq.QuoteIdentifier(sch) + "." + pq.QuoteIdentifier(tbl)
	} else {
		return nil, fmt.Errorf("invalid table format %q", table)
	}

	// 2. Resolve and quote time column
	physTimeCol := spec.TimeColumn
	if mapped, ok := columnMap[spec.TimeColumn]; ok {
		physTimeCol = mapped
	}
	if _, err := validateIdent(physTimeCol); err != nil {
		return nil, fmt.Errorf("time_column %q: %w", physTimeCol, err)
	}
	quotedTimeCol := pq.QuoteIdentifier(physTimeCol)

	// 3. Resolve and quote group-by columns
	quotedGroupCols := make([]string, len(spec.GroupBy))
	physGroupCols := make([]string, len(spec.GroupBy))
	for i, g := range spec.GroupBy {
		phys := g
		if mapped, ok := columnMap[g]; ok {
			phys = mapped
		}
		if _, err := validateIdent(phys); err != nil {
			return nil, fmt.Errorf("group_by column %q: %w", phys, err)
		}
		physGroupCols[i] = phys
		quotedGroupCols[i] = pq.QuoteIdentifier(phys)
	}

	// 4. Resolve and quote metric column
	var aggExpr string
	physMetricCol := ""
	if spec.Aggregate == "count" && spec.Field == "" {
		aggExpr = "COUNT(*)"
	} else {
		physMetricCol = spec.Field
		if mapped, ok := columnMap[spec.Field]; ok {
			physMetricCol = mapped
		}
		if _, err := validateIdent(physMetricCol); err != nil {
			return nil, fmt.Errorf("field %q: %w", physMetricCol, err)
		}
		quotedMetricCol := pq.QuoteIdentifier(physMetricCol)
		aggExpr = fmt.Sprintf("%s(%s)", strings.ToUpper(spec.Aggregate), quotedMetricCol)
	}

	// 5. Build comparison operator
	var op string
	switch spec.Comparison {
	case "greater_than":
		op = ">"
	case "greater_or_equal":
		op = ">="
	case "less_than":
		op = "<"
	case "less_or_equal":
		op = "<="
	case "equals":
		op = "="
	case "not_equals":
		op = "<>"
	default:
		return nil, fmt.Errorf("unsupported comparison %q", spec.Comparison)
	}

	// 6. Window interval string (Postgres format)
	var intervalStr string
	switch spec.Window {
	case "1h", "hour":
		intervalStr = "interval '1 hour'"
	case "1d", "day":
		intervalStr = "interval '1 day'"
	default:
		intervalStr = "interval '1 hour'"
	}

	dateTruncArg := "'hour'"
	if spec.Window == "1d" || spec.Window == "day" {
		dateTruncArg = "'day'"
	}

	selectItems := append([]string{}, quotedGroupCols...)
	selectItems = append(selectItems,
		fmt.Sprintf("date_trunc(%s, %s) AS window_start", dateTruncArg, quotedTimeCol),
		fmt.Sprintf("date_trunc(%s, %s) + %s AS window_end", dateTruncArg, quotedTimeCol, intervalStr),
		fmt.Sprintf("%s AS metric_value", aggExpr),
	)

	groupByItems := append([]string{}, quotedGroupCols...)
	groupByItems = append(groupByItems, fmt.Sprintf("date_trunc(%s, %s)", dateTruncArg, quotedTimeCol))

	query := fmt.Sprintf(`
		SELECT %s
		FROM %s
		WHERE tenant_id = $1 AND %s >= $2 AND %s < $3
		GROUP BY %s
		HAVING %s %s %f
	`,
		strings.Join(selectItems, ", "),
		quotedTable,
		quotedTimeCol, quotedTimeCol,
		strings.Join(groupByItems, ", "),
		aggExpr, op, spec.Threshold,
	)

	return &WindowSQLResult{
		Query:     strings.TrimSpace(query),
		GroupCols: physGroupCols,
		MetricCol: physMetricCol,
		TimeCol:   physTimeCol,
	}, nil
}
