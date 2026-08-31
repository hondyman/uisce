package querybuilder

import (
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
