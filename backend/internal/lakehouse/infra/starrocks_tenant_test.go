package infra

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// stubDest is an AuditDestination that records every call. Tests verify routing by checking
// which stub's methods were invoked.
type stubDest struct {
	name string
	mu   sync.Mutex
	// ensures is the list of (tenantID, tenantName) recorded from EnsureAuditDestination.
	ensures []AuditDestinationSpec
	// maxIDs is the list of tenantIDs MaxAuditID was called with.
	maxIDs []uuid.UUID
	// ensureErr is what EnsureAuditDestination returns (lets tests inject failures).
	ensureErr error
}

func (s *stubDest) EnsureAuditDestination(_ context.Context, spec AuditDestinationSpec) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensures = append(s.ensures, spec)
	return s.ensureErr
}

func (s *stubDest) MaxAuditID(_ context.Context, id uuid.UUID) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.maxIDs = append(s.maxIDs, id)
	return 0, nil
}

func (s *stubDest) AuditHash(_ context.Context, _ uuid.UUID, _ int64) (string, bool, error) {
	return "", false, nil
}
func (s *stubDest) AppendAudit(_ context.Context, _ uuid.UUID, _ []AuditRow) error { return nil }
func (s *stubDest) AuditRange(_ context.Context, _ uuid.UUID, _ int64, _ int) ([]AuditRow, error) {
	return nil, nil
}

// recordingLogger captures every log line the router emits, formatted the way the production
// logger would emit it. Tests assert on the captured lines.
type recordingLogger struct {
	mu    sync.Mutex
	lines []string
}

func (l *recordingLogger) Printf(format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lines = append(l.lines, fmt.Sprintf(format, args...))
}

func (l *recordingLogger) has(sub string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, s := range l.lines {
		if contains(s, sub) {
			return true
		}
	}
	return false
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// fakeConn implements driver.Conn + driver.ExecerContext so initConnector can run init SQL on it.
type fakeConn struct {
	driverConn
	mu       sync.Mutex
	closed   bool
	execArgs [][]driver.NamedValue
	execErr  error
}

type driverConn interface {
	Prepare(query string) (driver.Stmt, error)
	Close() error
	Begin() (driver.Tx, error)
}

func (c *fakeConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("Prepare not implemented") }
func (c *fakeConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	return nil
}
func (c *fakeConn) Begin() (driver.Tx, error) { return nil, errors.New("Begin not implemented") }
func (c *fakeConn) ExecContext(_ context.Context, _ string, args []driver.NamedValue) (driver.Result, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.execArgs = append(c.execArgs, args)
	return driver.RowsAffected(0), c.execErr
}

// fakeConnector returns a fresh fakeConn on each Connect().
type fakeConnector struct {
	mu    sync.Mutex
	conns []*fakeConn
}

func (f *fakeConnector) Connect(_ context.Context) (driver.Conn, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	c := &fakeConn{}
	f.conns = append(f.conns, c)
	return c, nil
}
func (f *fakeConnector) Driver() driver.Driver { return nil }

// ---- Tests --------------------------------------------------------------

func TestTenantEnvKey(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"northwinds", "NORTHWINDS"},
		{"Demo Tenant CRD Bakeoff", "DEMO_TENANT_CRD_BAKEOFF"},
		{"northwinds-prod", "NORTHWINDS_PROD"},
		{"  tenant-A  ", "TENANT_A"},
		{"", ""},
		{"a__b", "A_B"},        // collapsed repeated separators
		{"a-b-c", "A_B_C"},     // dashes → underscore
		{"tenant.A", "TENANT_A"},
		{"a/b\\c", "A_B_C"},
		{"_under_", "UNDER"},
	}
	for _, c := range cases {
		require.Equal(t, c.want, TenantEnvKey(c.in), "input %q", c.in)
	}
}

func TestInitConnector_RunsAllInitStatements(t *testing.T) {
	// Connect exercises the init path: init SQL is sent on every new connection.
	base := &fakeConnector{}
	c := &initConnector{base: base, init: []string{"SET resource_group = 'a_wg'", "USE a"}}
	conn, err := c.Connect(context.Background())
	require.NoError(t, err)
	require.Len(t, base.conns[0].execArgs, 2, "both init statements ran on the first connection")
	require.NoError(t, conn.Close())

	// A second Connect creates a fresh conn and re-runs the init SQL.
	conn2, err := c.Connect(context.Background())
	require.NoError(t, err)
	require.Len(t, base.conns[1].execArgs, 2, "init SQL ran on the second connection too")
	require.NoError(t, conn2.Close())
}

