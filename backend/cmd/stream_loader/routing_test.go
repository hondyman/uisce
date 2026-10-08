package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	kafka "github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	tenantA = "99e99e99-99e9-49e9-89e9-99e99e99e999"
	tenantB = "88e88e88-88e8-48e8-88e8-88e88e88e888"
)

func sampleRoutes() []TenantRoute {
	return []TenantRoute{
		{TenantID: tenantA, TenantName: "acme", Database: "tenant_99e99e99", User: "t_acme",
			Password: "pw-a", KeyColumns: []string{"id"}},
		{TenantID: tenantB, TenantName: "globex", Database: "tenant_88e88e88", User: "t_globex",
			Password: "pw-b", KeyColumns: []string{"id"}},
	}
}

func routingGatekeeper(t *testing.T, cfg Config) *StreamingGatekeeper {
	t.Helper()
	gk := newTestGatekeeper(cfg)
	routes := sampleRoutes()
	gk.router = &routerCache{router: NewTenantRouter(routes, "alpha")}
	return gk
}

// ---- normalisation and lookup ----

func TestNormalizeTenantIDCollapsesHyphenation(t *testing.T) {
	// Postgres emits the hyphenated uuid; StarRocks identifiers drop the hyphens. If
	// the two forms did not resolve to one key, a tenant's rows would split across
	// two destinations and neither copy would be complete.
	assert.Equal(t, normalizeTenantID(tenantA), normalizeTenantID(strings.ReplaceAll(tenantA, "-", "")))
	assert.Equal(t, normalizeTenantID("  "+tenantA+"  "), normalizeTenantID(tenantA))
	assert.Equal(t, "", normalizeTenantID("   "))
}

func TestRouterResolvesBothTenantIDForms(t *testing.T) {
	r := NewTenantRouter(sampleRoutes(), "alpha")

	for _, form := range []string{tenantA, strings.ReplaceAll(tenantA, "-", ""), strings.ToUpper(tenantA)} {
		got, err := r.Route(form)
		require.NoError(t, err)
		assert.Equal(t, "tenant_99e99e99", got.Database)
	}

	assert.True(t, r.Enabled())
	assert.Equal(t, 2, r.Len())
	assert.True(t, r.Known(tenantB))
}

func TestRouterFailsClosed(t *testing.T) {
	r := NewTenantRouter(sampleRoutes(), "alpha")

	_, err := r.Route("")
	assert.ErrorIs(t, err, ErrTenantUnattributed,
		"an empty tenant id has no destination and must not fall back to one")

	_, err = r.Route("deadbeef-0000-0000-0000-000000000000")
	assert.ErrorIs(t, err, ErrNoTenantRoute)
	assert.Contains(t, err.Error(), "deadbeef",
		"the DLQ payload must name the tenant that had no route")

	assert.False(t, NewTenantRouter(nil, "alpha").Enabled())
}

// ---- credential store ----

func TestLoadTenantRoutes(t *testing.T) {
	dir := t.TempDir()
	write := func(name string, body string) {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600))
	}
	write("a.json", `{"tenant_id":"`+tenantA+`","tenant_name":"acme","database":"tenant_a","user":"t_a","password":"pw"}`)
	write("b.json", `{"tenant_id":"`+tenantB+`","database":"tenant_b","user":"t_b","password":"pw"}`)
	// A route with no database would send rows to the wrong place; it is skipped whole.
	write("half.json", `{"tenant_id":"x","user":"t_x"}`)
	write("broken.json", `{not json`)
	write("notes.txt", `ignored`)

	routes, err := LoadTenantRoutes(dir)
	require.NoError(t, err)
	require.Len(t, routes, 2)
	assert.Equal(t, tenantB, routes[0].TenantID, "routes are sorted so startup logging is stable")
	assert.Equal(t, tenantA, routes[1].TenantID)
}

func TestLoadTenantRoutesMissingDirectoryIsNotAnError(t *testing.T) {
	// A single-tenant deployment legitimately has no credential directory; whether
	// that is fatal is the caller's decision, not the loader's.
	routes, err := LoadTenantRoutes(filepath.Join(t.TempDir(), "absent"))
	require.NoError(t, err)
	assert.Empty(t, routes)
}

