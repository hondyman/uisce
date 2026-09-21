package calcengine

import (
	"fmt"
	"strings"
)

// QueryDialect abstracts placeholder binding and identifier quoting for the
// engine that will execute the final hot/cold UNION query. This keeps the
// watermark router backend-agnostic while guaranteeing that the combined SQL
// uses a single, consistent placeholder scheme.
type QueryDialect interface {
	BindPlaceholder(index int) string
	QuoteIdentifier(name string) string
	// RequiresOrderByForLimit reports whether the dialect requires an ORDER BY
	// clause whenever LIMIT is used (e.g., StarRocks/DataFusion/Iceberg).
	RequiresOrderByForLimit() bool
}

// PostgresQueryDialect produces PostgreSQL-style numbered placeholders and
// double-quoted identifiers.
type PostgresQueryDialect struct{}

func (PostgresQueryDialect) BindPlaceholder(index int) string {
	return fmt.Sprintf("$%d", index)
}

func (PostgresQueryDialect) RequiresOrderByForLimit() bool { return false }

func (PostgresQueryDialect) QuoteIdentifier(name string) string {
	parts := strings.Split(name, ".")
	for i, p := range parts {
		parts[i] = `"` + p + `"`
	}
	return strings.Join(parts, ".")
}

// StarRocksQueryDialect produces positional '?' placeholders and double-quoted
// identifiers for StarRocks/Apache DataFusion execution paths.
type StarRocksQueryDialect struct{}

func (StarRocksQueryDialect) BindPlaceholder(index int) string {
	return "?"
}

func (StarRocksQueryDialect) RequiresOrderByForLimit() bool { return true }

func (StarRocksQueryDialect) QuoteIdentifier(name string) string {
	parts := strings.Split(name, ".")
	for i, p := range parts {
		parts[i] = `"` + p + `"`
	}
	return strings.Join(parts, ".")
}

// DataFusionQueryDialect produces positional '?' placeholders and double-quoted
// identifiers for Apache DataFusion execution paths.
type DataFusionQueryDialect = StarRocksQueryDialect

// SQLServerQueryDialect produces '@pN' placeholders and bracket-quoted
// identifiers for SQL Server execution paths.
type SQLServerQueryDialect struct{}

func (SQLServerQueryDialect) BindPlaceholder(index int) string {
	return fmt.Sprintf("@p%d", index)
}

func (SQLServerQueryDialect) RequiresOrderByForLimit() bool { return false }

func (SQLServerQueryDialect) QuoteIdentifier(name string) string {
	parts := strings.Split(name, ".")
	for i, p := range parts {
		parts[i] = `[` + p + `]`
	}
	return strings.Join(parts, ".")
}
