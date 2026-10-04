package tenantschema

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/hondyman/uisce/backend/internal/scanner"
	"github.com/hondyman/uisce/backend/models"
	"github.com/stretchr/testify/require"
)

type fixture struct{ nodes []*models.CatalogNode }

func js(v interface{}) json.RawMessage { b, _ := json.Marshal(v); return b }

func (f *fixture) schema(name string, extra map[string]interface{}) {
	p := map[string]interface{}{"definitions_captured": true, "definitions_version": scanner.DefinitionsVersion}
	for k, v := range extra {
		p[k] = v
	}
	f.nodes = append(f.nodes, &models.CatalogNode{NodeTypeID: scanner.NODE_TYPE_SCHEMA, NodeName: name, QualifiedPath: "/" + name, Properties: js(p)})
}

func (f *fixture) table(schema, name string, extra map[string]interface{}) {
	p := map[string]interface{}{"schema": schema}
	for k, v := range extra {
		p[k] = v
	}
	f.nodes = append(f.nodes, &models.CatalogNode{NodeTypeID: scanner.NODE_TYPE_TABLE, NodeName: name, QualifiedPath: "/" + schema + "/" + name, Properties: js(p)})
}

func (f *fixture) col(schema, table, name, ft string, ord int, extra map[string]interface{}) {
	p := map[string]interface{}{"is_physical_column": true, "format_type": ft, "ordinal_position": ord, "is_nullable": false}
	for k, v := range extra {
		p[k] = v
	}
	f.nodes = append(f.nodes, &models.CatalogNode{NodeTypeID: scanner.NODE_TYPE_COLUMN, NodeName: name, QualifiedPath: "/" + schema + "/" + table + "/" + name, Properties: js(p)})
}

func good() *fixture {
	f := &fixture{}
	f.schema("a", map[string]interface{}{
		"extensions": []interface{}{map[string]interface{}{"name": "uuid-ossp", "version": "1.1", "schema": "public"}, map[string]interface{}{"name": "vector", "version": "0.8", "schema": "public"}},
		"routines":   []interface{}{map[string]interface{}{"name": "f", "arguments": "", "definition": "CREATE OR REPLACE FUNCTION a.f() RETURNS trigger LANGUAGE plpgsql AS $$ begin return new; end $$"}},
	})
	f.schema("b", nil)
	f.table("a", "order", map[string]interface{}{
		"constraints": []interface{}{
			map[string]interface{}{"name": "order_pkey", "type": "p", "definition": "PRIMARY KEY (id)"},
			map[string]interface{}{"name": "fk_o", "type": "f", "definition": "FOREIGN KEY (x) REFERENCES b.t(id) ON DELETE CASCADE"},
			map[string]interface{}{"name": "chk", "type": "c", "definition": "CHECK ((x > 0))"},
		},
		"indexes":  []interface{}{map[string]interface{}{"name": "ix", "definition": "CREATE INDEX ix ON a.\"order\" USING btree (x)"}},
		"triggers": []interface{}{map[string]interface{}{"name": "trg", "definition": "CREATE TRIGGER trg BEFORE INSERT ON a.\"order\" FOR EACH ROW EXECUTE FUNCTION a.f()"}},
	})
	f.col("a", "order", "id", "uuid", 1, map[string]interface{}{"default_value": "gen_random_uuid()"})
	f.col("a", "order", "x", "integer", 2, map[string]interface{}{"is_nullable": true})
	f.table("b", "t", map[string]interface{}{"constraints": []interface{}{map[string]interface{}{"name": "t_pkey", "type": "p", "definition": "PRIMARY KEY (id)"}}})
	f.col("b", "t", "id", "uuid", 1, nil)
	return f
}

func compile(t *testing.T, f *fixture) (*Plan, error) {
	t.Helper()
	return Compile(f.nodes, Options{Schemas: []string{"a", "b"}})
}

