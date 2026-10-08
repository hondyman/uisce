package main

import (
	"database/sql"
	"sync"
)

// maxTenantDBHandles bounds how many StarRocks query connections one loader holds at
// once.
//
// Routing introduced one handle per destination DSN, which means one per tenant. Left
// unbounded that is a latent outage rather than a slow degradation: every handle is a
// MySQL-protocol session on the FE, and the projected tenant count is in the hundreds,
// so a burst of active tenants would take the FE's connection memory with it. Eight
// covers the realistic window (the flush loop walks destinations serially and each
// delete is a single statement) while putting a hard ceiling on the FE's exposure.
const maxTenantDBHandles = 8

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

// evictIdleLocked closes least-recently-used idle handles until the pool is within its
// cap. A handle nobody has used yet is closed immediately, since eviction right after
// opening is the only point where the cap is actually meaningful.
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
			// Everything is in use. Let it exceed the cap this once.
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
