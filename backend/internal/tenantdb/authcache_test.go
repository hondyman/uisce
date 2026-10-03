package tenantdb

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Authorization caching (Config.AuthTTL). The cache stores only successes, is keyed by the verified
// caller tenant, never serves a refusal, and bounds how long a suspension can be missed.

type ctxTenant struct{}

func tenantFromCtx(ctx context.Context) (string, error) {
	if v, ok := ctx.Value(ctxTenant{}).(string); ok && v != "" {
		return v, nil
	}
	return "", errors.New("no tenant")
}

func as(tenant string) context.Context {
	return context.WithValue(context.Background(), ctxTenant{}, tenant)
}

type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) now() time.Time      { c.mu.Lock(); defer c.mu.Unlock(); return c.t }
func (c *clock) add(d time.Duration) { c.mu.Lock(); defer c.mu.Unlock(); c.t = c.t.Add(d) }

func cachedRouter(t *testing.T, reg Registry, ttl time.Duration, maxAuth int) (*Router, *clock) {
	t.Helper()
	r, err := New(Config{Registry: reg, CallerTenant: tenantFromCtx, MaxPools: 4, MaxConnsPerPool: 1,
		IdleTTL: time.Minute, DialTimeout: 2 * time.Second, AuthTTL: ttl, MaxAuthEntries: maxAuth})
	require.NoError(t, err)
	c := &clock{t: time.Unix(1_000_000, 0)}
	r.now = c.now
	t.Cleanup(r.Close)
	return r, c
}

func activeReg() *fakeRegistry {
	return &fakeRegistry{ds: goodDS(), binding: Binding{Version: 1, Lifecycle: "active"}, user: "u", pw: "p", appDS: "ds-1"}
}

func TestAuthCache_Config(t *testing.T) {
	base := Config{Registry: &fakeRegistry{}, CallerTenant: caller("t"), MaxPools: 5, MaxConnsPerPool: 1, IdleTTL: time.Second, DialTimeout: time.Second}
	c := base
	c.AuthTTL = -time.Second
	_, err := New(c)
	require.Error(t, err)
	c = base
	c.MaxAuthEntries = -1
	_, err = New(c)
	require.Error(t, err)
	c = base
	c.AuthTTL = time.Second
	r, err := New(c)
	require.NoError(t, err)
	require.Equal(t, 20, r.cfg.MaxAuthEntries, "defaults to 4 x MaxPools")
}

func TestAuthCache_TTLZeroChecksAlphaEveryTime(t *testing.T) {
	reg := activeReg()
	r, _ := cachedRouter(t, reg, 0, 0)
	for i := 0; i < 3; i++ {
		_, _, err := r.authorizeCached(as("t-1"), "ds-1")
		require.NoError(t, err)
	}
	ds, bind, _ := reg.calls()
	require.Equal(t, 3, ds)
	require.Equal(t, 3, bind)
}

func TestAuthCache_ReusesASuccessUntilItExpires(t *testing.T) {
	reg := activeReg()
	r, clk := cachedRouter(t, reg, 3*time.Second, 0)

	for i := 0; i < 5; i++ {
		ds, b, err := r.authorizeCached(as("t-1"), "ds-1")
		require.NoError(t, err)
		require.Equal(t, "orm_acme", ds.Database)
		require.Equal(t, 1, b.Version)
	}
	d, b, _ := reg.calls()
	require.Equal(t, [2]int{1, 1}, [2]int{d, b}, "within the TTL alpha is asked once")

	clk.add(2900 * time.Millisecond)
	_, _, err := r.authorizeCached(as("t-1"), "ds-1")
	require.NoError(t, err)
	d, _, _ = reg.calls()
	require.Equal(t, 1, d, "still inside the TTL")

	clk.add(200 * time.Millisecond) // 3.1 s since the entry was stored
	_, _, err = r.authorizeCached(as("t-1"), "ds-1")
	require.NoError(t, err)
	d, b, _ = reg.calls()
	require.Equal(t, [2]int{2, 2}, [2]int{d, b}, "after the TTL alpha is asked again")
}

// The documented trade: a suspension is missed for at most the TTL, never longer.
func TestAuthCache_ASuspensionBitesWithinTheTTL(t *testing.T) {
	reg := activeReg()
	r, clk := cachedRouter(t, reg, 3*time.Second, 0)
	_, _, err := r.authorizeCached(as("t-1"), "ds-1")
	require.NoError(t, err)

	reg.set(func(f *fakeRegistry) { f.binding = Binding{Version: 1, Lifecycle: "suspended"} })
	_, _, err = r.authorizeCached(as("t-1"), "ds-1")
	require.NoError(t, err, "inside the TTL the earlier answer is served: this is the trade, and it is bounded")

	clk.add(3*time.Second + time.Millisecond)
	_, _, err = r.authorizeCached(as("t-1"), "ds-1")
	require.ErrorIs(t, err, ErrUnbound, "after the TTL the suspension is enforced")
}