func TestCheckTenantStoreDrift(t *testing.T) {
	routes := []TenantRoute{
		{TenantID: tenantA, TenantName: "acme", Database: "d", User: "u"},
		{TenantID: "cccccccc-cccc-4ccc-8ccc-cccccccccccc", TenantName: "ghost", Database: "d", User: "u"},
	}
	upstream := []string{tenantA, tenantB, "77e77e77-77e7-47e7-87e7-77e77e77e777"}

	d := CheckTenantStoreDrift(routes, upstream)
	assert.Equal(t, []string{"77e77e77-77e7-47e7-87e7-77e77e77e777", tenantB}, d.MissingCredentials,
		"tenants present in Postgres with no StarRocks route would DLQ every row")
	assert.Equal(t, []string{"ghost"}, d.OrphanCredentials)
}

// ---- event classification ----

func TestClassifyDataEvents(t *testing.T) {
	upsert := json.RawMessage(`{"schema":{},"payload":{"before":null,"after":{"id":"1","tenant_id":"` + tenantA + `"},"op":"c","ts_ms":1}}`)
	ev, err := decodeRecord(upsert)
	require.NoError(t, err)
	assert.Equal(t, classData, classify(ev, upsert))

	del := json.RawMessage(`{"schema":{},"payload":{"before":{"id":"1","tenant_id":"` + tenantA + `"},"after":null,"op":"d","ts_ms":2}}`)
	ev, err = decodeRecord(del)
	require.NoError(t, err)
	assert.Equal(t, classData, classify(ev, del),
		"a delete carries its row in before and must still be routed")
}

func TestClassifyNonDataEventsBeforeRouting(t *testing.T) {
	// Every one of these has no payload.after and therefore no tenant_id. If routing
	// ran first, all of them would land in the DLQ and page somebody.
	cases := []struct {
		name string
		raw  string
		want eventClass
	}{
		{"kafka tombstone", `null`, classTombstone},
		{"empty record", ``, classTombstone},
		{"compacted null image", `{"schema":{},"payload":{"before":null,"after":null,"op":"c","ts_ms":1}}`, classTombstone},
		{"heartbeat", `{"schema":null,"payload":{"ts_ms":1700000000000,"serverName":"alpha","status":"UP"}}`, classHeartbeat},
		{"schema change", `{"schema":{},"payload":{"source":{"db":"alpha"},"ts_ms":1,"schema":{"type":"struct"}},"op":"m"}`, classSchemaChange},
		{"not json at all", `{`, classUnknown},
		{"no payload", `{"schema":null}`, classUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ev, _ := decodeRecord([]byte(tc.raw))
			assert.Equal(t, tc.want, classify(ev, []byte(tc.raw)))
		})
	}
}

func TestHandleMessageSkipsHeartbeatInsteadOfDLQing(t *testing.T) {
	gk := routingGatekeeper(t, Config{Topic: "orm_oms.orm.order", StarRocksTable: "orm_order",
		PrimaryKeys: []string{"id"}})

	raw := json.RawMessage(`{"schema":null,"payload":{"ts_ms":1700000000000,"status":"UP"}}`)
	batched := gk.handleMessage(context.Background(), newBatchSet("t"), kafka.Message{Topic: "orm_oms.orm.order", Value: raw})

	assert.False(t, batched)
	assert.Equal(t, int64(1), gk.metrics.SkippedHeartbeats.Load())
	assert.Equal(t, int64(0), gk.metrics.DLQEmitted.Load(),
		"a heartbeat is liveness, not a tenant failure")
	assert.Equal(t, int64(0), gk.metrics.TenantUnattributed.Load())
}

// ---- routing gate ----

func TestRouteFailsClosedOnMissingTenantID(t *testing.T) {
	gk := routingGatekeeper(t, Config{Topic: "t", StarRocksTable: "orm_order", PrimaryKeys: []string{"id"}})

	_, ok := gk.route(context.Background(), map[string]interface{}{"id": "1"}, kafka.Message{})
	assert.False(t, ok, "a data event with no tenant must not be batched")
	assert.Equal(t, int64(1), gk.metrics.TenantUnattributed.Load())
	assert.Equal(t, int64(1), gk.metrics.DLQEmitted.Load())
}

