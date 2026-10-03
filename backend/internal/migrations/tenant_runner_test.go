package migrations

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
)

var tid = "11111111-2222-3333-4444-555555555555"

func target(app string) Target { return Target{TenantID: tid, App: app} }

func TestParseTarget(t *testing.T) {
	got, err := ParseTarget("tenant:" + tid + ":orm")
	require.NoError(t, err)
	require.Equal(t, Target{TenantID: tid, App: "orm"}, got)
	require.Equal(t, "tenant:"+tid+":orm", got.String())

	_, err = ParseTarget("alpha")
	require.ErrorIs(t, err, ErrAlphaTarget)

	for _, bad := range []string{
		"", "tenant", "tenant:" + tid, "tenant:" + tid + ":orm:extra", "tenant:not-a-uuid:orm",
		"tenant:" + tid + ":../etc", "tenant:" + tid + ":Orm", "tenant:" + tid + ":", "tenant:" + tid + ":a/b",
		"tenant:" + tid + ":0orm", "tenant:" + tid + ":" + string(make([]byte, 40)), "other:" + tid + ":orm",
	} {
		_, err := ParseTarget(bad)
		require.Error(t, err, bad)
		require.NotErrorIs(t, err, ErrAlphaTarget, bad)
	}
}

func TestAnalyze(t *testing.T) {
	tg := target("orm")
	sha := map[string]string{"001.up.sql": "a", "002.up.sql": "b", "003.up.sql": "c"}
	names := []string{"001.up.sql", "002.up.sql", "003.up.sql"}
	ap := func(n, s string) AppliedFile { return AppliedFile{Filename: n, SHA256: s} }

	t.Run("fresh: everything pending", func(t *testing.T) {
		r := analyze(tg, names, sha, nil)
		require.Equal(t, names, r.Pending)
		require.False(t, r.Done)
		require.Empty(t, r.Drift)
	})
	t.Run("current: done", func(t *testing.T) {
		r := analyze(tg, names, sha, []AppliedFile{ap("001.up.sql", "a"), ap("002.up.sql", "b"), ap("003.up.sql", "c")})
		require.True(t, r.Done)
		require.Empty(t, r.Pending)
	})
	t.Run("a changed applied file is drift", func(t *testing.T) {
		r := analyze(tg, names, sha, []AppliedFile{ap("001.up.sql", "OLD")})
		require.Equal(t, []Drift{{Filename: "001.up.sql", Kind: DriftChanged, RecordedSHA: "OLD", CurrentSHA: "a"}}, r.Drift)
		require.False(t, r.Done)
	})
	t.Run("an applied file that vanished is drift", func(t *testing.T) {
		r := analyze(tg, []string{"002.up.sql"}, map[string]string{"002.up.sql": "b"}, []AppliedFile{ap("001.up.sql", "a")})
		require.Equal(t, []Drift{{Filename: "001.up.sql", Kind: DriftMissing, RecordedSHA: "a"}}, r.Drift)
	})
	t.Run("a new file sorting before an applied one is out of order, and not pending", func(t *testing.T) {
		r := analyze(tg, []string{"000.up.sql", "002.up.sql"}, map[string]string{"000.up.sql": "z", "002.up.sql": "b"},
			[]AppliedFile{ap("002.up.sql", "b")})
		require.Equal(t, []Drift{{Filename: "000.up.sql", Kind: DriftOutOfOrder}}, r.Drift)
		require.Empty(t, r.Pending, "an out-of-order file must never be queued to run")
	})
}

// --- fleet logic, no database --------------------------------------------------------------

type fakeApplier struct {
	mu    sync.Mutex
	calls []string
	fail  map[string]bool
}

func (f *fakeApplier) Apply(_ context.Context, _ *sql.DB, t Target) (Report, error) {
	f.mu.Lock()
	f.calls = append(f.calls, t.TenantID)
	f.mu.Unlock()
	if f.fail[t.TenantID] {
		return Report{Target: t.String(), Error: "boom"}, errors.New("boom")
	}
	return Report{Target: t.String(), Done: true}, nil
}

