package boresolver

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
)

// SQLRowSource is a RowSource (see host_runtime_executor.go) that fetches
// host-runtime calc inputs with ONE batched, tenant-scoped query per node —
// grouping rows by entity inside the query engine (via a JSON array
// aggregate) rather than pulling raw ungrouped rows and grouping them in
// application code.
//
// This is the throughput-critical difference for a multi-tenant platform:
// fetching per-entity (an N+1 query loop) does not scale past a handful of
// entities, let alone millions of positions across every tenant on a shared
// StarRocks/Iceberg cluster. A single "GROUP BY entity, aggregate into an
// array" query lets the column store do the grouping and returns one row
// per entity instead of one row per cash flow.
//
// Aggregation is rendered as a JSON array cast to text rather than a native
// SQL array type deliberately: native array wire formats differ per driver
// (lib/pq's array encoding, Snowflake VARIANT, DataFusion's list type), so
// scanning them requires a driver-specific type per dialect. A JSON-text
// column is the one thing every engine and every database/sql driver can
// return uniformly, and it's what the Go side actually needs to unmarshal.
//
// Every term referenced by a host-runtime CalcNode's formula must resolve
// (via Resolver) to the SAME physical table — cross-table cash-flow series
// (e.g. amount on one table, date on another via a join) aren't supported
// here; that would need the same join-injection machinery
// bo_sql_generator.go uses for Rule 7 tenant scoping
// (InjectTenantScopingToGraph), which this intentionally doesn't duplicate
// yet.
type SQLRowSource struct {
	DB       *sqlx.DB
	Resolver *Resolver // resolves term keys -> physical table/column (see resolver.go)

	// EntityTerm is the term key whose physical column groups rows into
	// entities (e.g. "customer_id", "fund_id") — one host-runtime result
	// per group.
	EntityTerm string

	// OrderTerm is the term key that defines row order within each
	// entity's aggregated series (e.g. "cashflow_date"). Order matters for
	// functions like IRR where flows[i] means "period i" — it does not
	// matter for XIRR (finlib re-sorts by date internally), but supplying
	// it always keeps results deterministic and auditable.
	OrderTerm string
}

// jsonArrayAggRenderer builds "aggregate valueCol into a JSON array ordered
// by orderCol, cast to text" SQL for one dialect. Dialects without an entry
// return an explicit error from FetchRows rather than emitting SQL that was
// never validated against that engine.
var jsonArrayAggRenderer = map[string]func(quote func(string) string, valueCol, orderCol string) string{
	// StarRocks maps to PostgresDialect in GetDialect (dialect.go) today,
	// so this single renderer already covers OLTP Postgres and StarRocks.
	"postgres": func(quote func(string) string, valueCol, orderCol string) string {
		return fmt.Sprintf("json_agg(%s ORDER BY %s)::text", quote(valueCol), quote(orderCol))
	},
}

// FetchRows implements RowSource. tenantID scopes the query via the same
// "<column> = <param>" convention bo_sql_generator.go's
// InjectTenantScopingToGraph uses for Rule 7 tenant isolation.
func (s *SQLRowSource) FetchRows(ctx context.Context, tenantID string, termKeys []string) (map[string][]CalcRow, error) {
	if s.DB == nil || s.Resolver == nil {
		return nil, fmt.Errorf("SQLRowSource: DB and Resolver are required")
	}
	if tenantID == "" {
		return nil, fmt.Errorf("SQLRowSource: tenantID is required")
	}
	if s.EntityTerm == "" || s.OrderTerm == "" {
		return nil, fmt.Errorf("SQLRowSource: EntityTerm and OrderTerm are required")
	}

	dialect := s.Resolver.Dialect
	if dialect == nil {
		dialect = PostgresDialect{}
	}
	renderAgg, ok := jsonArrayAggRenderer[dialect.Name()]
	if !ok {
		return nil, fmt.Errorf("SQLRowSource: batched row fetch is not yet implemented for dialect %q", dialect.Name())
	}

	entityMapping, _, err := s.Resolver.ResolveTerm(s.EntityTerm)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve entity term %q: %w", s.EntityTerm, err)
	}
	orderMapping, _, err := s.Resolver.ResolveTerm(s.OrderTerm)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve order term %q: %w", s.OrderTerm, err)
	}
	if orderMapping.Table != entityMapping.Table {
		return nil, fmt.Errorf("SQLRowSource: order term %q resolves to table %q, expected %q (entity table)", s.OrderTerm, orderMapping.Table, entityMapping.Table)
	}

	table := entityMapping.Table
	columns := make(map[string]string, len(termKeys)) // termKey -> physical column
	for _, term := range termKeys {
		mapping, _, err := s.Resolver.ResolveTerm(term)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve term %q: %w", term, err)
		}
		if mapping.Table != table {
			return nil, fmt.Errorf("SQLRowSource: term %q resolves to table %q, expected %q (all terms in a host-runtime calc must share one physical table)", term, mapping.Table, table)
		}
		columns[term] = mapping.Column
	}

	selectCols := []string{fmt.Sprintf("%s AS entity_id", dialect.QuoteIdent(entityMapping.Column))}
	for _, term := range termKeys {
		selectCols = append(selectCols, fmt.Sprintf("%s AS %s", renderAgg(dialect.QuoteIdent, columns[term], orderMapping.Column), dialect.QuoteIdent(term)))
	}

	query := fmt.Sprintf(
		"SELECT %s FROM %s WHERE %s = $1 GROUP BY %s",
		strings.Join(selectCols, ", "),
		dialect.QuoteIdent(table),
		dialect.QuoteIdent("tenant_id"),
		dialect.QuoteIdent(entityMapping.Column),
	)

	rows, err := s.DB.QueryxContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("SQLRowSource: query failed: %w", err)
	}
	defer rows.Close()

	result := make(map[string][]CalcRow)
	for rows.Next() {
		raw := make(map[string]any)
		if err := rows.MapScan(raw); err != nil {
			return nil, fmt.Errorf("SQLRowSource: scan failed: %w", err)
		}

		entityID := fmt.Sprintf("%v", raw["entity_id"])

		seriesByTerm := make(map[string][]any, len(termKeys))
		seriesLen := -1
		for _, term := range termKeys {
			var series []any
			if err := json.Unmarshal(asBytes(raw[term]), &series); err != nil {
				return nil, fmt.Errorf("SQLRowSource: failed to decode aggregated series for %q (entity %s): %w", term, entityID, err)
			}
			seriesByTerm[term] = series
			if seriesLen == -1 {
				seriesLen = len(series)
			} else if len(series) != seriesLen {
				return nil, fmt.Errorf("SQLRowSource: aggregated series length mismatch for entity %s: %q has %d values, expected %d", entityID, term, len(series), seriesLen)
			}
		}

		entityRows := make([]CalcRow, seriesLen)
		for i := 0; i < seriesLen; i++ {
			row := make(CalcRow, len(termKeys))
			for _, term := range termKeys {
				row[term] = seriesByTerm[term][i]
			}
			entityRows[i] = row
		}
		result[entityID] = entityRows
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("SQLRowSource: row iteration failed: %w", err)
	}

	return result, nil
}

func asBytes(v any) []byte {
	switch b := v.(type) {
	case []byte:
		return b
	case string:
		return []byte(b)
	default:
		return []byte(fmt.Sprintf("%v", v))
	}
}

