package boresolver

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// securityRepo returns a mock repository with a three-level BO graph:
// orders -> customers -> regions.
func securityRepo() *MockBORepository {
	return &MockBORepository{
		BODefinitions: map[string]*BODefinition{
			"bo_orders": {
				ID:           "bo_orders",
				DrivingTable: "orders",
				Fields: []BOField{
					{ID: "f_id", Name: "id", PhysicalColumn: "orders.id"},
					{ID: "f_total", Name: "total", PhysicalColumn: "orders.total_amount"},
					{ID: "f_cust_id", Name: "customer_id", PhysicalColumn: "orders.customer_id",
						Type: "reference", ReferenceBOID: "bo_customers"},
				},
			},
			"bo_customers": {
				ID:           "bo_customers",
				DrivingTable: "customers",
				Fields: []BOField{
					{ID: "f_id", Name: "id", PhysicalColumn: "customers.id"},
					{ID: "f_name", Name: "name", PhysicalColumn: "customers.name"},
					{ID: "f_region_id", Name: "region_id", PhysicalColumn: "customers.region_id",
						Type: "reference", ReferenceBOID: "bo_regions"},
				},
			},
			"bo_regions": {
				ID:           "bo_regions",
				DrivingTable: "regions",
				Fields: []BOField{
					{ID: "f_id", Name: "id", PhysicalColumn: "regions.id"},
					{ID: "f_name", Name: "name", PhysicalColumn: "regions.name"},
				},
			},
		},
	}
}

func TestTenantScoping_SingleObject(t *testing.T) {
	repo := securityRepo()
	generator, err := NewBOSQLGenerator(repo, "postgres")
	require.NoError(t, err)

	req := SQLGenerationRequest{
		BusinessObjectID: "bo_orders",
		SelectedFields:   []string{"id", "total"},
		TenantID:         "tenant-alpha",
		Limit:            10,
	}

	sql, args, err := generator.GenerateSQL(req)
	require.NoError(t, err)

	// Root driving table is isolated.
	assert.Contains(t, sql, "t0.tenant_id = $1")
	assert.True(t, strings.HasPrefix(extractWhere(sql), "t0.tenant_id = $1"),
		"tenant predicate must lead the WHERE cluster, not be appended as a trailing filter")
	assert.Equal(t, []interface{}{"tenant-alpha"}, args)
}

func TestTenantScoping_MultiJoin(t *testing.T) {
	repo := securityRepo()
	generator, err := NewBOSQLGenerator(repo, "postgres")
	require.NoError(t, err)

	req := SQLGenerationRequest{
		BusinessObjectID: "bo_orders",
		SelectedFields:   []string{"customer_id.region_id.name"},
		TenantID:         "tenant-alpha",
		Limit:            10,
	}

	sql, args, err := generator.GenerateSQL(req)
	require.NoError(t, err)

	// Every alias in the join graph must be isolated. Placeholders are
	// numbered by TEXTUAL order of appearance, not by the order the tenant
	// predicates were built in: the two JOIN conditions (t1, t2) are
	// written before the root's WHERE predicate (t0) in the assembled
	// SQL, even though the root predicate is built first - see params.go.
	// For $N this is only a numbering difference, never a binding bug
	// (each $N is self-indexing) - but the numbering itself must still be
	// deterministic and match the text, since a positional ("?") dialect
	// depends on exactly this ordering to bind correctly.
	assert.Contains(t, sql, "t1.tenant_id = $1")
	assert.Contains(t, sql, "t2.tenant_id = $2")
	assert.Contains(t, sql, "t0.tenant_id = $3")

	// Each JOIN condition must have the tenant check structurally ANDed.
	assert.Contains(t, sql, "ON (t0.customer_id = t1.id) AND t1.tenant_id = $1")
	assert.Contains(t, sql, "ON (t1.region_id = t2.id) AND t2.tenant_id = $2")

	// Args must align 1:1 with placeholders.
	require.Len(t, args, 3)
	assert.Equal(t, "tenant-alpha", args[0])
	assert.Equal(t, "tenant-alpha", args[1])
	assert.Equal(t, "tenant-alpha", args[2])
}