func fleetTargets(n int) []Target {
	out := make([]Target, n)
	for i := range out {
		out[i] = Target{TenantID: uuid.NewString(), App: "orm"}
	}
	return out
}

func noDB(context.Context, Target) (*sql.DB, func(), error) { return nil, nil, nil }

func TestFleet_ABadWaveStopsTheRollout(t *testing.T) {
	ts := fleetTargets(6)
	ap := &fakeApplier{fail: map[string]bool{ts[2].TenantID: true}} // second wave
	f := &Fleet{Runner: ap, Connect: noDB, WaveSize: 2}

	rep, err := f.Apply(context.Background(), ts)
	require.NoError(t, err)
	require.False(t, rep.Done)
	require.Equal(t, 1, rep.Failed)
	require.Equal(t, 2, rep.Skipped)
	require.Len(t, rep.Waves, 3)
	require.ElementsMatch(t, []string{ts[0].TenantID, ts[1].TenantID, ts[2].TenantID, ts[3].TenantID}, ap.calls,
		"the third wave must never be touched")
	for _, r := range rep.Waves[2] {
		require.True(t, r.Skipped)
		require.False(t, r.Report.Done)
	}
}

func TestFleet_ToleratesFailuresUpToTheThreshold(t *testing.T) {
	ts := fleetTargets(4)
	ap := &fakeApplier{fail: map[string]bool{ts[0].TenantID: true}}
	rep, err := (&Fleet{Runner: ap, Connect: noDB, WaveSize: 2, MaxWaveFailures: 1}).Apply(context.Background(), ts)
	require.NoError(t, err)
	require.Equal(t, 1, rep.Failed)
	require.Zero(t, rep.Skipped)
	require.False(t, rep.Done, "a rollout with a failed target is never Done")
	require.Len(t, ap.calls, 4)
}

func TestFleet_AllGoodIsDone(t *testing.T) {
	ap := &fakeApplier{}
	rep, err := (&Fleet{Runner: ap, Connect: noDB, WaveSize: 3, Concurrency: 2}).Apply(context.Background(), fleetTargets(7))
	require.NoError(t, err)
	require.True(t, rep.Done)
	require.Len(t, ap.calls, 7)
}

func TestFleet_ConnectFailureIsAFailedTarget(t *testing.T) {
	ts := fleetTargets(3)
	conn := func(_ context.Context, tg Target) (*sql.DB, func(), error) {
		if tg.TenantID == ts[0].TenantID {
			return nil, nil, errors.New("no route")
		}
		return nil, nil, nil
	}
	ap := &fakeApplier{}
	rep, err := (&Fleet{Runner: ap, Connect: conn, WaveSize: 1}).Apply(context.Background(), ts)
	require.NoError(t, err)
	require.Equal(t, 1, rep.Failed)
	require.Equal(t, 2, rep.Skipped)
	require.Empty(t, ap.calls, "nothing runs for a target that could not be reached, and the rollout stops")
	require.Contains(t, rep.Waves[0][0].Report.Error, "no route")
}

func TestFleet_CancellationSkipsTheRest(t *testing.T) {
	ts := fleetTargets(4)
	ctx, cancel := context.WithCancel(context.Background())
	var n int32
	ap := &cancelAfter{cancel: cancel, after: 2, n: &n}
	rep, err := (&Fleet{Runner: ap, Connect: noDB, WaveSize: 1}).Apply(ctx, ts)
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 2, rep.Skipped)
	require.False(t, rep.Done)
}

type cancelAfter struct {
	cancel context.CancelFunc
	after  int32
	n      *int32
}

func (c *cancelAfter) Apply(_ context.Context, _ *sql.DB, t Target) (Report, error) {
	if atomic.AddInt32(c.n, 1) == c.after {
		c.cancel()
	}
	return Report{Target: t.String(), Done: true}, nil
}

