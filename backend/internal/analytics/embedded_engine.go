package analytics

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// EmbeddedEngine runs validation evaluation in-process against a continuously
// refreshed rule snapshot cache. Zero network overhead on the evaluation path.
//
// Invalidation is hybrid (belt and suspenders):
//   - Postgres LISTEN 'catalog_changed' for real-time invalidation (best effort;
//     the dedicated listener connection can drop and reconnect)
//   - jittered background polling of a cheap version query (authoritative)
//
// Consistency contract:
//   - evaluations always run against one internally-consistent frozen snapshot
//   - after a rule change, staleness is bounded by the poll interval (default
//     30s); NOTIFY typically shrinks this to milliseconds
//   - cold cache + DB unavailable = error (fail closed); a warm but stale
//     snapshot keeps serving (fail open for availability) and increments
//     StaleServes in metrics
type EmbeddedEngine struct {
	db       *sqlx.DB
	dsn      string
	tenantID string
	svc      *ValidationRuleService
	loader   ContextLoader

	pollInterval time.Duration
	notifyOn     bool // false disables the listener (e.g., non-Postgres tests)

	mu       sync.Mutex // guards snapshot map writes; reads go through atomic pointers
	snapshot map[snapshotKey]*atomic.Pointer[RuleSnapshot]
	versions map[snapshotKey]string

	Metrics EngineMetrics

	listenCtx    context.Context
	listenCancel context.CancelFunc
	wg           sync.WaitGroup

	started atomic.Bool
}

type snapshotKey struct {
	bo     string
	domain string
	timing string
}

type EngineMetrics struct {
	CacheHits        int64     `json:"cache_hits"`
	CacheMisses      int64     `json:"cache_misses"` // refreshes issued
	StaleServes      int64     `json:"stale_serves"` // evaluated while refresh was pending/failed
	RefreshErrors    int64     `json:"refresh_errors"`
	ListenReconnects int64     `json:"listen_reconnects"`
	LastRefresh      time.Time `json:"last_refresh"`
	LastVersion      string    `json:"last_version"`
}

type EmbeddedOption func(*EmbeddedEngine)

func WithPollInterval(d time.Duration) EmbeddedOption  { return func(e *EmbeddedEngine) { e.pollInterval = d } }
func WithoutNotify() EmbeddedOption                    { return func(e *EmbeddedEngine) { e.notifyOn = false } }
func WithContextLoader(l ContextLoader) EmbeddedOption { return func(e *EmbeddedEngine) { e.loader = l } }
func WithDSN(dsn string) EmbeddedOption                { return func(e *EmbeddedEngine) { e.dsn = dsn } }

// NewEmbeddedEngine constructs the engine. Call Start before use.
func NewEmbeddedEngine(db *sqlx.DB, tenantID string, svc *ValidationRuleService, opts ...EmbeddedOption) *EmbeddedEngine {
	e := &EmbeddedEngine{
		db:           db,
		tenantID:     tenantID,
		svc:          svc,
		pollInterval: 30 * time.Second,
		notifyOn:     true,
		snapshot:     map[snapshotKey]*atomic.Pointer[RuleSnapshot]{},
		versions:     map[snapshotKey]string{},
	}
	for _, o := range opts {
		o(e)
	}
	return e
}

// Start launches the poller and (optionally) the NOTIFY listener. Idempotent.
func (e *EmbeddedEngine) Start(ctx context.Context) {
	if !e.started.CompareAndSwap(false, true) {
		return
	}
	e.listenCtx, e.listenCancel = context.WithCancel(ctx)

	e.wg.Add(1)
	go e.pollLoop()
	if e.notifyOn && (e.dsn != "" || e.db != nil) {
		e.wg.Add(1)
		go e.listenLoop()
	}
}

// Close stops background workers.
func (e *EmbeddedEngine) Close() {
	if e.listenCancel != nil {
		e.listenCancel()
	}
	e.wg.Wait()
}

// SnapshotVersion returns the current catalog version for a key — monotonic
// enough for debugging ("which rules were active when this decision was made")
// and for tests. Format: "updated@<unixnano>:count<n>".
func (e *EmbeddedEngine) SnapshotVersion(ctx context.Context, boName, domain, timing string) (string, error) {
	return e.catalogVersion(ctx, snapshotKey{bo: boName, domain: domain, timing: timing})
}

// catalogVersion is the authoritative staleness probe. One indexed query.
func (e *EmbeddedEngine) catalogVersion(ctx context.Context, key snapshotKey) (string, error) {
	if e.db == nil {
		return "", fmt.Errorf("embedded engine: database connection is nil")
	}
	var (
		maxUpdated sql.NullTime
		count      int
	)
	gold, err := goldCopyTenantID(ctx, e.db)
	if err != nil {
		return "", err
	}
	tenants := visibleTenants(e.tenantID, gold)
	err = e.db.GetContext(ctx, &maxUpdated, `
		SELECT MAX(updated_at) FROM catalog_node n
		JOIN catalog_node_type nt ON n.node_type_id = nt.id
		WHERE nt.catalog_type_name = 'validation_rule'
		  AND n.tenant_id = ANY($1::uuid[])
		  AND ($2 = '' OR n.properties->>'bo_name' = $2)
	`, pq.Array(tenants), key.bo)
	if err != nil {
		return "", fmt.Errorf("embedded engine: version probe: %w", err)
	}
	err = e.db.GetContext(ctx, &count, `
		SELECT COUNT(*) FROM catalog_node n
		JOIN catalog_node_type nt ON n.node_type_id = nt.id
		WHERE nt.catalog_type_name = 'validation_rule'
		  AND n.tenant_id = ANY($1::uuid[])
		  AND ($2 = '' OR n.properties->>'bo_name' = $2)
	`, pq.Array(tenants), key.bo)
	if err != nil {
		return "", fmt.Errorf("embedded engine: version probe count: %w", err)
	}
	v := "none"
	if maxUpdated.Valid {
		v = fmt.Sprintf("upd@%d:count%d", maxUpdated.Time.UnixNano(), count)
	}
	return v, nil
}

