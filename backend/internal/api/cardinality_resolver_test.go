package api

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/stretchr/testify/require"
)

// constraintRows builds the (constraint_name, column_name) rows returned by
// isColumnSetUnique's query for a table with the given PK/UNIQUE constraints.
func constraintRows(constraints map[string][]string) *sqlmock.Rows {
	rows := sqlmock.NewRows([]string{"constraint_name", "column_name"})
	for name, cols := range constraints {
		for _, c := range cols {
			rows.AddRow(name, c)
		}
	}
	return rows
}

func TestResolveEdgeCardinality_OneToOne(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// source table (orders): FK column "customer_id" IS covered by a UNIQUE constraint
	mock.ExpectQuery("SELECT tc.constraint_name, kcu.column_name").
		WithArgs("public", "orders").
		WillReturnRows(constraintRows(map[string][]string{
			"orders_pkey":        {"id"},
			"orders_customer_uk": {"customer_id"},
		}))

	// target table (customers): FK column "id" IS covered by its PK
	mock.ExpectQuery("SELECT tc.constraint_name, kcu.column_name").
		WithArgs("public", "customers").
		WillReturnRows(constraintRows(map[string][]string{
			"customers_pkey": {"id"},
		}))

	r := NewCardinalityResolver(db)
	got, err := r.ResolveEdgeCardinality(context.Background(),
		"public", "orders", []string{"customer_id"},
		"public", "customers", []string{"id"},
	)
	require.NoError(t, err)
	require.Equal(t, models.CardinalityOneToOne, got)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestResolveEdgeCardinality_ManyToOne(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// source table (order_items): FK column "order_id" NOT covered by any unique constraint alone
	mock.ExpectQuery("SELECT tc.constraint_name, kcu.column_name").
		WithArgs("public", "order_items").
		WillReturnRows(constraintRows(map[string][]string{
			"order_items_pkey": {"id"},
		}))

	// target table (orders): FK column "id" covered by PK
	mock.ExpectQuery("SELECT tc.constraint_name, kcu.column_name").
		WithArgs("public", "orders").
		WillReturnRows(constraintRows(map[string][]string{
			"orders_pkey": {"id"},
		}))

	r := NewCardinalityResolver(db)
	got, err := r.ResolveEdgeCardinality(context.Background(),
		"public", "order_items", []string{"order_id"},
		"public", "orders", []string{"id"},
	)
	require.NoError(t, err)
	require.Equal(t, models.CardinalityManyToOne, got)
	require.Equal(t, models.CardinalityOneToMany, got.Inverse())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDetectJunctionTable_ClassicManyToMany(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// order_items has two FKs (order_id -> orders.id, product_id -> products.id),
	// and its own PK is the composite (order_id, product_id).
	mock.ExpectQuery("SELECT tc.constraint_name, kcu.column_name, ccu.table_name").
		WithArgs("public", "order_items").
		WillReturnRows(sqlmock.NewRows([]string{"constraint_name", "column_name", "foreign_table", "foreign_column"}).
			AddRow("fk_order", "order_id", "orders", "id").
			AddRow("fk_product", "product_id", "products", "id"))

	// junction-key coverage check: order_items' PK covers (order_id, product_id)
	mock.ExpectQuery("SELECT tc.constraint_name, kcu.column_name").
		WithArgs("public", "order_items").
		WillReturnRows(constraintRows(map[string][]string{
			"order_items_pkey": {"order_id", "product_id"},
		}))

	r := NewCardinalityResolver(db)
	junction, err := r.DetectJunctionTable(context.Background(), "public", "order_items")
	require.NoError(t, err)
	require.NotNil(t, junction)
	require.ElementsMatch(t, []string{junction.ParentA, junction.ParentB}, []string{"orders", "products"})
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDetectJunctionTable_NotAJunction(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// A plain child table with only one FK isn't a junction table.
	mock.ExpectQuery("SELECT tc.constraint_name, kcu.column_name, ccu.table_name").
		WithArgs("public", "orders").
		WillReturnRows(sqlmock.NewRows([]string{"constraint_name", "column_name", "foreign_table", "foreign_column"}).
			AddRow("fk_customer", "customer_id", "customers", "id"))

	r := NewCardinalityResolver(db)
	junction, err := r.DetectJunctionTable(context.Background(), "public", "orders")
	require.NoError(t, err)
	require.Nil(t, junction)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestParseCardinality_LegacyStrings(t *testing.T) {
	cases := map[string]models.Cardinality{
		"one-to-many":  models.CardinalityOneToMany,
		"1:N":          models.CardinalityOneToMany,
		"MANY_TO_MANY": models.CardinalityManyToMany,
		"n:m":          models.CardinalityManyToMany,
		"1:1":          models.CardinalityOneToOne,
		"bogus":        models.CardinalityUnknown,
	}
	for in, want := range cases {
		require.Equal(t, want, models.ParseCardinality(in), "input=%s", in)
	}
}