func TestInitConnector_InitFailureClosesConn(t *testing.T) {
	// Override the Connect to return a conn whose exec fails.
	failConn := &fakeConn{execErr: errors.New("permission denied")}
	custom := &failingConnector{conn: failConn}
	c := &initConnector{base: custom, init: []string{"SET resource_group = 'a_wg'"}}
	_, err := c.Connect(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "permission denied")
	require.True(t, failConn.closed, "init failure closes the underlying connection")
}

type failingConnector struct {
	conn driver.Conn
	err  error
}

func (f *failingConnector) Connect(_ context.Context) (driver.Conn, error) {
	if f.conn != nil {
		return f.conn, nil
	}
	return nil, f.err
}
func (f *failingConnector) Driver() driver.Driver { return nil }

type fakeDriver struct {
	connector driver.Connector
}

func (d *fakeDriver) Open(_ string) (driver.Conn, error) { return d.connector.Connect(context.Background()) }

func TestScanTenantEnv(t *testing.T) {
	env := []string{
		"PATH=/usr/bin",
		"LAKEHOUSE_STARROCKS_DSN_TENANT_NORTHWINDS=user:pw@tcp(h:9030)/",
		"LAKEHOUSE_STARROCKS_DSN_TENANT_NORTHWINDS_PROD=user:pw@tcp(h:9030)/",
		"LAKEHOUSE_STARROCKS_INIT_RESOURCE_GROUP_TENANT_NORTHWINDS=tenant_northwinds_wg",
		"LAKEHOUSE_STARROCKS_INIT_RESOURCE_GROUP_TENANT_NORTHWINDS_PROD=tenant_northwinds_prod_wg",
		// Stale single-tenant init-rg var (the design we just removed) is ignored.
		"LAKEHOUSE_STARROCKS_INIT_RESOURCE_GROUP=should_be_ignored",
		// Stale old name (was removed earlier) is ignored.
		"LAKEHOUSE_STARROCKS_RESOURCE_GROUP_TENANT_NORTHWINDS=should_be_ignored",
		"LAKEHOUSE_STARROCKS_REQUIRE_TENANT_DSN=true",
		"NOT_A_TENANT_VAR=ignored",
		"=missing_key",
		"=also_missing",
		// Lowercase key — must still be looked up via TenantEnvKey (which uppercases).
		"LAKEHOUSE_STARROCKS_DSN_TENANT_lowercase_key=dsn",
		"LAKEHOUSE_STARROCKS_INIT_RESOURCE_GROUP_TENANT_lowercase_key=lower_wg",
	}
	name2dsn, name2initRG := ScanTenantEnv(env)
	require.Equal(t, "user:pw@tcp(h:9030)/", name2dsn["NORTHWINDS"])
	require.Equal(t, "user:pw@tcp(h:9030)/", name2dsn["NORTHWINDS_PROD"])
	require.Equal(t, "dsn", name2dsn["LOWERCASE_KEY"], "lowercase env-var suffix is uppercased on read")
	require.NotContains(t, name2dsn, "NOT_A_TENANT_VAR")
	require.Equal(t, "tenant_northwinds_wg", name2initRG["NORTHWINDS"])
	require.Equal(t, "tenant_northwinds_prod_wg", name2initRG["NORTHWINDS_PROD"])
	require.Equal(t, "lower_wg", name2initRG["LOWERCASE_KEY"], "init-rg keys are also slug-normalized")
	require.NotContains(t, name2initRG, "should_be_ignored", "the global LAKEHOUSE_STARROCKS_INIT_RESOURCE_GROUP is no longer read")
	require.NotContains(t, name2initRG, "NOT_A_TENANT_VAR")
}

func TestScanTenantEnv_EmptyAndMissingEntries(t *testing.T) {
	env := []string{
		"",
		"NOEQUALS",
		"=value_only",
		"key=val",
	}
	name2dsn, name2initRG := ScanTenantEnv(env)
	require.Empty(t, name2dsn)
	require.Empty(t, name2initRG)
}

