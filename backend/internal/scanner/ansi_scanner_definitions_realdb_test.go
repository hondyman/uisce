package scanner

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/url"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/models"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
)

// Against a real PostgreSQL: every class of definition is recorded, exactly as the server deparses it, and nothing
// that belongs to something else is. SCANNER_TEST_ADMIN_DSN is a postgres:// URL with CREATEDB; the test creates and
// drops its own database.
func TestProcessDefinitions_AgainstARealServer(t *testing.T) {
	adminDSN := os.Getenv("SCANNER_TEST_ADMIN_DSN")
	if adminDSN == "" {
		t.Skip("SCANNER_TEST_ADMIN_DSN not set")
	}
	ctx := context.Background()
	admin, err := sql.Open("pgx", adminDSN)
	require.NoError(t, err)
	name := "scanner_defs_test"
	_, _ = admin.ExecContext(ctx, `DROP DATABASE IF EXISTS `+name)
	_, err = admin.ExecContext(ctx, `CREATE DATABASE `+name)
	require.NoError(t, err)
	// Cleanups run last-in first-out, so the connection to the new database (registered later) closes first.
	t.Cleanup(func() {
		_, _ = admin.Exec(`DROP DATABASE IF EXISTS ` + name)
		admin.Close()
	})
	u, err := url.Parse(adminDSN)
	require.NoError(t, err)
	u.Path = "/" + name
	db, err := sql.Open("pgx", u.String())
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, `
		CREATE SCHEMA scan_defs;
		CREATE EXTENSION pg_trgm WITH SCHEMA scan_defs;
		CREATE FUNCTION scan_defs.touch() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN NEW.note := 'x'; RETURN NEW; END $$;
		CREATE FUNCTION scan_defs.double(i int) RETURNS int LANGUAGE sql IMMUTABLE AS $$ SELECT i * 2 $$;
		CREATE TABLE scan_defs.quote (
			id uuid NOT NULL, quote_time timestamptz NOT NULL, bid numeric(18,9), note text,
			CONSTRAINT quote_pkey PRIMARY KEY (id, quote_time),
			CONSTRAINT chk_quote_bid CHECK (bid >= 0),
			CONSTRAINT quote_note_uq UNIQUE (id, quote_time, note)
		) PARTITION BY RANGE (quote_time);
		CREATE TABLE scan_defs.quote_default PARTITION OF scan_defs.quote DEFAULT;
		CREATE TABLE scan_defs.quote_2026 PARTITION OF scan_defs.quote FOR VALUES FROM ('2026-01-01') TO ('2027-01-01');
		CREATE INDEX idx_quote_time ON scan_defs.quote (quote_time DESC);
		CREATE TRIGGER trg_quote_touch BEFORE INSERT ON scan_defs.quote FOR EACH ROW EXECUTE FUNCTION scan_defs.touch();
		CREATE TABLE scan_defs.plain (id uuid PRIMARY KEY, label text, tags text[], codes character varying(4)[], at time(3), UNIQUE (label));
		CREATE INDEX idx_plain_lower ON scan_defs.plain (lower(label));
		CREATE INDEX idx_plain_trgm ON scan_defs.plain USING gin (label scan_defs.gin_trgm_ops);
		CREATE UNIQUE INDEX uq_plain_partial ON scan_defs.plain (label) WHERE label IS NOT NULL;
		CREATE TABLE scan_defs.bare (id uuid PRIMARY KEY);`)
	require.NoError(t, err)

	s, err := NewAnsiScanner(db, uuid.New(), uuid.New(), "src", nil, true, []string{"scan_defs"})
	require.NoError(t, err)
	nodes, _, err := s.ExtractMetadata()
	require.NoError(t, err)
	by := map[string]*models.CatalogNode{}
	for _, n := range nodes {
		by[n.QualifiedPath] = n
	}
	p := func(path string) map[string]interface{} {
		require.Contains(t, by, path)
		var m map[string]interface{}
		require.NoError(t, json.Unmarshal(by[path].Properties, &m))
		return m
	}
	names := func(v interface{}) []string {
		var out []string
		for _, e := range v.([]interface{}) {
			out = append(out, e.(map[string]interface{})["name"].(string))
		}
		return out
	}

	q := p("/scan_defs/quote")
	require.Equal(t, []interface{}{
		map[string]interface{}{"name": "chk_quote_bid", "type": "c", "definition": "CHECK ((bid >= (0)::numeric))"},
		map[string]interface{}{"name": "quote_note_uq", "type": "u", "definition": "UNIQUE (id, quote_time, note)"},
		map[string]interface{}{"name": "quote_pkey", "type": "p", "definition": "PRIMARY KEY (id, quote_time)"},
	}, q["constraints"], "primary, unique and check, in the server's words, with the key columns in key order")
	require.Equal(t, []string{"idx_quote_time"}, names(q["indexes"]), "not the primary key, not the unique constraint")
	require.Equal(t, map[string]interface{}{"key": "RANGE (quote_time)"}, q["partition"])
	require.Equal(t, []string{"trg_quote_touch"}, names(q["triggers"]))

	// A partition records what ties it to its parent, and neither the inherited check, nor the cloned trigger, nor the index copy.
	for path, bound := range map[string]string{
		"/scan_defs/quote_default": "DEFAULT",
		"/scan_defs/quote_2026":    "FOR VALUES FROM ('2026-01-01 00:00:00+00') TO ('2027-01-01 00:00:00+00')",
	} {
		c := p(path)
		require.Equal(t, "scan_defs.quote", c["partition"].(map[string]interface{})["parent"], path)
		require.Contains(t, c["partition"].(map[string]interface{})["bound"], bound[:7], path)
		require.Equal(t, []interface{}{}, c["constraints"], path+": inherited from the parent, so none of its own")
		require.Equal(t, []interface{}{}, c["triggers"], path+": a clone of the parent's trigger, so none of its own")
		require.Equal(t, []interface{}{}, c["indexes"], path+": a copy of the parent's index, so none of its own")
	}

	pl := p("/scan_defs/plain")
	require.ElementsMatch(t, []string{"idx_plain_lower", "idx_plain_trgm", "uq_plain_partial"}, names(pl["indexes"]),
		"expression, gin and partial unique indexes are recorded; the unique CONSTRAINT's index is not")
	for _, e := range pl["indexes"].([]interface{}) {
		m := e.(map[string]interface{})
		if m["name"] == "uq_plain_partial" {
			require.Equal(t, true, m["unique"])
			require.Contains(t, m["definition"], "WHERE")
		}
		if m["name"] == "idx_plain_trgm" {
			require.Equal(t, "gin", m["method"])
		}
	}
	require.Equal(t, []interface{}{}, p("/scan_defs/bare")["indexes"])

	// exact types, and the key as the server has it
	require.Equal(t, "numeric(18,9)", p("/scan_defs/quote/bid")["format_type"])
	require.Equal(t, "timestamp with time zone", p("/scan_defs/quote/quote_time")["format_type"])
	require.Equal(t, "text[]", p("/scan_defs/plain/tags")["format_type"], "information_schema says ARRAY and loses the element type")
	require.Equal(t, "character varying(4)[]", p("/scan_defs/plain/codes")["format_type"])
	require.Equal(t, "time(3) without time zone", p("/scan_defs/plain/at")["format_type"], "precision information_schema does not give for time")
	for _, c := range []string{"/scan_defs/quote_default", "/scan_defs/quote_2026"} {
		require.Equal(t, []interface{}{}, p(c)["constraints"], c+": the primary key, unique and check are the parent's")
	}
	for _, path := range []string{"/scan_defs/quote", "/scan_defs/plain"} {
		require.Nil(t, p(path)["persistence"], path)
		require.Nil(t, p(path)["options"], path)
	}

	// Every node the scan wrote carries this scan's id, and every structural key is present.
	scanID := s.scanID.String()
	for _, n := range nodes {
		require.Equal(t, scanID, func() string {
			var m map[string]interface{}
			_ = json.Unmarshal(n.Properties, &m)
			v, _ := m["scan_id"].(string)
			return v
		}(), n.QualifiedPath)
	}
	for _, path := range []string{"/scan_defs/bare", "/scan_defs/plain", "/scan_defs/quote"} {
		for _, k := range []string{"constraints", "indexes", "triggers", "partition", "persistence", "options"} {
			require.Contains(t, p(path), k, "%s must always carry %s", path, k)
		}
	}
	for _, k := range []string{"default_value", "column_comment", "max_length", "precision", "scale", "generated", "identity", "collation"} {
		require.Contains(t, p("/scan_defs/bare/id"), k, "a column always carries %s, null when it has none", k)
	}

	sc := p("/scan_defs")
	require.Equal(t, true, sc["definitions_captured"])
	var rn []string
	for _, r := range sc["routines"].([]interface{}) {
		rn = append(rn, r.(map[string]interface{})["name"].(string))
	}
	require.ElementsMatch(t, []string{"touch", "double"}, rn, "the extension's own functions (pg_trgm) are not the schema's")
}