func TestRouteFailsClosedOnUnknownTenant(t *testing.T) {
	// Unknown covers both provisioning lag and a deprovisioned tenant whose events are
	// still in flight. Same outcome for both: history preserved, signal preserved.
	gk := routingGatekeeper(t, Config{Topic: "t", StarRocksTable: "orm_order", PrimaryKeys: []string{"id"}})

	_, ok := gk.route(context.Background(), map[string]interface{}{"id": "1", "tenant_id": tenantB + "x"}, kafka.Message{})
	assert.False(t, ok)
	assert.Equal(t, int64(1), gk.metrics.TenantUnknown.Load())
}

func TestRouteResolvesPerTenantDatabase(t *testing.T) {
	gk := routingGatekeeper(t, Config{Topic: "t", StarRocksTable: "orm_order",
		StarRocksHTTP: "http://starrocks-fe:8030", StarRocksDB: "oms",
		StarRocksUser: "root", StarRocksPassword: "rootpw", PrimaryKeys: []string{"id"}})

	route, ok := gk.route(context.Background(), map[string]interface{}{"id": "1", "tenant_id": tenantB}, kafka.Message{})
	require.True(t, ok)
	assert.Equal(t, "tenant_88e88e88", route.Database)
	assert.Equal(t, "t_globex", route.User)

	cfg := route.ConfigFor(gk.cfg)
	assert.Equal(t, "tenant_88e88e88", cfg.StarRocksDB)
	assert.Equal(t, "t_globex", cfg.StarRocksUser)
	assert.Contains(t, cfg.QueryDSN, "/tenant_88e88e88",
		"the delete DSN must name the tenant's own database, not the shared one")
	assert.NotContains(t, cfg.QueryDSN, "rootpw", "the shared root credential must not leak into a tenant route")
}

func TestRouteSingleDestinationWhenRoutingDisabled(t *testing.T) {
	gk := newTestGatekeeper(Config{Topic: "t", StarRocksDB: "oms", StarRocksUser: "root",
		StarRocksTable: "orm_order", PrimaryKeys: []string{"id"}})

	route, ok := gk.route(context.Background(), map[string]interface{}{"id": "1"}, kafka.Message{})
	require.True(t, ok)
	assert.Equal(t, "oms", route.Database, "an unconfigured deployment keeps its single destination")
}

// ---- per-tenant accumulator ----

func TestBatchSetKeyedByTenant(t *testing.T) {
	router := NewTenantRouter(sampleRoutes(), "alpha")
	s := newBatchSet("orm_oms.orm.order")

	ra, _ := router.Route(tenantA)
	rb, _ := router.Route(tenantB)

	for i := 0; i < 3; i++ {
		s.get(ra).ops = append(s.get(ra).ops, pendingOp{key: "a"})
	}
	s.get(rb).ops = append(s.get(rb).ops, pendingOp{key: "b"})

	assert.Equal(t, 4, s.pending(), "the row budget is checked across all destinations")
	assert.Len(t, s.batches(), 2, "one flush window emits one load per active tenant")
	assert.Same(t, s.get(ra), s.get(ra), "the same tenant must reuse its accumulator")
	assert.False(t, s.empty(), "a window holding four ops is not empty")
	assert.True(t, newBatchSet("t").empty(), "a fresh window flushes nothing")

	s.get(ra).messages = append(s.get(ra).messages, kafka.Message{Offset: 10})
	s.get(ra).messages = append(s.get(ra).messages, kafka.Message{Offset: 42})
	// A batch spanning several partitions takes first and last offset, which is still
	// stable for a replay of the same window.
	assert.Equal(t, "orm_oms_orm_order_10_42", s.get(ra).label("orm_oms.orm.order"))
}

// ---- response taxonomy ----

func TestClassifyResponseTaxonomy(t *testing.T) {
	out := starRocksResponse{Database: "tenant_a", Table: "orm_order"}

	cases := []struct {
		name   string
		status int
		body   starRocksResponse
		want   loadOutcome
	}{
		{"success", 200, starRocksResponse{Status: "Success"}, outcomeOK},
		{"duplicate label", 200, starRocksResponse{Status: "Label Already Exists"}, outcomeOK},
		{"publish timeout", 200, starRocksResponse{Status: "Publish Timeout"}, outcomeOK},
		{"filtered rows", 200, starRocksResponse{Status: "Fail", Msg: "too many filtered rows"}, outcomeRowRejected},
		{"cross-db denial", 401, out, outcomeFatal},
		{"forbidden", 403, out, outcomeFatal},
		{"unknown database", 404, out, outcomeFatal},
		{"fe error", 500, out, outcomeTransport},
		{"bad gateway", 502, out, outcomeTransport},
		{"unreadable 200", 200, starRocksResponse{}, outcomeTransport},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := classifyResponse(tc.status, tc.body)
			assert.Equal(t, tc.want, got)
			if tc.want == outcomeOK {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			assert.Equal(t, tc.want.retryable(), got.retryable(),
				"only transport failures may be retried")
		})
	}
}

