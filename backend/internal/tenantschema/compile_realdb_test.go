package tenantschema

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/scanner"
	"github.com/hondyman/uisce/backend/models"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
)

// End to end, against a real server: scan a source database, compile a plan from the SCAN ALONE, apply it to an empty
// database, and compare the two structures object by object. The source is never read by the compiler; what it did not
// record is not in the target, and the comparison below is what notices.
//
// TENANTSCHEMA_TEST_ADMIN_DSN is a postgres:// URL with CREATEDB (the test creates and drops its own databases).
// Optionally TENANTSCHEMA_TEST_DDL_DIR (a directory of *.up.sql applied in name order) and TENANTSCHEMA_TEST_SCHEMAS
// (comma separated, in template order) replace the built-in fixture, to run the same proof on a real template.

const fixtureDDL = `
CREATE EXTENSION IF NOT EXISTS "uuid-ossp" WITH SCHEMA public;
CREATE SCHEMA ref_x;
CREATE SCHEMA core_x;
CREATE FUNCTION core_x.touch() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN NEW.note := 'x'; RETURN NEW; END $$;
CREATE FUNCTION core_x.double(i int) RETURNS int LANGUAGE sql IMMUTABLE AS $$ SELECT i * 2 $$;
CREATE TABLE ref_x.currency (code character(3) PRIMARY KEY, name varchar(60) NOT NULL, owner_account uuid);
CREATE TABLE core_x."order" (
	id uuid DEFAULT public.uuid_generate_v4() NOT NULL, tenant_id uuid NOT NULL, ccy character(3),
	qty numeric(18,4) DEFAULT 0 NOT NULL, tags text[], codes character varying(4)[], placed time(3), meta jsonb DEFAULT '{}'::jsonb NOT NULL,
	CONSTRAINT order_pkey PRIMARY KEY (id),
	CONSTRAINT order_tenant_ccy_uq UNIQUE (tenant_id, ccy),
	CONSTRAINT chk_order_qty CHECK (qty >= 0),
	CONSTRAINT fk_order_ccy FOREIGN KEY (ccy) REFERENCES ref_x.currency(code) ON DELETE SET NULL ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED
);
CREATE INDEX idx_order_meta ON core_x."order" USING gin (meta);
CREATE INDEX idx_order_expr ON core_x."order" (lower(ccy)) WHERE qty > 0;
CREATE TRIGGER trg_order_touch BEFORE INSERT ON core_x."order" FOR EACH ROW EXECUTE FUNCTION core_x.touch();
CREATE TABLE core_x.account (id uuid PRIMARY KEY, name text NOT NULL);
ALTER TABLE ref_x.currency ADD CONSTRAINT fk_currency_owner FOREIGN KEY (owner_account) REFERENCES core_x.account(id) ON DELETE CASCADE;
CREATE TABLE core_x.quote (
	id uuid NOT NULL, quote_time timestamptz NOT NULL, bid numeric(18,9), order_id uuid,
	CONSTRAINT quote_pkey PRIMARY KEY (id, quote_time),
	CONSTRAINT chk_quote_bid CHECK (bid >= 0),
	CONSTRAINT fk_quote_order FOREIGN KEY (order_id) REFERENCES core_x."order"(id) ON DELETE CASCADE
) PARTITION BY RANGE (quote_time);
CREATE TABLE core_x.quote_default PARTITION OF core_x.quote DEFAULT;
CREATE TABLE core_x.quote_2026 PARTITION OF core_x.quote FOR VALUES FROM ('2026-01-01') TO ('2027-01-01');
CREATE INDEX idx_quote_time ON core_x.quote (quote_time DESC);
CREATE TABLE core_x.bare (id uuid PRIMARY KEY);
`

func openAdmin(t *testing.T) (*sql.DB, *url.URL) {
	t.Helper()
	dsn := os.Getenv("TENANTSCHEMA_TEST_ADMIN_DSN")
	if dsn == "" {
		t.Skip("TENANTSCHEMA_TEST_ADMIN_DSN not set")
	}
	u, err := url.Parse(dsn)
	require.NoError(t, err, "TENANTSCHEMA_TEST_ADMIN_DSN must be a postgres:// URL")
	admin, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { admin.Close() })
	return admin, u
}