func TestCompile_BuildsTheStructureInDependencyOrder(t *testing.T) {
	p, err := compile(t, good())
	require.NoError(t, err)
	var phases []string
	for _, s := range p.Statements {
		if len(phases) == 0 || phases[len(phases)-1] != s.Phase {
			phases = append(phases, s.Phase)
		}
	}
	require.Equal(t, []string{"prelude", "extension", "schema", "routine", "table", "key", "check", "foreign key", "index", "trigger", "epilogue"}, phases,
		"keys before the foreign keys that need them, foreign keys after every table exists, triggers after the routines they call")
	sql := p.SQL()
	require.Contains(t, sql, `CREATE TABLE "a"."order" (`+"\n"+`    "id" uuid DEFAULT gen_random_uuid() NOT NULL,`+"\n"+`    "x" integer`+"\n"+`);`, "identifiers are quoted, the order of columns is the scan's, a default and NOT NULL are kept")
	require.Contains(t, sql, `ALTER TABLE "a"."order" ADD CONSTRAINT "fk_o" FOREIGN KEY (x) REFERENCES b.t(id) ON DELETE CASCADE;`, "the server's own words, verbatim")
	require.Equal(t, 2, p.Tables)
	require.Equal(t, p.Hash(), p.Hash())
	require.Len(t, p.Hash(), 64)
}

func TestCompile_OnlyAllowListedExtensionsAreCreated(t *testing.T) {
	p, err := compile(t, good())
	require.NoError(t, err)
	require.Contains(t, p.SQL(), `CREATE EXTENSION IF NOT EXISTS "uuid-ossp" WITH SCHEMA "public";`)
	require.NotContains(t, p.SQL(), "vector", "an extension that is not on the allow-list is not created by a tenant's structure")
	none, err := Compile(good().nodes, Options{Schemas: []string{"a", "b"}, Extensions: []string{}})
	require.NoError(t, err)
	require.NotContains(t, none.SQL(), "CREATE EXTENSION", "an explicit empty list means none")
}

func TestCompile_RefusesAScanThatDidNotRecordEverything(t *testing.T) {
	t.Run("definitions not captured", func(t *testing.T) {
		f := good()
		f.nodes[0].Properties = js(map[string]interface{}{"definitions_captured": false, "definitions_error": "permission denied"})
		_, err := compile(t, f)
		require.ErrorIs(t, err, ErrScanNotComplete)
		require.ErrorContains(t, err, "a")
	})
	t.Run("an older shape", func(t *testing.T) {
		f := good()
		f.nodes[1].Properties = js(map[string]interface{}{"definitions_captured": true, "definitions_version": 1})
		_, err := compile(t, f)
		require.ErrorIs(t, err, ErrScanNotComplete)
	})
	t.Run("a schema that was never scanned", func(t *testing.T) {
		_, err := Compile(good().nodes, Options{Schemas: []string{"a", "b", "ghost"}})
		require.ErrorIs(t, err, ErrScanNotComplete)
		require.ErrorContains(t, err, "ghost (not in the scan)")
	})
	t.Run("a column with no recorded type", func(t *testing.T) {
		f := good()
		f.col("a", "order", "y", "", 3, nil)
		_, err := compile(t, f)
		require.ErrorIs(t, err, ErrScanNotComplete)
		require.ErrorContains(t, err, "/a/order/y")
	})
}

func TestCompile_RefusesWhatItCannotReproduce(t *testing.T) {
	for name, tc := range map[string]struct {
		mutate func(*fixture)
		want   string
	}{
		"generated column":      {func(f *fixture) { f.col("a", "order", "g", "integer", 3, map[string]interface{}{"generated": "s"}) }, "generated"},
		"identity column":       {func(f *fixture) { f.col("a", "order", "i", "integer", 3, map[string]interface{}{"identity": "a"}) }, "identity"},
		"non-default collation": {func(f *fixture) { f.col("a", "order", "c", "text", 3, map[string]interface{}{"collation": "C"}) }, "collation"},
		"unlogged table": {func(f *fixture) {
			f.table("a", "u", map[string]interface{}{"persistence": "u"})
		}, "persistence"},
		"storage options": {func(f *fixture) { f.table("a", "o", map[string]interface{}{"options": "fillfactor=70"}) }, "storage options"},
	} {
		t.Run(name, func(t *testing.T) {
			f := good()
			tc.mutate(f)
			_, err := compile(t, f)
			require.ErrorIs(t, err, ErrUnsupported)
			require.ErrorContains(t, err, tc.want)
		})
	}
}

