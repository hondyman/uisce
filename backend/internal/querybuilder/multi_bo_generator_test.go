package querybuilder

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/hondyman/uisce/backend/internal/boresolver"
)

func TestBuildMultiBOSQL_JoinsAndTenantScoping(t *testing.T) {
	primary := &boresolver.BODefinition{
		ID:           "bo-account",
		DrivingTable: "account",
		Fields: []boresolver.BOField{
			{ID: "f1", Name: "account_id", Type: "string", PhysicalColumn: "account.account_id"},
		},
	}
	related := &boresolver.BODefinition{
		ID:           "bo-household",
		DrivingTable: "household",
		Fields: []boresolver.BOField{
			{ID: "f2", Name: "household_name", Type: "string", PhysicalColumn: "household.household_name"},
		},
	}

	qd := &boresolver.QueryDef{
		Context: boresolver.QueryContext{
			BOID:         "bo-account",
			RelatedBOIDs: []string{"bo-household"},
		},
		Query: boresolver.QueryRequest{
			Dimensions: []boresolver.DimensionDef{
				{TermNodeID: "account_id", Alias: "Account", BOID: "bo-account"},
				{TermNodeID: "household_name", Alias: "Household", BOID: "bo-household"},
			},
			Limit: 50,
		},
	}

	path := &analytics.JoinPath{
		Steps: []analytics.JoinPathStep{
			{
				LeftTable: "account", LeftAlias: "t0", LeftColumn: "household_id",
				RightTable: "household", RightAlias: "t1", RightColumn: "id",
				JoinType: "LEFT", Cardinality: "M:1",
			},
		},
	}

	gen, _ := boresolver.NewBOSQLGenerator(nil, "postgres")
	sql, args, columns, err := buildMultiBOSQL(gen, primary, []joinedBO{
		{BOID: "bo-household", BODef: related, Path: path, Cardinality: path.TraversalCardinality()},
	}, qd, "tenant-123")
	if err != nil {
		t.Fatalf("buildMultiBOSQL failed: %v", err)
	}

	if !strings.Contains(sql, "FROM account AS t0") {
		t.Errorf("expected base table t0, got: %s", sql)
	}
	if !strings.Contains(sql, "LEFT JOIN household AS t1 ON t0.household_id = t1.id") {
		t.Errorf("expected join clause, got: %s", sql)
	}
	if !strings.Contains(sql, "t0.tenant_id = $") || !strings.Contains(sql, "t1.tenant_id = $") {
		t.Errorf("expected tenant scoping on every joined table, got: %s", sql)
	}
	if len(args) != 2 {
		t.Errorf("expected 2 tenant-scoping args, got %d: %v", len(args), args)
	}
	if len(columns) != 2 || columns[1].Cardinality != "one" {
		t.Fatalf("expected household_name column cardinality 'one' (M:1 join), got: %+v", columns)
	}
	if columns[0].BOID != "bo-account" || columns[1].BOID != "bo-household" {
		t.Errorf("expected columns tagged with their source BOID, got: %+v", columns)
	}
}

