package api

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

// TestDetectJunctionTable_CanonicalOrderRegardlessOfScanOrder proves the
// fix for a real cross-producer risk flagged in review: two independent
// producers can call DetectJunctionTable for the same junction table (the
// discovery engine's discoverJunctionRelationship, which persists to
// catalog_edge, and the backfill runner's BackfillJunctionRelationships,
// which persists to entity_relationship). Both tables' uniqueness/conflict
// targets are directional — (tenant_datasource_id, source_node_id,
// edge_type_id, target_node_id) for catalog_edge, (tenant_datasource_id,
// source_entity_id, target_entity_id, relationship_type) for
// entity_relationship — so if ParentA/ParentB weren't canonically ordered,
// a rescan (or a second producer) that happened to observe the underlying
// FK constraints in a different row order could emit a mirrored (B,A) row
// alongside an existing (A,B) row, since neither table's constraint
// catches the reverse pair as a duplicate.
//
// This test forces DetectJunctionTable to see its two FK rows in reversed
// order across two calls and asserts ParentA/ParentB still land the same
// way both times — proving the fix holds regardless of scan/row order, not
// just regardless of which caller invokes it.
func TestDetectJunctionTable_CanonicalOrderRegardlessOfScanOrder(t *testing.T) {
	// Constraint names are deliberately chosen so their alphabetical order
	// is the OPPOSITE of the target tables' alphabetical order
	// ("aaa_..." -> products, "zzz_..." -> orders). This is what makes the
	// test discriminating: sort.Strings(order) alone (the pre-fix
	// behavior) would pick ParentA/ParentB by constraint-name order, which
	// here disagrees with table-name order — so a version of
	// DetectJunctionTable without the table-name canonicalization swap
	// would report ParentA="products" here, failing the table-name
	// assertions below. (Constraint names like "fk_order"/"fk_product"
	// happen to already sort the same as their tables and would NOT catch
	// a regression — verified by temporarily reverting the fix locally.)
	scenarios := []struct {
		name    string
		fkOrder [][4]string // constraint_name, column_name, foreign_table, foreign_column
	}{
		{
			name: "constraint scan order: products-pointing constraint first",
			fkOrder: [][4]string{
				{"aaa_constraint_to_products", "product_id", "products", "id"},
				{"zzz_constraint_to_orders", "order_id", "orders", "id"},
			},
		},
		{
			name: "constraint scan order: orders-pointing constraint first",
			fkOrder: [][4]string{
				{"zzz_constraint_to_orders", "order_id", "orders", "id"},
				{"aaa_constraint_to_products", "product_id", "products", "id"},
			},
		},
	}

	var results []*JunctionTable
	for _, sc := range scenarios {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)

		fkRows := sqlmock.NewRows([]string{"constraint_name", "column_name", "foreign_table", "foreign_column"})
		for _, r := range sc.fkOrder {
			fkRows.AddRow(r[0], r[1], r[2], r[3])
		}
		mock.ExpectQuery("SELECT tc.constraint_name, kcu.column_name, ccu.table_name").
			WithArgs("public", "order_items").
			WillReturnRows(fkRows)

		mock.ExpectQuery("SELECT tc.constraint_name, kcu.column_name").
			WithArgs("public", "order_items").
			WillReturnRows(constraintRows(map[string][]string{
				"order_items_pkey": {"order_id", "product_id"},
			}))

		r := NewCardinalityResolver(db)
		junction, err := r.DetectJunctionTable(context.Background(), "public", "order_items")
		require.NoError(t, err, sc.name)
		require.NotNil(t, junction, sc.name)
		results = append(results, junction)
		require.NoError(t, mock.ExpectationsWereMet(), sc.name)
		db.Close()
	}

	require.Equal(t, results[0].ParentA, results[1].ParentA,
		"ParentA must be the same table regardless of FK constraint scan order")
	require.Equal(t, results[0].ParentB, results[1].ParentB,
		"ParentB must be the same table regardless of FK constraint scan order")
	// Also pin the actual canonical direction (lexicographic table name),
	// so a future change to the ordering rule fails loudly here.
	require.Equal(t, "orders", results[0].ParentA)
	require.Equal(t, "products", results[0].ParentB)
}