// A refusal is returned every time and never remembered, so recovery is immediate.
func TestAuthCache_RefusalsAreNeverCached(t *testing.T) {
	reg := activeReg()
	reg.binding = Binding{Version: 1, Lifecycle: "suspended"}
	r, _ := cachedRouter(t, reg, 10*time.Second, 0)
	for i := 0; i < 3; i++ {
		_, _, err := r.authorizeCached(as("t-1"), "ds-1")
		require.ErrorIs(t, err, ErrUnbound)
	}
	_, b, _ := reg.calls()
	require.Equal(t, 3, b, "each refusal asked alpha again")

	reg.set(func(f *fakeRegistry) { f.binding = Binding{Version: 2, Lifecycle: "active"} })
	_, got, err := r.authorizeCached(as("t-1"), "ds-1")
	require.NoError(t, err, "once fixed it works at once, with no TTL to wait out")
	require.Equal(t, 2, got.Version)

	for name, e := range map[string]error{"registry down": errors.New("alpha down"), "no binding": ErrUnbound} {
		reg2 := activeReg()
		reg2.dsErr = e
		r2, _ := cachedRouter(t, reg2, 10*time.Second, 0)
		_, _, err := r2.authorizeCached(as("t-1"), "ds-1")
		require.Error(t, err, name)
		reg2.set(func(f *fakeRegistry) { f.dsErr = nil })
		_, _, err = r2.authorizeCached(as("t-1"), "ds-1")
		require.NoError(t, err, "%s: an error must not have been cached", name)
	}
}

// Keyed by the verified caller: tenant A's cached authorization can never be served to tenant B.
func TestAuthCache_ANeverServesOneTenantsEntryToAnother(t *testing.T) {
	reg := activeReg() // ds-1 belongs to t-1
	r, _ := cachedRouter(t, reg, 10*time.Second, 0)
	_, _, err := r.authorizeCached(as("t-1"), "ds-1")
	require.NoError(t, err)

	_, _, err = r.authorizeCached(as("t-2"), "ds-1")
	require.ErrorIs(t, err, ErrTenantMismatch, "t-2 must be checked on its own, not served t-1's entry")
	d, _, _ := reg.calls()
	require.Equal(t, 2, d, "t-2 consulted alpha")

	_, _, err = r.authorizeCached(as("t-1"), "ds-1")
	require.NoError(t, err)
	_, _, err = r.authorizeCached(context.Background(), "ds-1")
	require.ErrorIs(t, err, ErrNoTenant, "no tenant in context is refused before the cache is consulted")
}

func TestAuthCache_InvalidationTakesEffectAtOnce(t *testing.T) {
	reg := activeReg()
	r, _ := cachedRouter(t, reg, time.Hour, 0)
	_, _, _ = r.authorizeCached(as("t-1"), "ds-1")
	_, _ = r.appDatasource(as("t-1"), "t-1", "core")
	reg.set(func(f *fakeRegistry) { f.binding = Binding{Version: 1, Lifecycle: "suspended"} })

	_, _, err := r.authorizeCached(as("t-1"), "ds-1")
	require.NoError(t, err, "still cached")
	r.InvalidateDatasource("ds-1")
	_, _, err = r.authorizeCached(as("t-1"), "ds-1")
	require.ErrorIs(t, err, ErrUnbound, "an in-process suspension can take effect immediately")

	reg.set(func(f *fakeRegistry) { f.binding = Binding{Version: 1, Lifecycle: "active"} })
	_, _, _ = r.authorizeCached(as("t-1"), "ds-1")
	reg.set(func(f *fakeRegistry) { f.binding = Binding{Version: 1, Lifecycle: "suspended"} })
	r.InvalidateTenant("t-1")
	_, _, err = r.authorizeCached(as("t-1"), "ds-1")
	require.ErrorIs(t, err, ErrUnbound)
	require.Zero(t, len(r.apps), "invalidating the tenant also forgets its app lookups")
}

// Probe is the last gate before activation and must always see alpha's current answer.
func TestAuthCache_ProbeNeverUsesTheCache(t *testing.T) {
	reg := activeReg()
	r, _ := cachedRouter(t, reg, time.Hour, 0)
	_, _, err := r.authorizeCached(as("t-1"), "ds-1")
	require.NoError(t, err)
	d0, b0, _ := reg.calls()

	reg.set(func(f *fakeRegistry) { f.binding = Binding{Version: 1, Lifecycle: "suspended"} })
	err = r.Probe(as("t-1"), "ds-1")
	require.ErrorIs(t, err, ErrUnbound, "a cached success must not let Probe pass a suspended binding")
	d1, b1, _ := reg.calls()
	require.Greater(t, d1, d0)
	require.Greater(t, b1, b0)
}