// TestBuildMultiBOSQL_AggregatesManySideToPreventFanOut covers the bug
// this generator used to have: selecting a bare column from BOTH the
// primary ("one") side and a "many"-cardinality related BO (order -> its
// many allocations) produced a flat LEFT JOIN, repeating every order
// column once per allocation row instead of aggregating. The fix must
// GROUP BY the one-side column and auto-aggregate the many-side one.
func TestBuildMultiBOSQL_AggregatesManySideToPreventFanOut(t *testing.T) {
	primary := &boresolver.BODefinition{
		ID:           "bo-order",
		DrivingTable: "order",
		Fields: []boresolver.BOField{
			{ID: "f1", Name: "order_id", Type: "string", PhysicalColumn: "order.id"},
		},
	}
	related := &boresolver.BODefinition{
		ID:           "bo-allocation",
		DrivingTable: "allocation",
		Fields: []boresolver.BOField{
			{ID: "f2", Name: "allocated_qty", Type: "number", PhysicalColumn: "allocation.qty"},
		},
	}

	qd := &boresolver.QueryDef{
		Context: boresolver.QueryContext{
			BOID:         "bo-order",
			RelatedBOIDs: []string{"bo-allocation"},
		},
		Query: boresolver.QueryRequest{
			Dimensions: []boresolver.DimensionDef{
				{TermNodeID: "order_id", Alias: "OrderID", BOID: "bo-order"},
			},
			Measures: []boresolver.MeasureDef{
				{TermNodeID: "allocated_qty", Alias: "AllocatedQty", BOID: "bo-allocation"},
			},
		},
	}

	path := &analytics.JoinPath{
		Steps: []analytics.JoinPathStep{
			{
				LeftTable: "order", LeftAlias: "t0", LeftColumn: "id",
				RightTable: "allocation", RightAlias: "t1", RightColumn: "order_id",
				JoinType: "LEFT", Cardinality: "1:M",
			},
		},
	}

	gen, _ := boresolver.NewBOSQLGenerator(nil, "postgres")
	sql, _, columns, err := buildMultiBOSQL(gen, primary, []joinedBO{
		{BOID: "bo-allocation", BODef: related, Path: path, Cardinality: path.TraversalCardinality()},
	}, qd, "tenant-123")
	if err != nil {
		t.Fatalf("buildMultiBOSQL failed: %v", err)
	}

	if !strings.Contains(sql, "SUM(t1.qty) AS \"AllocatedQty\"") {
		t.Errorf("expected the many-side measure to be auto-wrapped in SUM, got: %s", sql)
	}
	if !strings.Contains(sql, "GROUP BY t0.id") {
		t.Errorf("expected GROUP BY on the one-side column, got: %s", sql)
	}
	if columns[1].Cardinality != "many" {
		t.Fatalf("expected allocated_qty column cardinality 'many', got: %+v", columns[1])
	}
}

// TestBuildMultiBOSQL_ManySideOnlyNoGroupBy confirms a query that selects
// ONLY many-side columns (a legitimate "list this order's allocations")
// is left as a plain join with no GROUP BY - there's nothing to
// deduplicate since no one-side value is being repeated.
func TestBuildMultiBOSQL_ManySideOnlyNoGroupBy(t *testing.T) {
	primary := &boresolver.BODefinition{
		ID:           "bo-order",
		DrivingTable: "order",
		Fields:       []boresolver.BOField{{ID: "f1", Name: "order_id", Type: "string", PhysicalColumn: "order.id"}},
	}
	related := &boresolver.BODefinition{
		ID:           "bo-allocation",
		DrivingTable: "allocation",
		Fields:       []boresolver.BOField{{ID: "f2", Name: "allocated_qty", Type: "number", PhysicalColumn: "allocation.qty"}},
	}
	qd := &boresolver.QueryDef{
		Context: boresolver.QueryContext{BOID: "bo-order", RelatedBOIDs: []string{"bo-allocation"}},
		Query: boresolver.QueryRequest{
			Dimensions: []boresolver.DimensionDef{{TermNodeID: "allocated_qty", Alias: "Qty", BOID: "bo-allocation"}},
		},
	}
	path := &analytics.JoinPath{Steps: []analytics.JoinPathStep{
		{LeftTable: "order", LeftAlias: "t0", LeftColumn: "id", RightTable: "allocation", RightAlias: "t1", RightColumn: "order_id", JoinType: "LEFT", Cardinality: "1:M"},
	}}

	gen, _ := boresolver.NewBOSQLGenerator(nil, "postgres")
	sql, _, _, err := buildMultiBOSQL(gen, primary, []joinedBO{
		{BOID: "bo-allocation", BODef: related, Path: path, Cardinality: path.TraversalCardinality()},
	}, qd, "tenant-123")
	if err != nil {
		t.Fatalf("buildMultiBOSQL failed: %v", err)
	}
	if strings.Contains(sql, "GROUP BY") {
		t.Errorf("expected no GROUP BY for a many-side-only listing query, got: %s", sql)
	}
	if strings.Contains(sql, "SUM(") {
		t.Errorf("expected no auto-aggregation for a many-side-only listing query, got: %s", sql)
	}
}

