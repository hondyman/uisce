package activities_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	uiscedb "github.com/hondyman/uisce/backend/internal/db"
	"github.com/hondyman/uisce/backend/internal/dscreds"
	"github.com/hondyman/uisce/backend/internal/migrations"
	"github.com/hondyman/uisce/backend/internal/provisioning"
	"github.com/hondyman/uisce/backend/internal/secrets"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/hondyman/uisce/backend/internal/temporal/activities"
	"github.com/hondyman/uisce/backend/internal/tenantdb"
)

// Phase 4a's closing gate: does the fleet hold up at hundreds of tenants?
//
// It provisions N tenants through the real saga activities (so provisioning cost is measured too),
// then drives the real tenantdb router over all of them: cold builds, warm resolves, concurrent
// resolves, the query path, the connections the cluster actually sees, and eviction under churn.
//
// Opt-in: SAGA_TEST_SCALE=1 plus the SAGA_TEST_* environment of the saga tests, against a
// DEDICATED hardened cluster with max_connections well above N (one connection per tenant pool is
// the floor). SAGA_TEST_SCALE_N sets N (default 200); SAGA_TEST_SCALE_OUT names a JSON file for the
// numbers. It is a measurement as much as a test: every number is logged, and the assertions are
// the properties that must hold at any size.

// stmtCounter counts every statement sent to alpha (including BEGIN, COMMIT and set_config).
type stmtCounter struct{ n int64 }

func (c *stmtCounter) TraceQueryStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryStartData) context.Context {
	atomic.AddInt64(&c.n, 1)
	return ctx
}
func (c *stmtCounter) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

type scaleTenant struct {
	tenant, ds, database string
	in                   provisioning.TenantDatabaseInput
	provision, probe     time.Duration
}

type scaleFixture struct {
	stmts      *stmtCounter
	admin, app *sql.DB
	cluster    activities.TenantDatabaseAdmin
	acts       *activities.TenantProvisioningActivities
	tenants    []scaleTenant
}

func pct(d []time.Duration, p float64) time.Duration {
	if len(d) == 0 {
		return 0
	}
	s := append([]time.Duration(nil), d...)
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	i := int(float64(len(s)-1) * p)
	return s[i]
}

type stat struct {
	N                  int     `json:"n"`
	P50, P95, P99, Max float64 // milliseconds
}

func summarize(d []time.Duration) stat {
	ms := func(x time.Duration) float64 { return float64(x.Microseconds()) / 1000 }
	return stat{N: len(d), P50: ms(pct(d, .50)), P95: ms(pct(d, .95)), P99: ms(pct(d, .99)), Max: ms(pct(d, 1))}
}

func (s stat) String() string {
	return fmt.Sprintf("n=%d p50=%.2fms p95=%.2fms p99=%.2fms max=%.2fms", s.N, s.P50, s.P95, s.P99, s.Max)
}