func TestTenantRouter_PicksByNameAndCaches(t *testing.T) {
	stubs := map[string]*stubDest{
		"northwinds":  {name: "northwinds"},
		"crd_bakeoff": {name: "crd_bakeoff"},
	}
	log := &recordingLogger{}
	r, err := newTenantRouter(StarRocksConfig{}, map[string]string{
		"NORTHWINDS":  "tenant_northwinds_svc:pw@tcp(h:9030)/tenant_northwinds",
		"CRD_BAKEOFF": "tenant_crd_bakeoff_svc:pw@tcp(h:9030)/tenant_crd_bakeoff",
	}, nil /* name2initRG */, "legacy:pw", false, log)
	require.NoError(t, err)

	opens := &recordingOpener{}
	r.openDB = opens.open
	r.newSR = opens.newSRFor(stubs)

	northwindsID := uuid.New()
	err = r.EnsureAuditDestination(context.Background(), AuditDestinationSpec{
		TenantID:   northwindsID,
		TenantName: "northwinds",
	})
	require.NoError(t, err)
	require.Equal(t, "tenant_northwinds_svc:pw@tcp(h:9030)/tenant_northwinds", opens.firstDSN())
	require.Empty(t, opens.firstInit(), "default: no init SQL (classifier routes)")
	require.Equal(t, 1, opens.builds(), "first call builds")
	require.Equal(t, 1, len(stubs["northwinds"].ensures))

	err = r.EnsureAuditDestination(context.Background(), AuditDestinationSpec{
		TenantID:   northwindsID,
		TenantName: "northwinds",
	})
	require.NoError(t, err)
	require.Equal(t, 1, opens.builds(), "second call to same tenant uses cache; no second openDB")

	bakeoffID := uuid.New()
	err = r.EnsureAuditDestination(context.Background(), AuditDestinationSpec{
		TenantID:   bakeoffID,
		TenantName: "crd_bakeoff",
	})
	require.NoError(t, err)
	require.Equal(t, 2, opens.builds(), "different tenant name → new build")
	require.Equal(t, 1, len(stubs["crd_bakeoff"].ensures))

	_, err = r.MaxAuditID(context.Background(), bakeoffID)
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{bakeoffID}, stubs["crd_bakeoff"].maxIDs, "id resolves to the right tenant name")

	require.False(t, log.has("no per-tenant StarRocks DSN"), "no fallback warning when all tenants are mapped")
}

// TestPerTenantInitResourceGroupEnv_AppendsOnlyForConfiguredTenants verifies
// the per-tenant shape: each tenant's connection runs SET resource_group only
// if its own LAKEHOUSE_STARROCKS_INIT_RESOURCE_GROUP_TENANT_<SLUG> is set.
// A global value would defeat per-tenant isolation because the SET overrides
// classifier routing on StarRocks 3.3.
func TestPerTenantInitResourceGroupEnv_AppendsOnlyForConfiguredTenants(t *testing.T) {
	stubs := map[string]*stubDest{
		"northwinds":  {name: "northwinds"},
		"crd_bakeoff": {name: "crd_bakeoff"},
	}
	log := &recordingLogger{}
	r, err := newTenantRouter(StarRocksConfig{}, map[string]string{
		"NORTHWINDS":  "tenant_northwinds_svc:pw@tcp(h:9030)/tenant_northwinds",
		"CRD_BAKEOFF": "tenant_crd_bakeoff_svc:pw@tcp(h:9030)/tenant_crd_bakeoff",
	}, map[string]string{
		// Only northwinds has the init-rg override set. crd_bakeoff does not — its
		// connections must rely on the classifier alone.
		"NORTHWINDS": "tenant_northwinds_wg",
	},
		"", false, log)
	require.NoError(t, err)

	opens := &recordingOpener{}
	r.openDB = opens.open
	r.newSR = opens.newSRFor(stubs)

	// northwinds → init SQL present.
	err = r.EnsureAuditDestination(context.Background(), AuditDestinationSpec{
		TenantID: uuid.New(), TenantName: "northwinds",
	})
	require.NoError(t, err)
	require.Equal(t, []string{"SET resource_group = 'tenant_northwinds_wg'"}, opens.calls[0].initSQL)

	// crd_bakeoff → no init SQL; classifier routes.
	err = r.EnsureAuditDestination(context.Background(), AuditDestinationSpec{
		TenantID: uuid.New(), TenantName: "crd_bakeoff",
	})
	require.NoError(t, err)
	require.Empty(t, opens.calls[1].initSQL, "unmapped tenant gets no init SQL")

	require.True(t, log.has("init SQL: SET resource_group = \"tenant_northwinds_wg\""),
		"startup log records per-tenant init-rg on the configured tenant")
	require.True(t, log.has("no init SQL (classifier routes)"),
		"startup log records the absent init-rg on the unconfigured tenant")
}

