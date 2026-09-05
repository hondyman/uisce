package scanner

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/models"
	"github.com/stretchr/testify/require"
)

// TestResolveFKCardinality_UsesRealConstraints proves the actual live
// scan-time cardinality writer (AnsiScanner.processForeignKeys, wired to
// the live POST /api/catalog/scan route via internal/metadata/catalog_scan_service.go)
// now resolves cardinality from real PK/unique constraints instead of the
// old inferFKCardinality heuristic, which unconditionally returned "N:1"
// for every foreign key regardless of the actual schema — meaning a real
// 1:1 relationship was always reported as N:1 to every consumer reading
// catalog_edge.properties->>'cardinality' (page designer, query builder).
func TestResolveFKCardinality_UsesRealConstraints(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	defer db.Close()
	mock.ExpectPing()

	s, err := NewAnsiScanner(db, uuid.New(), uuid.New(), "test_source", nil, false, nil)
	require.NoError(t, err)
	require.NotNil(t, s.cardinality, "AnsiScanner must construct a cardinality resolver so scans stop guessing N:1 for every FK")

	// source table's FK column IS covered by a unique constraint -> 1:1
	mock.ExpectQuery("SELECT tc.constraint_name, kcu.column_name").
		WithArgs("public", "accounts").
		WillReturnRows(sqlmock.NewRows([]string{"constraint_name", "column_name"}).
			AddRow("accounts_pkey", "id").
			AddRow("accounts_user_uk", "user_id"))
	mock.ExpectQuery("SELECT tc.constraint_name, kcu.column_name").
		WithArgs("public", "users").
		WillReturnRows(sqlmock.NewRows([]string{"constraint_name", "column_name"}).
			AddRow("users_pkey", "id"))

	got := s.resolveFKCardinality("public", "accounts", "public", "users", []map[string]interface{}{
		{"source_column": "user_id", "target_column": "id"},
	})
	require.Equal(t, "ONE_TO_ONE", got, "a real 1:1 FK relationship must resolve to ONE_TO_ONE, not the old hardcoded N:1")
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestResolveFKCardinality_FallsBackWithoutColumns proves the heuristic
// fallback still fires when column info isn't available, rather than
// erroring the whole scan.
func TestResolveFKCardinality_FallsBackWithoutColumns(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	defer db.Close()
	mock.ExpectPing()

	s, err := NewAnsiScanner(db, uuid.New(), uuid.New(), "test_source", nil, false, nil)
	require.NoError(t, err)

	got := s.resolveFKCardinality("public", "orders", "public", "customers", nil)
	require.Equal(t, "N:1", got)
}

// TestResolveFKCardinality_FallsBackOnQueryError proves a resolver query
// failure (e.g. a non-Postgres source whose information_schema behaves
// differently, or a transient connection error) degrades to the old
// heuristic rather than propagating an error — a scan that used to succeed
// (with a merely-imprecise cardinality) must keep succeeding, never abort
// because cardinality resolution failed.
func TestResolveFKCardinality_FallsBackOnQueryError(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	defer db.Close()
	mock.ExpectPing()

	s, err := NewAnsiScanner(db, uuid.New(), uuid.New(), "test_source", nil, false, nil)
	require.NoError(t, err)

	mock.ExpectQuery("SELECT tc.constraint_name, kcu.column_name").
		WithArgs("public", "orders").
		WillReturnError(fmt.Errorf("driver: this data source does not support information_schema.table_constraints"))

	got := s.resolveFKCardinality("public", "orders", "public", "customers", []map[string]interface{}{
		{"source_column": "customer_id", "target_column": "id"},
	})
	require.Equal(t, "N:1", got, "a resolver query error must fall back to the heuristic, not propagate")
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestDetectAndAppendJunctionEdges_EmitsManyToMany proves the scanner's
// junction-detection pass — the fix for the flagship "M:M relationship
// renders as an embedded grid" scenario — actually appends a synthesized
// MANY_TO_MANY edge to s.edges for a real junction table, not just for the
// unrelated FK edges the main relationship loop already creates.
func TestDetectAndAppendJunctionEdges_EmitsManyToMany(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	defer db.Close()
	mock.ExpectPing()

	s, err := NewAnsiScanner(db, uuid.New(), uuid.New(), "test_source", nil, false, nil)
	require.NoError(t, err)

	mock.ExpectQuery("SELECT tc.constraint_name, kcu.column_name, ccu.table_name").
		WithArgs("public", "order_items").
		WillReturnRows(sqlmock.NewRows([]string{"constraint_name", "column_name", "foreign_table", "foreign_column"}).
			AddRow("fk_order", "order_id", "orders", "id").
			AddRow("fk_product", "product_id", "products", "id"))
	mock.ExpectQuery("SELECT tc.constraint_name, kcu.column_name").
		WithArgs("public", "order_items").
		WillReturnRows(sqlmock.NewRows([]string{"constraint_name", "column_name"}).
			AddRow("order_items_pkey", "order_id").
			AddRow("order_items_pkey", "product_id"))

	edgesBefore := len(s.edges)
	s.detectAndAppendJunctionEdges([]scannedTableRef{{schema: "public", table: "order_items"}})

	require.Equal(t, edgesBefore+1, len(s.edges), "expected exactly one synthesized junction edge to be appended")
	newEdge := s.edges[len(s.edges)-1]
	require.Equal(t, "many_to_many_junction", newEdge.EdgeTypeName)

	var props map[string]interface{}
	require.NoError(t, json.Unmarshal(newEdge.Properties, &props))
	require.Equal(t, "MANY_TO_MANY", props["cardinality"])
	require.Equal(t, "order_items", props["junction_table"])
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestDetectAndAppendJunctionEdges_TagsRawEdges proves the two raw FK edges
// (junction -> parentA, junction -> parentB) that the main relationship
// loop creates get tagged with "junction_table" once the junction pattern
// is detected — otherwise a consumer reading catalog_edge can't tell a
// junction-side raw edge apart from an ordinary FK edge, or associate it
// with the synthesized MANY_TO_MANY edge.
func TestDetectAndAppendJunctionEdges_TagsRawEdges(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	defer db.Close()
	mock.ExpectPing()

	tenantDatasourceID := uuid.New()
	s, err := NewAnsiScanner(db, uuid.New(), tenantDatasourceID, "test_source", nil, false, nil)
	require.NoError(t, err)

	junctionTableID := generateID(tenantDatasourceID.String(), "test_source", NODE_TYPE_TABLE.String(), "/public/order_items")
	ordersTableID := generateID(tenantDatasourceID.String(), "test_source", NODE_TYPE_TABLE.String(), "/public/orders")
	productsTableID := generateID(tenantDatasourceID.String(), "test_source", NODE_TYPE_TABLE.String(), "/public/products")

	rawProps, err := json.Marshal(map[string]interface{}{"cardinality": "MANY_TO_ONE", "source_table": "order_items", "target_table": "orders"})
	require.NoError(t, err)
	s.edges = append(s.edges,
		models.CatalogEdge{ID: uuid.New(), SourceNodeID: junctionTableID, TargetNodeID: ordersTableID, Properties: rawProps, EdgeTypeName: "foreign_key"},
		models.CatalogEdge{ID: uuid.New(), SourceNodeID: junctionTableID, TargetNodeID: productsTableID, Properties: rawProps, EdgeTypeName: "foreign_key"},
	)
	edgesBeforeJunctionPass := len(s.edges)

	mock.ExpectQuery("SELECT tc.constraint_name, kcu.column_name, ccu.table_name").
		WithArgs("public", "order_items").
		WillReturnRows(sqlmock.NewRows([]string{"constraint_name", "column_name", "foreign_table", "foreign_column"}).
			AddRow("fk_order", "order_id", "orders", "id").
			AddRow("fk_product", "product_id", "products", "id"))
	mock.ExpectQuery("SELECT tc.constraint_name, kcu.column_name").
		WithArgs("public", "order_items").
		WillReturnRows(sqlmock.NewRows([]string{"constraint_name", "column_name"}).
			AddRow("order_items_pkey", "order_id").
			AddRow("order_items_pkey", "product_id"))

	s.detectAndAppendJunctionEdges([]scannedTableRef{{schema: "public", table: "order_items"}})

	require.Equal(t, edgesBeforeJunctionPass+1, len(s.edges), "raw edges must not be duplicated or removed, only tagged")

	for _, edge := range s.edges[:edgesBeforeJunctionPass] {
		var props map[string]interface{}
		require.NoError(t, json.Unmarshal(edge.Properties, &props))
		require.Equal(t, "order_items", props["junction_table"], "raw junction-side edge %s must be tagged with junction_table", edge.ID)
	}
	require.NoError(t, mock.ExpectationsWereMet())
}
