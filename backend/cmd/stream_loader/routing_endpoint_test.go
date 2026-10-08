package main

// Tests for the two defects the second-tenant fixture found on its first live run:
// deletes that could not be attributed because Postgres was not configured to send
// the row, and keyed DELETEs that dialled an endpoint written for someone else's
// network namespace.

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	kafka "github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- DSN endpoint resolution ----
//
// The provisioning script writes each route's DSN with its own SR_HOST, which on the
// deploy host is 127.0.0.1. That is correct for the script and wrong for a loader in
// a container, where 127.0.0.1 is the container's own loopback. Observed live on
// 2026-10-08: every keyed DELETE retried "dial tcp 127.0.0.1:9030: connect:
// connection refused" three times and then dead-lettered, leaving deleted rows in
// the tenant database forever.

func TestDSNForHostRewritesOnlyTheAddress(t *testing.T) {
	r := TenantRoute{
		TenantID: tenantA,
		Database: "tenant_a",
		User:     "t_a",
		Password: "s3cr3t",
		DSN:      "t_a:s3cr3t@tcp(127.0.0.1:9030)/tenant_a",
	}

	got := r.DSNForHost("starrocks-fe")

	assert.Equal(t, "t_a:s3cr3t@tcp(starrocks-fe:9030)/tenant_a", got,
		"credentials and database belong to the provisioning script; only the endpoint moves")
}

func TestDSNForHostPreservesAnExplicitPort(t *testing.T) {
	// 9030 is StarRocks' query port, not the only one it can listen on. A rewrite
	// that hardcoded it would silently break a non-default deployment.
	r := TenantRoute{DSN: "u:p@tcp(10.0.0.5:9031)/db"}
	assert.Equal(t, "u:p@tcp(starrocks-fe:9031)/db", r.DSNForHost("starrocks-fe"))
}

func TestDSNForHostPreservesDSNParameters(t *testing.T) {
	// Params are how parseTime, loc and friends are set. Dropping them changes the
	// driver's behaviour on the very connection that now succeeds.
	r := TenantRoute{DSN: "u:p@tcp(127.0.0.1:9030)/db?parseTime=true&loc=UTC"}
	assert.Equal(t, "u:p@tcp(starrocks-fe:9030)/db?parseTime=true&loc=UTC", r.DSNForHost("starrocks-fe"))
}

func TestDSNForHostLeavesUnusableInputAlone(t *testing.T) {
	// Returning the original rather than a best guess keeps a broken DSN broken in a
	// way the connection error names, instead of silently reshaping it into a
	// different target.
	assert.Equal(t, "not-a-dsn", TenantRoute{DSN: "not-a-dsn"}.DSNForHost("starrocks-fe"),
		"a DSN with no @tcp( clause is left untouched")
	assert.Equal(t, "u:p@tcp(127.0.0.1:9030)/db", TenantRoute{DSN: "u:p@tcp(127.0.0.1:9030)/db"}.DSNForHost(""),
		"no host configured means no rewrite")
	assert.Equal(t, "", TenantRoute{}.DSNForHost("starrocks-fe"),
		"a route with no DSN stays empty")
}

