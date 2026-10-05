package activities_test

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/dscreds"
	"github.com/hondyman/uisce/backend/internal/scanner"
	"github.com/hondyman/uisce/backend/internal/tenantschema"
	"github.com/hondyman/uisce/backend/models"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
)

// How long does onboarding a tenant's structure actually take, on a REAL template? This scans a real source database once
// (structure only; not timed: in production alpha already holds it), then times the saga steps that matter for
// "onboard instantly" against the real cluster rig: planning (load and compile), applying (one transaction, plus the grants),
// and the probe through tenantdb.
//
// TENANT_STRUCTURE_TIMING_SOURCE_DSN is a postgres:// URL of the source and TENANT_STRUCTURE_TIMING_SCHEMAS the template's
// schemas, comma separated. It needs the same SAGA_TEST_* environment as the other rig tests.

type timingStore struct {
	schemas string
	nodes   []*models.CatalogNode
}

func (s *timingStore) Owner(context.Context, string) (string, error)    { return "gold", nil }
func (s *timingStore) IsGoldCopy(context.Context, string) (bool, error) { return true, nil }
func (s *timingStore) Read(context.Context, string, string) (string, []*models.CatalogNode, error) {
	return s.schemas, s.nodes, nil
}

func TestStructure_OnboardingTimingOnARealTemplate(t *testing.T) {
	dsn, schemas := os.Getenv("TENANT_STRUCTURE_TIMING_SOURCE_DSN"), os.Getenv("TENANT_STRUCTURE_TIMING_SCHEMAS")
	if dsn == "" || schemas == "" {
		t.Skip("TENANT_STRUCTURE_TIMING_SOURCE_DSN / _SCHEMAS not set")
	}
	src, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	sc, err := scanner.NewAnsiScanner(src, uuid.New(), uuid.New(), "src", nil, true, strings.Split(schemas, ","))
	require.NoError(t, err)
	sc.SkipDataProfile()
	nodes, _, err := sc.ExtractMetadata()
	require.NoError(t, err)

	r := newSagaRig(t)
	r.acts.Templates = &tenantschema.Loader{Store: &timingStore{schemas: schemas, nodes: nodes}}
	in := r.provisioned(t)
	in.TemplateDatasourceID = uuid.NewString()
	ctx := context.Background()

	t0 := time.Now()
	plan, err := r.acts.PlanTenantStructure(ctx, in)
	require.NoError(t, err)
	planTime := time.Since(t0)
	in.StructureHash = plan.Hash

	t1 := time.Now()
	rep, err := r.acts.ApplyTenantStructure(ctx, in)
	require.NoError(t, err)
	require.True(t, rep.Done)
	applyTime := time.Since(t1)

	t2 := time.Now()
	require.NoError(t, r.acts.ProbeTenantDatabase(ctx, in))
	probeTime := time.Since(t2)

	t3 := time.Now()
	again, err := r.acts.ApplyTenantStructure(ctx, in)
	require.NoError(t, err)
	require.Empty(t, again.Ran)
	reapplyTime := time.Since(t3)

	t.Logf("template: %d tables, %d statements, %d schemas", plan.Tables, plan.Statements, len(plan.Schemas))
	t.Logf("plan (load %d nodes + compile): %v", len(nodes), planTime.Round(time.Millisecond))
	t.Logf("apply (one transaction + grants): %v", applyTime.Round(time.Millisecond))
	t.Logf("probe through tenantdb:          %v", probeTime.Round(time.Millisecond))
	t.Logf("re-apply (a resumed run):        %v", reapplyTime.Round(time.Millisecond))
	t.Logf("structure total (plan+apply+probe): %v", (planTime + applyTime + probeTime).Round(time.Millisecond))

	// and the tenant can really use it
	path, _ := dscreds.CanonicalPath(dscreds.KindDatasource, r.tenant, in.DatasourceID)
	stored, err := r.sec.GetMap(ctx, path)
	require.NoError(t, err)
	role := r.asRole(t, stored[dscreds.KeyPassword])
	var n int
	require.NoError(t, role.QueryRow(`SELECT count(*) FROM pg_class WHERE relkind IN ('r','p') AND relnamespace IN (SELECT oid FROM pg_namespace WHERE nspname = ANY(string_to_array($1, ',')))`, schemas).Scan(&n))
	require.Equal(t, plan.Tables, n)
	_, err = role.Exec(`CREATE TABLE mdm.sneaky (id int)`)
	require.Error(t, err, "no DDL for the tenant's role")
}