func newScaleFixture(t *testing.T, n int) *scaleFixture {
	t.Helper()
	adminDSN, appDSN := os.Getenv("SAGA_TEST_ALPHA_ADMIN_DSN"), os.Getenv("SAGA_TEST_ALPHA_APP_DSN")
	host, user := os.Getenv("SAGA_TEST_PG_HOST"), os.Getenv("SAGA_TEST_PG_USER")
	port, _ := strconv.Atoi(os.Getenv("SAGA_TEST_PG_PORT"))
	if os.Getenv("SAGA_TEST_SCALE") != "1" || adminDSN == "" || appDSN == "" || host == "" || user == "" || port == 0 {
		t.Skip("SAGA_TEST_SCALE=1 and SAGA_TEST_* not set")
	}
	adm, err := sql.Open("pgx", adminDSN)
	require.NoError(t, err)
	t.Cleanup(func() { adm.Close() })
	adm.SetMaxOpenConns(4)
	var name string
	require.NoError(t, adm.QueryRow(`SELECT current_database()`).Scan(&name))
	require.Contains(t, name, "saga", "refusing to drop the public schema of %q", name)

	migDir := "../../../db/migrations/"
	b1, err := os.ReadFile(migDir + "20261206_001_tenant_datasource_binding.up.sql")
	require.NoError(t, err)
	b2, err := os.ReadFile(migDir + "20261210_001_binding_pg_credential.up.sql")
	require.NoError(t, err)
	for _, q := range []string{`DROP SCHEMA public CASCADE`, `CREATE SCHEMA public`, sagaStub, string(b1), string(b2), sagaPolicies} {
		_, err := adm.Exec(q)
		require.NoError(t, err)
	}
	stmts := &stmtCounter{}
	appCfg, err := pgx.ParseConfig(appDSN)
	require.NoError(t, err)
	appCfg.Tracer = stmts
	app := stdlib.OpenDB(*appCfg)
	app.SetMaxOpenConns(16)
	t.Cleanup(func() { app.Close() })
	var appUser string
	require.NoError(t, app.QueryRow(`SELECT current_user`).Scan(&appUser))
	_, err = adm.Exec(fmt.Sprintf(`REVOKE CONNECT ON DATABASE "%s" FROM PUBLIC; GRANT CONNECT ON DATABASE "%s" TO "%s"`, name, name, appUser))
	require.NoError(t, err)

	cluster := activities.TenantDatabaseAdmin{Host: host, Port: port, User: user, Password: "x"}
	sec := secrets.NewMemoryProvider()
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "orm"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "orm", "0001_notes.up.sql"), []byte(isoMigration), 0o644))
	t.Setenv("DATABASE_URL", fmt.Sprintf("postgres://%s:%s@%s:%d/postgres?sslmode=disable", cluster.User, cluster.Password, cluster.Host, cluster.Port))

	f := &scaleFixture{stmts: stmts, admin: adm, app: app, cluster: cluster}
	f.acts = &activities.TenantProvisioningActivities{
		ControlDB: sqlx.NewDb(app, "pgx"), Logger: zap.NewNop().Sugar(),
		Secrets: sec, TenantDB: cluster, Creds: dscreds.NewResolver(sec, dscreds.WithCacheTTL(0)),
		Migrations: &migrations.TenantRunner{Root: root},
	}
	t.Cleanup(func() {
		c, err := cluster.Open("postgres")
		if err != nil {
			return
		}
		defer c.Close()
		for _, tn := range f.tenants {
			_, _ = c.Exec(`DROP DATABASE IF EXISTS ` + tn.database + ` WITH (FORCE)`)
			_, _ = c.Exec(`DROP ROLE IF EXISTS ` + tn.database + `_app`)
		}
	})
	return f
}

