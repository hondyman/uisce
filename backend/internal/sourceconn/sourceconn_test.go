package sourceconn

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type ctxTenant struct{}

func as(tenant string) context.Context {
	return context.WithValue(context.Background(), ctxTenant{}, tenant)
}
func tenantFromCtx(ctx context.Context) (string, error) {
	if v, ok := ctx.Value(ctxTenant{}).(string); ok && v != "" {
		return v, nil
	}
	return "", errors.New("no tenant")
}

type fakeReg struct {
	mu    sync.Mutex
	calls int
	src   Source
	err   error
	gate  chan struct{}
}

func (f *fakeReg) Source(ctx context.Context, tenantID, id string, p Policy) (Source, error) {
	if g := f.gate; g != nil {
		select {
		case <-g:
		case <-ctx.Done():
			return Source{}, ctx.Err()
		}
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	return f.src, f.err
}
func (f *fakeReg) count() int            { f.mu.Lock(); defer f.mu.Unlock(); return f.calls }
func (f *fakeReg) set(fn func(*fakeReg)) { f.mu.Lock(); defer f.mu.Unlock(); fn(f) }

type fakeCreds struct {
	mu    sync.Mutex
	calls int
	out   []byte
	err   error
}

func (f *fakeCreds) Hydrate(context.Context, Source) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	return f.out, f.err
}
func (f *fakeCreds) count() int { f.mu.Lock(); defer f.mu.Unlock(); return f.calls }

type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) now() time.Time      { c.mu.Lock(); defer c.mu.Unlock(); return c.t }
func (c *clock) add(d time.Duration) { c.mu.Lock(); defer c.mu.Unlock(); c.t = c.t.Add(d) }

func newConn(t *testing.T, reg Registry, cr Credentials, ttl time.Duration) (*Connector, *clock) {
	t.Helper()
	c, err := New(Config{Registry: reg, Credentials: cr, CallerTenant: tenantFromCtx, MaxPools: 4, MaxConnsPerPool: 2,
		IdleTTL: time.Minute, DialTimeout: 2 * time.Second, AuthTTL: ttl})
	require.NoError(t, err)
	clk := &clock{t: time.Unix(1_000_000, 0)}
	c.now = clk.now
	t.Cleanup(c.Close)
	return c, clk
}

func okReg() (*fakeReg, *fakeCreds) {
	return &fakeReg{src: Source{ID: "ds-1", TenantID: "t-1", Config: []byte(`{}`)}},
		&fakeCreds{out: []byte(`{"host":"h","port":5432,"database":"d","username":"u"}`)}
}

func TestNew_RequiresEverything(t *testing.T) {
	reg, cr := okReg()
	ok := Config{Registry: reg, Credentials: cr, CallerTenant: tenantFromCtx, MaxPools: 1, MaxConnsPerPool: 1, IdleTTL: time.Second, DialTimeout: time.Second}
	_, err := New(ok)
	require.NoError(t, err)
	for name, mut := range map[string]func(*Config){
		"registry":    func(c *Config) { c.Registry = nil },
		"credentials": func(c *Config) { c.Credentials = nil },
		"caller":      func(c *Config) { c.CallerTenant = nil },
		"pools":       func(c *Config) { c.MaxPools = 0 },
		"conns":       func(c *Config) { c.MaxConnsPerPool = 0 },
		"idle":        func(c *Config) { c.IdleTTL = 0 },
		"dial":        func(c *Config) { c.DialTimeout = 0 },
		"negative":    func(c *Config) { c.AuthTTL = -1 },
	} {
		c := ok
		mut(&c)
		_, err := New(c)
		require.Error(t, err, name)
	}
}

func TestAuthorize_FailsClosed(t *testing.T) {
	t.Run("no tenant never reaches the registry", func(t *testing.T) {
		reg, cr := okReg()
		c, _ := newConn(t, reg, cr, 0)
		_, err := c.Pool(context.Background(), "ds-1", OwnerOnly)
		require.ErrorIs(t, err, ErrNoTenant)
		require.Zero(t, reg.count())
	})
	t.Run("the registry's refusals reach the caller and leave no pool", func(t *testing.T) {
		for name, e := range map[string]error{"not found": ErrNotFound, "not allowed": ErrNotAllowed, "bad config": ErrBadConfig, "down": errors.New("alpha down")} {
			reg, cr := okReg()
			reg.err = e
			c, _ := newConn(t, reg, cr, 0)
			p, err := c.Pool(as("t-1"), "ds-1", OwnerOnly)
			require.ErrorIs(t, err, e, name)
			require.Nil(t, p)
			require.Zero(t, c.Size())
			require.Zero(t, cr.count(), "%s: credentials must not be fetched for a datasource the caller may not use", name)
		}
	})
	t.Run("a credential failure is an error and no pool", func(t *testing.T) {
		reg, cr := okReg()
		cr.err = errors.New("vault down")
		c, _ := newConn(t, reg, cr, 0)
		p, err := c.Pool(as("t-1"), "ds-1", OwnerOnly)
		require.ErrorContains(t, err, "vault down")
		require.Nil(t, p)
	})
	t.Run("unusable details are refused before any dial", func(t *testing.T) {
		reg, cr := okReg()
		cr.out = []byte(`{"host":"h"}`)
		c, _ := newConn(t, reg, cr, 0)
		_, err := c.Pool(as("t-1"), "ds-1", OwnerOnly)
		require.ErrorIs(t, err, ErrBadConfig)
	})
}