// TestBuildMultiBOSQL_RejectsMultiBranchFanOut is the regression test for
// the double-counting bug this generator had: selecting measures from TWO
// independently "many"-cardinality related BOs at once (order -> its many
// line_items, AND order -> its many shipments) used to flatten both into
// one JOIN, cross-multiplying the root's rows before any aggregation ran.
// With 3 line_items and 2 shipments, that produced 3*2=6 joined rows for
// one order, so SUM(line_items.amount) counted every line item once per
// shipment (double- or triple-counted) instead of once - a silent wrong
// number, not a crash. This must now fail loudly with ErrUnsupportedFanOut
// instead of emitting that query.
func TestBuildMultiBOSQL_RejectsMultiBranchFanOut(t *testing.T) {
	primary := &boresolver.BODefinition{
		ID:           "bo-order",
		DrivingTable: "order",
		Fields: []boresolver.BOField{
			{ID: "f1", Name: "order_id", Type: "string", PhysicalColumn: "order.id"},
		},
	}
	lineItems := &boresolver.BODefinition{
		ID:           "bo-line-item",
		DrivingTable: "line_items",
		Fields: []boresolver.BOField{
			{ID: "f2", Name: "amount", Type: "number", PhysicalColumn: "line_items.amount"},
		},
	}
	shipments := &boresolver.BODefinition{
		ID:           "bo-shipment",
		DrivingTable: "shipments",
		Fields: []boresolver.BOField{
			{ID: "f3", Name: "shipment_id", Type: "string", PhysicalColumn: "shipments.id"},
		},
	}

	qd := &boresolver.QueryDef{
		Context: boresolver.QueryContext{
			BOID:         "bo-order",
			RelatedBOIDs: []string{"bo-line-item", "bo-shipment"},
		},
		Query: boresolver.QueryRequest{
			Measures: []boresolver.MeasureDef{
				{TermNodeID: "amount", Alias: "TotalAmount", BOID: "bo-line-item", Aggregation: "SUM"},
				{TermNodeID: "shipment_id", Alias: "ShipmentCount", BOID: "bo-shipment", Aggregation: "COUNT"},
			},
		},
	}

	lineItemPath := &analytics.JoinPath{Steps: []analytics.JoinPathStep{
		{LeftTable: "order", LeftAlias: "t0", LeftColumn: "id", RightTable: "line_items", RightAlias: "t1", RightColumn: "order_id", JoinType: "LEFT", Cardinality: "1:M"},
	}}
	shipmentPath := &analytics.JoinPath{Steps: []analytics.JoinPathStep{
		{LeftTable: "order", LeftAlias: "t0", LeftColumn: "id", RightTable: "shipments", RightAlias: "t2", RightColumn: "order_id", JoinType: "LEFT", Cardinality: "1:M"},
	}}

	gen, _ := boresolver.NewBOSQLGenerator(nil, "postgres")
	sql, args, columns, err := buildMultiBOSQL(gen, primary, []joinedBO{
		{BOID: "bo-line-item", BODef: lineItems, Path: lineItemPath, Cardinality: lineItemPath.TraversalCardinality()},
		{BOID: "bo-shipment", BODef: shipments, Path: shipmentPath, Cardinality: shipmentPath.TraversalCardinality()},
	}, qd, "tenant-123")

	if err == nil {
		t.Fatalf("expected ErrUnsupportedFanOut, got a query instead: sql=%q args=%v columns=%v", sql, args, columns)
	}
	var fanOutErr *ErrUnsupportedFanOut
	if !errors.As(err, &fanOutErr) {
		t.Fatalf("expected *ErrUnsupportedFanOut, got %T: %v", err, err)
	}
	if !strings.Contains(fanOutErr.Error(), "bo-line-item") || !strings.Contains(fanOutErr.Error(), "bo-shipment") {
		t.Fatalf("expected error to name both branches, got: %v", fanOutErr)
	}
	if sql != "" || args != nil || columns != nil {
		t.Fatalf("expected no partial query on the error path, got sql=%q args=%v columns=%v", sql, args, columns)
	}
}

