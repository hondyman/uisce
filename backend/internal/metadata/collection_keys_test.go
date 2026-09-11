package metadata

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

// TestCollectionKeyDriftGuard verifies that every collection key which
// analytics.ListSemanticFields advertises (via appendCollectionFields ->
// models.CollectionKeysForBO) is actually delivered by the corresponding
// BO context loader. If a second developer adds a collection key to
// models.CollectionKeysForBO but forgets to implement delivery in
// loadOrderContext, this test fails — closing the drift at the point
// of authorship rather than at the editor's autocomplete render.
func TestCollectionKeyDriftGuard(t *testing.T) {
	t.Run("order BO delivers every collection key it advertises", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		orderID := "test-order-001"
		targetQty := 100.0

		// loadOrderContext makes multiple queries; set up all of them.
		// Missing queries produce warnings but the function continues,
		// so we check only that the collection key is delivered.

		// 1. SELECT COALESCE(SUM(target_qty), 0) FROM orm.order_allocation
		sumRows := sqlmock.NewRows([]string{"coalesce"}).AddRow(0)
		mock.ExpectQuery("SELECT COALESCE").
			WithArgs(orderID).
			WillReturnRows(sumRows)

		// 2. SELECT ... FROM orm.order_allocation (collection rows)
		allocRows := sqlmock.NewRows([]string{"target_qty", "allocated_qty", "status", "created_at"}).
			AddRow(targetQty, targetQty, "PENDING", "2026-01-01T00:00:00Z")
		mock.ExpectQuery("SELECT target_qty, allocated_qty, status, created_at FROM orm.order_allocation").
			WithArgs(orderID).
			WillReturnRows(allocRows)

		// 3. SELECT COALESCE(SUM(routed_qty), 0) FROM orm.placement
		routedRows := sqlmock.NewRows([]string{"coalesce"}).AddRow(0)
		mock.ExpectQuery("SELECT COALESCE.+FROM orm.placement").
			WithArgs(orderID).
			WillReturnRows(routedRows)

		// 4. SELECT a.status, a.is_discretionary FROM orm.order_allocation oa JOIN orm.account a
		accountRows := sqlmock.NewRows([]string{"status", "is_discretionary"}).AddRow("ACTIVE", true)
		mock.ExpectQuery("SELECT a.status, a.is_discretionary.+FROM orm.order_allocation oa JOIN orm.account a").
			WithArgs(orderID).
			WillReturnRows(accountRows)

		xdb := sqlx.NewDb(mockDB, "postgres")
		svc := &BusinessObjectService{db: xdb}
		data := map[string]interface{}{
			"id":         orderID,
			"target_qty": targetQty,
		}

		svc.loadOrderContext(context.Background(), xdb, data)

		for _, key := range models.CollectionKeysForBO("order") {
			val, ok := data[key]
			if !ok {
				t.Errorf("collection key %q advertised by models.CollectionKeysForBO for BO %q "+
					"was not delivered by loadOrderContext — drift detected", key, "order")
				continue
			}
			// loadOrderContext stores []map[string]interface{}; this is the
			// canonical type the evaluator receives and the type assertion in
			// evalDottedFieldRef handles.
			slice, ok := val.([]map[string]interface{})
			if !ok {
				t.Errorf("collection key %q has wrong type %T — expected []map[string]interface{}", key, val)
				continue
			}
			_ = slice
		}
	})

	t.Run("empty collection delivered when no allocation rows exist", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		orderID := "test-order-empty"

		sumRows := sqlmock.NewRows([]string{"coalesce"}).AddRow(0)
		mock.ExpectQuery("SELECT COALESCE").
			WithArgs(orderID).
			WillReturnRows(sumRows)

		mock.ExpectQuery("SELECT target_qty, allocated_qty, status, created_at FROM orm.order_allocation").
			WithArgs(orderID).
			WillReturnRows(sqlmock.NewRows([]string{"target_qty", "allocated_qty", "status", "created_at"}))

		routedRows := sqlmock.NewRows([]string{"coalesce"}).AddRow(0)
		mock.ExpectQuery("SELECT COALESCE.+FROM orm.placement").
			WithArgs(orderID).
			WillReturnRows(routedRows)

		mock.ExpectQuery("SELECT a.status, a.is_discretionary.+FROM orm.order_allocation oa JOIN orm.account a").
			WithArgs(orderID).
			WillReturnRows(sqlmock.NewRows([]string{"status", "is_discretionary"}).AddRow("ACTIVE", true))

		xdb := sqlx.NewDb(mockDB, "postgres")
		svc := &BusinessObjectService{db: xdb}
		data := map[string]interface{}{
			"id":         orderID,
			"target_qty": 100.0,
		}

		svc.loadOrderContext(context.Background(), xdb, data)

		val, ok := data["OrderAllocations"]
		if !ok {
			t.Fatal("OrderAllocations not present in data after loadOrderContext")
		}
		slice, ok := val.([]map[string]interface{})
		if !ok {
			t.Fatalf("OrderAllocations has wrong type %T, expected []map[string]interface{}", val)
		}
		if len(slice) != 0 {
			t.Errorf("OrderAllocations should be empty slice for order with no allocations, got len=%d", len(slice))
		}
	})
}
