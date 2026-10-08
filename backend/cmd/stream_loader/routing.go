package main

// Per-tenant routing for the CDC loader.
//
// Decision A: tenant isolation in StarRocks is database-per-tenant with DB-scoped
// grants. Verified live on 3.3.22:
//
//	own-db   stream load  -> HTTP 200 Status=Success
//	cross-db stream load  -> HTTP 401 Access denied
//	cross-db SELECT       -> ERROR 5203 Access denied
//
// Two traps this file exists to encode, both absent from the pre-routing loader:
//
//  1. Not every event carries a row. Debezium heartbeats, schema-change events and
//     tombstones have no `payload.after`. If the first thing routing does is read
//     `tenant_id`, the first heartbeat after a quiet period lands in the DLQ and
//     pages someone for nothing. Events are classified BEFORE routing.
//
//  2. A flush window can span many tenants. The accumulator is keyed by
//     (tenant_id, table), so one window emits one load per active tenant rather
//     than one per table. Labels are unique per database, and the Kafka
//     partition/offset range stays in the label so a re-processed batch after a
//     rebalance stays idempotent at each destination.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	kafka "github.com/segmentio/kafka-go"
)

// TenantRoute is one provisioned tenant: which StarRocks database its rows belong
// in, and the principal allowed to write there and nowhere else.
type TenantRoute struct {
	TenantID   string   `json:"tenant_id"`
	TenantName string   `json:"tenant_name"`
	Database   string   `json:"database"`
	User       string   `json:"user"`
	Password   string   `json:"password"`
	DSN        string   `json:"dsn"`
	KeyColumns []string `json:"key_columns"`
}

// TenantRouter resolves a tenant to its destination. It is built once at startup
// and never mutated, so lookups from the flush path need no lock; the credential
// files on disk are only re-read by a restart.
type TenantRouter struct {
	byTenant map[string]TenantRoute
	// sourceDB records which Postgres database the captured events came from.
	//
	// The target topology is control plane / data plane: `alpha` holds the metadata
	// for every tenant (public.tenants, tenant_product_datasource, tenant_lakehouse),
	// and each tenant's business data lives in its own Postgres database named by
	// tenant_product_datasource.config. So this seam is load-bearing, not hypothetical.
	//
	// What is live today is narrower than the target, and the gap matters:
	//
	//   - the connector captures only `alpha`, and `alpha.orm.*` is single-tenant
	//     today, discriminated by a tenant_id column. after.tenant_id is therefore
	//     the correct discriminator for the connector as it is configured now, which
	//     is what this loader implements.
	//   - of 9 tenant_product_datasource rows, 3 carry a real host/database; only
	//     one of those is bound to a tenant instance, and none is captured by CDC.
	//   - tenant_datasource_binding, the control plane's own registry of per-tenant
	//     starrocks_database / starrocks_role, holds 0 rows.
	//
	// When the connector is pointed at tenant data planes, three things change and
	// this comment is the checklist:
	//
	//  1. The discriminator moves from after.tenant_id to source.db. A tenant's own
	//     database need not carry a tenant_id column at all, and this loader would
	//     then dead-letter every row as unattributed.
	//  2. Debezium's Postgres connector takes one database per connector, so N tenant
	//     databases means N connectors and N logical replication slots. Slot lifecycle
	//     (create with the tenant, drop on deprovision, lag alerting for every one)
	//     becomes part of provisioning rather than a one-off.
	//  3. The credential files under the routes directory stop being the source of
	//     truth for database and role names; tenant_datasource_binding is, with its
	//     UNIQUE constraints already refusing to bind one physical database to two
	//     tenants. The files remain the carrier of the password, which the control
	//     plane deliberately does not store.
	//
	// The seam is here so that change does not require touching the flush path.
	//
	// TRIGGER CONDITIONS -- rebuild the connector when either of these becomes true:
	//
	//   1. tenant_datasource_binding gains a row for a datasource with a live host that
	//      needs analytics. That is the first declaration of a per-tenant StarRocks
	//      destination, and nothing in the CDC path will be routed to it.
	//   2. A product decision commits data planes as the topology.
	//
	// Until one fires, after.tenant_id over alpha is the correct implementation of the
	// topology as configured, and the N-connector rebuild is speculative
	// infrastructure. Deferring it is safe by construction, not by hope: the failure
	// mode of the wrong discriminator is loud (every row dead-lettered as
	// unattributed, tenant_unattributed climbing), which is the property fail-closed
	// routing was chosen for.
	//
	// Do not rely on someone reading this. topology_tripwire.go turns condition 1 into
	// an alarm at startup and every five minutes, so the premise is checked against the
	// control plane rather than remembered.
	sourceDB string
}