// ValidateRecord evaluates one record against the cached snapshot, refreshing
// it first if the catalog version has moved.
func (e *EmbeddedEngine) ValidateRecord(ctx context.Context, boName, domain, timing string, record map[string]any) (*RecordEvaluation, error) {
	snap, err := e.snapshotFor(ctx, boName, domain, timing)
	if err != nil {
		return nil, err
	}
	return EvaluateRecord(ctx, snap, record, e.loader)
}

// ValidateBatch runs a batch against one cached snapshot (worker pool inside).
func (e *EmbeddedEngine) ValidateBatch(ctx context.Context, boName, domain, timing string, records []map[string]any) (*EvaluateBatchResult, error) {
	snap, err := e.snapshotFor(ctx, boName, domain, timing)
	if err != nil {
		return nil, err
	}
	return EvaluateBatchWithSnapshot(ctx, snap, records, e.loader)
}

// snapshotFor returns the cached snapshot, refreshing when the version moved.
func (e *EmbeddedEngine) snapshotFor(ctx context.Context, boName, domain, timing string) (*RuleSnapshot, error) {
	key := snapshotKey{bo: boName, domain: domain, timing: timing}

	e.mu.Lock()
	ptr, ok := e.snapshot[key]
	knownVersion := e.versions[key]
	e.mu.Unlock()

	if ok && ptr != nil {
		snap := ptr.Load()
		if snap != nil {
			// Version probe: if it fails (DB down), serve the warm snapshot
			// and count it — availability beats freshness for warm caches.
			v, verr := e.catalogVersion(ctx, key)
			if verr != nil {
				atomic.AddInt64(&e.Metrics.StaleServes, 1)
				return snap, nil
			}
			if v == knownVersion {
				atomic.AddInt64(&e.Metrics.CacheHits, 1)
				return snap, nil
			}
			// Version moved → refresh below.
		}
	}

	atomic.AddInt64(&e.Metrics.CacheMisses, 1)
	if e.svc == nil {
		if ok && ptr != nil {
			snap := ptr.Load()
			if snap != nil {
				atomic.AddInt64(&e.Metrics.StaleServes, 1)
				return snap, nil
			}
		}
		return nil, fmt.Errorf("embedded engine: cold cache refresh failed (fail closed): validation rule service is nil")
	}

	snap, err := e.svc.LoadRuleSnapshot(ctx, e.tenantID, boName, domain, timing)
	if err != nil {
		atomic.AddInt64(&e.Metrics.RefreshErrors, 1)
		if ok && ptr != nil {
			// Warm stale snapshot on refresh failure (fail open for warm cache).
			snap := ptr.Load()
			if snap != nil {
				atomic.AddInt64(&e.Metrics.StaleServes, 1)
				return snap, nil
			}
		}
		return nil, fmt.Errorf("embedded engine: cold cache refresh failed (fail closed): %w", err)
	}
	v, _ := e.catalogVersion(ctx, key)

	e.mu.Lock()
	if !ok || ptr == nil {
		ptr = &atomic.Pointer[RuleSnapshot]{}
		e.snapshot[key] = ptr
	}
	e.versions[key] = v
	e.mu.Unlock()
	ptr.Store(snap)

	e.Metrics.LastRefresh = time.Now().UTC()
	e.Metrics.LastVersion = v
	return snap, nil
}

// Invalidate drops cached snapshots — called by the NOTIFY listener and
// available for tests/manual control. The next Validate* call re-probes.
func (e *EmbeddedEngine) Invalidate() {
	e.mu.Lock()
	e.snapshot = map[snapshotKey]*atomic.Pointer[RuleSnapshot]{}
	e.versions = map[snapshotKey]string{}
	e.mu.Unlock()
}

// --- Background workers ---

func (e *EmbeddedEngine) pollLoop() {
	defer e.wg.Done()
	interval := e.pollInterval
	if interval <= 0 {
		interval = 30 * time.Second
	}
	for {
		jitter := time.Duration(float64(interval) * (0.8 + 0.4*rand.Float64()))
		select {
		case <-e.listenCtx.Done():
			return
		case <-time.After(jitter):
			e.Invalidate() // cheap: next evaluation re-probes versions
		}
	}
}

// listenLoop holds a dedicated listener connection for LISTEN catalog_changed.
func (e *EmbeddedEngine) listenLoop() {
	defer e.wg.Done()

	connStr := e.dsn
	if connStr == "" && e.db != nil {
		// If DSN wasn't passed explicitly, listenLoop gracefully exits if unavailable
		return
	}

	listener := pq.NewListener(connStr, 5*time.Second, time.Minute, func(ev pq.ListenerEventType, err error) {
		if err != nil {
			atomic.AddInt64(&e.Metrics.ListenReconnects, 1)
		}
	})
	defer listener.Close()

	if err := listener.Listen("catalog_changed"); err != nil {
		return
	}

	for {
		select {
		case <-e.listenCtx.Done():
			return
		case notif, ok := <-listener.Notify:
			if !ok {
				return
			}
			if notif == nil {
				continue
			}
			var payload struct {
				Op     string `json:"op"`
				Type   string `json:"type"`
				Tenant string `json:"tenant"`
			}
			if json.Unmarshal([]byte(notif.Extra), &payload) == nil {
				if payload.Type == "validation_rule" || payload.Type == "catalog_node_type" {
					e.Invalidate()
				}
			}
		}
	}
}