func freshDB(t *testing.T, admin *sql.DB, u *url.URL, name string) *sql.DB {
	t.Helper()
	_, _ = admin.Exec(`DROP DATABASE IF EXISTS ` + name)
	_, err := admin.Exec(`CREATE DATABASE ` + name)
	require.NoError(t, err)
	// Registered before the connection to the new database, so cleanups (last in, first out) close that first.
	t.Cleanup(func() { _, _ = admin.Exec(`DROP DATABASE IF EXISTS ` + name) })
	v := *u
	v.Path = "/" + name
	db, err := sql.Open("pgx", v.String())
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

func sourceAndSchemas(t *testing.T, db *sql.DB) []string {
	t.Helper()
	ctx := context.Background()
	if dir := os.Getenv("TENANTSCHEMA_TEST_DDL_DIR"); dir != "" {
		files, err := os.ReadDir(dir)
		require.NoError(t, err)
		for _, f := range files {
			if strings.HasSuffix(f.Name(), ".up.sql") {
				b, err := os.ReadFile(dir + "/" + f.Name())
				require.NoError(t, err)
				_, err = db.ExecContext(ctx, string(b))
				require.NoError(t, err, f.Name())
			}
		}
		return strings.Split(os.Getenv("TENANTSCHEMA_TEST_SCHEMAS"), ",")
	}
	_, err := db.ExecContext(ctx, fixtureDDL)
	require.NoError(t, err)
	return []string{"ref_x", "core_x"}
}

// fingerprint describes a structure in terms that must be identical between source and target. Names the server
// invents for a partition's copy of something are normalised, because they are not part of the design.
func fingerprint(t *testing.T, db *sql.DB, schemas []string) map[string][]string {
	t.Helper()
	in := "('" + strings.Join(schemas, "','") + "')"
	nsoid := "(select oid from pg_namespace where nspname in " + in + ")"
	q := map[string]string{
		"tables": `select n.nspname||'.'||c.relname||':'||c.relkind::text||':'||c.relpersistence::text||':'||coalesce(array_to_string(c.reloptions,','),'')
		             from pg_class c join pg_namespace n on n.oid=c.relnamespace where c.relkind in ('r','p') and n.nspname in ` + in,
		"columns": `select n.nspname||'.'||c.relname||'.'||a.attname||':'||a.attnum||':'||format_type(a.atttypid,a.atttypmod)||':'||a.attnotnull::text||':'||coalesce(pg_get_expr(d.adbin,d.adrelid),'')||':'||a.attgenerated::text||a.attidentity::text
		             from pg_attribute a join pg_class c on c.oid=a.attrelid join pg_namespace n on n.oid=c.relnamespace left join pg_attrdef d on d.adrelid=a.attrelid and d.adnum=a.attnum
		            where a.attnum>0 and not a.attisdropped and c.relkind in ('r','p') and n.nspname in ` + in,
		// keys, checks and foreign keys on ordinary tables and parents keep their names; a partition's clone is skipped
		"constraints": `select c.relnamespace::regnamespace::text||'.'||c.relname||':'||k.conname||':'||k.contype::text||':'||pg_get_constraintdef(k.oid)
		             from pg_constraint k join pg_class c on c.oid=k.conrelid where k.contype in ('p','u','c','f','x') and k.conislocal and k.conparentid=0 and c.relnamespace in ` + nsoid,
		// every index, by definition with its name removed and ON ONLY removed, so a partition's own copy is compared too
		"indexes": `select regexp_replace(regexp_replace(indexdef,'INDEX \S+ ON','INDEX ON'),' ON ONLY ',' ON ') from pg_indexes where schemaname in ` + in,
		"partitions": `select n.nspname||'.'||c.relname||':'||coalesce(pg_get_partkeydef(c.oid),'')||':'||coalesce(pg_get_expr(c.relpartbound,c.oid),'')||':'||coalesce((select h.inhparent::regclass::text from pg_inherits h where h.inhrelid=c.oid),'')
		             from pg_class c join pg_namespace n on n.oid=c.relnamespace where c.relkind in ('r','p') and (c.relkind='p' or c.relispartition) and n.nspname in ` + in,
		"routines": `select p.oid::regprocedure::text||':'||md5(pg_get_functiondef(p.oid)) from pg_proc p where p.prokind in ('f','p') and p.pronamespace in ` + nsoid +
			` and not exists (select 1 from pg_depend d where d.objid=p.oid and d.classid='pg_proc'::regclass and d.deptype='e')`,
		"triggers": `select c.relnamespace::regnamespace::text||'.'||c.relname||':'||t.tgname||':'||pg_get_triggerdef(t.oid) from pg_trigger t join pg_class c on c.oid=t.tgrelid
		            where not t.tgisinternal and t.tgparentid=0 and c.relnamespace in ` + nsoid,
	}
	out := map[string][]string{}
	for k, sqlText := range q {
		rows, err := db.Query(sqlText)
		require.NoError(t, err, k)
		var vals []string
		for rows.Next() {
			var s string
			require.NoError(t, rows.Scan(&s))
			vals = append(vals, s)
		}
		require.NoError(t, rows.Err())
		rows.Close()
		sortStrings(vals)
		out[k] = vals
	}
	return out
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

func scanOf(t *testing.T, db *sql.DB, schemas []string) []*models.CatalogNode {
	t.Helper()
	s, err := scanner.NewAnsiScanner(db, uuid.New(), uuid.New(), "src", nil, true, schemas)
	require.NoError(t, err)
	s.SkipDataProfile() // the structure is the same without reading every column's data, which is what makes a large source slow
	nodes, _, err := s.ExtractMetadata() // closes db
	require.NoError(t, err)
	return nodes
}

func TestCompile_TheTargetEqualsTheSource_FromTheScanAlone(t *testing.T) {
	admin, u := openAdmin(t)
	src := freshDB(t, admin, u, "tenantschema_src")
	schemas := sourceAndSchemas(t, src)
	want := fingerprint(t, src, schemas)
	require.NotEmpty(t, want["tables"])
	nodes := scanOf(t, src, schemas) // the compiler gets nothing but these

	// What a real alpha holds after rescans: nodes of an older scan for things that have since vanished from the source.
	// A column that is gone, and a whole table that is gone, both still "active". They must not reach the target.
	first := schemas[0]
	var tbl string
	for _, n := range nodes {
		if n.NodeTypeID == scanner.NODE_TYPE_TABLE {
			var m map[string]interface{}
			require.NoError(t, json.Unmarshal(n.Properties, &m))
			if m["schema"] == first {
				tbl = n.NodeName
				break
			}
		}
	}
	require.NotEmpty(t, tbl)
	stale := func(typ uuid.UUID, name, path string, p map[string]interface{}) {
		p["scan_id"] = "an-older-scan"
		b, _ := json.Marshal(p)
		nodes = append(nodes, &models.CatalogNode{NodeTypeID: typ, NodeName: name, QualifiedPath: path, Properties: b})
	}
	stale(scanner.NODE_TYPE_COLUMN, "vanished", "/"+first+"/"+tbl+"/vanished", map[string]interface{}{"is_physical_column": true, "format_type": "text", "ordinal_position": 99, "is_nullable": true})
	stale(scanner.NODE_TYPE_TABLE, "ghost_table", "/"+first+"/ghost_table", map[string]interface{}{"schema": first})
	stale(scanner.NODE_TYPE_COLUMN, "id", "/"+first+"/ghost_table/id", map[string]interface{}{"is_physical_column": true, "format_type": "uuid", "ordinal_position": 1})

	plan, err := Compile(nodes, Options{Schemas: schemas})
	require.NoError(t, err)
	require.Equal(t, len(want["tables"]), plan.Tables)

	if f := os.Getenv("TENANTSCHEMA_TEST_DUMP_SQL"); f != "" {
		require.NoError(t, os.WriteFile(f, []byte(plan.SQL()), 0o600))
	}
	dst := freshDB(t, admin, u, "tenantschema_dst")
	tx, err := dst.Begin()
	require.NoError(t, err)
	_, err = tx.Exec(plan.SQL())
	require.NoError(t, err, "the plan applies to an empty database in one transaction")
	require.NoError(t, tx.Commit())

	got := fingerprint(t, dst, schemas)
	for k := range want {
		require.Equal(t, want[k], got[k], "%s differ between the source and the database built from its scan", k)
	}

	// Deterministic: the same nodes in any order give the same plan.
	shuffled := append([]*models.CatalogNode(nil), nodes...)
	rand.New(rand.NewSource(7)).Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
	again, err := Compile(shuffled, Options{Schemas: schemas})
	require.NoError(t, err)
	require.Equal(t, plan.Hash(), again.Hash(), "node order must not change the plan")
	fmt.Printf("compiled %d tables, %d statements, hash %s\n", plan.Tables, len(plan.Statements), plan.Hash()[:12])
}
