package api

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/stretchr/testify/require"
)

// TestDiscoverJunctionRelationship_EmitsManyToMany verifies that
// ForeignKeyDiscoveryEngine actually calls DetectJunctionTable and, when a
// junction table is found with both parent entities registered, emits a
// synthesized MANY_TO_MANY EntityRelationshipFromFK — the gap flagged in
// review: DetectJunctionTable existed and was unit-tested, but nothing in
// the discovery engine ever called it.
func TestDiscoverJunctionRelationship_EmitsManyToMany(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	tenantID := "22222222-2222-2222-2222-222222222222"
	datasourceID := "33333333-3333-3333-3333-333333333333"

	// DetectJunctionTable: order_items has two FKs (order_id -> orders,
	// product_id -> products) whose union is covered by its own PK.
	mock.ExpectQuery("SELECT tc.constraint_name, kcu.column_name, ccu.table_name").
		WithArgs("public", "order_items").
		WillReturnRows(sqlmock.NewRows([]string{"constraint_name", "column_name", "foreign_table", "foreign_column"}).
			AddRow("fk_order", "order_id", "orders", "id").
			AddRow("fk_product", "product_id", "products", "id"))

	mock.ExpectQuery("SELECT tc.constraint_name, kcu.column_name").
		WithArgs("public", "order_items").
		WillReturnRows(constraintRows(map[string][]string{
			"order_items_pkey": {"order_id", "product_id"},
		}))

	// findEntityByBackingTable("orders")
	mock.ExpectQuery("SELECT (.+) FROM public.entities e").
		WithArgs("orders", tenantID, datasourceID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "description", "created_at"}).
			AddRow("44444444-4444-4444-4444-444444444444", "Orders", "", time.Now()))

	// findEntityByBackingTable("products")
	mock.ExpectQuery("SELECT (.+) FROM public.entities e").
		WithArgs("products", tenantID, datasourceID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "description", "created_at"}).
			AddRow("55555555-5555-5555-5555-555555555555", "Products", "", time.Now()))

	engine := NewForeignKeyDiscoveryEngine(db)
	rel, err := engine.discoverJunctionRelationship(context.Background(), tenantID, datasourceID, "order_items")
	require.NoError(t, err)
	require.NotNil(t, rel)
	require.Equal(t, string(models.CardinalityManyToMany), rel.Cardinality)
	require.Equal(t, "fk_junction", rel.DiscoveryCode)
	require.Equal(t, "Orders", rel.SourceEntityName)
	require.Equal(t, "Products", rel.TargetEntityName)
	require.Equal(t, "order_items", rel.EdgeProperties["junction_table"])
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestDiscoverJunctionRelationship_NotAJunction verifies a plain child
// table (single FK) produces no synthesized relationship.
func TestDiscoverJunctionRelationship_NotAJunction(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery("SELECT tc.constraint_name, kcu.column_name, ccu.table_name").
		WithArgs("public", "orders").
		WillReturnRows(sqlmock.NewRows([]string{"constraint_name", "column_name", "foreign_table", "foreign_column"}).
			AddRow("fk_customer", "customer_id", "customers", "id"))

	engine := NewForeignKeyDiscoveryEngine(db)
	rel, err := engine.discoverJunctionRelationship(context.Background(), "t", "d", "orders")
	require.NoError(t, err)
	require.Nil(t, rel)
	require.NoError(t, mock.ExpectationsWereMet())
}
