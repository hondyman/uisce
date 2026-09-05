package api

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/stretchr/testify/require"
)

// TestDetectJunctionTable_NorthwindOrderDetails exercises DetectJunctionTable
// against the real order_details/orders/products shape from the Northwind
// fixture used elsewhere in this repo
// (backend/migrations/20260902_northwind_customer_cashflows_fixture.sql,
// backend/schema-drift.diff): order_details_pkey PRIMARY KEY (order_id,
// product_id).
//
// NOTE: this is an sqlmock simulation of that shape, not a live run against
// a seeded Postgres instance — no live database was available to run this
// against. It also assumes order_details carries FOREIGN KEY constraints to
// orders and products; per backend/schema-drift.diff, the fixture's actual
// order_details table currently has NO such constraints (only its composite
// PK), and products.product_id is uuid while order_details.product_id is
// integer — a type mismatch that would make a real FK invalid even if one
// were added. So DetectJunctionTable correctly returns nil against the
// fixture as it stands today; this test documents the intended behavior
// once order_details is given valid FK constraints, and should be re-run
// against a live database as part of closing that separate data-modeling
// gap.
func TestDetectJunctionTable_NorthwindOrderDetails(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery("SELECT tc.constraint_name, kcu.column_name, ccu.table_name").
		WithArgs("public", "order_details").
		WillReturnRows(sqlmock.NewRows([]string{"constraint_name", "column_name", "foreign_table", "foreign_column"}).
			AddRow("order_details_order_id_fkey", "order_id", "orders", "id").
			AddRow("order_details_product_id_fkey", "product_id", "products", "product_id"))

	mock.ExpectQuery("SELECT tc.constraint_name, kcu.column_name").
		WithArgs("public", "order_details").
		WillReturnRows(constraintRows(map[string][]string{
			"order_details_pkey": {"order_id", "product_id"},
		}))

	r := NewCardinalityResolver(db)
	junction, err := r.DetectJunctionTable(context.Background(), "public", "order_details")
	require.NoError(t, err)
	require.NotNil(t, junction)
	require.ElementsMatch(t, []string{junction.ParentA, junction.ParentB}, []string{"orders", "products"})
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestParseCardinality_NorthwindDefaultBeforeFix(t *testing.T) {
	// businessobject_service.go's RelationshipResult query used to always
	// default missing cardinality to the loose '1:N' string; confirm that
	// value still normalizes correctly through the shared parser so old
	// rows read through the new code path don't regress.
	require.Equal(t, models.CardinalityOneToMany, models.ParseCardinality("1:N"))
}