func TestClassifyResponseJudgesStatusBeforeBody(t *testing.T) {
	// A denial must never be masked by a well-formed reply that happens to claim
	// success, or the loader would commit offsets for rows that were refused.
	got, err := classifyResponse(401, starRocksResponse{Status: "Success", Database: "other", Table: "orm_order"})
	assert.Equal(t, outcomeFatal, got)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "other.orm_order", "the alert must name the database that refused the write")
}

func TestOutcomeOfUnknownErrorIsRetryable(t *testing.T) {
	assert.Equal(t, outcomeTransport, OutcomeOf(assert.AnError),
		"an error with no classification is one we have never seen a decision on")
	_, fatal := FatalLoadError(assert.AnError)
	assert.False(t, fatal)
}

// ---- flush behaviour ----

func newRoutedGK(t *testing.T, ts *httptest.Server) (*StreamingGatekeeper, *tenantBatch) {
	t.Helper()
	gk := routingGatekeeper(t, Config{
		Topic: "orm_oms.orm.order", StarRocksHTTP: ts.URL, StarRocksDB: "oms",
		StarRocksTable: "orm_order", PrimaryKeys: []string{"id"}, StrictMode: true,
	})
	dest := &tenantBatch{route: TenantRoute{TenantID: tenantA, Database: "tenant_99e99e99",
		User: "t_acme", Password: "pw", KeyColumns: []string{"id"}},
		ops:      []pendingOp{{key: "k1", row: json.RawMessage(`{"id":"k1"}`)}},
		messages: []kafka.Message{{Topic: "orm_oms.orm.order", Offset: 1}}}
	return gk, dest
}

// flushed runs one destination the way the flush window does and returns the
// messages it settled -- or nothing if it did not settle.
func flushed(gk *StreamingGatekeeper, dest *tenantBatch) []kafka.Message {
	if gk.flushDestination(context.Background(), dest, "test") {
		return dest.messages
	}
	gk.floor.block(dest.messages)
	return nil
}

func TestFlushDestinationDeadLettersOnCrossDatabaseDenial(t *testing.T) {
	var attempts int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		// Verified live on 3.3.22: a cross-database stream load answers 401 with no
		// body at all.
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer ts.Close()

	gk, dest := newRoutedGK(t, ts)
	committed := flushed(gk, dest)

	assert.Equal(t, 1, attempts, "a denial that can never succeed must not be retried")
	assert.Len(t, committed, 1, "the offsets are settled: the rows are dead-lettered, not replayed forever")
	assert.Equal(t, int64(1), gk.metrics.FatalLoads.Load())
	assert.Equal(t, int64(1), gk.metrics.DLQEmitted.Load())
	assert.Equal(t, int64(0), gk.metrics.BatchesFlushed.Load())
}

func TestFlushDestinationRetriesTransportErrors(t *testing.T) {
	var attempts int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write([]byte(`{"Status":"Success"}`))
	}))
	defer ts.Close()

	gk, dest := newRoutedGK(t, ts)
	committed := flushed(gk, dest)

	assert.Equal(t, 3, attempts)
	assert.Len(t, committed, 1)
	assert.Equal(t, int64(1), gk.metrics.BatchesFlushed.Load())
	assert.Equal(t, int64(0), gk.metrics.FatalLoads.Load())
}

func TestFlushDestinationDeadLettersWhenTransportNeverRecovers(t *testing.T) {
	var attempts int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer ts.Close()

	gk, dest := newRoutedGK(t, ts)
	committed := flushed(gk, dest)

	assert.Equal(t, maxFlushAttempts, attempts, "retries are bounded so one bad destination cannot stall the window")
	assert.Len(t, committed, 1, "an unreachable destination must not hold offsets back forever")
	assert.Equal(t, int64(1), gk.metrics.DLQEmitted.Load())
	assert.Equal(t, int64(0), gk.metrics.BatchesFlushed.Load())
}