func TestTenantScoping_CombinesWithExistingFilters(t *testing.T) {
	repo := securityRepo()
	generator, err := NewBOSQLGenerator(repo, "postgres")
	require.NoError(t, err)

	req := SQLGenerationRequest{
		BusinessObjectID: "bo_orders",
		SelectedFields:   []string{"id"},
		TenantID:         "tenant-alpha",
		Filters: []FilterClause{{
			FieldID:  "total",
			Operator: ">",
			Value:    100,
		}},
		Limit: 10,
	}

	sql, args, err := generator.GenerateSQL(req)
	require.NoError(t, err)

	// The filter's value (100) is bound, never inlined - this test used
	// to assert "t0.total_amount > 100" as a literal with args of length
	// 1, which was stale even before params.go: CompileFilterPredicate
	// has always parameterized filter values regardless of type, so a
	// numeric filter combined with tenant scoping was always 2 args, not
	// 1. Root tenant scoping (built in GenerateSQL step 6) is spliced to
	// appear before the filter clause (built in step 5) in the text, so
	// it gets $1 and the filter gets $2 - textual order, not creation
	// order; see params.go.
	where := extractWhere(sql)
	assert.True(t, strings.HasPrefix(where, "t0.tenant_id = $1"))
	assert.Contains(t, where, " AND ")
	assert.Contains(t, sql, "t0.total_amount > $2")
	require.Len(t, args, 2)
	assert.Equal(t, "tenant-alpha", args[0])
	assert.Equal(t, 100, args[1])
}

func TestTenantScoping_NoTenantID_NoScoping(t *testing.T) {
	repo := securityRepo()
	generator, err := NewBOSQLGenerator(repo, "postgres")
	require.NoError(t, err)

	req := SQLGenerationRequest{
		BusinessObjectID: "bo_orders",
		SelectedFields:   []string{"id", "total"},
		Limit:            10,
	}

	sql, args, err := generator.GenerateSQL(req)
	require.NoError(t, err)

	assert.NotContains(t, sql, "tenant_id")
	assert.Empty(t, args)
}

func TestTenantScoping_SemanticRequest(t *testing.T) {
	repo := securityRepo()
	generator, err := NewBOSQLGenerator(repo, "postgres")
	require.NoError(t, err)

	semanticReq := &SemanticSQLGenerationRequest{
		Datasource: "orders",
		Select:     []SemanticField{{Term: "id"}},
		Limit:      10,
	}

	sql, args, err := generator.GenerateSQLFromSemantic(semanticReq, "tenant-alpha", "ds-postgres")
	require.NoError(t, err)

	assert.Contains(t, sql, "t0.tenant_id = $1")
	assert.Equal(t, []interface{}{"tenant-alpha"}, args)
}

func TestTenantScoping_DialectTokens(t *testing.T) {
	repo := securityRepo()

	tests := []struct {
		dialect string
		want    string
	}{
		{"postgres", "t0.tenant_id = $1"},
		{"snowflake", "t0.tenant_id = ?"},
		{"sqlserver", "t0.tenant_id = @p1"},
	}

	for _, tc := range tests {
		t.Run(tc.dialect, func(t *testing.T) {
			generator, err := NewBOSQLGenerator(repo, tc.dialect)
			require.NoError(t, err)

			req := SQLGenerationRequest{
				BusinessObjectID: "bo_orders",
				SelectedFields:   []string{"id"},
				TenantID:         "tenant-alpha",
			}

			sql, _, err := generator.GenerateSQL(req)
			require.NoError(t, err)
			assert.Contains(t, sql, tc.want)
		})
	}
}

// extractWhere is a small helper to pull the WHERE clause from a generated query.
func extractWhere(sql string) string {
	idx := strings.Index(sql, "WHERE ")
	if idx == -1 {
		return ""
	}
	return strings.TrimSpace(sql[idx+len("WHERE "):])
}
