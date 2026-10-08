package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeDB is a *sql.DB that never talks to anything: sql.Open never dials, so a handle
// is inert until a statement runs. That is all the pool needs, since eviction is about
// handle lifetime rather than connectivity.
//
// sql.Register panics if a name is registered twice, and these tests open many handles
// per test, so registrations are memoised by name.
func fakeDB(t *testing.T) *sql.DB {
	t.Helper()
	name := fmt.Sprintf("poolfake-%s", t.Name())
	if _, loaded := fakeDrivers.LoadOrStore(name, true); !loaded {
		sql.Register(name, fakeDriver{})
	}
	db, err := sql.Open(name, "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

var fakeDrivers sync.Map

type fakeDriver struct{}

func (fakeDriver) Open(string) (driver.Conn, error) { return fakeConn{}, nil }

type fakeConn struct{}

func (fakeConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("unused") }
func (fakeConn) Close() error                        { return nil }
func (fakeConn) Begin() (driver.Tx, error)           { return nil, errors.New("unused") }

func TestPoolBoundsOpenHandlesAsTenantsGrow(t *testing.T) {
	// This is the property the archguard inventory's reason asserts: the connection
	// count must not grow with the tenant count, or a few hundred tenants take the FE's
	// connection memory with them.
	p := newQueryDBPool(4)
	defer p.close()

	for i := 0; i < 50; i++ {
		_, release, err := p.acquire(fmt.Sprintf("dsn-%d", i), func(string) (*sql.DB, error) {
			return fakeDB(t), nil
		})
		require.NoError(t, err)
		release()
	}
	assert.Equal(t, 4, p.len(), "50 destinations must not produce 50 FE sessions")
}

func TestPoolReusesAHandleForTheSameDSN(t *testing.T) {
	p := newQueryDBPool(4)
	defer p.close()

	opens := 0
	open := func(string) (*sql.DB, error) { opens++; return fakeDB(t), nil }

	a, relA, err := p.acquire("same", open)
	require.NoError(t, err)
	b, relB, err := p.acquire("same", open)
	require.NoError(t, err)

	assert.Same(t, a, b, "the same destination must reuse its handle")
	assert.Equal(t, 1, opens, "a repeated DSN must not open a second session")
	assert.Equal(t, 1, p.len())

	relA()
	relB()
}

func TestPoolEvictsLeastRecentlyUsed(t *testing.T) {
	p := newQueryDBPool(2)
	defer p.close()

	open := func(string) (*sql.DB, error) { return fakeDB(t), nil }
	for _, dsn := range []string{"a", "b"} {
		_, release, err := p.acquire(dsn, open)
		require.NoError(t, err)
		release()
	}
	// Touch "a" so "b" becomes the least recently used.
	_, release, err := p.acquire("a", open)
	require.NoError(t, err)
	release()

	_, release, err = p.acquire("c", open)
	require.NoError(t, err)
	release()

	assert.Equal(t, 2, p.len())
	_, stillThere := p.handles["a"]
	assert.True(t, stillThere, "the recently used handle must survive eviction")
	_, evicted := p.handles["b"]
	assert.False(t, evicted, "the least recently used handle is the one that goes")
}

func TestPoolNeverClosesAHandleInUse(t *testing.T) {
	// Evicting a handle another destination is mid-statement on would surface as an
	// intermittent "connection is already closed" on a healthy tenant. The pool
	// overflows instead.
	p := newQueryDBPool(1)
	defer p.close()

	open := func(string) (*sql.DB, error) { return fakeDB(t), nil }

	held, release, err := p.acquire("in-use", open)
	require.NoError(t, err)
	defer release()

	_, releaseB, err := p.acquire("other", open)
	require.NoError(t, err)
	defer releaseB()

	assert.Equal(t, 2, p.len(), "the cap yields to correctness when everything is in flight")
	require.NotNil(t, held)
	assert.NoError(t, held.PingContext(context.Background()))
}

func TestPoolReleaseIsIdempotent(t *testing.T) {
	// A double release would decrement another destination's reference count and let
	// the pool close a live handle.
	p := newQueryDBPool(2)
	defer p.close()

	open := func(string) (*sql.DB, error) { return fakeDB(t), nil }
	_, release, err := p.acquire("dsn", open)
	require.NoError(t, err)

	release()
	release()

	p.mu.Lock()
	defer p.mu.Unlock()
	assert.Equal(t, 0, p.handles["dsn"].refs)
}

func TestPoolPropagatesOpenFailure(t *testing.T) {
	p := newQueryDBPool(2)
	defer p.close()

	_, release, err := p.acquire("bad", func(string) (*sql.DB, error) {
		return nil, errors.New("cannot reach StarRocks")
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot reach StarRocks")
	assert.Equal(t, 0, p.len(), "a failed open must not leave a handle behind")
	release() // must be safe even on the error path
}

func TestPoolIsConcurrencySafe(t *testing.T) {
	// The flush loop is single-goroutine today, but the pool is shared state and a
	// race here would be a latent bug rather than a current one.
	p := newQueryDBPool(4)
	defer p.close()
	open := func(string) (*sql.DB, error) { return fakeDB(t), nil }

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				_, release, err := p.acquire(fmt.Sprintf("dsn-%d", i%9), open)
				if err == nil {
					release()
				}
			}
		}(i)
	}
	wg.Wait()
	// The count is deliberately not asserted: with nine destinations in flight at once
	// the cap is expected to yield. What this test really checks is that -race finds
	// nothing in the shared map and the refcount bookkeeping.
	assert.GreaterOrEqual(t, p.len(), 1)
}