func TestFlushDestinationIsolatesOneTenantFromAnother(t *testing.T) {
	// Tenant A's database does not exist; tenant B's works. B must still land.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tenant_missing/orm_order/_stream_load" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write([]byte(`{"Status":"Success"}`))
	}))
	defer ts.Close()

	gk, good := newRoutedGK(t, ts)
	bad := &tenantBatch{route: TenantRoute{TenantID: "gone", Database: "tenant_missing",
		User: "t_gone", KeyColumns: []string{"id"}},
		ops:      []pendingOp{{key: "k9", row: json.RawMessage(`{"id":"k9"}`)}},
		messages: []kafka.Message{{Topic: "orm_oms.orm.order", Offset: 2}}}

	committed := flushed(gk, bad)
	committed = append(committed, flushed(gk, good)...)

	assert.Equal(t, int64(1), gk.metrics.FatalLoads.Load())
	assert.Equal(t, int64(1), gk.metrics.BatchesFlushed.Load(), "a sibling tenant's failure must not stall this one")
	assert.Len(t, committed, 2)
}

func TestWriteBatchTargetsTheTenantDatabaseAndPrincipal(t *testing.T) {
	var gotPath, gotUser, gotPass string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotUser, gotPass, _ = r.BasicAuth()
		w.Write([]byte(`{"Status":"Success"}`))
	}))
	defer ts.Close()

	gk, dest := newRoutedGK(t, ts)
	require.NoError(t, gk.writeBatch(context.Background(), dest))

	assert.Equal(t, "/api/tenant_99e99e99/orm_order/_stream_load", gotPath)
	assert.Equal(t, "t_acme", gotUser)
	assert.Equal(t, "pw", gotPass,
		"the tenant's own credential is used, so the grants are the isolation boundary")
}

// ---- offset commit floor ----

func TestOffsetFloorStopsACommitSteppingOverUnlandedRows(t *testing.T) {
	// Several tenants share one partition. kafka-go commits the highest offset it is
	// handed, so committing tenant B's rows at offset 9 while tenant A's rows at
	// offset 5 are still replayable would lose A's rows permanently.
	f := newOffsetFloor()
	msg := func(part int, off int64) kafka.Message {
		return kafka.Message{Topic: "orm_oms.orm.order", Partition: part, Offset: off}
	}

	f.block([]kafka.Message{msg(0, 5)})

	safe := f.safe([]kafka.Message{msg(0, 3), msg(0, 4), msg(0, 9), msg(0, 10)})
	require.Len(t, safe, 2, "only offsets strictly below the floor may be committed")
	assert.Equal(t, int64(3), safe[0].Offset)
	assert.Equal(t, int64(4), safe[1].Offset)
}

func TestOffsetFloorIsPerPartition(t *testing.T) {
	// A stuck tenant on one partition must not freeze every other partition: the
	// others are healthy and holding their offsets back buys nothing.
	f := newOffsetFloor()
	f.block([]kafka.Message{{Topic: "t", Partition: 0, Offset: 5}})

	safe := f.safe([]kafka.Message{
		{Topic: "t", Partition: 0, Offset: 9},
		{Topic: "t", Partition: 1, Offset: 9},
		{Topic: "other", Partition: 0, Offset: 9},
	})
	assert.Len(t, safe, 2, "other partitions and topics stay committable")
}

func TestOffsetFloorOnlyMovesDown(t *testing.T) {
	// Settling an earlier batch must not clear a later unsettled one.
	f := newOffsetFloor()
	f.block([]kafka.Message{{Topic: "t", Partition: 0, Offset: 9}})
	f.block([]kafka.Message{{Topic: "t", Partition: 0, Offset: 3}})

	assert.Len(t, f.safe([]kafka.Message{{Topic: "t", Partition: 0, Offset: 20}}), 0)
}

func TestOffsetFloorGuardsSingleMessageCommit(t *testing.T) {
	// The consumer loop commits skipped and dead-lettered messages one at a time.
	// Without this check a skip arriving after a blocked batch would commit past it.
	f := newOffsetFloor()
	assert.False(t, f.blocked(kafka.Message{Topic: "t", Partition: 0, Offset: 1}))
	f.block([]kafka.Message{{Topic: "t", Partition: 0, Offset: 5}})
	assert.False(t, f.blocked(kafka.Message{Topic: "t", Partition: 0, Offset: 4}))
	assert.True(t, f.blocked(kafka.Message{Topic: "t", Partition: 0, Offset: 5}))
	assert.True(t, f.blocked(kafka.Message{Topic: "t", Partition: 0, Offset: 9}))
	assert.False(t, f.blocked(kafka.Message{Topic: "t", Partition: 1, Offset: 9}))
}

