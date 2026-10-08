package main

import (
	"database/sql"
	"errors"
	"fmt"
	"sync"
)

// maxTenantDBHandles bounds how many StarRocks query connections one loader holds at
// once.
//
// Routing introduced one handle per destination DSN, which means one per tenant. Left
// unbounded that is a latent outage rather than a slow degradation: every handle is a
// MySQL-protocol session on the FE, and the projected tenant count is in the hundreds,
// so a burst of active tenants would take the FE's connection memory with it.
//
// FE budget, checked rather than assumed: starrocks-fe has no qe_max_connection in
// fe.conf, so the StarRocks default applies (1024 per FE). Eight handles across the
// five loader replicas is ~40 sessions, a low single-digit percentage of that, on top
// of the keyed-DELETE traffic itself. The cap protects the loader from itself and is
// also comfortably inside the FE's headroom -- but if the FE is ever tuned DOWN, this
// constant is the first thing to revisit.
//
// Exhaustion is a LOAD condition and is deliberately surfaced as retryable, never as a
// dead-letter. errPoolExhausted is a plain error, so OutcomeOf classifies it as
// outcomeTransport: the flush loop retries with backoff like any other transient
// failure. Dead-lettering a row because the pool was momentarily busy would route a
// load problem into the permanent-loss path, which is the opposite of what it is.
const maxTenantDBHandles = 8

// errPoolExhausted is returned when every handle is in use and the cap is reached.
// Transient by construction: the caller retries, and the rows are unaffected.
var errPoolExhausted = errors.New("starrocks handle pool exhausted (all handles in use)")

// queryDBPool is a bounded, least-recently-used pool of StarRocks query handles keyed
// by DSN.
//
// Handles are reference counted so eviction cannot close a connection another
// destination is mid-statement on. If every handle is busy the pool overflows its cap
// temporarily rather than closing something in use; the next idle acquisition brings it
// back under the limit. An occasional overflow is strictly better than either a
// spurious "too many open files" or a closed-underneath-you connection.
type queryDBPool struct {
	mu      sync.Mutex
	max     int
	handles map[string]*pooledDB
	// order is least-recently-used first, most-recently-used last.
	order []string
}

type pooledDB struct {
	db   *sql.DB
	refs int
}

func newQueryDBPool(max int) *queryDBPool {
	if max < 1 {
		max = 1
	}
	return &queryDBPool{max: max, handles: map[string]*pooledDB{}}
}

// acquire returns a handle for dsn, opening it if needed, plus a release function.
//
// The release must be called exactly once. It is returned rather than deferred inside
// acquire so the caller can hold the handle across the statement without holding a
// pool lock for the duration of network I/O.
func (p *queryDBPool) acquire(dsn string, open func(string) (*sql.DB, error)) (*sql.DB, func(), error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if h, ok := p.handles[dsn]; ok {
		h.refs++
		p.touch(dsn)
		return h.db, p.releaser(dsn), nil
	}

	// At the cap with everything in flight, refuse rather than open a ninth session.
	// The caller retries (see errPoolExhausted); opening anyway would just move the
	// exhaustion onto the FE, which is the party with no headroom to spare.
	if len(p.handles) >= p.max && !p.hasIdleLocked() {
		return nil, func() {}, fmt.Errorf("%w (cap %d)", errPoolExhausted, p.max)
	}

	db, err := open(dsn)
	if err != nil {
		return nil, func() {}, err
	}
	h := &pooledDB{db: db, refs: 1}
	p.handles[dsn] = h
	p.order = append(p.order, dsn)
	p.touch(dsn)
	p.evictIdleLocked()
	return db, p.releaser(dsn), nil
}

// releaser builds the release closure. It is a method call per acquire rather than a
// shared closure so the decrement always lands on the same pool.
func (p *queryDBPool) releaser(dsn string) func() {
	var once sync.Once
	return func() {
		once.Do(func() {
			p.mu.Lock()
			defer p.mu.Unlock()
			if h, ok := p.handles[dsn]; ok && h.refs > 0 {
				h.refs--
			}
		})
	}
}

// touch moves dsn to the most-recently-used end.
func (p *queryDBPool) touch(dsn string) {
	for i, k := range p.order {
		if k == dsn {
			p.order = append(p.order[:i], p.order[i+1:]...)
			break
		}
	}
	p.order = append(p.order, dsn)
}

// hasIdleLocked reports whether any handle is free to be evicted.
func (p *queryDBPool) hasIdleLocked() bool {
	for _, h := range p.handles {
		if h.refs == 0 {
			return true
		}
	}
	return false
}

// evictIdleLocked closes least-recently-used idle handles until the pool is within its
// cap.
func (p *queryDBPool) evictIdleLocked() {
	for len(p.order) > p.max {
		victim := ""
		victimIdx := -1
		for i, dsn := range p.order {
			if h, ok := p.handles[dsn]; ok && h.refs == 0 {
				victim, victimIdx = dsn, i
				break
			}
		}
		if victimIdx < 0 {
			// Everything is in use and acquire() has already refused to exceed the
			// cap, so this is unreachable in practice. Leave it alone rather than close
			// a connection somebody is using.
			return
		}
		p.order = append(p.order[:victimIdx], p.order[victimIdx+1:]...)
		if h, ok := p.handles[victim]; ok {
			_ = h.db.Close()
			delete(p.handles, victim)
		}
	}
}

// len reports the number of open handles. Test-facing, and the value that backs the
// "connections do not grow with the tenant count" claim in the archguard inventory.
func (p *queryDBPool) len() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.handles)
}

func (p *queryDBPool) close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for dsn, h := range p.handles {
		_ = h.db.Close()
		delete(p.handles, dsn)
	}
	p.order = nil
}