func TestAuthCache_ReuseExpiryAndKeys(t *testing.T) {
	reg, cr := okReg()
	c, clk := newConn(t, reg, cr, 3*time.Second)
	key := authKey{"t-1", "ds-1", OwnerOnly}

	for i := 0; i < 4; i++ {
		_, err := c.authorize(as("t-1"), key)
		require.NoError(t, err)
	}
	require.Equal(t, 1, reg.count(), "within the TTL alpha is asked once")
	require.Equal(t, 1, cr.count())

	clk.add(3*time.Second + time.Millisecond)
	_, err := c.authorize(as("t-1"), key)
	require.NoError(t, err)
	require.Equal(t, 2, reg.count(), "after the TTL it is asked again")

	// A cached OwnerOnly authorization must never satisfy a broader policy, nor another tenant.
	_, err = c.authorize(as("t-1"), authKey{"t-1", "ds-1", OwnerOrGoldCopy})
	require.NoError(t, err)
	require.Equal(t, 3, reg.count(), "a different policy is a different decision")
	_, err = c.authorize(as("t-2"), authKey{"t-2", "ds-1", OwnerOnly})
	require.NoError(t, err)
	require.Equal(t, 4, reg.count(), "a different tenant is a different decision")
}

func TestAuthCache_RefusalsAreNeverCachedAndInvalidationIsImmediate(t *testing.T) {
	reg, cr := okReg()
	reg.err = ErrNotAllowed
	c, _ := newConn(t, reg, cr, time.Hour)
	key := authKey{"t-1", "ds-1", OwnerOnly}
	for i := 0; i < 3; i++ {
		_, err := c.authorize(as("t-1"), key)
		require.ErrorIs(t, err, ErrNotAllowed)
	}
	require.Equal(t, 3, reg.count(), "each refusal asked alpha again")
	reg.set(func(f *fakeReg) { f.err = nil })
	_, err := c.authorize(as("t-1"), key)
	require.NoError(t, err, "once fixed it works at once")

	reg.set(func(f *fakeReg) { f.err = ErrNotAllowed })
	_, err = c.authorize(as("t-1"), key)
	require.NoError(t, err, "still cached: the documented, bounded trade")
	c.InvalidateDatasource("ds-1")
	_, err = c.authorize(as("t-1"), key)
	require.ErrorIs(t, err, ErrNotAllowed, "an in-process change takes effect at once")
}

func TestAuthCache_ConcurrentLookupsAreCoalescedAndACancelledCallerDoesNotFailOthers(t *testing.T) {
	reg, cr := okReg()
	reg.gate = make(chan struct{})
	c, _ := newConn(t, reg, cr, time.Hour)
	key := authKey{"t-1", "ds-1", OwnerOnly}

	const n = 30
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := c.authorize(as("t-1"), key); errs <- err }()
	}
	time.Sleep(120 * time.Millisecond)
	close(reg.gate)
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	require.Equal(t, 1, reg.count(), "thirty concurrent callers cost alpha one lookup")

	reg2, cr2 := okReg()
	reg2.gate = make(chan struct{})
	c2, _ := newConn(t, reg2, cr2, time.Hour)
	ctx1, cancel := context.WithCancel(as("t-1"))
	first := make(chan error, 1)
	go func() { _, err := c2.authorize(ctx1, key); first <- err }()
	time.Sleep(40 * time.Millisecond)
	second := make(chan error, 1)
	go func() { _, err := c2.authorize(as("t-1"), key); second <- err }()
	time.Sleep(40 * time.Millisecond)
	cancel()
	require.ErrorIs(t, <-first, context.Canceled)
	close(reg2.gate)
	require.NoError(t, <-second, "the shared lookup is not cancelled with the first caller")
}

func TestAuthorize_CredentialHashFollowsTheHydratedDetails(t *testing.T) {
	reg, cr := okReg()
	c, _ := newConn(t, reg, cr, 0)
	key := authKey{"t-1", "ds-1", OwnerOnly}
	a1, err := c.authorize(as("t-1"), key)
	require.NoError(t, err)
	a1b, _ := c.authorize(as("t-1"), key)
	require.Equal(t, a1.key, a1b.key)

	cr.mu.Lock()
	cr.out = []byte(`{"host":"h","port":5432,"database":"d","username":"u","password":"rotated"}`)
	cr.mu.Unlock()
	a2, err := c.authorize(as("t-1"), key)
	require.NoError(t, err)
	require.NotEqual(t, a1.key.credHash, a2.key.credHash, "a rotated credential must get a different pool key")
	require.NotContains(t, a2.key.credHash, "rotated", "the credential itself is never part of a key")
}

// Through the public entry point: a cached authorization under one policy must never satisfy a
// different one. The dial fails (nothing listens), but the registry count still shows whether the
// second call was served from the first call's decision.
func TestPool_ACachedDecisionNeverCrossesPolicies(t *testing.T) {
	reg, cr := okReg()
	cr.out = []byte(`{"host":"127.0.0.1","port":1,"database":"d","username":"u"}`)
	c, _ := newConn(t, reg, cr, time.Hour)

	_, err := c.Pool(as("t-1"), "ds-1", OwnerOnly)
	require.Error(t, err, "nothing listens on the port")
	require.Equal(t, 1, reg.count())
	_, _ = c.Pool(as("t-1"), "ds-1", OwnerOnly)
	require.Equal(t, 1, reg.count(), "the same policy reuses the decision")
	_, _ = c.Pool(as("t-1"), "ds-1", OwnerOrGoldCopy)
	require.Equal(t, 2, reg.count(), "OwnerOrGoldCopy must be decided on its own, not served OwnerOnly's answer")
	_, _ = c.Pool(as("t-2"), "ds-1", OwnerOnly)
	require.Equal(t, 3, reg.count(), "another tenant must be decided on its own")
}