func TestCompile_PartitionsFollowTheirParent_AndTheParentsIndexCoversThem(t *testing.T) {
	f := good()
	f.table("a", "quote", map[string]interface{}{
		"partition": map[string]interface{}{"key": "RANGE (t)"},
		"indexes":   []interface{}{map[string]interface{}{"name": "ixq", "definition": "CREATE INDEX ixq ON ONLY a.quote USING btree (t DESC)"}},
	})
	f.col("a", "quote", "t", "timestamp with time zone", 1, nil)
	f.table("a", "quote_default", map[string]interface{}{"partition": map[string]interface{}{"parent": "a.quote", "bound": "DEFAULT"}})
	f.col("a", "quote_default", "t", "timestamp with time zone", 1, nil)
	p, err := compile(t, f)
	require.NoError(t, err)
	sql := p.SQL()
	require.Contains(t, sql, `CREATE TABLE "a"."quote" (`)
	require.Contains(t, sql, `) PARTITION BY RANGE (t);`)
	require.Contains(t, sql, `CREATE TABLE "a"."quote_default" PARTITION OF "a"."quote" DEFAULT;`, "a partition lists no columns of its own")
	require.Less(t, strings.Index(sql, `CREATE TABLE "a"."quote" (`), strings.Index(sql, `PARTITION OF "a"."quote"`), "the parent first")
	require.Contains(t, sql, "CREATE INDEX ixq ON a.quote USING btree (t DESC);", "ON ONLY would leave the parent's index invalid and the partitions unindexed")
	require.NotContains(t, sql, "ON ONLY")
}

func TestCompile_AnOrdinaryTablesIndexIsLeftAsRecorded(t *testing.T) {
	f := good()
	f.nodes = append(f.nodes, &models.CatalogNode{NodeTypeID: scanner.NODE_TYPE_TABLE, NodeName: "w", QualifiedPath: "/a/w", Properties: js(map[string]interface{}{
		"schema": "a", "indexes": []interface{}{map[string]interface{}{"name": "iw", "definition": "CREATE INDEX iw ON ONLY a.w USING btree (x)"}}})})
	f.col("a", "w", "x", "integer", 1, nil)
	p, err := compile(t, f)
	require.NoError(t, err)
	require.Contains(t, p.SQL(), "CREATE INDEX iw ON ONLY a.w", "only a partitioned parent's index is rewritten; this one is the server's text")
}

func TestCompile_RefusesAnInconsistentScan(t *testing.T) {
	f := good()
	f.table("a", "orphan", map[string]interface{}{"partition": map[string]interface{}{"parent": "a.gone", "bound": "DEFAULT"}})
	_, err := compile(t, f)
	require.ErrorIs(t, err, ErrBadScan)
	require.ErrorContains(t, err, "a.gone")

	f = good()
	f.table("a", "empty", nil)
	_, err = compile(t, f)
	require.ErrorIs(t, err, ErrBadScan, "a table with no columns is not a table")

	_, err = Compile(good().nodes, Options{})
	require.Error(t, err, "a template that names no schemas")
}

func TestCompile_IgnoresWhatIsNotInTheTemplate(t *testing.T) {
	f := good()
	f.schema("other", map[string]interface{}{"definitions_captured": false}) // unscanned or incomplete, but not asked for
	f.table("other", "x", map[string]interface{}{"persistence": "u"})
	p, err := compile(t, f)
	require.NoError(t, err)
	require.NotContains(t, p.SQL(), "other")
}

func TestCompile_IsDeterministic(t *testing.T) {
	a, err := compile(t, good())
	require.NoError(t, err)
	f := good()
	for i, j := 0, len(f.nodes)-1; i < j; i, j = i+1, j-1 {
		f.nodes[i], f.nodes[j] = f.nodes[j], f.nodes[i]
	}
	b, err := compile(t, f)
	require.NoError(t, err)
	require.Equal(t, a.Hash(), b.Hash())
}

func TestQuoting(t *testing.T) {
	require.Equal(t, `"order"`, qident("order"))
	require.Equal(t, `"a""b"`, qident(`a"b`), "a quote in a name is doubled, never trusted")
	require.Equal(t, `"s"."t"`, qname("s", "t"))
}