func TestFleet_RejectsBadInput(t *testing.T) {
	ok := Target{TenantID: tid, App: "orm"}
	for name, tc := range map[string]struct {
		f  *Fleet
		ts []Target
	}{
		"no runner":   {&Fleet{Connect: noDB, WaveSize: 1}, []Target{ok}},
		"no connect":  {&Fleet{Runner: &fakeApplier{}, WaveSize: 1}, []Target{ok}},
		"no wave":     {&Fleet{Runner: &fakeApplier{}, Connect: noDB}, []Target{ok}},
		"duplicate":   {&Fleet{Runner: &fakeApplier{}, Connect: noDB, WaveSize: 1}, []Target{ok, ok}},
		"bad target":  {&Fleet{Runner: &fakeApplier{}, Connect: noDB, WaveSize: 1}, []Target{{TenantID: "x", App: "orm"}}},
		"path as app": {&Fleet{Runner: &fakeApplier{}, Connect: noDB, WaveSize: 1}, []Target{{TenantID: tid, App: "../x"}}},
	} {
		_, err := tc.f.Apply(context.Background(), tc.ts)
		require.Error(t, err, name)
	}
}

// --- against a real Postgres ---------------------------------------------------------------

// TENANT_MIGRATE_TEST_DSN names a superuser on a SCRATCH database; the test drops its public
// schema and ivy_meta before each case.
func scratch(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TENANT_MIGRATE_TEST_DSN")
	if dsn == "" {
		t.Skip("TENANT_MIGRATE_TEST_DSN not set")
	}
	db, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	var name string
	require.NoError(t, db.QueryRow(`SELECT current_database()`).Scan(&name))
	require.Contains(t, name, "tdb", "refusing to drop schemas of %q", name)
	for _, q := range []string{`DROP SCHEMA IF EXISTS ivy_meta CASCADE`, `DROP SCHEMA public CASCADE`, `CREATE SCHEMA public`} {
		_, err := db.Exec(q)
		require.NoError(t, err)
	}
	return db
}

func appDir(t *testing.T, files map[string]string) *TenantRunner {
	t.Helper()
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "orm"), 0o755))
	for n, body := range files {
		require.NoError(t, os.WriteFile(filepath.Join(root, "orm", n), []byte(body), 0o644))
	}
	return &TenantRunner{Root: root}
}

func write(t *testing.T, r *TenantRunner, name, body string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(r.Root, "orm", name), []byte(body), 0o644))
}

func count(t *testing.T, db *sql.DB, q string) int {
	t.Helper()
	var n int
	require.NoError(t, db.QueryRow(q).Scan(&n))
	return n
}

func exists(t *testing.T, db *sql.DB, rel string) bool {
	t.Helper()
	var ok bool
	require.NoError(t, db.QueryRow(`SELECT to_regclass($1) IS NOT NULL`, rel).Scan(&ok))
	return ok
}

var threeFiles = map[string]string{
	"001_counter.up.sql": `CREATE TABLE counter (n int); INSERT INTO counter VALUES (1);`,
	"002_orders.up.sql":  `CREATE TABLE orders (id int primary key);`,
	"003_items.up.sql":   `CREATE TABLE items (id int primary key);`,
}

func TestRunner_AppliesPendingOnceAndOnlyOnce(t *testing.T) {
	db, ctx := scratch(t), context.Background()
	r := appDir(t, threeFiles)

	rep, err := r.Inspect(ctx, db, target("orm"))
	require.NoError(t, err)
	require.Len(t, rep.Pending, 3)
	require.False(t, exists(t, db, "ivy_meta.migration_log"), "Inspect must not change the database")

	rep, err = r.Apply(ctx, db, target("orm"))
	require.NoError(t, err)
	require.True(t, rep.Done)
	require.Len(t, rep.Ran, 3)
	require.Len(t, rep.Applied, 3)
	require.True(t, exists(t, db, "items"))

	rep, err = r.Apply(ctx, db, target("orm"))
	require.NoError(t, err)
	require.True(t, rep.Done)
	require.Empty(t, rep.Ran, "an applied file must never run again")
	require.Equal(t, 1, count(t, db, `SELECT count(*) FROM counter`))
}