// Many goroutines finding an entry expired at the same moment cost alpha one round.
func TestAuthCache_ConcurrentRefreshesAreCoalesced(t *testing.T) {
	reg := activeReg()
	reg.gate = make(chan struct{})
	r, _ := cachedRouter(t, reg, time.Hour, 0)

	const n = 40
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, err := r.authorizeCached(as("t-1"), "ds-1")
			errs <- err
		}()
	}
	time.Sleep(150 * time.Millisecond) // all n are now waiting on the one in-flight lookup
	close(reg.gate)
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	d, b, _ := reg.calls()
	require.Equal(t, [2]int{1, 1}, [2]int{d, b}, "forty concurrent callers must cost alpha one lookup")
}

func TestAuthCache_ACancelledCallerDoesNotFailTheOthers(t *testing.T) {
	reg := activeReg()
	reg.gate = make(chan struct{})
	r, _ := cachedRouter(t, reg, time.Hour, 0)

	ctx1, cancel := context.WithCancel(as("t-1"))
	first := make(chan error, 1)
	go func() { _, _, err := r.authorizeCached(ctx1, "ds-1"); first <- err }()
	time.Sleep(50 * time.Millisecond)
	second := make(chan error, 1)
	go func() { _, _, err := r.authorizeCached(as("t-1"), "ds-1"); second <- err }()
	time.Sleep(50 * time.Millisecond)

	cancel()
	require.ErrorIs(t, <-first, context.Canceled, "the cancelled caller gets its own cancellation")
	close(reg.gate)
	require.NoError(t, <-second, "the shared lookup must not be cancelled with the first caller")
}

func TestAuthCache_IsBounded(t *testing.T) {
	reg := activeReg()
	r, clk := cachedRouter(t, reg, time.Hour, 3)
	for i := 0; i < 10; i++ {
		clk.add(time.Millisecond)
		_, _, err := r.authorizeCached(as("t-1"), "ds-"+string(rune('a'+i)))
		require.NoError(t, err)
	}
	r.mu.Lock()
	size := len(r.auth) + len(r.apps)
	r.mu.Unlock()
	require.LessOrEqual(t, size, 3, "the cache must not grow without bound")
	require.Greater(t, size, 0)
}

func TestAuthCache_TheAppLookupIsCachedToo(t *testing.T) {
	reg := activeReg()
	r, clk := cachedRouter(t, reg, 3*time.Second, 0)
	for i := 0; i < 4; i++ {
		id, err := r.appDatasource(as("t-1"), "t-1", "core")
		require.NoError(t, err)
		require.Equal(t, "ds-1", id)
	}
	_, _, app := reg.calls()
	require.Equal(t, 1, app)
	clk.add(4 * time.Second)
	_, err := r.appDatasource(as("t-1"), "t-1", "core")
	require.NoError(t, err)
	_, _, app = reg.calls()
	require.Equal(t, 2, app)

	reg.set(func(f *fakeRegistry) { f.appErr = ErrAmbiguousApp })
	clk.add(4 * time.Second)
	_, err = r.appDatasource(as("t-1"), "t-1", "core")
	require.ErrorIs(t, err, ErrAmbiguousApp)
	reg.set(func(f *fakeRegistry) { f.appErr = nil })
	_, err = r.appDatasource(as("t-1"), "t-1", "core")
	require.NoError(t, err, "a failed lookup is never cached")
}

func TestAuthCache_CloseForgetsEverything(t *testing.T) {
	reg := activeReg()
	r, _ := cachedRouter(t, reg, time.Hour, 0)
	_, _, _ = r.authorizeCached(as("t-1"), "ds-1")
	_, _ = r.appDatasource(as("t-1"), "t-1", "core")
	r.Close()
	require.Zero(t, len(r.auth)+len(r.apps))
}

// End to end on a real pool: a suspension is enforced within the TTL, on the real clock.
func TestAuthCache_RealPoolSuspensionIsEnforcedWithinTheTTL(t *testing.T) {
	e := realPG(t)
	reg := &fakeRegistry{ds: e.ds("ds-a", "t-1", e.dbA), binding: Binding{Version: 1, Lifecycle: "active"}, user: e.user, pw: e.pw, appDS: "ds-a"}
	r, err := New(Config{Registry: reg, CallerTenant: tenantFromCtx, MaxPools: 2, MaxConnsPerPool: 2,
		IdleTTL: time.Minute, DialTimeout: 2 * time.Second, AuthTTL: 300 * time.Millisecond})
	require.NoError(t, err)
	t.Cleanup(r.Close)

	for i := 0; i < 5; i++ {
		p, err := r.ResolveApp(as("t-1"), "core")
		require.NoError(t, err)
		require.Equal(t, e.dbA, p.Database())
	}
	d, b, app := reg.calls()
	require.Equal(t, [3]int{1, 1, 1}, [3]int{d, b, app}, "five resolves on a warm pool cost alpha one authorization")

	reg.set(func(f *fakeRegistry) { f.binding = Binding{Version: 1, Lifecycle: "suspended"} })
	time.Sleep(400 * time.Millisecond)
	_, err = r.ResolveApp(as("t-1"), "core")
	require.ErrorIs(t, err, ErrUnbound, "after the TTL a suspended tenant is refused")
}
