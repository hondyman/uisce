package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	kafka "github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeBindings returns a fixed row count or error, so the tripwire's decision can be
// exercised without a live control plane.
type fakeBindings struct {
	n       int64
	err     error
	queried int
}

func (f *fakeBindings) CountBindings(context.Context) (int64, error) {
	f.queried++
	return f.n, f.err
}

func routedCfg() Config {
	return Config{TenantRoutesDir: "/etc/uisce/sr-tenants", SourceDatabase: controlPlaneDatabase}
}

func TestTripwireSilentWhileBindingTableIsEmpty(t *testing.T) {
	// An empty registry means the old discriminator is still the correct one.
	f := &fakeBindings{n: 0}
	v := checkTopologyTripwire(context.Background(), f, routedCfg())

	assert.False(t, v.fired)
	assert.Equal(t, 1, f.queried, "the check must actually read the registry")
}

func TestTripwireFiresOnceBindingsExist(t *testing.T) {
	f := &fakeBindings{n: 3}
	v := checkTopologyTripwire(context.Background(), f, routedCfg())

	require.True(t, v.fired, "a declared per-tenant analytics destination invalidates the tenant_id discriminator")
	assert.Equal(t, int64(3), v.bindings)
	// The alert has to be actionable without opening the source.
	assert.Contains(t, v.reason, "source.db")
	assert.Contains(t, v.reason, "replication slot")
	assert.Contains(t, v.reason, controlPlaneDatabase)
}

func TestTripwireQuietWhenDiscriminatorAlreadyFlipped(t *testing.T) {
	// If CDC_SOURCE_DB no longer names the control plane, routing is already keyed on
	// source.db, so the bindings are expected rather than alarming.
	cfg := routedCfg()
	cfg.SourceDatabase = "tenant_northwinds"
	v := checkTopologyTripwire(context.Background(), &fakeBindings{n: 3}, cfg)

	assert.False(t, v.fired)
	assert.Contains(t, v.reason, "already keyed on source.db")
}

func TestTripwireQuietWithoutRouting(t *testing.T) {
	// No per-tenant routing means no discriminator in play, so the registry's contents
	// say nothing about this process.
	f := &fakeBindings{n: 5}
	v := checkTopologyTripwire(context.Background(), f, Config{SourceDatabase: controlPlaneDatabase})

	assert.False(t, v.fired)
	assert.Equal(t, 0, f.queried, "no routing means no reason to query at all")
}

func TestTripwireDoesNotFireOnQueryError(t *testing.T) {
	// A briefly unreachable control plane is not evidence of a topology change, and an
	// alarm that cries wolf on a transient is one people learn to ignore.
	v := checkTopologyTripwire(context.Background(),
		&fakeBindings{err: errors.New("connection refused")}, routedCfg())

	assert.False(t, v.fired)
	assert.Contains(t, v.reason, "connection refused")
}

func TestTripwireQuietWithoutControlPlaneConnection(t *testing.T) {
	v := checkTopologyTripwire(context.Background(), nil, routedCfg())
	assert.False(t, v.fired)
	assert.Contains(t, v.reason, "no control-plane connection")
}

func TestReportTripwireCountsTheFiring(t *testing.T) {
	metrics := &GatekeeperMetrics{}
	reportTopologyTripwire(context.Background(), &fakeBindings{n: 1}, routedCfg(), metrics)
	assert.Equal(t, int64(1), metrics.TopologyTripwireFired.Load())

	reportTopologyTripwire(context.Background(), &fakeBindings{n: 0}, routedCfg(), metrics)
	assert.Equal(t, int64(1), metrics.TopologyTripwireFired.Load(),
		"a quiet check must not inflate the counter")
}

// ---- DLQ acknowledgement gating ----

func TestEmitDLQReportsSuccessWithoutAWriter(t *testing.T) {
	// No DLQ configured at all is a deployment choice, not a transient fault: it is
	// counted so it is visible, but it does not stall the stream forever.
	gk := newTestGatekeeper(Config{Topic: "t"})
	require.NoError(t, gk.emitDLQ(context.Background(), "ERR_X", "detail", nil, nil))

	assert.Equal(t, int64(1), gk.metrics.DLQUnavailable.Load())
	assert.Equal(t, int64(1), gk.metrics.DLQEmitted.Load())
}

func TestEmitDLQReportsFailureWhenTheBrokerIsUnreachable(t *testing.T) {
	gk := newTestGatekeeper(Config{Topic: "t", DLQTopic: "dlq"})
	gk.dlqWriter = &kafka.Writer{
		Addr:         kafka.TCP("127.0.0.1:1"),
		Topic:        "dlq",
		WriteTimeout: 100 * time.Millisecond,
		MaxAttempts:  1,
	}
	defer gk.dlqWriter.Close()

	err := gk.emitDLQ(context.Background(), "ERR_X", "detail", nil, nil)
	require.Error(t, err, "an unacknowledged dead-letter must be reported, not swallowed")
	assert.Contains(t, err.Error(), "dlq")
}

func TestFatalLoadIsUnsettledWhenTheDeadLetterCannotBeWritten(t *testing.T) {
	// The dead-letter is the only durable record of a fatal batch. If it never lands,
	// the batch is not terminal, and committing its offsets would discard rows that
	// then exist nowhere -- the exact loss the offset floor exists to prevent.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// A destination that does not exist: fatal, and never retried.
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	gk := newTestGatekeeper(Config{
		Topic: "orm_oms.orm.order", StarRocksHTTP: ts.URL, StarRocksDB: "oms",
		StarRocksTable: "orm_order", PrimaryKeys: []string{"id"}, StrictMode: true,
	})
	gk.dlqWriter = &kafka.Writer{
		Addr:         kafka.TCP("127.0.0.1:1"),
		Topic:        "dlq",
		WriteTimeout: 100 * time.Millisecond,
		MaxAttempts:  1,
	}
	defer gk.dlqWriter.Close()

	dest := &tenantBatch{
		route: TenantRoute{TenantID: tenantA, Database: "tenant_99e99e99", KeyColumns: []string{"id"}},
		ops:   []pendingOp{{key: "k1", row: []byte(`{"id":"k1"}`)}},
		messages: []kafka.Message{
			{Topic: "orm_oms.orm.order", Partition: 0, Offset: 4},
		},
	}

	settled := gk.flushDestination(context.Background(), dest, "test")
	assert.False(t, settled, "the rows exist in neither StarRocks nor the DLQ yet")
	assert.Equal(t, int64(1), gk.metrics.FatalLoads.Load())
}

func TestFatalLoadSettlesWhenTheDeadLetterIsAcknowledged(t *testing.T) {
	// The same fatal batch, but the DLQ write is a no-op that succeeds.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	gk := newTestGatekeeper(Config{
		Topic: "orm_oms.orm.order", StarRocksHTTP: ts.URL, StarRocksDB: "oms",
		StarRocksTable: "orm_order", PrimaryKeys: []string{"id"}, StrictMode: true,
	})
	dest := &tenantBatch{
		route: TenantRoute{TenantID: tenantA, Database: "tenant_99e99e99", KeyColumns: []string{"id"}},
		ops:   []pendingOp{{key: "k1", row: []byte(`{"id":"k1"}`)}},
		messages: []kafka.Message{
			{Topic: "orm_oms.orm.order", Partition: 0, Offset: 4},
		},
	}

	assert.True(t, gk.flushDestination(context.Background(), dest, "test"),
		"a dead-lettered row is terminal; replaying it forever would stall the partition")
}
