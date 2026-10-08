package main

// Connector configuration drift: three lists that must agree, and nothing checks
// that they do.
//
// `debezium/orm-oms-connector.json` names the tables Debezium captures. Each of
// those tables then needs three other things to exist, and none of them is derived
// from the first:
//
//   1. a stream-loader service in docker-compose.remote.yml, or the change events
//      for that table have no consumer and vanish at the broker;
//   2. a destination table in migrations/starrocks/002_cdc_orm_tables.sql, or the
//      loader has nowhere to write;
//   3. REPLICA IDENTITY FULL in Postgres, or -- exactly as found on 2026-10-08 --
//      every DELETE for that table dead-letters as ERR_TENANT_UNAVAILABLE_ON_DELETE
//      while inserts and updates sail through.
//
// Add a table to table.include.list and get (1) right but (3) wrong, and the
// pipeline reports success for weeks. Nothing turns red: the connector accepts the
// table, the loader writes rows, and the one operation that cannot be routed is the
// one nobody runs.
//
// This is the three-week-silent-absence class, and it recurs because the lists are
// maintained in four different files by convention rather than by mechanism.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// heartbeatTable is Debezium's own liveness store. It is in table.include.list by
// necessity and is deliberately not a CDC target: nothing consumes its topic, it has
// no StarRocks destination, and it has no tenant_id. Excluding it is correct, not a
// special case being waved through.
const heartbeatTable = "orm.debezium_heartbeat"

type connectorConfig struct {
	Name   string            `json:"name"`
	Config map[string]string `json:"config"`
}

// repoRoot walks up from the test's working directory to the module workspace root,
// identified by go.work. Walking rather than hardcoding a depth keeps this working
// when the package moves.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.work")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	t.Fatal("could not locate repository root (no go.work found walking upwards)")
	return ""
}

func readConnector(t *testing.T) connectorConfig {
	t.Helper()
	path := filepath.Join(repoRoot(t), "debezium", "orm-oms-connector.json")
	b, err := os.ReadFile(path)
	require.NoError(t, err, "the connector config is the source of truth for what is captured")
	var c connectorConfig
	require.NoError(t, json.Unmarshal(b, &c), "connector config must stay parseable")
	return c
}

// capturedTables is everything in table.include.list except the heartbeat store.
func capturedTables(t *testing.T) []string {
	t.Helper()
	list := readConnector(t).Config["table.include.list"]
	require.NotEmpty(t, list, "table.include.list must be set; an empty list captures nothing")

	var out []string
	for _, raw := range strings.Split(list, ",") {
		name := strings.TrimSpace(raw)
		if name == "" || name == heartbeatTable {
			continue
		}
		out = append(out, name)
	}
	require.NotEmpty(t, out, "every captured table was the heartbeat store, which cannot be right")
	return out
}

// destinationTable maps a captured source table to its StarRocks table name:
// `orm.order_allocation` -> `orm_order_allocation`. The first "." -- the schema
// separator -- becomes "_", and underscores inside the name are left alone. The same
// rule the compose services and 002 use, so if that rule ever changes this check is
// the thing that notices.
//
// Note this is the schema separator, not a path separator: filepath.Split is the
// wrong tool here and returns "orm.order" unchanged.
func destinationTable(source string) string {
	return strings.Replace(source, ".", "_", 1)
}

func topicPrefix(t *testing.T) string {
	t.Helper()
	prefix := readConnector(t).Config["topic.prefix"]
	require.NotEmpty(t, prefix, "topic.prefix must be set; topics are derived from it")
	return prefix
}