// TestBuildMultiBOSQL_TagsSharedOwnership is the regression test for the
// hazard the cardinality trace's Finding 1/3 identified: a lookup column
// (Cardinality "one" - no fan-out, no aggregation triggers, the flat join
// looks completely unremarkable) can still have shared root ownership if
// any hop back to the primary BO is M:1/M:M. Per-row output is correct;
// the danger is a caller summing/averaging this column across returned
// rows, which double-counts whenever two rows share the same zone. This
// column carries no measure - it's the bare "warehouse_zone_name" from
// the cardinality trace's shape-A example (two M:1 hops), deliberately
// not the customer/region story already worked through by hand elsewhere.
func TestBuildMultiBOSQL_TagsSharedOwnership(t *testing.T) {
	primary := &boresolver.BODefinition{
		ID:           "bo-order",
		DrivingTable: "orders",
		Fields:       []boresolver.BOField{{ID: "f1", Name: "order_id", Type: "string", PhysicalColumn: "orders.id"}},
	}
	warehouse := &boresolver.BODefinition{
		ID:           "bo-warehouse-zone",
		DrivingTable: "zones",
		Fields:       []boresolver.BOField{{ID: "f2", Name: "zone_name", Type: "string", PhysicalColumn: "zones.name"}},
	}

	qd := &boresolver.QueryDef{
		Context: boresolver.QueryContext{BOID: "bo-order", RelatedBOIDs: []string{"bo-warehouse-zone"}},
		Query: boresolver.QueryRequest{
			Dimensions: []boresolver.DimensionDef{
				{TermNodeID: "order_id", Alias: "OrderID", BOID: "bo-order"},
				{TermNodeID: "zone_name", Alias: "ZoneName", BOID: "bo-warehouse-zone"},
			},
		},
	}

	// orders -(M:1)-> warehouses -(M:1)-> zones: shape A. Two different
	// warehouses can share the same zone, and two different orders can
	// share the same warehouse - either hop alone makes this "shared".
	path := &analytics.JoinPath{Steps: []analytics.JoinPathStep{
		{LeftTable: "orders", LeftAlias: "t0", LeftColumn: "warehouse_id", RightTable: "warehouses", RightAlias: "t1", RightColumn: "id", JoinType: "LEFT", Cardinality: "M:1"},
		{LeftTable: "warehouses", LeftAlias: "t1", LeftColumn: "zone_id", RightTable: "zones", RightAlias: "t2", RightColumn: "id", JoinType: "LEFT", Cardinality: "M:1"},
	}}

	gen, _ := boresolver.NewBOSQLGenerator(nil, "postgres")
	_, _, columns, err := buildMultiBOSQL(gen, primary, []joinedBO{
		{BOID: "bo-warehouse-zone", BODef: warehouse, Path: path, Cardinality: path.TraversalCardinality()},
	}, qd, "tenant-123")
	if err != nil {
		t.Fatalf("buildMultiBOSQL failed: %v", err)
	}

	if columns[1].Cardinality != "one" {
		t.Fatalf("expected zone_name Cardinality 'one' (no fan-out - both hops are to-one going down), got: %+v", columns[1])
	}
	if columns[1].RootOwnership != "shared" {
		t.Fatalf("expected zone_name RootOwnership 'shared' (M:1 hops going up), got: %+v", columns[1])
	}
	// The primary BO's own column is trivially its own unique owner.
	if columns[0].RootOwnership != "unique" && columns[0].RootOwnership != "" {
		t.Fatalf("expected the primary column's ownership to be unique (or the empty-means-unique default), got: %q", columns[0].RootOwnership)
	}
}