// ErrNoTenantRoute is returned for a tenant with no provisioned route. It covers
// both provisioning lag (tenant exists in Postgres, script has not run) and
// deprovisioned tenants whose events are still in flight -- deliberately the same
// outcome, so history is preserved while the signal stays visible.
var ErrNoTenantRoute = errors.New("no StarRocks route provisioned for tenant")

// ErrTenantUnattributed is returned when a data event carries no tenant_id at
// all. Fail-closed: there is no default database, because a default destination is
// exactly the silent-misrouting failure this work exists to prevent.
var ErrTenantUnattributed = errors.New("data event carries no tenant_id")

func NewTenantRouter(routes []TenantRoute, sourceDB string) *TenantRouter {
	m := make(map[string]TenantRoute, len(routes))
	for _, r := range routes {
		m[normalizeTenantID(r.TenantID)] = r
	}
	return &TenantRouter{byTenant: m, sourceDB: sourceDB}
}

// LoadTenantRoutes reads the credential files written by
// scripts/provision_starrocks_tenants.sh. A missing or unreadable directory is
// not an error here: the caller decides whether per-tenant routing is enabled,
// because a single-tenant deployment may legitimately have none.
func LoadTenantRoutes(dir string) ([]TenantRoute, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read tenant route directory %s: %w", dir, err)
	}
	var out []TenantRoute
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var r TenantRoute
		if err := json.Unmarshal(b, &r); err != nil || r.Database == "" || r.User == "" {
			// A malformed credential file is skipped rather than half-applied: a
			// route missing its database would send rows to the wrong place.
			continue
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TenantID < out[j].TenantID })
	return out, nil
}

// normalizeTenantID accepts the forms a tenant id arrives in: with hyphens (the
// Postgres uuid text form) and without (the StarRocks identifier form used by
// cube_materializer.go). Both must resolve to the same route, or a tenant's rows
// would split across two destinations.
func normalizeTenantID(id string) string {
	return strings.ToLower(strings.TrimSpace(strings.ReplaceAll(id, "-", "")))
}

// Route resolves the destination for a tenant id.
func (t *TenantRouter) Route(tenantID string) (TenantRoute, error) {
	key := normalizeTenantID(tenantID)
	if key == "" {
		return TenantRoute{}, ErrTenantUnattributed
	}
	r, ok := t.byTenant[key]
	if !ok {
		return TenantRoute{}, fmt.Errorf("%w: %s", ErrNoTenantRoute, tenantID)
	}
	return r, nil
}

func (t *TenantRouter) Enabled() bool { return len(t.byTenant) > 0 }

func (t *TenantRouter) Len() int { return len(t.byTenant) }

// Known reports whether a tenant has a route. Used by the provisioning drift
// check to distinguish "never provisioned" from "deprovisioned".
func (t *TenantRouter) Known(tenantID string) bool {
	_, ok := t.byTenant[normalizeTenantID(tenantID)]
	return ok
}

// ---- event classification ----

type eventClass string

const (
	// classData is a change event carrying a row: op=c/r/u (after) or op=d
	// (before). These are the only events that are routed.
	classData eventClass = "data"
	// classHeartbeat is Debezium's liveness event. It has no payload.after.
	classHeartbeat eventClass = "heartbeat"
	// classSchemaChange carries payload.schema with no row.
	classSchemaChange eventClass = "schema_change"
	// classTombstone is a null-valued record from a deleted key.
	classTombstone eventClass = "tombstone"
	// classUnknown is anything else with no row. Skipped rather than DLQ'd: an
	// unrecognised envelope is not evidence of a tenant misroute.
	classUnknown eventClass = "unknown"
)

