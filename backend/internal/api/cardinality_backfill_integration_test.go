package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"testing"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

// TestBackfillJunctionRelationships_IsIdempotent runs the full backfill
// (re-resolve + junction detection/insert) against a real Postgres
// database, twice, and asserts the second run makes no further changes.
// This is the check sqlmock cannot do: sqlmock validates SQL shape, not
// schema, so it can't catch a CHECK-constraint violation or a missing
// unique index behind an ON CONFLICT target.
//
// Skipped unless TEST_DATABASE_URL is set, so it runs automatically
// wherever a real Postgres is available (CI with a service container, or a
// developer machine with the schema migrated) and skips harmlessly
// everywhere else, including this sandbox.
//
// Self-provisioning: this test creates its own junction-shaped tables
// (uniquely named per run) and its own entity_attribute /
// entity_attribute_column_mapping / entity_relationship rows, rather than
// depending on whatever pre-existing data (e.g. the Northwind fixture)
// happens to be in the target database. That keeps it independent of that
// fixture's own state or repair, and gives deterministic before/after
// assertions instead of only "the row count didn't change" on unknown
// pre-existing data. It still requires the baseline platform schema
// (006_relationship_discovery_schema.sql,
// 20251226_restructure_entity_schema_robust.sql, and at least one row in
// tenants / tenant_product_datasource) already migrated/seeded — that's a
// normal precondition for an integration test against this app, not
// something this test can or should provision itself.
func TestBackfillJunctionRelationships_IsIdempotent(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping live-database integration test")
	}

	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer db.Close()
	ctx := context.Background()
	require.NoError(t, db.PingContext(ctx))

	tenantID, tenantDatasourceID, ok := lookupAnyTenantScope(ctx, db)
	if !ok {
		t.Skip("no row in tenants/tenant_product_datasource; nothing to scope this test's fixture data to")
	}

	fx := newJunctionFixture(t, db, tenantID, tenantDatasourceID)
	defer fx.cleanup(t, db)
	fx.create(t, db)

	resolver := NewCardinalityResolver(db)

	// First run: re-resolve existing rows, then insert/refresh junction rows.
	_, err = ReresolveExistingRelationships(ctx, db, resolver)
	require.NoError(t, err, "re-resolve must not fail against the real schema/CHECK constraints")

	firstInserted, err := BackfillJunctionRelationships(ctx, db, resolver)
	require.NoError(t, err, "junction backfill must not fail against the real schema/CHECK constraints — "+
		"this is the case sqlmock cannot catch (e.g. a bad literal against entity_relationship_valid_cardinality, "+
		"or a missing unique index behind the ON CONFLICT target)")

	// Deterministic assertion (not just "row count is stable" on unknown
	// pre-existing data): our own fixture's junction table must have
	// produced exactly one MANY_TO_MANY_JUNCTION row between our two
	// parent entities.
	junctionRows := fx.countOwnJunctionRows(t, db)
	require.Equal(t, 1, junctionRows, "expected exactly one synthesized MANY_TO_MANY_JUNCTION row for this fixture's junction table")

	rowCountAfterFirst := countEntityRelationshipRows(t, db)

	// Second run against the same data: must not error, duplicate rows, or
	// change the row count — proving ON CONFLICT actually resolves against
	// a real unique index rather than erroring or silently duplicating.
	_, err = ReresolveExistingRelationships(ctx, db, resolver)
	require.NoError(t, err)

	secondInserted, err := BackfillJunctionRelationships(ctx, db, resolver)
	require.NoError(t, err)

	rowCountAfterSecond := countEntityRelationshipRows(t, db)
	junctionRowsAfterSecond := fx.countOwnJunctionRows(t, db)

	require.Equal(t, rowCountAfterFirst, rowCountAfterSecond,
		"second run must not change entity_relationship's row count (ON CONFLICT should update in place, not insert duplicates)")
	require.Equal(t, 1, junctionRowsAfterSecond, "second run must still show exactly one junction row for this fixture, not a mirrored duplicate")
	require.Equal(t, firstInserted, secondInserted,
		"both runs should report the same number of junction relationships found in this fixed dataset")
}

