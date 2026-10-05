package querybuilder

import (
	"fmt"
	"os"
	"strings"
)

// defaultCubeSourceCatalog is the StarRocks external catalog that mirrors
// Postgres (JDBC) for cube materialize FROM clauses. Override with
// CUBE_SOURCE_CATALOG (lab/prod often "pg_alpha").
func defaultCubeSourceCatalog() string {
	if v := strings.TrimSpace(os.Getenv("CUBE_SOURCE_CATALOG")); v != "" {
		return sanitizeIdentifier(v)
	}
	return "pg_alpha"
}

// QualifyCubeSourceTable maps a BO driver path like "/orm/account" to a
// StarRocks-qualified table "pg_alpha.orm.account" for MV DDL. Bare
// "schema.table" / "catalog.schema.table" names pass through unchanged.
// Empty input stays empty.
func QualifyCubeSourceTable(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	// Already a dotted SQL name (schema.table or catalog.schema.table).
	if !strings.HasPrefix(raw, "/") && strings.Contains(raw, ".") {
		return raw
	}

	path := strings.TrimPrefix(raw, "/")
	parts := make([]string, 0, 3)
	for _, p := range strings.Split(path, "/") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id := sanitizeIdentifier(p)
		if id == "" {
			continue
		}
		parts = append(parts, id)
	}
	if len(parts) == 0 {
		return raw
	}
	cat := defaultCubeSourceCatalog()
	if cat == "" {
		return strings.Join(parts, ".")
	}
	return fmt.Sprintf("%s.%s", cat, strings.Join(parts, "."))
}