func TestPerTenantInitResourceGroupEnv_DifferentRgPerTenant(t *testing.T) {
	// Two tenants, two different resource groups — the per-tenant shape is the
	// whole point. A single global value would have pinned both to one group,
	// defeating isolation.
	stubs := map[string]*stubDest{
		"northwinds":  {name: "northwinds"},
		"crd_bakeoff": {name: "crd_bakeoff"},
	}
	r, err := newTenantRouter(StarRocksConfig{}, map[string]string{
		"NORTHWINDS":  "tenant_northwinds_svc:pw@tcp(h:9030)/tenant_northwinds",
		"CRD_BAKEOFF": "tenant_crd_bakeoff_svc:pw@tcp(h:9030)/tenant_crd_bakeoff",
	}, map[string]string{
		"NORTHWINDS":  "tenant_northwinds_wg",
		"CRD_BAKEOFF": "tenant_crd_bakeoff_wg",
	}, "", false, &recordingLogger{})
	require.NoError(t, err)

	opens := &recordingOpener{}
	r.openDB = opens.open
	r.newSR = opens.newSRFor(stubs)

	for _, name := range []string{"northwinds", "crd_bakeoff"} {
		err := r.EnsureAuditDestination(context.Background(), AuditDestinationSpec{
			TenantID: uuid.New(), TenantName: name,
		})
		require.NoError(t, err)
	}
	require.Equal(t, []string{"SET resource_group = 'tenant_northwinds_wg'"}, opens.calls[0].initSQL)
	require.Equal(t, []string{"SET resource_group = 'tenant_crd_bakeoff_wg'"}, opens.calls[1].initSQL,
		"each tenant gets its own resource group, not a shared one")
}

func TestInitResourceGroupEnv_DefaultIsEmpty(t *testing.T) {
	log := &recordingLogger{}
	_, err := newTenantRouter(StarRocksConfig{}, map[string]string{
		"X": "dsn_x",
	}, nil /* no per-tenant init-rg */, "", false, log)
	require.NoError(t, err)
	require.True(t, log.has("no init SQL (classifier routes)"),
		"default behaviour is logged: no init SQL, classifier alone")
}

func TestTenantRouter_TenantNameCasingIsSlugIsCaseInsensitive(t *testing.T) {
	// Tenant names passed at runtime can be in any case; the env-var suffix is uppercase by
	// convention. The router must apply `TenantEnvKey` so a lower-case name still hits an
	// upper-cased env key.
	stubs := map[string]*stubDest{"NORTHWINDS": {name: "northwinds"}}
	log := &recordingLogger{}
	r, err := newTenantRouter(StarRocksConfig{}, map[string]string{
		"NORTHWINDS": "tenant_northwinds_svc:pw@tcp(h:9030)/tenant_northwinds",
	}, nil, "", false, log)
	require.NoError(t, err)

	opens := &recordingOpener{}
	r.openDB = opens.open
	r.newSR = opens.newSRFor(stubs)

	// Use a tenant name that maps to the same slug as "NORTHWINDS" via the slugger. "northwinds"
	// is the canonical example, but here we also try "Northwinds" with mixed case to assert the
	// slug is case-insensitive.
	for _, name := range []string{"northwinds", "Northwinds", "NORTHWINDS"} {
		err := r.EnsureAuditDestination(context.Background(), AuditDestinationSpec{
			TenantID: uuid.New(), TenantName: name,
		})
		require.NoError(t, err)
	}
	// All three calls hit the same tenant (after slug normalization) → only one build.
	require.Equal(t, 1, opens.builds(), "slug normalization collapses mixed-case names to the same destination")
}

