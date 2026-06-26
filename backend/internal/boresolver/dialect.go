package boresolver

import (
	"fmt"
	"strings"
)

// Dialect defines the interface for SQL dialect-specific operations
type Dialect interface {
	Name() string
	QuoteIdent(ident string) string
	QuoteLiteral(lit string) string

	// Operators
	OpAdd() string
	OpSub() string
	OpMul() string
	OpDiv() string

	// Null-safe division
	SafeDiv(numerator string, denominator string) string

	// Generic Function support
	Func(name string, args ...string) string
}

// ============================================================================
// Postgres Dialect
// ============================================================================

type PostgresDialect struct{}

func (d PostgresDialect) Name() string { return "postgres" }

func (d PostgresDialect) QuoteIdent(s string) string {
	parts := strings.Split(s, ".")
	quoted := make([]string, len(parts))
	for i, p := range parts {
		quoted[i] = `"` + p + `"`
	}
	return strings.Join(quoted, ".")
}

func (d PostgresDialect) QuoteLiteral(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func (d PostgresDialect) OpAdd() string { return "+" }
func (d PostgresDialect) OpSub() string { return "-" }
func (d PostgresDialect) OpMul() string { return "*" }
func (d PostgresDialect) OpDiv() string { return "/" }

func (d PostgresDialect) SafeDiv(n, dnm string) string {
	return fmt.Sprintf("CASE WHEN %s = 0 THEN NULL ELSE %s / %s END", dnm, n, dnm)
}

func (d PostgresDialect) Func(name string, args ...string) string {
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
		// args: unit, value, expr
		return fmt.Sprintf("(%s + INTERVAL '%s %s')", args[2], args[1], args[0])
	case "date_diff":
		// args: unit, start, end
		return fmt.Sprintf("EXTRACT(EPOCH FROM (%s - %s))", args[2], args[1])
	case "case_when":
		// args: cond1, val1, cond2, val2, ..., default
		if len(args) < 3 || len(args)%2 == 0 {
			return "NULL"
		}
		var parts []string
		for i := 0; i < len(args)-1; i += 2 {
			parts = append(parts, fmt.Sprintf("WHEN %s THEN %s", args[i], args[i+1]))
		}
		elseExpr := args[len(args)-1]
		return fmt.Sprintf("(CASE %s ELSE %s END)", strings.Join(parts, " "), elseExpr)
	}
	// Default
	return fmt.Sprintf("%s(%s)", name, strings.Join(args, ", "))
}

// ============================================================================
// Snowflake Dialect
// ============================================================================

type SnowflakeDialect struct{}

func (d SnowflakeDialect) Name() string { return "snowflake" }

func (d SnowflakeDialect) QuoteIdent(s string) string {
	parts := strings.Split(s, ".")
	quoted := make([]string, len(parts))
	for i, p := range parts {
		quoted[i] = `"` + strings.ToUpper(p) + `"`
	}
	return strings.Join(quoted, ".")
}

func (d SnowflakeDialect) QuoteLiteral(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func (d SnowflakeDialect) OpAdd() string { return "+" }
func (d SnowflakeDialect) OpSub() string { return "-" }
func (d SnowflakeDialect) OpMul() string { return "*" }
func (d SnowflakeDialect) OpDiv() string { return "/" }

func (d SnowflakeDialect) SafeDiv(n, dnm string) string {
	return fmt.Sprintf("IFF(%s = 0, NULL, %s / %s)", dnm, n, dnm)
}

func (d SnowflakeDialect) Func(name string, args ...string) string {
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
		return fmt.Sprintf("DATEADD(%s, %s, %s)", args[0], args[1], args[2])
	case "date_diff":
		return fmt.Sprintf("DATEDIFF(%s, %s, %s)", args[0], args[1], args[2])
	case "case_when":
		if len(args) < 3 || len(args)%2 == 0 {
			return "NULL"
		}
		var parts []string
		for i := 0; i < len(args)-1; i += 2 {
			parts = append(parts, fmt.Sprintf("WHEN %s THEN %s", args[i], args[i+1]))
		}
		elseExpr := args[len(args)-1]
		return fmt.Sprintf("CASE %s ELSE %s END", strings.Join(parts, " "), elseExpr)
	}
	return fmt.Sprintf("%s(%s)", strings.ToUpper(name), strings.Join(args, ", "))
}

// ============================================================================
// SQL Server Dialect
// ============================================================================

type SQLServerDialect struct{}

func (d SQLServerDialect) Name() string { return "sqlserver" }

func (d SQLServerDialect) QuoteIdent(s string) string {
	parts := strings.Split(s, ".")
	quoted := make([]string, len(parts))
	for i, p := range parts {
		quoted[i] = `[` + p + `]`
	}
	return strings.Join(quoted, ".")
}

func (d SQLServerDialect) QuoteLiteral(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func (d SQLServerDialect) OpAdd() string { return "+" }
func (d SQLServerDialect) OpSub() string { return "-" }
func (d SQLServerDialect) OpMul() string { return "*" }
func (d SQLServerDialect) OpDiv() string { return "/" }

func (d SQLServerDialect) SafeDiv(n, dnm string) string {
	return fmt.Sprintf("CASE WHEN %s = 0 THEN NULL ELSE %s / %s END", dnm, n, dnm)
}

func (d SQLServerDialect) Func(name string, args ...string) string {
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
		return fmt.Sprintf("DATEADD(%s, %s, %s)", args[0], args[1], args[2])
	case "date_diff":
		return fmt.Sprintf("DATEDIFF(%s, %s, %s)", args[0], args[1], args[2])
	case "case_when":
		if len(args) < 3 || len(args)%2 == 0 {
			return "NULL"
		}
		var parts []string
		for i := 0; i < len(args)-1; i += 2 {
			parts = append(parts, fmt.Sprintf("WHEN %s THEN %s", args[i], args[i+1]))
		}
		elseExpr := args[len(args)-1]
		return fmt.Sprintf("CASE %s ELSE %s END", strings.Join(parts, " "), elseExpr)
	}
	return fmt.Sprintf("%s(%s)", name, strings.Join(args, ", "))
}
