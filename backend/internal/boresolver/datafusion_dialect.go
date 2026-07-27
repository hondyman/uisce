package boresolver

import (
	"fmt"
	"strings"
)

// DataFusionIcebergDialect implements Dialect for Apache DataFusion querying Iceberg REST catalogs.
// DataFusion uses double-quote identifier quoting (ANSI SQL) and standard '?' parameter placeholders for Flight SQL.
type DataFusionIcebergDialect struct{}

func (d DataFusionIcebergDialect) Name() string {
	return "datafusion_iceberg"
}

// QuoteIdent wraps identifiers in standard double quotes for DataFusion.
func (d DataFusionIcebergDialect) QuoteIdent(name string) string {
	clean := strings.Trim(name, "\"")
	return fmt.Sprintf("\"%s\"", clean)
}

// QuoteLiteral quotes a string literal for DataFusion.
func (d DataFusionIcebergDialect) QuoteLiteral(lit string) string {
	return "'" + strings.ReplaceAll(lit, "'", "''") + "'"
}

func (d DataFusionIcebergDialect) OpAdd() string { return "+" }
func (d DataFusionIcebergDialect) OpSub() string { return "-" }
func (d DataFusionIcebergDialect) OpMul() string { return "*" }
func (d DataFusionIcebergDialect) OpDiv() string { return "/" }

// SafeDiv uses NULLIF to prevent division by zero.
func (d DataFusionIcebergDialect) SafeDiv(numerator, denominator string) string {
	return fmt.Sprintf("(%s / NULLIF(%s, 0))", numerator, denominator)
}

// Func implements DataFusion-specific functions.
func (d DataFusionIcebergDialect) Func(name string, args ...string) string {
	switch strings.ToLower(name) {
	case "coalesce":
		return fmt.Sprintf("COALESCE(%s)", strings.Join(args, ", "))
	case "abs":
		return fmt.Sprintf("ABS(%s)", args[0])
	case "round":
		return fmt.Sprintf("ROUND(%s)", strings.Join(args, ", "))
	case "cast":
		return fmt.Sprintf("CAST(%s AS %s)", args[0], args[1])
	case "date_add":
		if len(args) >= 3 {
			return fmt.Sprintf("DATE_ADD('%s', %s, %s)", args[0], args[1], args[2])
		}
	case "date_diff":
		if len(args) >= 2 {
			return fmt.Sprintf("DATE_DIFF('%s', %s, %s)", args[0], args[1], args[2])
		}
	case "case_when":
		if len(args) >= 3 && len(args)%2 == 1 {
			var parts []string
			for i := 0; i < len(args)-1; i += 2 {
				parts = append(parts, fmt.Sprintf("WHEN %s THEN %s", args[i], args[i+1]))
			}
			return fmt.Sprintf("(CASE %s ELSE %s END)", strings.Join(parts, " "), args[len(args)-1])
		}
	}
	return fmt.Sprintf("%s(%s)", strings.ToUpper(name), strings.Join(args, ", "))
}

// FormatQualifiedPath converts catalog paths (e.g., ["tenant_alpha", "default", "customer_orders"])
// into 3-tier ANSI double-quoted paths ("tenant_alpha"."default"."customer_orders").
func (d DataFusionIcebergDialect) FormatQualifiedPath(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	quoted := make([]string, len(parts))
	for i, part := range parts {
		cleanPart := strings.TrimPrefix(part, "/")
		quoted[i] = d.QuoteIdent(cleanPart)
	}
	return strings.Join(quoted, ".")
}

// BindPlaceholder returns standard parameter placeholders ('?') used by DataFusion Flight SQL / Arrow.
func (d DataFusionIcebergDialect) BindPlaceholder(index int) string {
	return "?"
}

// BuildLimitOffset formats standard LIMIT ... OFFSET pagination for DataFusion execution plans.
func (d DataFusionIcebergDialect) BuildLimitOffset(limit, offset int) string {
	if limit <= 0 && offset <= 0 {
		return ""
	}
	if offset > 0 {
		if limit <= 0 {
			limit = 2147483647
		}
		return fmt.Sprintf("LIMIT %d OFFSET %d", limit, offset)
	}
	return fmt.Sprintf("LIMIT %d", limit)
}

// RequiresOrderByForLimit indicates that DataFusion does NOT strictly require ORDER BY clauses for LIMIT.
func (d DataFusionIcebergDialect) RequiresOrderByForLimit() bool {
	return false
}

// SupportsBitemporalAsOf returns true because Iceberg REST catalogs support TIMESTAMP AS OF syntax.
func (d DataFusionIcebergDialect) SupportsBitemporalAsOf() bool {
	return true
}

// FormatTableSnapshot returns the Iceberg snapshot or time-travel syntax for DataFusion query planning.
func (d DataFusionIcebergDialect) FormatTableSnapshot(tableQualifiedPath string, asOfIsoTimestamp string) string {
	if asOfIsoTimestamp == "" {
		return tableQualifiedPath
	}
	return fmt.Sprintf("%s FOR SYSTEM_TIME AS OF TIMESTAMP '%s'", tableQualifiedPath, asOfIsoTimestamp)
}