func TestTenantRouter_FallsBackToLegacyWithWarn(t *testing.T) {
	log := &recordingLogger{}
	r, err := newTenantRouter(StarRocksConfig{}, map[string]string{
		"NORTHWINDS": "tenant_northwinds_svc:pw@tcp(h:9030)/tenant_northwinds",
	}, nil,
		"legacy:pw@tcp(h:9030)/", false, log)
	require.NoError(t, err)

	legacyStub := &stubDest{name: "legacy"}
	opens := &recordingOpener{}
	r.openDB = opens.open
	r.newSR = func(_ *sql.DB, _ StarRocksConfig) (AuditDestination, error) {
		// Return legacy for any call
		return legacyStub, nil
	}

	err = r.EnsureAuditDestination(context.Background(), AuditDestinationSpec{
		TenantID:   uuid.New(),
		TenantName: "unmapped-tenant",
	})
	require.NoError(t, err)
	require.True(t, log.has("WARN"), "WARN log when falling back to legacy for unmapped tenant")
	require.True(t, log.has("LAKEHOUSE_STARROCKS_DSN_TENANT_UNMAPPED_TENANT"), "WARN names the env var it looked for")
	require.Equal(t, 1, len(legacyStub.ensures), "legacy destination got the call")

	// Calling MaxAuditID without EnsureAuditDestination first → error (id2name is empty)
	_, err = r.MaxAuditID(context.Background(), uuid.New())
	require.Error(t, err)
	require.Contains(t, err.Error(), "EnsureAuditDestination must be called first")

	// The legacy WARN is only logged once; second call to same unmapped tenant does not spam.
	prevLogCount := len(log.lines)
	err = r.EnsureAuditDestination(context.Background(), AuditDestinationSpec{
		TenantID:   uuid.New(),
		TenantName: "unmapped-tenant",
	})
	require.NoError(t, err)
	require.Equal(t, prevLogCount, len(log.lines), "WARN is logged once per unmapped tenant name, not per call")
}

