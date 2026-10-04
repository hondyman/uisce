package activities_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"github.com/hondyman/uisce/backend/internal/dscreds"
	"github.com/hondyman/uisce/backend/internal/provisioning"
)

// Concurrency: provisioning is a Temporal workflow, so many tenants are provisioned, and rolled
// back, at the same time against one cluster. The isolation check lists every other database and
// tries to connect to each, which makes it sensitive to other tenants' databases at every moment of
// THEIR provisioning: just created, not yet closed to PUBLIC, mid-rollback, dropped.
//
// Same environment as the other saga tests (SAGA_TEST_*), against a dedicated hardened cluster.

// rigs builds n tenants that share one alpha and one secrets store, leaving every tenant's
// database for the caller to create.
func concurrentRigs(t *testing.T, n int) []*sagaRig {
	t.Helper()
	base := newSagaRig(t)
	rigs := make([]*sagaRig, 0, n)
	for i := 0; i < n; i++ {
		r := newSagaRigOpts(t, base, false)
		dir := filepath.Join(r.acts.Migrations.Root, "orm")
		require.NoError(t, os.MkdirAll(dir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "0001_notes.up.sql"), []byte(isoMigration), 0o644))
		rigs = append(rigs, r)
	}
	return rigs
}

// provisionAll runs the whole provisioning chain for a rig and returns the first error and the input
// the later steps used.
func (r *sagaRig) provisionAll(ctx context.Context) (provisioning.TenantDatabaseInput, error) {
	if err := r.acts.CreateTenantDatabase(ctx, r.database); err != nil {
		return provisioning.TenantDatabaseInput{}, fmt.Errorf("create: %w", err)
	}
	b, err := r.acts.BindTenantDatabase(ctx, r.in())
	if err != nil {
		return provisioning.TenantDatabaseInput{}, fmt.Errorf("bind: %w", err)
	}
	in := r.in()
	in.DatasourceID = b.DatasourceID
	if err := r.acts.ProvisionTenantDatabaseAccess(ctx, in); err != nil {
		return in, fmt.Errorf("access: %w", err)
	}
	if _, err := r.acts.ApplyTenantMigrations(ctx, in); err != nil {
		return in, fmt.Errorf("migrate: %w", err)
	}
	if err := r.acts.ProbeTenantDatabase(ctx, in); err != nil {
		return in, fmt.Errorf("probe: %w", err)
	}
	if err := r.acts.ActivateTenantDatabase(ctx, in); err != nil {
		return in, fmt.Errorf("activate: %w", err)
	}
	return in, nil
}

// Many tenants provisioned at once: every one must succeed, with no spurious refusal from the
// isolation check seeing another tenant's half-provisioned database, and every one must end up
// reaching its own database and no other.
func TestConcurrentProvision_EveryTenantSucceedsAndStaysIsolated(t *testing.T) {
	const n = 12
	rigs := concurrentRigs(t, n)
	ctx := context.Background()

	start := make(chan struct{})
	var wg sync.WaitGroup
	errs := make([]error, n)
	ins := make([]provisioning.TenantDatabaseInput, n)
	for i := range rigs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			ins[i], errs[i] = rigs[i].provisionAll(ctx)
		}(i)
	}
	close(start)
	wg.Wait()
	for i, e := range errs {
		require.NoError(t, e, "tenant %d must provision successfully while %d others do", i, n-1)
	}

	// Alpha: every binding active, every role and credential distinct.
	var active, distinctRoles int
	require.NoError(t, rigs[0].admin.QueryRow(
		`SELECT count(*) FILTER (WHERE lifecycle_state = 'active'), count(DISTINCT pg_role) FROM tenant_datasource_binding`).Scan(&active, &distinctRoles))
	require.Equal(t, n, active)
	require.Equal(t, n, distinctRoles)
	passwords := map[string]bool{}
	for i, r := range rigs {
		path, _ := dscreds.CanonicalPath(dscreds.KindDatasource, r.tenant, ins[i].DatasourceID)
		c, err := r.sec.GetMap(ctx, path)
		require.NoError(t, err)
		require.False(t, passwords[c[dscreds.KeyPassword]], "two tenants must never share a credential")
		passwords[c[dscreds.KeyPassword]] = true
	}

	// The matrix: each tenant's role reaches its own database and every other tenant's refuses it.
	for i, ri := range rigs {
		path, _ := dscreds.CanonicalPath(dscreds.KindDatasource, ri.tenant, ins[i].DatasourceID)
		creds, err := ri.sec.GetMap(ctx, path)
		require.NoError(t, err)
		for j, rj := range rigs {
			c, err := pgx.Connect(ctx, fmt.Sprintf("postgres://%s:%s@%s:%d/%s", creds[dscreds.KeyUsername], creds[dscreds.KeyPassword],
				ri.cluster.Host, ri.cluster.Port, rj.database))
			if i == j {
				require.NoError(t, err, "tenant %d must reach its own database", i)
			} else {
				require.Error(t, err, "tenant %d's role must not reach tenant %d's database", i, j)
				require.Contains(t, err.Error(), "permission denied")
			}
			if err == nil {
				c.Close(ctx)
			}
		}
	}
}