// The merge into alpha is `target.properties || source.properties`, which keeps every key the new scan does not mention.
// This is why the scan always writes the structural keys: against PostgreSQL itself, an index that is gone from the source
// is overwritten by an empty list, and a default that is gone by a null; a missing key would leave both in alpha.
func TestMergeSemantics_ExplicitEmptiesOverwriteWhatWasDropped(t *testing.T) {
	adminDSN := os.Getenv("SCANNER_TEST_ADMIN_DSN")
	if adminDSN == "" {
		t.Skip("SCANNER_TEST_ADMIN_DSN not set")
	}
	db, err := sql.Open("pgx", adminDSN)
	require.NoError(t, err)
	defer db.Close()
	old := `{"indexes":[{"name":"ix"}],"constraints":[{"name":"chk"}],"default_value":"now()","max_length":10,"title":"Kept"}`
	t.Run("a scan that omits the keys leaves the stale values (the hazard)", func(t *testing.T) {
		var got string
		require.NoError(t, db.QueryRow(`SELECT ($1::jsonb || '{"data_type":"text"}'::jsonb)::text`, old).Scan(&got))
		require.Contains(t, got, `"indexes"`)
		require.Contains(t, got, `"default_value"`)
	})
	t.Run("a scan that writes them overwrites them and keeps what a person added", func(t *testing.T) {
		fresh := `{"indexes":[],"constraints":[],"default_value":null,"max_length":null,"data_type":"text"}`
		var idx, cons string
		var def, max sql.NullString
		var title string
		require.NoError(t, db.QueryRow(`SELECT (m->'indexes')::text, (m->'constraints')::text, m->>'default_value', m->>'max_length', m->>'title'
			FROM (SELECT $1::jsonb || $2::jsonb AS m) x`, old, fresh).Scan(&idx, &cons, &def, &max, &title))
		require.Equal(t, "[]", idx)
		require.Equal(t, "[]", cons)
		require.False(t, def.Valid, "a dropped default is null, not still now()")
		require.False(t, max.Valid)
		require.Equal(t, "Kept", title, "a key only a person set (a title, a mapping) is untouched")
	})
}

