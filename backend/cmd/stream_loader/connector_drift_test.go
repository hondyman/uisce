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
	"regexp"
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

// loaderServiceTables is the inverse input: every table that has a running stream
// loader, derived from CDC_TOPIC in compose rather than from table.include.list.
func loaderServiceTables(t *testing.T) []string {
	t.Helper()
	compose := readRepoFile(t, "docker-compose.remote.yml")
	prefix := topicPrefix(t)

	re := regexp.MustCompile(`CDC_TOPIC:\s*` + regexp.QuoteMeta(prefix) + `\.(\S+)`)
	matches := re.FindAllStringSubmatch(compose, -1)
	require.NotEmpty(t, matches, "no CDC_TOPIC entries found for prefix %q", prefix)

	var out []string
	for _, m := range matches {
		out = append(out, strings.TrimSpace(m[1]))
	}
	return out
}

// The captured -> consumed direction is covered above. This is the other one, and it
// is the shape of the original three-week bug: a table configured *everywhere except*
// the capture list.
//
// `order` had a loader service, a destination in 002 and a REPLICA IDENTITY
// migration, and was absent from table.include.list. Every forward check passed. The
// loader consumed a topic nothing produced, so the table simply never received a row
// -- and because the loader is silent when idle rather than loud when starved, that
// looked exactly like "no orders today".
//
// So: anything wired to a loader must also be captured. The capture list is the only
// entry point to the pipeline, and a consumer pointed at a topic that is never
// produced is a pipeline that will not say so.
func TestEveryLoaderServiceTableIsActuallyCaptured(t *testing.T) {
	captured := map[string]bool{}
	for _, table := range capturedTables(t) {
		captured[table] = true
	}

	for _, table := range loaderServiceTables(t) {
		if table == heartbeatTable {
			continue
		}
		assert.True(t, captured[table],
			"%s has a stream-loader service consuming %s.%s, but it is missing from "+
				"table.include.list -- the loader is watching a topic nothing produces, so "+
				"the table silently never receives a row",
			table, topicPrefix(t), table)
	}
}

// The same inverse question for destinations: a table created in the canonical DDL
// that nothing captures is provisioned into every tenant database and never written
// to. Under provisioning that is pure cost paid N times, and it is also a second
// surface that looks like a live table.
func TestEveryDestinationTableIsCapturedOrDeliberatelyExempt(t *testing.T) {
	ddl := readRepoFile(t, filepath.Join("migrations", "starrocks", "002_cdc_orm_tables.sql"))

	re := regexp.MustCompile(`CREATE TABLE IF NOT EXISTS oms\.([a-z_]+)`)
	dests := re.FindAllStringSubmatch(ddl, -1)
	require.NotEmpty(t, dests, "no destination tables parsed from the canonical DDL")

	captured := map[string]bool{}
	for _, table := range capturedTables(t) {
		captured[destinationTable(table)] = true
	}

	// Tables the DDL creates for reasons other than CDC may legitimately exist
	// without being captured. Today there are none; the exemption is an explicit,
	// reviewable list rather than a comment, so adding one is a deliberate act.
	exempt := map[string]string{}

	for _, m := range dests {
		dest := m[1]
		if captured[dest] {
			continue
		}
		if reason, ok := exempt[dest]; ok {
			t.Logf("destination %s is not captured (%s)", dest, reason)
			continue
		}
		assert.Fail(t, "uncaptured destination table",
			"oms.%s is created by 002_cdc_orm_tables.sql and therefore provisioned into "+
				"every tenant database, but no captured table maps to it. Either capture the "+
				"source table or add %q to the exempt map with the reason.", dest, dest)
	}
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