func (f *scaleFixture) provisionOne(t *testing.T, i int) scaleTenant {
	t.Helper()
	ctx := context.Background()
	tn := scaleTenant{tenant: uuid.NewString(), ds: uuid.NewString(), database: fmt.Sprintf("tdb_saga_s%04d_%s", i, uuid.NewString()[:6])}
	inst, prod, cat := uuid.NewString(), uuid.NewString(), uuid.NewString()
	cfg := `{"host":"gold-host","port":5432,"database":"gold_copy_db"}`

	tx, err := f.app.Begin()
	require.NoError(t, err)
	_, err = tx.Exec(`SELECT set_config('uisce.current_tenant', $1, true)`, tn.tenant)
	require.NoError(t, err)
	for _, q := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO tenants (id, name, code, status) VALUES ($1, 'n', $2, 'provisioning')`, []any{tn.tenant, "c" + tn.tenant[:8]}},
		{`INSERT INTO tenant_instance (id, tenant_id) VALUES ($1, $2)`, []any{inst, tn.tenant}},
		{`INSERT INTO tenant_product (id, datasource_id) VALUES ($1, $2)`, []any{prod, inst}},
		{`INSERT INTO alpha_datasource (id, datasource_code) VALUES ($1, 'orm')`, []any{cat}},
		{`INSERT INTO tenant_product_datasource (id, tenant_product_id, alpha_datasource_id, config) VALUES ($1, $2, $3, $4::jsonb)`, []any{tn.ds, prod, cat, cfg}},
	} {
		_, err := tx.Exec(q.sql, q.args...)
		require.NoError(t, err, q.sql)
	}
	require.NoError(t, tx.Commit())

	start := time.Now()
	in := provisioning.TenantDatabaseInput{TenantID: tn.tenant, InstanceID: inst, App: "orm", DatabaseName: tn.database, GoldCopyDatabase: "gold_copy_db"}
	require.NoError(t, f.acts.CreateTenantDatabase(ctx, tn.database))
	b, err := f.acts.BindTenantDatabase(ctx, in)
	require.NoError(t, err)
	in.DatasourceID = b.DatasourceID
	tn.in = in
	f.tenants = append(f.tenants, tn) // registered for cleanup before the steps that can fail
	require.NoError(t, f.acts.ProvisionTenantDatabaseAccess(ctx, in))
	rep, err := f.acts.ApplyTenantMigrations(ctx, in)
	require.NoError(t, err)
	require.True(t, rep.Done)
	ps := time.Now()
	require.NoError(t, f.acts.ProbeTenantDatabase(ctx, in))
	tn.probe = time.Since(ps)
	require.NoError(t, f.acts.ActivateTenantDatabase(ctx, in))
	tn.provision = time.Since(start)
	f.tenants[len(f.tenants)-1] = tn
	return tn
}

type countingRegistry struct {
	*tenantdb.AlphaRegistry
	creds int64
}

func (c *countingRegistry) Credentials(ctx context.Context, ds tenantdb.Datasource) (string, string, error) {
	atomic.AddInt64(&c.creds, 1)
	return c.AlphaRegistry.Credentials(ctx, ds)
}

func (f *scaleFixture) router(t *testing.T, maxPools int, maxConns int32) (*tenantdb.Router, *countingRegistry) {
	t.Helper()
	reg := &countingRegistry{AlphaRegistry: &tenantdb.AlphaRegistry{
		DB: f.app, Resolver: security.NewDBDatasourceResolver(sqlx.NewDb(f.app, "pgx")), Creds: f.acts.Creds,
	}}
	r, err := tenantdb.New(tenantdb.Config{
		Registry: reg, CallerTenant: uiscedb.GetTenantIDFromCtx,
		MaxPools: maxPools, MaxConnsPerPool: maxConns, IdleTTL: 10 * time.Minute, DialTimeout: 5 * time.Second,
	})
	require.NoError(t, err)
	t.Cleanup(r.Close)
	return r, reg
}

// tenantConns counts server connections held by tenant roles, as the cluster sees them.
func (f *scaleFixture) tenantConns(t *testing.T) int {
	t.Helper()
	var n int
	require.NoError(t, f.admin.QueryRow(`SELECT count(*) FROM pg_stat_activity WHERE usename LIKE 'tdb_saga_s%_app'`).Scan(&n))
	return n
}

func TestScale_FleetOfTenants(t *testing.T) {
	n := 200
	if v, err := strconv.Atoi(os.Getenv("SAGA_TEST_SCALE_N")); err == nil && v > 0 {
		n = v
	}
	f := newScaleFixture(t, n)
	out := map[string]any{"tenants": n, "gomaxprocs": runtime.GOMAXPROCS(0)}

	// ---- 1. provisioning, through the real saga activities ------------------------------------
	t.Logf("== provisioning %d tenants (real saga activities)", n)
	provStart := time.Now()
	for i := 0; i < n; i++ {
		f.provisionOne(t, i)
		if (i+1)%50 == 0 {
			t.Logf("   provisioned %d/%d (probe of tenant %d took %v)", i+1, n, i+1, f.tenants[i].probe.Round(time.Millisecond))
		}
	}
	provTotal := time.Since(provStart)
	var probes, provs []time.Duration
	for _, tn := range f.tenants {
		probes = append(probes, tn.probe)
		provs = append(provs, tn.provision)
	}
	at := func(i int) time.Duration {
		if i >= len(f.tenants) {
			i = len(f.tenants) - 1
		}
		return f.tenants[i].probe
	}
	t.Logf("PROVISION total=%v per-tenant: %s", provTotal.Round(time.Millisecond), summarize(provs))
	t.Logf("PROBE (isolation check tries every other database): %s", summarize(probes))
	t.Logf("PROBE growth: tenant#1=%v #%d=%v #%d=%v #%d=%v", at(0).Round(time.Millisecond),
		n/4, at(n/4-1).Round(time.Millisecond), n/2, at(n/2-1).Round(time.Millisecond), n, at(n-1).Round(time.Millisecond))
	out["provision"] = summarize(provs)
	out["probe"] = summarize(probes)
	out["probe_first_ms"] = at(0).Milliseconds()
	out["probe_last_ms"] = at(n - 1).Milliseconds()

	ctx := func(tn scaleTenant) context.Context {
		return uiscedb.WithTenantContextToCtx(context.Background(), tn.tenant)
	}

	// ---- 2. cold: first touch of every tenant -------------------------------------------------
	router, reg := f.router(t, n+8, 4)
	var cold []time.Duration
	for _, tn := range f.tenants {
		s := time.Now()
		p, err := router.Resolve(ctx(tn), tn.ds)
		require.NoError(t, err)
		cold = append(cold, time.Since(s))
		require.Equal(t, tn.database, p.Database())
	}
	require.Equal(t, n, router.Size(), "one pool per tenant, owned by the router")
	t.Logf("COLD Resolve (build pool + alpha lookups): %s", summarize(cold))
	require.Less(t, pct(cold, .99), 5*time.Second, "a cold build must stay inside the dial budget")
	out["cold"] = summarize(cold)
	require.Equal(t, int64(n), atomic.LoadInt64(&reg.creds), "exactly one credential fetch per cold build")

	// ---- 3. warm, single goroutine, and what each Resolve costs alpha -------------------------
	st0 := atomic.LoadInt64(&f.stmts.n)
	var warm []time.Duration
	const rounds = 10
	for r := 0; r < rounds; r++ {
		for _, tn := range f.tenants {
			s := time.Now()
			_, err := router.Resolve(ctx(tn), tn.ds)
			require.NoError(t, err)
			warm = append(warm, time.Since(s))
		}
	}
	st1 := atomic.LoadInt64(&f.stmts.n)
	calls := int64(rounds * n)
	perResolve := float64(st1-st0) / float64(calls)
	t.Logf("WARM Resolve (cache hit), 1 goroutine: %s", summarize(warm))
	t.Logf("ALPHA statements per warm Resolve: %.1f (%d over %d calls)", perResolve, st1-st0, calls)
	out["warm"] = summarize(warm)
	out["alpha_statements_per_resolve"] = perResolve
	require.Equal(t, int64(n), atomic.LoadInt64(&reg.creds), "a cache hit must not fetch a credential or build a pool")
	require.Equal(t, n, router.Size())

	// ---- 4. warm, concurrent -------------------------------------------------------------------
	const workers, perWorker = 32, 400
	var mu sync.Mutex
	var conc []time.Duration
	var errs int64
	var wg sync.WaitGroup
	cs := time.Now()
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(seed int64) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(seed))
			local := make([]time.Duration, 0, perWorker)
			for i := 0; i < perWorker; i++ {
				tn := f.tenants[rng.Intn(n)]
				s := time.Now()
				_, err := router.Resolve(ctx(tn), tn.ds)
				if err != nil {
					atomic.AddInt64(&errs, 1)
					continue
				}
				local = append(local, time.Since(s))
			}
			mu.Lock()
			conc = append(conc, local...)
			mu.Unlock()
		}(int64(w))
	}
	wg.Wait()
	concWall := time.Since(cs)
	t.Logf("WARM Resolve, %d goroutines: %s (wall %v, %.0f resolves/s)", workers, summarize(conc), concWall.Round(time.Millisecond), float64(workers*perWorker)/concWall.Seconds())
	out["warm_concurrent"] = summarize(conc)
	out["warm_concurrent_per_sec"] = float64(workers*perWorker) / concWall.Seconds()
	require.Zero(t, atomic.LoadInt64(&errs), "no resolve may fail under concurrency")

	// ---- 5. the query path on warm pools -------------------------------------------------------
	var q []time.Duration
	for i := 0; i < 2000; i++ {
		tn := f.tenants[rand.Intn(n)]
		p, err := router.Resolve(ctx(tn), tn.ds)
		require.NoError(t, err)
		s := time.Now()
		require.NoError(t, p.WithTx(context.Background(), func(tx pgx.Tx) error {
			var x int
			return tx.QueryRow(context.Background(), `SELECT 1`).Scan(&x)
		}))
		q = append(q, time.Since(s))
	}
	t.Logf("QUERY (begin, SELECT 1, commit) on a warm pool: %s", summarize(q))
	out["query"] = summarize(q)

	// ---- 5b. one hot tenant must not exceed its pool's cap --------------------------------------
	{
		hot := f.tenants[0]
		var hw sync.WaitGroup
		for i := 0; i < 24; i++ {
			hw.Add(1)
			go func() {
				defer hw.Done()
				p, err := router.Resolve(ctx(hot), hot.ds)
				if err != nil {
					return
				}
				_ = p.WithTx(context.Background(), func(tx pgx.Tx) error {
					_, err := tx.Exec(context.Background(), `SELECT pg_sleep(0.4)`)
					return err
				})
			}()
		}
		time.Sleep(200 * time.Millisecond) // all 24 are now queued on, or holding, the pool
		var hotConns int
		require.NoError(t, f.admin.QueryRow(`SELECT count(*) FROM pg_stat_activity WHERE usename = $1`, hot.database+"_app").Scan(&hotConns))
		hw.Wait()
		t.Logf("HOT TENANT: 24 concurrent transactions held %d server connections (MaxConnsPerPool=4)", hotConns)
		out["hot_tenant_conns"] = hotConns
		require.LessOrEqual(t, hotConns, 4, "one tenant must never exceed its pool's cap, however hard it is driven")
		require.GreaterOrEqual(t, hotConns, 2, "the pool must actually have grown past one connection, or the cap was not exercised")
	}

	// ---- 6. what the cluster and the process hold ---------------------------------------------
	var ms runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&ms)
	conns := f.tenantConns(t)
	t.Logf("FOOTPRINT: pools=%d server connections from tenant roles=%d heap=%.1fMB goroutines=%d", router.Size(), conns, float64(ms.HeapAlloc)/1e6, runtime.NumGoroutine())
	out["pools"], out["server_conns"], out["heap_mb"], out["goroutines"] = router.Size(), conns, float64(ms.HeapAlloc)/1e6, runtime.NumGoroutine()
	require.LessOrEqual(t, conns, n*4, "connections must stay within MaxConnsPerPool per pool")
	require.GreaterOrEqual(t, conns, n, "every tenant's pool is connected (one connection each is the floor)")

	router.Close()
	time.Sleep(time.Second)
	require.Zero(t, f.tenantConns(t), "closing the router must release every tenant connection")

	// ---- 7. churn: more tenants than pool slots ------------------------------------------------
	const slots = 50
	if n > slots {
		small, sreg := f.router(t, slots, 4)
		var churn []time.Duration
		maxSize, maxConns := 0, 0
		rng := rand.New(rand.NewSource(7))
		for i := 0; i < 3000; i++ {
			tn := f.tenants[rng.Intn(n)]
			s := time.Now()
			_, err := small.Resolve(ctx(tn), tn.ds)
			require.NoError(t, err)
			churn = append(churn, time.Since(s))
			if sz := small.Size(); sz > maxSize {
				maxSize = sz
			}
			if i%500 == 499 {
				if c := f.tenantConns(t); c > maxConns {
					maxConns = c
				}
			}
		}
		builds := atomic.LoadInt64(&sreg.creds)
		t.Logf("CHURN (%d slots, %d tenants, uniform access): %s", slots, n, summarize(churn))
		t.Logf("CHURN: pool builds=%d of 3000 (hit rate %.0f%%), max pools=%d, max server connections sampled=%d", builds, 100*(1-float64(builds)/3000), maxSize, maxConns)
		out["churn"] = summarize(churn)
		out["churn_builds"], out["churn_max_pools"], out["churn_max_conns"] = builds, maxSize, maxConns
		require.LessOrEqual(t, maxSize, slots, "the pool count must stay bounded under churn")
		require.LessOrEqual(t, maxConns, slots*4+4, "evicted pools must release their connections")
	}

	if path := os.Getenv("SAGA_TEST_SCALE_OUT"); path != "" {
		b, _ := json.MarshalIndent(out, "", "  ")
		require.NoError(t, os.WriteFile(path, b, 0o644))
	}
}
