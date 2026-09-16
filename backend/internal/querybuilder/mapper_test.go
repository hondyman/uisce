package querybuilder

import (
	"testing"

	"github.com/hondyman/uisce/backend/internal/boresolver"
)

// TestMapQueryDefToSemanticRequest_UsesBOIDNotDrivingTable covers the bug
// this mapper used to have: it set SemanticSQLGenerationRequest.Datasource
// to boDef.DrivingTable (a physical datasource path like "/orm/order"),
// which ResolveSemanticRequest then matched against
// business_objects.bo_key (a plain name like "order") via
// GetBOByTechnicalName - a lookup that could never succeed since a path
// is never a bo_key. The mapper already has the resolved BODefinition in
// hand, so it should pass BusinessObjectID and skip that name-based
// lookup entirely, not merely pass a "more correct" string into Datasource.
func TestMapQueryDefToSemanticRequest_UsesBOIDNotDrivingTable(t *testing.T) {
	boDef := &boresolver.BODefinition{
		ID:           "bo-order-uuid",
		DrivingTable: "/orm/order", // a datasource path, never a valid bo_key
		Fields: []boresolver.BOField{
			{ID: "f1", Name: "target_qty", PhysicalColumn: "order.target_qty"},
		},
	}
	qd := &boresolver.QueryDef{
		Query: boresolver.QueryRequest{
			Dimensions: []boresolver.DimensionDef{{TermNodeID: "target_qty", Alias: "TargetQty"}},
			Limit:      10,
		},
	}

	req, err := mapQueryDefToSemanticRequest(qd, boDef, nil)
	if err != nil {
		t.Fatalf("mapQueryDefToSemanticRequest failed: %v", err)
	}

	if req.BusinessObjectID != "bo-order-uuid" {
		t.Errorf("expected BusinessObjectID to be set from boDef.ID, got %q", req.BusinessObjectID)
	}
	if req.Datasource != "" {
		t.Errorf("expected Datasource to be left empty (BusinessObjectID takes precedence in ResolveSemanticRequest), got %q", req.Datasource)
	}
}