// classify decides what an envelope is BEFORE any tenant routing happens. Getting
// this order wrong is how a heartbeat becomes a 2am page.
//
// Note that decodeRecord cannot be trusted to make this distinction: it treats a
// missing payload.after as a tombstone, and a Debezium heartbeat has no
// payload.after at all. Classification therefore re-reads the raw envelope's
// payload keys rather than inferring from decodedEvent.
func classify(ev decodedEvent, raw []byte) eventClass {
	if ev.isDelete && ev.before != nil {
		return classData
	}
	if !ev.isDelete && ev.after != nil {
		return classData
	}

	// A record with no value at all is a Kafka tombstone, as distinct from a
	// data-shaped envelope whose image is explicitly JSON null.
	if isNullish(raw) {
		return classTombstone
	}
	var probe struct {
		Schema  json.RawMessage            `json:"schema"`
		Payload map[string]json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil || probe.Payload == nil {
		return classUnknown
	}
	_, hasAfter := probe.Payload["after"]
	_, hasBefore := probe.Payload["before"]
	if hasAfter || hasBefore {
		// Data-shaped but the image did not decode to a row: a compacted-topic
		// tombstone or an unreadable row. Either way there is nothing to route.
		return classTombstone
	}
	if _, isSchemaChange := probe.Payload["schema"]; isSchemaChange {
		return classSchemaChange
	}
	// A well-formed Debezium payload with no row image and no schema block is the
	// heartbeat Debezium emits on __debezium-heartbeat.<topic>.
	return classHeartbeat
}

// ---- flush outcome taxonomy ----

type loadOutcome int

const (
	// outcomeOK means the rows landed (or were already applied under this label).
	outcomeOK loadOutcome = iota
	// outcomeRowRejected means StarRocks refused specific rows. The batch is
	// bisected to isolate them; the rest still land.
	outcomeRowRejected
	// outcomeFatal means the batch can never succeed as addressed: an
	// authorization failure, or a destination that does not exist. Retrying would
	// either hammer the FE or spin forever, so these go to the DLQ.
	outcomeFatal
	// outcomeTransport means a 5xx, timeout or refused connection. Retryable.
	outcomeTransport
)

func (o loadOutcome) retryable() bool { return o == outcomeTransport }

func (o loadOutcome) String() string {
	switch o {
	case outcomeOK:
		return "ok"
	case outcomeRowRejected:
		return "row_rejected"
	case outcomeFatal:
		return "fatal"
	case outcomeTransport:
		return "transport"
	}
	return "unknown"
}

// classifyResponse turns a stream-load reply into an outcome. The table this
// encodes was measured on 3.3.22 and matters because the failure modes look
// alike from the outside but need opposite handling:
//
//	200 Success / Label Already Exists / Publish Timeout -> ok
//	200 Fail (filtered rows)                              -> bisect
//	401, 403                                              -> fatal (denied destination)
//	4xx database/table not found                          -> fatal (wrong destination)
//	5xx, timeout, refused                                 -> transport (retry)
//
// The HTTP status is judged before the body on purpose. A cross-database write
// answers 401 with no body, and an authorization denial must never be masked by a
// well-formed reply that happens to say Success.
func classifyResponse(status int, out starRocksResponse) (loadOutcome, error) {
	switch {
	case status == 401 || status == 403:
		// Cross-database writes answer 401 with NO JSON body, which is why this is
		// judged from the status code and never inferred from an empty Status.
		return outcomeFatal, fmt.Errorf("starrocks denied the write (HTTP %d) for %s.%s: "+
			"either the tenant's grants are missing or the row was routed to the wrong database",
			status, out.Database, out.Table)
	case status >= 500:
		return outcomeTransport, fmt.Errorf("starrocks transport failure (HTTP %d) for %s.%s: %s",
			status, out.Database, out.Table, out.Msg)
	case status >= 400:
		// Provisioning lag, or a routing bug that named a database nobody made.
		// Retryable in principle, but retrying a wrong destination only hammers the
		// FE, so it is DLQ'd with enough context to tell it apart from a 401.
		return outcomeFatal, fmt.Errorf("starrocks rejected the destination (HTTP %d) for %s.%s: %s",
			status, out.Database, out.Table, out.Msg)
	}

	switch out.Status {
	case "Success", "Label Already Exists", "Publish Timeout":
		return outcomeOK, nil
	case "Fail":
		return outcomeRowRejected, fmt.Errorf("stream load rejected: %s (%s)", out.Status, out.Msg)
	}
	// A 2xx we cannot read is not a decision we can act on. Treating it as success
	// would commit offsets for rows nobody confirmed; treating it as fatal would
	// dead-letter data over a malformed reply. Retrying is the only safe reading.
	return outcomeTransport, fmt.Errorf("starrocks replied HTTP %d with unrecognised status %q for %s.%s",
		status, out.Status, out.Database, out.Table)
}

// ---- per-tenant accumulator ----

// tenantBatch is one (tenant, table) destination's pending work. Because the
// loader runs one table per process, the tenant id alone keys the accumulator.
//
// The 2k-row / 10s budget applies per batch, so a window with many active tenants
// emits one load each. At the projected scale (hundreds of tenants, of which a
// handful are active in any window) that is a handful of small loads rather than
// one large one. loads_per_flush is exported as a metric so the multiplication is
// visible if that assumption ever stops holding, rather than silently degrading
// FE health.
type tenantBatch struct {
	route    TenantRoute
	ops      []pendingOp
	messages []kafka.Message
}

func (b *tenantBatch) empty() bool { return len(b.ops) == 0 }

// label is derived from the offset range so a batch re-processed after a consumer
// group rebalance is rejected as a duplicate at each destination. StarRocks scopes
// labels per database, so the same sanitized label is safe across tenants, but
// keeping the offsets in makes the replay case obvious rather than accidental.
func (b *tenantBatch) label(topic string) string {
	if len(b.messages) == 0 {
		return ""
	}
	first, last := b.messages[0], b.messages[len(b.messages)-1]
	return fmt.Sprintf("%s_%d_%d", sanitizeLabel(topic), first.Offset, last.Offset)
}

// batchSet is the flush window's accumulator, keyed by normalized tenant id.
type batchSet struct {
	byTenant map[string]*tenantBatch
	order    []string
	topic    string
}

func newBatchSet(topic string) *batchSet {
	return &batchSet{byTenant: map[string]*tenantBatch{}, topic: topic}
}

func (s *batchSet) get(route TenantRoute) *tenantBatch {
	key := normalizeTenantID(route.TenantID)
	b, ok := s.byTenant[key]
	if !ok {
		b = &tenantBatch{route: route}
		s.byTenant[key] = b
		s.order = append(s.order, key)
	}
	return b
}

func (s *batchSet) empty() bool { return len(s.order) == 0 }

// pending is the total across every destination, which is what the 2k-row budget
// is checked against so one busy tenant cannot starve the others of a flush.
func (s *batchSet) pending() int {
	n := 0
	for _, k := range s.order {
		n += len(s.byTenant[k].ops)
	}
	return n
}

func (s *batchSet) batches() []*tenantBatch {
	out := make([]*tenantBatch, 0, len(s.order))
	for _, k := range s.order {
		out = append(out, s.byTenant[k])
	}
	return out
}

// ---- credential store drift ----

// TenantStoreDrift compares the credential directory against the authoritative
// tenant list. This turns "the loader DLQ'd an unknown tenant" from a mystery into
// a reportable condition: either a tenant was provisioned in Postgres and never
// here, or one here no longer exists upstream.
type TenantStoreDrift struct {
	MissingCredentials []string // in Postgres, no credential file
	OrphanCredentials  []string // credential file, no longer in Postgres
}

func CheckTenantStoreDrift(routes []TenantRoute, postgresTenants []string) TenantStoreDrift {
	have := map[string]string{}
	for _, r := range routes {
		have[normalizeTenantID(r.TenantID)] = r.TenantName
	}
	upstream := map[string]bool{}
	var d TenantStoreDrift
	for _, t := range postgresTenants {
		key := normalizeTenantID(t)
		upstream[key] = true
		if _, ok := have[key]; !ok {
			d.MissingCredentials = append(d.MissingCredentials, t)
		}
	}
	for key, name := range have {
		if !upstream[key] {
			d.OrphanCredentials = append(d.OrphanCredentials, name)
		}
	}
	sort.Strings(d.MissingCredentials)
	sort.Strings(d.OrphanCredentials)
	return d
}

// ---- offset commit floor ----

// offsetFloor tracks, per topic-partition, the lowest offset whose destination has
// not settled.
//
// It exists because a flush window can now contain several tenants spread across
// the same partitions, and kafka-go's CommitMessages commits the highest offset it
// is given for each partition. If tenant A's rows at offset 5 are still replayable
// while tenant B's rows at offset 9 loaded fine, committing B's message would commit
// past A's and lose them permanently. Under single-destination routing this could not
// happen -- the window either succeeded or it did not -- so per-tenant partial
// success is what introduced the hazard.
type offsetFloor struct {
	mu     sync.Mutex
	byPart map[string]map[int]floorMark
}

// floorMark is where a partition is stuck, and when commits last stopped making
// progress there.
//
// lastAdvance resets whenever the floor MOVES, which is what makes it the paging
// metric: it answers "when did commits stop progressing" rather than "when did the
// current row arrive". A never-resetting clock reports one continuous hold spanning a
// stall, a recovery and a fresh stall -- recovery becomes invisible, so the alert never
// clears, and triage points at a cause that predates the real one.
//
// Re-blocking the SAME offset deliberately does not reset it: commits are stuck exactly
// where they already were, so that stall has not recovered and the clock must keep
// running.
type floorMark struct {
	offset      int64
	lastAdvance time.Time
}

func newOffsetFloor() *offsetFloor {
	return &offsetFloor{byPart: map[string]map[int]floorMark{}}
}

// block lowers the floor for every partition these messages touch. Offsets are
// lowered, never raised: a settled batch earlier in the partition must not clear a
// later unsettled one.
func (f *offsetFloor) block(msgs []kafka.Message) {
	if f == nil {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	now := time.Now()
	for _, m := range msgs {
		parts := f.byPart[m.Topic]
		if parts == nil {
			parts = map[int]floorMark{}
			f.byPart[m.Topic] = parts
		}
		// Only a genuine MOVE of the floor restarts the stall clock. Re-blocking the
		// same offset means commits are still stuck where they already were, so the
		// clock must keep running -- that is a stall that has not recovered.
		if cur, ok := parts[m.Partition]; !ok || m.Offset < cur.offset {
			parts[m.Partition] = floorMark{offset: m.Offset, lastAdvance: now}
		}
	}
}

// release clears the stall for any partition whose blocked offset has now been
// committed past. Without it a partition that stalled, recovered and was never
// re-stalled would keep reporting its old stall age forever, and the alert would
// never clear.
func (f *offsetFloor) release(committed []kafka.Message) {
	if f == nil {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, m := range committed {
		parts := f.byPart[m.Topic]
		if parts == nil {
			continue
		}
		// The blocked offset is not committable while it is blocked, so its appearance
		// in a committed set IS its resolution.
		if mark, ok := parts[m.Partition]; ok && m.Offset >= mark.offset {
			delete(parts, m.Partition)
		}
		if len(parts) == 0 {
			delete(f.byPart, m.Topic)
		}
	}
}

// holds reports, per topic-partition, how long commits have made no progress: the
// paging metric. It resets when the floor moves and clears when the partition commits
// past it, so a recovered partition stops alerting.
func (f *offsetFloor) holds() map[string]float64 {
	if f == nil {
		return nil
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.byPart) == 0 {
		return nil
	}
	now := time.Now()
	out := make(map[string]float64, len(f.byPart))
	for topic, parts := range f.byPart {
		for part, mark := range parts {
			out[fmt.Sprintf("%s-%d", topic, part)] = now.Sub(mark.lastAdvance).Seconds()
		}
	}
	return out
}

// safe returns the subset of msgs that may be committed without stepping over an
// unsettled offset. A nil floor has no restrictions, which is the single-destination
// behaviour and also keeps a partially built gatekeeper from panicking.
func (f *offsetFloor) safe(msgs []kafka.Message) []kafka.Message {
	if f == nil {
		return msgs
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]kafka.Message, 0, len(msgs))
	for _, m := range msgs {
		if parts := f.byPart[m.Topic]; parts != nil {
			if mark, ok := parts[m.Partition]; ok && m.Offset >= mark.offset {
				continue
			}
		}
		out = append(out, m)
	}
	return out
}

// blocked reports whether committing this single message would step over an
// unsettled offset. The consumer loop commits skipped and dead-lettered messages one
// at a time, and without this check a skip arriving after a blocked batch would
// commit past it.
func (f *offsetFloor) blocked(m kafka.Message) bool {
	if f == nil {
		return false
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if parts := f.byPart[m.Topic]; parts != nil {
		if mark, ok := parts[m.Partition]; ok && m.Offset >= mark.offset {
			return true
		}
	}
	return false
}

// routerCache allows the router to be rebuilt from disk without restarting, which
// matters because provisioning runs as a separate job from the loader.
type routerCache struct {
	mu     sync.RWMutex
	router *TenantRouter
	loaded time.Time
}

// get returns the current router, or nil when per-tenant routing is off. The nil
// receiver is deliberate: a gatekeeper built without a router is the normal
// single-destination deployment, not an error.
func (c *routerCache) get() *TenantRouter {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.router
}

func (c *routerCache) set(r *TenantRouter) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.router = r
	c.loaded = time.Now()
}

// ---- per-destination configuration ----

// ConfigFor derives the loader config that talks to this tenant's database. Only
// the destination fields change: topic, keys, batching and rule engine stay shared,
// because one loader process still handles one table on one topic.
//
// The derived QueryDSN must name the tenant's own database. Reusing the parent's
// DSN here would be a silent cross-database write that the grants correctly deny --
// which is safe, but turns every row into a 401 instead of a load.
func (r TenantRoute) ConfigFor(parent Config) Config {
	cfg := parent
	cfg.StarRocksDB = r.Database
	cfg.StarRocksUser = r.User
	cfg.StarRocksPassword = r.Password
	if len(r.KeyColumns) > 0 {
		cfg.PrimaryKeys = r.KeyColumns
	}
	if r.DSN != "" {
		cfg.QueryDSN = r.DSN
	} else if host := hostOf(parent.StarRocksHTTP); host != "" {
		cfg.QueryDSN = fmt.Sprintf("%s:%s@tcp(%s:9030)/%s", r.User, r.Password, host, r.Database)
	}
	return cfg
}

// Target renders the destination as db.table for alert payloads, so 2am triage can
// tell a routing bug from an expired credential without re-deriving the mapping.
func (r TenantRoute) Target(table string) string { return r.Database + "." + table }

// ---- fatal-load error carrying ----

// loadError attaches the taxonomy outcome to a transport error. classifyResponse
// alone is not enough at the call site: the flush loop needs to know whether to
// retry, the DLQ needs the tenant and target, and tests need the outcome. Carrying
// it on the error keeps that from being reconstructed from a message string.
type loadError struct {
	outcome loadOutcome
	status  int
	resp    starRocksResponse
	target  string
	tenant  string
	err     error
}

func (e *loadError) Error() string { return e.err.Error() }
func (e *loadError) Unwrap() error { return e.err }

// Outcome reports the taxonomy class, or outcomeTransport for a plain network error
// that never reached a response. Defaulting to retryable is correct: an error with
// no classification is an error we have never seen the server answer.
func OutcomeOf(err error) loadOutcome {
	var le *loadError
	if errors.As(err, &le) {
		return le.outcome
	}
	return outcomeTransport
}

// FatalLoadError extracts the classified error, if this was one.
func FatalLoadError(err error) (*loadError, bool) {
	var le *loadError
	if errors.As(err, &le) && !le.outcome.retryable() {
		return le, true
	}
	return nil, false
}