func TestUnsettledDestinationHoldsOnlyItsOwnOffsets(t *testing.T) {
	// End to end: tenant A cannot load, tenant B can. B's rows must not be committed
	// past A's, because both live on partition 0.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tenant_stuck/orm_order/_stream_load" {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte(`{"Status":"Success"}`))
	}))
	defer ts.Close()

	gk := routingGatekeeper(t, Config{
		Topic: "orm_oms.orm.order", StarRocksHTTP: ts.URL, StarRocksDB: "oms",
		StarRocksTable: "orm_order", PrimaryKeys: []string{"id"}, StrictMode: true,
	})

	stuck := &tenantBatch{route: TenantRoute{TenantID: "stuck", Database: "tenant_stuck",
		User: "t_stuck", KeyColumns: []string{"id"}},
		ops:      []pendingOp{{key: "k1", row: json.RawMessage(`{"id":"k1"}`)}},
		messages: []kafka.Message{{Topic: "orm_oms.orm.order", Partition: 0, Offset: 4}}}
	healthy := &tenantBatch{route: TenantRoute{TenantID: tenantB, Database: "tenant_88e88e88",
		User: "t_globex", KeyColumns: []string{"id"}},
		ops:      []pendingOp{{key: "k2", row: json.RawMessage(`{"id":"k2"}`)}},
		messages: []kafka.Message{{Topic: "orm_oms.orm.order", Partition: 0, Offset: 6}}}

	// Pretend the stuck destination exhausted its flush window without settling.
	gk.floor.block(stuck.messages)
	require.True(t, gk.flushDestination(context.Background(), healthy, "test"))

	assert.Empty(t, gk.floor.safe(healthy.messages),
		"the healthy tenant shares partition 0 with the stuck one; committing it would skip offset 4")
	assert.Equal(t, int64(1), gk.metrics.BatchesFlushed.Load())
}

// ---- hold-clock semantics ----

func TestFloorStallClockResetsWhenTheFloorMoves(t *testing.T) {
	// A never-resetting clock reports one continuous hold across a stall, a recovery
	// and a fresh stall: recovery becomes invisible and triage points at a cause that
	// predates the real one.
	f := newOffsetFloor()
	msg := func(off int64) kafka.Message {
		return kafka.Message{Topic: "t", Partition: 0, Offset: off}
	}

	f.block([]kafka.Message{msg(5)})
	first := f.holds()["t-0"]
	assert.GreaterOrEqual(t, first, 0.0)

	// The same partition blocks LOWER: the floor moved, so commits stopped making
	// progress again and the stall clock must restart.
	f.block([]kafka.Message{msg(3)})
	second := f.holds()["t-0"]
	assert.LessOrEqual(t, second, first, "a moved floor restarts the stall clock")
	assert.Less(t, second, 1.0, "the reset clock is fresh, not a continuation")
}

func TestFloorStallClockKeepsRunningWhenTheSameOffsetStaysBlocked(t *testing.T) {
	// Re-blocking the same offset means commits are stuck where they already were. This
	// is a stall that has NOT recovered, and the clock must keep counting.
	f := newOffsetFloor()
	msg := kafka.Message{Topic: "t", Partition: 0, Offset: 5}

	f.block([]kafka.Message{msg})
	time.Sleep(20 * time.Millisecond)
	f.block([]kafka.Message{msg})

	assert.GreaterOrEqual(t, f.holds()["t-0"], 0.015,
		"an unresolved stall must keep ageing, otherwise a stuck tenant looks healthy")
}

func TestFloorClearsOnRecovery(t *testing.T) {
	// Without this, a partition that stalled once and recovered would keep reporting its
	// old stall age forever and the alert would never clear.
	f := newOffsetFloor()
	f.block([]kafka.Message{{Topic: "t", Partition: 0, Offset: 5}})
	require.Contains(t, f.holds(), "t-0")

	f.release([]kafka.Message{{Topic: "t", Partition: 0, Offset: 5}})
	assert.Empty(t, f.holds(), "committing past the floor clears the stall")
}