// 1. Every captured table has a stream-loader service.
//
// The check is on CDC_TOPIC rather than on the service name, because the topic is
// what Debezium actually emits and what a consumer must match on. A service renamed
// for clarity is fine; a table captured with no matching consumer is not.
func TestEveryCapturedTableHasALoaderService(t *testing.T) {
	compose := readRepoFile(t, "docker-compose.remote.yml")
	prefix := topicPrefix(t)

	for _, table := range capturedTables(t) {
		topic := prefix + "." + table
		assert.Contains(t, compose, "CDC_TOPIC: "+topic,
			"%s is captured by the connector but nothing consumes %s", table, topic)
	}

	// And the destination each service writes to must match the derived name, or the
	// loader is pointed at a table that does not exist.
	for _, table := range capturedTables(t) {
		assert.Contains(t, compose, "STARROCKS_TABLE: "+destinationTable(table),
			"the loader for %s must write to %s", table, destinationTable(table))
	}
}

// 1b. No loader service may write to the shared `oms` schema once per-tenant routing
// is enabled. The routing flip sets STREAM_LOAD_TENANT_ROUTES_DIR and clears
// STARROCKS_DB on every loader service in the same commit. If STARROCKS_DB is set to
// "oms" on any loader, the loader either hasn't been flipped or was reverted — and
// because routing is per-service, one un-flipped loader causes split-brain data that
// is not caught by any other check.
//
// The mutation form of this check: flip one service back to STARROCKS_DB: oms, run the
// test, confirm it fails. A permanently passing check is a ritual, not a guard.
func TestNoLoaderWritesToSharedSchema(t *testing.T) {
	compose := readRepoFile(t, "docker-compose.remote.yml")

	// Only the loader services are in scope; other services (provisioning, backfill)
	// may legitimately target oms during transition.
	loaderPrefixes := []string{
		"uisce-stream-loader-execution",
		"uisce-stream-loader-order",
		"uisce-stream-loader-placement",
		"uisce-stream-loader-order-allocation",
		"uisce-stream-loader-execution-allocation",
	}

	for _, svc := range loaderPrefixes {
		// Extract the environment block for this service. If the service is removed
		// entirely, that is also a finding — but a different one (nothing to check).
		idx := strings.Index(compose, svc+":")
		if idx == -1 {
			t.Logf("%s: service removed from compose; skipping", svc)
			continue
		}
		// Find the next top-level key (starts at column 0, followed by ':')
		// to bound the environment block.
		block := compose[idx:]
		if nextIdx := strings.Index(block[1:], "\n[^ ]"); nextIdx != -1 {
			block = block[:nextIdx+1]
		}

		assert.NotContains(t, block, "STARROCKS_DB: oms",
			"%s still points at the shared oms schema — it has not been flipped "+
				"to per-tenant routing, or was reverted. Run the routing flip before "+
				"this test. See AGENTS.md: Testing disciplines.", svc)

		// Routing-enabled loaders must have the routes directory set, otherwise
		// the empty STARROCKS_DB causes an immediate fatal at startup and the
		// tripwire has already caught it — but this assertion makes the requirement
		// explicit in the test contract.
		assert.Contains(t, block, "STREAM_LOAD_TENANT_ROUTES_DIR:",
			"%s has STARROCKS_DB cleared but does not set STREAM_LOAD_TENANT_ROUTES_DIR; "+
				"the loader will fatal at startup with no usable route", svc)
	}
}

// 2. Every captured table has a StarRocks destination in the canonical DDL.
//
// This is the file scripts/provision_starrocks_tenants.sh replays for every tenant
// database, so a table missing here is missing from all N of them, not one.
func TestEveryCapturedTableHasADestinationInCanonicalDDL(t *testing.T) {
	ddl := readRepoFile(t, filepath.Join("migrations", "starrocks", "002_cdc_orm_tables.sql"))

	for _, table := range capturedTables(t) {
		dest := destinationTable(table)
		assert.Contains(t, ddl, "CREATE TABLE IF NOT EXISTS oms."+dest,
			"%s is captured but %s is not created by 002_cdc_orm_tables.sql, so the "+
				"loader has no table to write and provisioning never creates one", table, dest)
	}
}