func TestRunner_ResumesAfterAPartialFailure(t *testing.T) {
	db, ctx := scratch(t), context.Background()
	r := appDir(t, map[string]string{
		"001_counter.up.sql": threeFiles["001_counter.up.sql"],
		"002_orders.up.sql":  threeFiles["002_orders.up.sql"],
		// Fails on its second statement: the first must roll back with it.
		"003_bad.up.sql": `CREATE TABLE half_done (id int); SELECT 1/0;`,
	})

	rep, err := r.Apply(ctx, db, target("orm"))
	require.Error(t, err)
	require.False(t, rep.Done)
	require.Equal(t, []string{"001_counter.up.sql", "002_orders.up.sql"}, rep.Ran)
	require.Equal(t, []string{"003_bad.up.sql"}, rep.Pending)
	require.NotEmpty(t, rep.Error)
	require.False(t, exists(t, db, "half_done"), "a failed file must roll back whole")
	require.Equal(t, 2, count(t, db, `SELECT count(*) FROM ivy_meta.migration_log`))

	// The failed file was never applied, so fixing it is legitimate. Applied files are untouched.
	write(t, r, "003_bad.up.sql", `CREATE TABLE half_done (id int);`)
	rep, err = r.Apply(ctx, db, target("orm"))
	require.NoError(t, err)
	require.True(t, rep.Done)
	require.Equal(t, []string{"003_bad.up.sql"}, rep.Ran, "resume must run only what was not applied")
	require.Equal(t, 1, count(t, db, `SELECT count(*) FROM counter`), "001 must not have run twice")
}

func TestRunner_ResumesAfterAKilledRun(t *testing.T) {
	db := scratch(t)
	r := appDir(t, threeFiles)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // the orchestrator dies before applying anything
	_, err := r.Apply(ctx, db, target("orm"))
	require.Error(t, err)

	rep, err := r.Apply(context.Background(), db, target("orm"))
	require.NoError(t, err)
	require.True(t, rep.Done)
	require.Equal(t, 1, count(t, db, `SELECT count(*) FROM counter`))
}

func TestRunner_DriftStopsEverything(t *testing.T) {
	ctx := context.Background()
	cases := map[string]func(t *testing.T, r *TenantRunner){
		"an applied file was edited": func(t *testing.T, r *TenantRunner) {
			write(t, r, "002_orders.up.sql", `CREATE TABLE orders (id int primary key, sneaky int);`)
		},
		"an applied file was deleted": func(t *testing.T, r *TenantRunner) {
			require.NoError(t, os.Remove(filepath.Join(r.Root, "orm", "002_orders.up.sql")))
		},
		"a file was inserted into the past": func(t *testing.T, r *TenantRunner) {
			write(t, r, "000_early.up.sql", `CREATE TABLE early (id int);`)
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			db := scratch(t)
			r := appDir(t, threeFiles)
			_, err := r.Apply(ctx, db, target("orm"))
			require.NoError(t, err)

			mutate(t, r)
			write(t, r, "004_new.up.sql", `CREATE TABLE brand_new (id int);`)

			rep, err := r.Apply(ctx, db, target("orm"))
			require.ErrorIs(t, err, ErrDrift)
			require.NotEmpty(t, rep.Drift)
			require.False(t, rep.Done)
			require.Empty(t, rep.Ran)
			require.False(t, exists(t, db, "brand_new"), "nothing may run while a target has drifted")
			require.False(t, exists(t, db, "early"))
		})
	}
}

func TestRunner_OnlyOneRunnerPerDatabase(t *testing.T) {
	db, ctx := scratch(t), context.Background()
	r := appDir(t, threeFiles)

	holder, err := db.Conn(ctx)
	require.NoError(t, err)
	defer holder.Close()
	var got bool
	require.NoError(t, holder.QueryRowContext(ctx, `SELECT pg_try_advisory_lock(hashtext('ivy_tenant_migrate'))`).Scan(&got))
	require.True(t, got)

	rep, err := r.Apply(ctx, db, target("orm"))
	require.ErrorIs(t, err, ErrTargetBusy)
	require.Empty(t, rep.Ran)
	require.False(t, exists(t, db, "counter"))
}

