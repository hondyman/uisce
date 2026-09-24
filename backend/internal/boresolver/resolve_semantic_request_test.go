package boresolver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestResolveSemanticRequest_BusinessObjectIDBypassesNameLookup covers the
// real-world bug this generator had: a caller with an already-resolved BO
// (querybuilder.mapQueryDefToSemanticRequest) had no way to tell
// ResolveSemanticRequest that, so it always went through
// GetBOByTechnicalName(Datasource, ...) - which only matches
// business_objects.bo_key, never a physical driver-table path like
// "/orm/order". Setting BusinessObjectID must resolve via GetBODefinition
// directly and never touch GetBOByTechnicalName at all (a mock whose
// GetBOByTechnicalName always errors proves this).
func TestResolveSemanticRequest_BusinessObjectIDBypassesNameLookup(t *testing.T) {
	repo := &MockBORepository{
		BODefinitions: map[string]*BODefinition{
			"bo-order-uuid": {
				ID:           "bo-order-uuid",
				DrivingTable: "/orm/order", // a datasource path - GetBOByTechnicalName would never match this
				Fields: []BOField{
					{ID: "f1", Name: "target_qty", PhysicalColumn: "target_qty"},
				},
			},
		},
	}

	generator, err := NewBOSQLGenerator(repo, "postgres")
	assert.NoError(t, err)

	req := &SemanticSQLGenerationRequest{
		BusinessObjectID: "bo-order-uuid",
		Select:           []SemanticField{{Term: "target_qty", Label: "TargetQty"}},
		Limit:            10,
	}

	resolved, err := generator.ResolveSemanticRequest(req, "tenant-1", "ds-1")
	assert.NoError(t, err)
	assert.Equal(t, "bo-order-uuid", resolved.BusinessObjectID)
	assert.Equal(t, []string{"f1"}, resolved.SelectedFields)
}

// TestResolveSemanticRequest_FallsBackToNameLookup confirms existing
// callers that only have a name (the public semantic-SQL API in
// bo_sql_routes.go) are unaffected by the BusinessObjectID fast path.
func TestResolveSemanticRequest_FallsBackToNameLookup(t *testing.T) {
	repo := &MockBORepository{
		BODefinitions: map[string]*BODefinition{
			"bo_orders": {
				ID:           "bo_orders",
				DrivingTable: "orders",
				Fields:       []BOField{{ID: "f1", Name: "id", PhysicalColumn: "id"}},
			},
		},
	}
	generator, err := NewBOSQLGenerator(repo, "postgres")
	assert.NoError(t, err)

	req := &SemanticSQLGenerationRequest{
		Datasource: "orders",
		Select:     []SemanticField{{Term: "id"}},
	}
	resolved, err := generator.ResolveSemanticRequest(req, "tenant-1", "ds-1")
	assert.NoError(t, err)
	assert.Equal(t, "bo_orders", resolved.BusinessObjectID)
}