// TestBuildMultiBOSQL_TagsSharedOwnership_WireJSON pins the same shape-A
// scenario above through the ACTUAL serialization boundary a caller would
// see - the exact map literal HandleGetPreview builds - rather than
// asserting on the Go struct fields alone. A previous round demonstrated
// this via a temporary, uncommitted scratch test; this is that
// demonstration made permanent and reproducible, since "I ran it once and
// it printed the right JSON" is not evidence once the file is deleted.
func TestBuildMultiBOSQL_TagsSharedOwnership_WireJSON(t *testing.T) {
	primary := &boresolver.BODefinition{
		ID:           "bo-order",
		DrivingTable: "orders",
		Fields:       []boresolver.BOField{{ID: "f1", Name: "order_id", Type: "string", PhysicalColumn: "orders.id"}},
	}
	warehouse := &boresolver.BODefinition{
		ID:           "bo-warehouse-zone",
		DrivingTable: "zones",
		Fields:       []boresolver.BOField{{ID: "f2", Name: "zone_name", Type: "string", PhysicalColumn: "zones.name"}},
	}
	qd := &boresolver.QueryDef{
		Context: boresolver.QueryContext{BOID: "bo-order", RelatedBOIDs: []string{"bo-warehouse-zone"}},
		Query: boresolver.QueryRequest{
			Dimensions: []boresolver.DimensionDef{
				{TermNodeID: "order_id", Alias: "OrderID", BOID: "bo-order"},
				{TermNodeID: "zone_name", Alias: "ZoneName", BOID: "bo-warehouse-zone"},
			},
		},
	}
	path := &analytics.JoinPath{Steps: []analytics.JoinPathStep{
		{LeftTable: "orders", LeftAlias: "t0", LeftColumn: "warehouse_id", RightTable: "warehouses", RightAlias: "t1", RightColumn: "id", JoinType: "LEFT", Cardinality: "M:1"},
		{LeftTable: "warehouses", LeftAlias: "t1", LeftColumn: "zone_id", RightTable: "zones", RightAlias: "t2", RightColumn: "id", JoinType: "LEFT", Cardinality: "M:1"},
	}}

	gen, _ := boresolver.NewBOSQLGenerator(nil, "postgres")
	_, _, columns, err := buildMultiBOSQL(gen, primary, []joinedBO{
		{BOID: "bo-warehouse-zone", BODef: warehouse, Path: path, Cardinality: path.TraversalCardinality()},
	}, qd, "tenant-123")
	if err != nil {
		t.Fatalf("buildMultiBOSQL failed: %v", err)
	}

	// The exact map literal HandleGetPreview builds around resp.Columns -
	// see saved_query_handler.go's HandleGetPreview.
	resp := map[string]interface{}{
		"columns":       columns,
		"hasRelatedBOs": len(qd.Context.RelatedBOIDs) > 0,
	}
	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshaling response: %v", err)
	}
	wire := string(b)

	if !strings.Contains(wire, `"name":"ZoneName"`) {
		t.Fatalf("expected ZoneName column in wire JSON, got: %s", wire)
	}
	if !strings.Contains(wire, `"cardinality":"one"`) {
		t.Fatalf("expected cardinality:one on the wire - the gate's grain signal - got: %s", wire)
	}
	if !strings.Contains(wire, `"rootOwnership":"shared"`) {
		t.Fatalf("expected rootOwnership:shared on the wire - the gate's ownership signal - got: %s", wire)
	}
	if !strings.Contains(wire, `"hasRelatedBOs":true`) {
		t.Fatalf("expected hasRelatedBOs:true (this query has a related BO) on the wire, got: %s", wire)
	}
}