func TestRunner_ConcurrentRunnersNeverDoubleApply(t *testing.T) {
	db, ctx := scratch(t), context.Background()
	r := appDir(t, threeFiles)
	var wg sync.WaitGroup
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := r.Apply(ctx, db, target("orm"))
			if err != nil {
				require.ErrorIs(t, err, ErrTargetBusy)
			}
		}()
	}
	wg.Wait()
	rep, err := r.Apply(ctx, db, target("orm"))
	require.NoError(t, err)
	require.True(t, rep.Done)
	require.Equal(t, 1, count(t, db, `SELECT count(*) FROM counter`))
	require.Equal(t, 3, count(t, db, `SELECT count(*) FROM ivy_meta.migration_log`))
}

func TestRunner_RefusesTransactionControl(t *testing.T) {
	db, ctx := scratch(t), context.Background()
	r := appDir(t, map[string]string{"001_x.up.sql": `CREATE TABLE x (id int); COMMIT;`})
	rep, err := r.Apply(ctx, db, target("orm"))
	require.Error(t, err)
	require.Empty(t, rep.Ran)
	require.False(t, exists(t, db, "x"))
}

func TestRunner_BaselineRecordsWithoutRunning(t *testing.T) {
	db, ctx := scratch(t), context.Background()
	r := appDir(t, threeFiles)

	rep, err := r.Baseline(ctx, db, target("orm"), "002_orders.up.sql")
	require.NoError(t, err)
	require.Len(t, rep.Applied, 2)
	require.True(t, rep.Applied[0].Baseline)
	require.Equal(t, []string{"003_items.up.sql"}, rep.Pending)
	require.False(t, exists(t, db, "counter"), "a baseline must not execute the files it records")

	// The runner then carries on from the baseline: only the later file runs.
	_, err = db.Exec(`CREATE TABLE orders (id int primary key)`) // what the clone already had
	require.NoError(t, err)
	rep, err = r.Apply(ctx, db, target("orm"))
	require.NoError(t, err)
	require.True(t, rep.Done)
	require.Equal(t, []string{"003_items.up.sql"}, rep.Ran)

	// A baseline never papers over a real history.
	_, err = r.Baseline(ctx, db, target("orm"), "003_items.up.sql")
	require.ErrorIs(t, err, ErrBaselineNotEmpty)
	_, err = r.Baseline(ctx, db, target("orm"), "nope.up.sql")
	require.Error(t, err)
}

func TestRunner_UnknownAppIsAnError(t *testing.T) {
	db := scratch(t)
	r := appDir(t, threeFiles)
	_, err := r.Apply(context.Background(), db, target("crm"))
	require.ErrorIs(t, err, ErrNoApp)
	_, err = r.Inspect(context.Background(), db, Target{TenantID: tid, App: "../orm"})
	require.Error(t, err)
}

// The file's SQL and its log row commit together or not at all: if the log insert fails after
// the file ran, the file's effects must roll back too, or the database holds a change the log
// does not know about.
func TestRunner_ALogFailureRollsTheFileBack(t *testing.T) {
	db, ctx := scratch(t), context.Background()
	r := appDir(t, threeFiles)
	for _, q := range []string{
		`CREATE SCHEMA ivy_meta`,
		`CREATE TABLE ivy_meta.migration_log (
			filename TEXT PRIMARY KEY, sha256 TEXT NOT NULL, baseline BOOLEAN NOT NULL DEFAULT false,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			CONSTRAINT refuse_002 CHECK (filename <> '002_orders.up.sql'))`,
	} {
		_, err := db.Exec(q)
		require.NoError(t, err)
	}

	rep, err := r.Apply(ctx, db, target("orm"))
	require.Error(t, err)
	require.Equal(t, []string{"001_counter.up.sql"}, rep.Ran)
	require.False(t, exists(t, db, "orders"), "002 ran but could not be logged; it must have rolled back")
	require.Equal(t, 1, count(t, db, `SELECT count(*) FROM ivy_meta.migration_log`))
}

// The repository's orm directory exists, so App="orm" is a valid app (even while it has no
// migrations) and an unknown app is still ErrNoApp.
func TestRepoOrmDirectoryIsAValidApp(t *testing.T) {
	r := &TenantRunner{Root: "../../db/tenant_migrations"}
	_, _, err := r.files(target("orm"))
	require.NoError(t, err)
	_, _, err = r.files(target("nosuchapp"))
	require.ErrorIs(t, err, ErrNoApp)
}