func countEntityRelationshipRows(t *testing.T, db *sql.DB) int {
	t.Helper()
	var count int
	require.NoError(t, db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM public.entity_relationship`).Scan(&count))
	return count
}

func lookupAnyTenantScope(ctx context.Context, db *sql.DB) (tenantID, tenantDatasourceID string, ok bool) {
	err := db.QueryRowContext(ctx, `
		SELECT tenant_id, id FROM public.tenant_product_datasource LIMIT 1
	`).Scan(&tenantID, &tenantDatasourceID)
	if err != nil {
		return "", "", false
	}
	return tenantID, tenantDatasourceID, true
}

// junctionFixture provisions a self-contained order_items-style junction
// table (with two real parent tables, real FK constraints, and matching
// entity_attribute/entity_attribute_column_mapping rows) under a unique
// name suffix so concurrent/repeated test runs don't collide, and this
// test's assertions don't depend on any other data already present in the
// target database.
type junctionFixture struct {
	suffix             string
	parentATable       string
	parentBTable       string
	junctionTable      string
	tenantID           string
	tenantDatasourceID string
	parentAEntityID    string
	parentBEntityID    string
	junctionEntityID   string
	catalogNodeIDs     []string
}

func newJunctionFixture(t *testing.T, db *sql.DB, tenantID, tenantDatasourceID string) *junctionFixture {
	t.Helper()
	suffix := fmt.Sprintf("cardtest_%d", rand.Intn(1_000_000))
	return &junctionFixture{
		suffix:             suffix,
		parentATable:       "parent_a_" + suffix,
		parentBTable:       "parent_b_" + suffix,
		junctionTable:      "junction_" + suffix,
		tenantID:           tenantID,
		tenantDatasourceID: tenantDatasourceID,
	}
}

func (fx *junctionFixture) create(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx := context.Background()

	_, err := db.ExecContext(ctx, fmt.Sprintf(`CREATE TABLE public.%s (id uuid PRIMARY KEY DEFAULT gen_random_uuid())`, fx.parentATable))
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, fmt.Sprintf(`CREATE TABLE public.%s (id uuid PRIMARY KEY DEFAULT gen_random_uuid())`, fx.parentBTable))
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, fmt.Sprintf(`
		CREATE TABLE public.%s (
			a_id uuid NOT NULL REFERENCES public.%s(id),
			b_id uuid NOT NULL REFERENCES public.%s(id),
			PRIMARY KEY (a_id, b_id)
		)
	`, fx.junctionTable, fx.parentATable, fx.parentBTable))
	require.NoError(t, err)

	fx.parentAEntityID = fx.insertEntity(t, db, fx.parentATable, "id")
	fx.parentBEntityID = fx.insertEntity(t, db, fx.parentBTable, "id")
	fx.junctionEntityID = fx.insertEntity(t, db, fx.junctionTable, "a_id")

	// BackfillJunctionRelationships' candidate-table discovery reads from
	// catalog_edge (properties->>'source_table'/'target_table'), not
	// entity_relationship — entity_relationship has no live discovery-time
	// writer (see cardinality_backfill.go's comment), so sourcing
	// candidates from it would find nothing on a fresh environment. Seed
	// the two raw FK edges (junction -> parentA, junction -> parentB) the
	// real scanner (internal/scanner/ansi_scanner.go) would have produced,
	// in the same shape it writes, so the junction table actually gets
	// considered.
	fx.insertRawFKEdge(t, db, fx.junctionTable, "a_id", fx.parentATable, "id")
	fx.insertRawFKEdge(t, db, fx.junctionTable, "b_id", fx.parentBTable, "id")
}

// insertRawFKEdge seeds a catalog_edge row in the shape
// internal/scanner/ansi_scanner.go's processForeignKeys writes it (id,
// tenant_id, tenant_datasource_id, source_node_id, target_node_id,
// properties, edge_type_id, edge_type_name), so
// BackfillJunctionRelationships' candidate discovery (which reads
// properties->>'source_table'/'target_table') actually finds it.
//
// catalog_edge.source_node_id/target_node_id carry a FOREIGN KEY to
// catalog_node (migrations/001003_add_catalog_edge_constraints.sql), so
// this also creates minimal catalog_node rows for the referenced tables.
// catalog_node's exact live column set has drifted across migrations in
// this codebase (confirmed elsewhere in this session); the columns used
// here (id, tenant_id, tenant_datasource_id, node_name, node_type_id) are
// the ones corroborated by matching usage across
// relationships_discovery.go, fk_discovery_engine.go and
// businessobject_service.go, but this is the one part of this fixture not
// independently verified against a live schema — if this INSERT fails
// with an unknown-column error when this test actually runs (i.e. once
// TEST_DATABASE_URL is set somewhere), that's the place to fix, not a
// sign the production query is wrong.
func (fx *junctionFixture) insertRawFKEdge(t *testing.T, db *sql.DB, sourceTable, sourceColumn, targetTable, targetColumn string) {
	t.Helper()
	ctx := context.Background()

	// Matches internal/scanner/ansi_scanner.go's NODE_TYPE_TABLE constant —
	// the live scanner's own fixed "this catalog_node represents a table"
	// type id, reused here rather than inventing a new one.
	const nodeTypeTable = "49a50271-ae58-4d3e-ae1c-2f5b89d89192"

	sourceNodeID := fx.ensureCatalogNode(t, db, sourceTable, nodeTypeTable)
	targetNodeID := fx.ensureCatalogNode(t, db, targetTable, nodeTypeTable)

	props, err := json.Marshal(map[string]interface{}{
		"edge_type_name": "foreign_key",
		"cardinality":    "MANY_TO_ONE",
		"source_table":   sourceTable,
		"target_table":   targetTable,
		"columns": []map[string]string{
			{"source_column": sourceColumn, "target_column": targetColumn},
		},
	})
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, `
		INSERT INTO public.catalog_edge (
			id, tenant_id, tenant_datasource_id, source_node_id, target_node_id,
			properties, edge_type_name, created_at
		) VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, 'foreign_key', now())
	`, fx.tenantID, fx.tenantDatasourceID, sourceNodeID, targetNodeID, props)
	require.NoError(t, err)
}

// ensureCatalogNode inserts (or reuses) a minimal catalog_node row for
// tableName scoped to this fixture's tenant/datasource, returning its id.
func (fx *junctionFixture) ensureCatalogNode(t *testing.T, db *sql.DB, tableName, nodeTypeID string) string {
	t.Helper()
	ctx := context.Background()

	var nodeID string
	err := db.QueryRowContext(ctx, `
		SELECT id FROM public.catalog_node
		WHERE tenant_datasource_id = $1 AND node_name = $2
		LIMIT 1
	`, fx.tenantDatasourceID, tableName).Scan(&nodeID)
	if err == nil {
		return nodeID
	}

	err = db.QueryRowContext(ctx, `
		INSERT INTO public.catalog_node (id, tenant_id, tenant_datasource_id, node_name, node_type_id, created_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, now())
		RETURNING id
	`, fx.tenantID, fx.tenantDatasourceID, tableName, nodeTypeID).Scan(&nodeID)
	require.NoError(t, err, "insertRawFKEdge's catalog_node column assumptions may not match the live schema — see its doc comment")

	fx.catalogNodeIDs = append(fx.catalogNodeIDs, nodeID)
	return nodeID
}

func (fx *junctionFixture) insertEntity(t *testing.T, db *sql.DB, tableName, columnName string) string {
	t.Helper()
	ctx := context.Background()

	var entityID string
	err := db.QueryRowContext(ctx, `
		INSERT INTO public.entity_attribute (tenant_id, tenant_datasource_id, entity_key, name)
		VALUES ($1, $2, $3, $3)
		RETURNING id
	`, fx.tenantID, fx.tenantDatasourceID, tableName).Scan(&entityID)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, `
		INSERT INTO public.entity_attribute_column_mapping
			(tenant_id, tenant_datasource_id, entity_attribute_id, table_name, column_name, is_primary_key)
		VALUES ($1, $2, $3, $4, $5, true)
	`, fx.tenantID, fx.tenantDatasourceID, entityID, tableName, columnName)
	require.NoError(t, err)

	return entityID
}

func (fx *junctionFixture) countOwnJunctionRows(t *testing.T, db *sql.DB) int {
	t.Helper()
	var count int
	err := db.QueryRowContext(context.Background(), `
		SELECT COUNT(*) FROM public.entity_relationship
		WHERE relationship_type = 'MANY_TO_MANY_JUNCTION'
		  AND source_entity_id = $1 AND target_entity_id = $2
	`, fx.parentAEntityID, fx.parentBEntityID).Scan(&count)
	require.NoError(t, err)
	return count
}

func (fx *junctionFixture) cleanup(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx := context.Background()
	// Best-effort cleanup; log rather than fail the test if something
	// upstream already failed and left partial state.
	for _, id := range []string{fx.parentAEntityID, fx.parentBEntityID, fx.junctionEntityID} {
		if id == "" {
			continue
		}
		_, _ = db.ExecContext(ctx, `
			DELETE FROM public.entity_relationship
			WHERE source_entity_id = $1 OR target_entity_id = $1
		`, id)
	}
	_, _ = db.ExecContext(ctx, `DELETE FROM public.entity_attribute_column_mapping WHERE table_name = ANY($1)`,
		pqStringArray([]string{fx.parentATable, fx.parentBTable, fx.junctionTable}))
	for _, id := range []string{fx.parentAEntityID, fx.parentBEntityID, fx.junctionEntityID} {
		if id == "" {
			continue
		}
		_, _ = db.ExecContext(ctx, `DELETE FROM public.entity_attribute WHERE id = $1`, id)
	}
	if len(fx.catalogNodeIDs) > 0 {
		_, _ = db.ExecContext(ctx, `
			DELETE FROM public.catalog_edge
			WHERE source_node_id = ANY($1::uuid[]) OR target_node_id = ANY($1::uuid[])
		`, pqStringArray(fx.catalogNodeIDs))
		_, _ = db.ExecContext(ctx, `DELETE FROM public.catalog_node WHERE id = ANY($1::uuid[])`, pqStringArray(fx.catalogNodeIDs))
	}
	_, _ = db.ExecContext(ctx, fmt.Sprintf(`DROP TABLE IF EXISTS public.%s`, fx.junctionTable))
	_, _ = db.ExecContext(ctx, fmt.Sprintf(`DROP TABLE IF EXISTS public.%s`, fx.parentATable))
	_, _ = db.ExecContext(ctx, fmt.Sprintf(`DROP TABLE IF EXISTS public.%s`, fx.parentBTable))
}

// pqStringArray formats a Go string slice as a Postgres text[] literal for
// use with ANY($1); avoids pulling in lib/pq's pq.Array just for cleanup.
func pqStringArray(ss []string) string {
	out := "{"
	for i, s := range ss {
		if i > 0 {
			out += ","
		}
		out += `"` + s + `"`
	}
	return out + "}"
}
