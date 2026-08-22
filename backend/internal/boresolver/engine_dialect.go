package boresolver

import (
	"fmt"
	"strings"
)

type DialectType string

const (
	DialectPostgreSQL DialectType = "POSTGRESQL"
	DialectSnowflake  DialectType = "SNOWFLAKE"
	DialectStarRocks  DialectType = "STARROCKS"
	DialectDuckDB     DialectType = "DUCKDB"
)

type EngineDialect interface {
	DateTrunc(unit, col string) string
	NullSafeCoalesce(col, fallback string) string
	JSONExtract(col, path string) string
	QuoteIdentifier(ident string) string
}

type PostgreSQLDialectImpl struct{}

func (d PostgreSQLDialectImpl) DateTrunc(unit, col string) string {
	return fmt.Sprintf("DATE_TRUNC('%s', %s)", unit, col)
}
func (d PostgreSQLDialectImpl) NullSafeCoalesce(col, fallback string) string {
	return fmt.Sprintf("COALESCE(%s, %s)", col, fallback)
}
func (d PostgreSQLDialectImpl) JSONExtract(col, path string) string {
	return fmt.Sprintf("%s->>'%s'", col, path)
}
func (d PostgreSQLDialectImpl) QuoteIdentifier(ident string) string {
	return fmt.Sprintf(`"%s"`, ident)
}

type SnowflakeDialectImpl struct{}

func (d SnowflakeDialectImpl) DateTrunc(unit, col string) string {
	return fmt.Sprintf("DATE_TRUNC('%s', %s)", strings.ToUpper(unit), col)
}
func (d SnowflakeDialectImpl) NullSafeCoalesce(col, fallback string) string {
	return fmt.Sprintf("IFNULL(%s, %s)", col, fallback)
}
func (d SnowflakeDialectImpl) JSONExtract(col, path string) string {
	return fmt.Sprintf("GET_PATH(%s, '%s')::VARCHAR", col, path)
}
func (d SnowflakeDialectImpl) QuoteIdentifier(ident string) string {
	return fmt.Sprintf(`"%s"`, strings.ToUpper(ident))
}

type StarRocksDialectImpl struct{}

func (d StarRocksDialectImpl) DateTrunc(unit, col string) string {
	return fmt.Sprintf("DATE_TRUNC('%s', %s)", unit, col)
}
func (d StarRocksDialectImpl) NullSafeCoalesce(col, fallback string) string {
	return fmt.Sprintf("COALESCE(%s, %s)", col, fallback)
}
func (d StarRocksDialectImpl) JSONExtract(col, path string) string {
	return fmt.Sprintf("json_query(%s, '$.%s')", col, path)
}
func (d StarRocksDialectImpl) QuoteIdentifier(ident string) string {
	return fmt.Sprintf("`%s`", ident)
}

type DuckDBDialectImpl struct{}

func (d DuckDBDialectImpl) DateTrunc(unit, col string) string {
	return fmt.Sprintf("DATE_TRUNC('%s', %s)", unit, col)
}
func (d DuckDBDialectImpl) NullSafeCoalesce(col, fallback string) string {
	return fmt.Sprintf("COALESCE(%s, %s)", col, fallback)
}
func (d DuckDBDialectImpl) JSONExtract(col, path string) string {
	return fmt.Sprintf("%s->>'%s'", col, path)
}
func (d DuckDBDialectImpl) QuoteIdentifier(ident string) string {
	return fmt.Sprintf(`"%s"`, ident)
}

func ResolveDialect(t DialectType) EngineDialect {
	switch t {
	case DialectSnowflake:
		return SnowflakeDialectImpl{}
	case DialectStarRocks:
		return StarRocksDialectImpl{}
	case DialectDuckDB:
		return DuckDBDialectImpl{}
	default:
		return PostgreSQLDialectImpl{}
	}
}