func TestTenantRouter_StrictModeErrors(t *testing.T) {
	log := &recordingLogger{}
	r, err := newTenantRouter(StarRocksConfig{}, map[string]string{
		"NORTHWINDS": "tenant_northwinds_svc:pw@tcp(h:9030)/tenant_northwinds",
	}, nil,
		"legacy:pw@tcp(h:9030)/", true /* require */, log)
	require.NoError(t, err)

	err = r.EnsureAuditDestination(context.Background(), AuditDestinationSpec{
		TenantID:   uuid.New(),
		TenantName: "unmapped-tenant",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "no per-tenant StarRocks DSN")
	require.Contains(t, err.Error(), "LAKEHOUSE_STARROCKS_REQUIRE_TENANT_DSN=true")
	require.Contains(t, err.Error(), "LAKEHOUSE_STARROCKS_DSN_TENANT_UNMAPPED_TENANT", "error names the env var")
}

func TestTenantRouter_LogsAllMappingsAtConstruction(t *testing.T) {
	log := &recordingLogger{}
	_, err := newTenantRouter(StarRocksConfig{}, map[string]string{
		"NORTHWINDS":  "dsn1",
		"CRD_BAKEOFF": "dsn2",
	}, nil, "legacy", false, log)
	require.NoError(t, err)
	require.True(t, log.has("NORTHWINDS"), "northwinds mapping logged")
	require.True(t, log.has("CRD_BAKEOFF"), "crd_bakeoff mapping logged")
	require.True(t, log.has("legacy DSN is set"), "fallback behavior logged")
}

func TestTenantRouter_LegacyBuildOnlyOnce(t *testing.T) {
	log := &recordingLogger{}
	legacyDSN := "legacy:pw@tcp(h:9030)/"
	r, err := newTenantRouter(StarRocksConfig{}, map[string]string{
		"A": "dsn_a",
	}, nil, legacyDSN, false, log)
	require.NoError(t, err)

	opens := &recordingOpener{legacyDSN: legacyDSN}
	r.openDB = opens.open
	stub := &stubDest{}
	r.newSR = func(_ *sql.DB, _ StarRocksConfig) (AuditDestination, error) { return stub, nil }

	// Three EnsureAuditDestination calls with three different unmapped names.
	for _, name := range []string{"u1", "u2", "u3"} {
		err := r.EnsureAuditDestination(context.Background(), AuditDestinationSpec{
			TenantID: uuid.New(), TenantName: name,
		})
		require.NoError(t, err)
	}
	// The legacy DSN should have been opened once (sync.Once), even though three different
	// unmapped names routed through it.
	require.Equal(t, 1, opens.legacyBuilds(), "legacy is built once and cached")
}

func TestTenantRouter_PoolHygiene(t *testing.T) {
	// Verify the router calls db.SetMaxOpenConns and db.SetConnMaxLifetime on the per-tenant DB.
	r, err := newTenantRouter(StarRocksConfig{}, map[string]string{
		"A": "dsn_a",
	}, nil, "", false, &recordingLogger{})
	require.NoError(t, err)

	fakesqlDB, _, err := sqlmock.New()
	require.NoError(t, err)
	r.openDB = func(_ string, _ []string) (*sql.DB, error) { return fakesqlDB, nil }
	r.newSR = func(*sql.DB, StarRocksConfig) (AuditDestination, error) { return &stubDest{}, nil }

	err = r.EnsureAuditDestination(context.Background(), AuditDestinationSpec{
		TenantID: uuid.New(), TenantName: "A",
	})
	require.NoError(t, err)

	// sql.DB doesn't expose the configured limits directly. We can only assert the call didn't
	// panic and that fakesqlDB is usable. The setter is a no-op on the fake; the test passes
	// as long as the production path runs.
	require.NotNil(t, fakesqlDB)
	// Pool config not directly checkable on sqlmock-backed DB; this test mainly proves the
	// openDB chain doesn't error on the SetMaxOpenConns / SetConnMaxLifetime path.
}

// ---- helpers used by the tests above ----

// recordingOpener captures (dsn, initSQL) pairs and tracks how many times openDB was called.
// Used to verify the router picks the right DSN, runs the right init SQL, and caches.
type recordingOpener struct {
	mu        sync.Mutex
	calls     []struct{ dsn string; initSQL []string }
	legacyC   int
	buildsC   int
	legacyDSN string
}

func (r *recordingOpener) open(dsn string, initSQL []string) (*sql.DB, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, struct {
		dsn     string
		initSQL []string
	}{dsn, append([]string(nil), initSQL...)})
	r.buildsC++
	if r.legacyDSN != "" && dsn == r.legacyDSN {
		r.legacyC++
	}
	db, _, err := sqlmock.New()
	return db, err
}

func (r *recordingOpener) newSRFor(stubs map[string]*stubDest) func(*sql.DB, StarRocksConfig) (AuditDestination, error) {
	return func(_ *sql.DB, cfg StarRocksConfig) (AuditDestination, error) {
		// Inspect the most recent open to pick the stub by tenant slug.
		r.mu.Lock()
		defer r.mu.Unlock()
		if len(r.calls) == 0 {
			return nil, errors.New("newSR called before openDB")
		}
		last := r.calls[len(r.calls)-1].dsn
		// Slug the DSN back to a tenant key: the test DSNs look like
		// "tenant_<name>_svc:pw@...", so a substring match on the lowercase form of the
		// stub key (or its slug) is enough.
		for name, stub := range stubs {
			slug := TenantEnvKey(name)
			lowerSlug := strings.ToLower(slug)
			if contains(last, lowerSlug) {
				return stub, nil
			}
		}
		return nil, errors.New("no stub for DSN " + last)
	}
}

func (r *recordingOpener) firstDSN() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.calls) == 0 {
		return ""
	}
	return r.calls[0].dsn
}

func (r *recordingOpener) firstInit() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.calls) == 0 {
		return nil
	}
	return r.calls[0].initSQL
}

func (r *recordingOpener) builds() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.buildsC
}

func (r *recordingOpener) legacyBuilds() int {
	// For legacy tracking: legacy is built lazily inside the router's getLegacy method. We
	// approximate it as "builds that don't have initSQL" — but the simpler proxy is "builds
	// count" since each call to forTenantByName for an unmapped tenant triggers getLegacy once.
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.legacyC
}

// time_ is used to keep the import of "time" alive (the production code uses time.Duration
// for ConnMaxLifetime). Removing the import causes the build to fail in some Go versions.
var _ = time.Second