// 3. Every captured table is covered by REPLICA IDENTITY FULL.
//
// The migration that matters is publication-driven, so the right assertion is not
// that a table name appears in some file -- it is that a migration exists which
// applies FULL to whatever the publication contains. That is the property that keeps
// working when the capture list changes.
func TestCapturedTablesAreCoveredByReplicaIdentityFull(t *testing.T) {
	root := repoRoot(t)

	migrations, err := filepath.Glob(filepath.Join(root, "backend", "db", "migrations",
		"2026*_cdc_replica_identity*.up.sql"))
	require.NoError(t, err)
	require.NotEmpty(t, migrations,
		"no replica identity migration found; without it every DELETE dead-letters under routing")

	var publicationDriven bool
	var targeted string
	for _, m := range migrations {
		body := readRepoFile(t, mustRel(t, root, m))
		if strings.Contains(body, "pg_publication_tables") {
			publicationDriven = true
		}
		// The targeted migration names tables literally, so a table appearing there
		// counts as covered even before the publication-driven one runs.
		for _, table := range capturedTables(t) {
			if strings.Contains(body, "'"+table+"'") || strings.Contains(body, "."+table+" ") {
				targeted += " " + table
			}
		}
	}

	assert.True(t, publicationDriven,
		"no migration derives replica identity from pg_publication_tables, so adding a "+
			"table to table.include.list silently reintroduces unroutable deletes")

	// Sanity-check the target list is non-empty, otherwise the assertion above would
	// pass on a config that captures nothing.
	assert.NotEmpty(t, targeted,
		"no captured table is named by any replica identity migration; the check above "+
			"would pass while every delete still dead-letters")
}

// The connector points at a publication by name. If that publication is recreated
// without the captured tables, or renamed, deletes stop flowing for exactly the
// tables this file cares about -- and nothing in the repo notices, because the
// publication lives in the database.
func TestConnectorNamesAPublicationAndSlot(t *testing.T) {
	cfg := readConnector(t).Config

	assert.NotEmpty(t, cfg["publication.name"],
		"a pgoutput connector with no publication captures nothing")
	assert.NotEmpty(t, cfg["slot.name"],
		"a pgoutput connector with no slot name would create one per restart and orphan WAL")

	// autocreate disabled means the publication is managed outside the connector. That
	// is a deliberate choice -- it stops Debezium silently widening the capture set --
	// but it also means a missing publication fails closed and quietly.
	if cfg["publication.autocreate.mode"] == "disabled" {
		assert.NotEmpty(t, cfg["publication.name"],
			"autocreate is disabled, so a nameless publication can never be created")
	}
}

// decimal.handling.mode must stay "string". Changing it alters how every numeric
// column is encoded in the event, which means the destination columns would receive
// a different representation from the one the loader, the backfill and the audit all
// assume. Nothing would fail loudly -- the columns would simply hold different
// bytes, and the value audit would disagree with itself.
func TestDecimalHandlingStaysString(t *testing.T) {
	cfg := readConnector(t).Config
	assert.Equal(t, "string", cfg["decimal.handling.mode"],
		"the loader, the backfill and cdc_value_audit.sh all encode decimals as "+
			"strings; any other mode changes the wire format for every numeric column")
}

// tombstones.on.delete=false is load-bearing rather than cosmetic: the loader
// classifies tombstones and skips them, and enabling them would add a record type
// per delete that carries no row at all.
func TestTombstonesRemainDisabled(t *testing.T) {
	cfg := readConnector(t).Config
	assert.Equal(t, "false", cfg["tombstones.on.delete"],
		"tombstones carry no row; the loader classifies and skips them, and enabling "+
			"them changes the event stream for a reason nothing here depends on")
}

func readRepoFile(t *testing.T, rel string) string {
	t.Helper()
	root := repoRoot(t)
	b, err := os.ReadFile(filepath.Join(root, rel))
	require.NoError(t, err, "reading %s", rel)
	return string(b)
}

func mustRel(t *testing.T, root, abs string) string {
	t.Helper()
	rel, err := filepath.Rel(root, abs)
	require.NoError(t, err)
	return rel
}