// A structure-only scan records the same structure and reads no data.
func TestSkipDataProfile_KeepsTheStructureAndReadsNoData(t *testing.T) {
	adminDSN := os.Getenv("SCANNER_TEST_ADMIN_DSN")
	if adminDSN == "" {
		t.Skip("SCANNER_TEST_ADMIN_DSN not set")
	}
	ctx := context.Background()
	admin, err := sql.Open("pgx", adminDSN)
	require.NoError(t, err)
	name := "scanner_skip_test"
	_, _ = admin.ExecContext(ctx, `DROP DATABASE IF EXISTS `+name)
	_, err = admin.ExecContext(ctx, `CREATE DATABASE `+name)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = admin.Exec(`DROP DATABASE IF EXISTS ` + name)
		admin.Close()
	})
	u, err := url.Parse(adminDSN)
	require.NoError(t, err)
	u.Path = "/" + name
	open := func() *sql.DB {
		d, err := sql.Open("pgx", u.String())
		require.NoError(t, err)
		return d
	}
	seed := open()
	_, err = seed.ExecContext(ctx, `CREATE SCHEMA sk; CREATE TABLE sk.t (id uuid PRIMARY KEY, label text);
		INSERT INTO sk.t SELECT gen_random_uuid(), 'v' || (i % 3) FROM generate_series(1, 30) i;`)
	require.NoError(t, err)
	seed.Close()

	scan := func(skip bool) map[string]interface{} {
		s, err := NewAnsiScanner(open(), uuid.New(), uuid.New(), "src", nil, true, []string{"sk"})
		require.NoError(t, err)
		if skip {
			s.SkipDataProfile()
		}
		nodes, _, err := s.ExtractMetadata()
		require.NoError(t, err)
		for _, n := range nodes {
			if n.QualifiedPath == "/sk/t/label" {
				var m map[string]interface{}
				require.NoError(t, json.Unmarshal(n.Properties, &m))
				return m
			}
		}
		t.Fatal("column node not found")
		return nil
	}
	full, lean := scan(false), scan(true)
	require.Contains(t, full, "sample_values", "a normal scan profiles the data")
	require.NotContains(t, lean, "sample_values", "a structure-only scan reads none")
	for _, k := range []string{"format_type", "data_type", "is_nullable", "scan_id"} {
		require.Equal(t, full[k] != nil, lean[k] != nil, k)
		require.NotNil(t, lean[k], k)
	}
}