// TestBuildMultiBOSQL_UnresolvedHop_DoesNotBecomeUnique is the negative
// test Finding 1 called for: nothing in buildMultiBOSQL's own plumbing
// (boOwnership map, ownershipOrDefault, QueryResultColumn construction)
// may collapse an "unresolved" path into "unique" on the way out. The
// happy-path test above (TestBuildMultiBOSQL_TagsSharedOwnership) only
// proves a DEFINITE "shared" hop survives - it says nothing about what
// happens to a hop the resolver couldn't classify at all.
func TestBuildMultiBOSQL_UnresolvedHop_DoesNotBecomeUnique(t *testing.T) {
	primary := &boresolver.BODefinition{
		ID:           "bo-order",
		DrivingTable: "orders",
		Fields:       []boresolver.BOField{{ID: "f1", Name: "order_id", Type: "string", PhysicalColumn: "orders.id"}},
	}
	related := &boresolver.BODefinition{
		ID:           "bo-warehouse",
		DrivingTable: "warehouses",
		Fields:       []boresolver.BOField{{ID: "f2", Name: "warehouse_name", Type: "string", PhysicalColumn: "warehouses.name"}},
	}

	qd := &boresolver.QueryDef{
		Context: boresolver.QueryContext{BOID: "bo-order", RelatedBOIDs: []string{"bo-warehouse"}},
		Query: boresolver.QueryRequest{
			Dimensions: []boresolver.DimensionDef{
				{TermNodeID: "order_id", Alias: "OrderID", BOID: "bo-order"},
				{TermNodeID: "warehouse_name", Alias: "WarehouseName", BOID: "bo-warehouse"},
			},
		},
	}

	// The resolver couldn't classify this hop's cardinality - an empty
	// string, not one of the four known values.
	path := &analytics.JoinPath{Steps: []analytics.JoinPathStep{
		{LeftTable: "orders", LeftAlias: "t0", LeftColumn: "warehouse_id", RightTable: "warehouses", RightAlias: "t1", RightColumn: "id", JoinType: "LEFT", Cardinality: ""},
	}}

	gen, _ := boresolver.NewBOSQLGenerator(nil, "postgres")
	_, _, columns, err := buildMultiBOSQL(gen, primary, []joinedBO{
		{BOID: "bo-warehouse", BODef: related, Path: path, Cardinality: path.TraversalCardinality()},
	}, qd, "tenant-123")
	if err != nil {
		t.Fatalf("buildMultiBOSQL failed: %v", err)
	}

	if columns[1].RootOwnership != "unresolved" {
		t.Fatalf("expected warehouse_name RootOwnership 'unresolved' for an unclassifiable hop, got %q (an unrecognized hop must never read back as \"unique\")", columns[1].RootOwnership)
	}
}

func TestBuildMultiBOSQL_RejectsUnsafeIdentifier(t *testing.T) {
	primary := &boresolver.BODefinition{
		ID:           "bo-account",
		DrivingTable: "account; DROP TABLE users;--",
		Fields: []boresolver.BOField{
			{ID: "f1", Name: "account_id", Type: "string", PhysicalColumn: "account_id"},
		},
	}
	qd := &boresolver.QueryDef{
		Context: boresolver.QueryContext{BOID: "bo-account"},
		Query: boresolver.QueryRequest{
			Dimensions: []boresolver.DimensionDef{{TermNodeID: "account_id", BOID: "bo-account"}},
		},
	}

	gen, _ := boresolver.NewBOSQLGenerator(nil, "postgres")
	_, _, _, err := buildMultiBOSQL(gen, primary, nil, qd, "tenant-123")
	if err == nil {
		t.Fatal("expected an error for an unsafe driving table identifier, got nil")
	}
}

func TestBuildMultiBOSQL_RejectsUnknownFilterOperator(t *testing.T) {
	primary := &boresolver.BODefinition{
		ID:           "bo-account",
		DrivingTable: "account",
		Fields: []boresolver.BOField{
			{ID: "f1", Name: "account_id", Type: "string", PhysicalColumn: "account_id"},
		},
	}
	qd := &boresolver.QueryDef{
		Context: boresolver.QueryContext{BOID: "bo-account"},
		Query: boresolver.QueryRequest{
			Dimensions: []boresolver.DimensionDef{{TermNodeID: "account_id", BOID: "bo-account"}},
			Filters: []boresolver.FilterDef{
				{TermNodeID: "account_id", Operator: "1=1; --", Value: "x", BOID: "bo-account"},
			},
		},
	}

	gen, _ := boresolver.NewBOSQLGenerator(nil, "postgres")
	_, _, _, err := buildMultiBOSQL(gen, primary, nil, qd, "tenant-123")
	if err == nil {
		t.Fatal("expected an error for an unsupported filter operator, got nil")
	}
}
