package nl_intelligence

import (
	"fmt"
	"strings"
)

// DialectEngine handles database-specific syntax adjustments
type DialectEngine struct{}

// FormatSQL adjusts a generic SQL query for a specific database
func (e *DialectEngine) FormatSQL(sql string, dialect string) string {
	sql = strings.TrimSpace(sql)

	switch strings.ToLower(dialect) {
	case "postgres":
		return e.formatPostgres(sql)
	case "snowflake":
		return e.formatSnowflake(sql)
	default:
		return sql
	}
}

func (e *DialectEngine) formatPostgres(sql string) string {
	// Postgres specific adjustments
	return sql
}

func (e *DialectEngine) formatSnowflake(sql string) string {
	// Snowflake specific adjustments (e.g., upper casing identifiers)
	return strings.ToUpper(sql)
}

// FormatCypher wraps raw Cypher in Apache AGE syntax
func (e *DialectEngine) FormatCypher(cypher string, graphName string) string {
	cypher = strings.TrimSpace(cypher)
	if strings.HasPrefix(strings.ToUpper(cypher), "SELECT") {
		return cypher // Already wrapped
	}

	return fmt.Sprintf(`SELECT * FROM cypher('%s', $$ %s $$) as (result agtype);`, graphName, cypher)
}