func TestLoadTenantRoutesAppliesTheQueryHost(t *testing.T) {
	dir := t.TempDir()
	body := `{"tenant_id":"` + tenantA + `","database":"tenant_a","user":"t_a","password":"pw",` +
		`"dsn":"t_a:pw@tcp(127.0.0.1:9030)/tenant_a"}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.json"), []byte(body), 0o600))

	routes, err := LoadTenantRoutes(dir, "starrocks-fe")
	require.NoError(t, err)
	require.Len(t, routes, 1)
	assert.Equal(t, "t_a:pw@tcp(starrocks-fe:9030)/tenant_a", routes[0].DSN,
		"every route read from disk is dialled from the loader's own network position")
}

// ---- delete attribution ----

// zeroFilledDeleteBefore is the `before` image of a real Debezium delete captured
// from orm-oms-connector-v2 on 2026-10-08, not a hand-written approximation. The
// tell is that every non-key column is its schema zero value: tenant_id "" rather
// than a uuid, trade_date 0, created_at at the epoch. Under REPLICA IDENTITY DEFAULT
// the logical decoding plugin sends only the key (`id`) and Debezium fills the rest.
func zeroFilledDeleteBefore() map[string]interface{} {
	return map[string]interface{}{
		"id":         "b2222222-2222-4222-8222-222222222222",
		"sec_id":     "0",
		"trade_date": float64(0),
		"tenant_id":  "",
		"side":       "",
		"status":     "NEW",
		"created_at": "1970-01-01T00:00:00.000000Z",
	}
}

func TestAttributionErrorNamesTheReplicaIdentityZeroFill(t *testing.T) {
	err := attributionError(zeroFilledDeleteBefore(), true)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrTenantUnavailableOnDelete,
		"a delete whose tenant field is present-but-empty is a Postgres configuration fault, not missing data")
	assert.NotErrorIs(t, err, ErrTenantUnattributed,
		"the two causes have different fixes and must not be collapsed into one reason code")
	assert.Contains(t, err.Error(), "REPLICA IDENTITY FULL",
		"the error carries its own fix; an operator reading a DLQ at 2am should not have to derive it")
}

func TestAttributionErrorTreatsAGenuinelyMissingTenantAsUnattributed(t *testing.T) {
	// No tenant_id column at all: the data is wrong, not the schema.
	assert.ErrorIs(t, attributionError(map[string]interface{}{"id": "x"}, true), ErrTenantUnattributed)

	// Present but explicitly null: a real NULL tenant, again not a zero fill.
	assert.ErrorIs(t, attributionError(map[string]interface{}{"id": "x", "tenant_id": nil}, true),
		ErrTenantUnattributed)

	// A zero fill on an insert or update cannot be replica identity, which only trims
	// the old image -- so it stays the plain unattributed case.
	assert.ErrorIs(t, attributionError(map[string]interface{}{"id": "x", "tenant_id": ""}, false),
		ErrTenantUnattributed)
}

func TestAttributionErrorAcceptsARealTenant(t *testing.T) {
	assert.NoError(t, attributionError(map[string]interface{}{"id": "x", "tenant_id": tenantA}, false))
	assert.NoError(t, attributionError(map[string]interface{}{"id": "x", "tenant_id": tenantB}, true))
	assert.ErrorIs(t, attributionError(nil, true), ErrTenantUnattributed,
		"a nil image carries no tenant either, so it cannot be attributed")
}

// The gate has to carry the distinction all the way to the DLQ record, because the
// DLQ entry is what an operator reads. Asserting only the helper would pass even if
// route() collapsed the two reasons back into one on the way out.
//
// On the counters: tenant_unavailable_on_delete is deliberately a SUBSET of
// tenant_unattributed, not a replacement. Making them exclusive would let the
// replica-identity fault silence an alert that is already wired up to fire on
// unattributed rows -- turning a schema fault into silence, which is the outcome this
// whole subsystem is built to prevent. The DLQ reason code is the exclusive half;
// the counters nest.
func TestRouteReportsTheReplicaIdentityFaultDistinctly(t *testing.T) {
	gk := routingGatekeeper(t, Config{Topic: "t", StarRocksTable: "orm_order", PrimaryKeys: []string{"id"}})

	_, ok := gk.route(context.Background(), zeroFilledDeleteBefore(), kafka.Message{}, true)

	assert.False(t, ok, "a delete that cannot be attributed must not be batched")
	assert.Equal(t, int64(1), gk.metrics.TenantUnavailableOnDelete.Load(),
		"counted under its own metric so the cause is visible without reading the DLQ")
	assert.Equal(t, int64(1), gk.metrics.TenantUnattributed.Load(),
		"and still counted as unattributed, so an existing alert on unattributed rows does not go quiet")
	assert.Equal(t, int64(1), gk.metrics.DLQEmitted.Load())
}

func TestRouteCountsAPlainMissingTenantOnlyOnce(t *testing.T) {
	gk := routingGatekeeper(t, Config{Topic: "t", StarRocksTable: "orm_order", PrimaryKeys: []string{"id"}})

	_, ok := gk.route(context.Background(), map[string]interface{}{"id": "1"}, kafka.Message{}, true)

	assert.False(t, ok)
	assert.Equal(t, int64(1), gk.metrics.TenantUnattributed.Load())
	assert.Equal(t, int64(0), gk.metrics.TenantUnavailableOnDelete.Load(),
		"a genuinely absent tenant is not the replica-identity fault and must not be reported as one")
}