// A database that has just been created must never make ANOTHER tenant's probe fail: creating one
// and closing it to PUBLIC must not leave an instant in which the probe could see it open. Tenants
// are created and rolled back in a tight loop while an already-provisioned tenant is probed over
// and over; not one probe may fail.
func TestConcurrentProvision_ADatabaseBeingCreatedNeverFailsAnotherTenantsProbe(t *testing.T) {
	rigs := concurrentRigs(t, 2)
	ctx := context.Background()
	steady := rigs[0]
	in, err := steady.provisionAll(ctx)
	require.NoError(t, err)

	churner := rigs[1].acts // creates and rolls back databases exactly as a provisioning saga does
	var stop, created int64
	var cw sync.WaitGroup
	cw.Add(1)
	go func() {
		defer cw.Done()
		for i := 0; atomic.LoadInt64(&stop) == 0; i++ {
			name := fmt.Sprintf("tdb_saga_churn_%d_%s", i, uuid.NewString()[:6])
			if err := churner.CreateTenantDatabase(ctx, name); err != nil {
				return
			}
			atomic.AddInt64(&created, 1)
			_ = churner.RollbackCreateTenantDatabase(ctx, name)
		}
	}()

	var failures []string
	probes := 0
	deadline := time.Now().Add(12 * time.Second)
	for time.Now().Before(deadline) && probes < 400 {
		probes++
		if err := steady.acts.ProbeTenantDatabase(ctx, in); err != nil {
			failures = append(failures, err.Error())
			if len(failures) >= 5 {
				break
			}
		}
	}
	atomic.StoreInt64(&stop, 1)
	cw.Wait()
	t.Logf("%d probes ran while %d databases were created and rolled back concurrently; %d failed", probes, atomic.LoadInt64(&created), len(failures))
	require.Empty(t, failures, "a probe must never fail because another tenant's database is being created or removed:\n%s", strings.Join(failures, "\n"))
	require.Greater(t, atomic.LoadInt64(&created), int64(5), "the churn must actually have run")
}

// An activity that dies after CREATE DATABASE, or after the REVOKE, leaves the database half
// created. Neither state may affect any OTHER tenant's provisioning (the half-created database is
// not connectable, so there is nothing for its probe to reach), and the retry must finish the job.
func TestConcurrentProvision_ACrashedCreateNeverAffectsOtherTenantsAndTheRetryFinishesIt(t *testing.T) {
	rigs := concurrentRigs(t, 2)
	ctx := context.Background()
	steady, crashed := rigs[0], rigs[1]
	in, err := steady.provisionAll(ctx)
	require.NoError(t, err)

	adm, err := crashed.cluster.Open("postgres")
	require.NoError(t, err)
	t.Cleanup(func() { adm.Close() }) // registered first, so it runs AFTER the role cleanup below
	_, _ = adm.Exec(`DROP ROLE IF EXISTS crash_probe_role`)
	_, err = adm.Exec(`CREATE ROLE crash_probe_role NOLOGIN`)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = adm.Exec(`DROP ROLE IF EXISTS crash_probe_role`) })
	db := crashed.database

	state := func() (allow bool, publicCanConnect bool) {
		require.NoError(t, adm.QueryRow(`SELECT datallowconn, has_database_privilege('crash_probe_role', datname, 'CONNECT') FROM pg_database WHERE datname = $1`, db).Scan(&allow, &publicCanConnect))
		return
	}

	// Crash after step 1: created, connections disabled, still open to PUBLIC.
	_, err = adm.Exec(`CREATE DATABASE ` + db + ` WITH ALLOW_CONNECTIONS false`)
	require.NoError(t, err)
	allow, open := state()
	require.False(t, allow)
	require.True(t, open, "this is the state a crash after CREATE leaves")
	require.NoError(t, steady.acts.ProbeTenantDatabase(ctx, in), "an unconnectable half-created database must not fail another tenant's probe")

	// Crash after step 2: closed to PUBLIC, connections still disabled.
	_, err = adm.Exec(`REVOKE CONNECT ON DATABASE ` + db + ` FROM PUBLIC`)
	require.NoError(t, err)
	require.NoError(t, steady.acts.ProbeTenantDatabase(ctx, in))

	// The retry (an "already exists" path) finishes: connections on, PUBLIC closed.
	require.NoError(t, crashed.acts.CreateTenantDatabase(ctx, db))
	allow, open = state()
	require.True(t, allow, "the retry must enable connections, or the tenant database is unusable")
	require.False(t, open, "and it must stay closed to PUBLIC")

	// And the whole chain now completes for that tenant.
	_, err = crashed.provisionAll(ctx)
	require.NoError(t, err)
	require.NoError(t, steady.acts.ProbeTenantDatabase(ctx, in))
}
