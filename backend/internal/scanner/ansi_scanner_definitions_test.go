package scanner

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/models"
	"github.com/stretchr/testify/require"
)

func defsScanner(t *testing.T, whitelist ...string) (*AnsiScanner, sqlmock.Sqlmock, map[string]*models.CatalogNode) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	s := &AnsiScanner{sourceDB: db, tenantDatasourceId: uuid.New(), sourceSystem: "crims", schemaWhitelist: whitelist}
	nodes := map[string]*models.CatalogNode{}
	add := func(path string, typ uuid.UUID, props string) {
		n := &models.CatalogNode{QualifiedPath: path, NodeTypeID: typ, Properties: json.RawMessage(props)}
		nodes[path] = n
		s.nodes = append(s.nodes, n)
	}
	add("/orm", NODE_TYPE_SCHEMA, `{"is_core":true}`)
	s.nodes[0].NodeName = "orm"
	add("/orm/quote", NODE_TYPE_TABLE, `{"schema":"orm","is_core":true}`)
	add("/orm/quote_default", NODE_TYPE_TABLE, `{"schema":"orm"}`)
	add("/orm/plain", NODE_TYPE_TABLE, `{"schema":"orm"}`)
	return s, mock, nodes
}

func props(t *testing.T, n *models.CatalogNode) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(n.Properties, &m))
	return m
}

func expectAll(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(`pg_constraint k`).WillReturnRows(sqlmock.NewRows([]string{"s", "t", "n", "d"}).
		AddRow("orm", "quote", "chk_quote_price", "CHECK ((bid_price >= (0)::numeric))").
		AddRow("orm", "ghost", "chk_ghost", "CHECK (true)")) // a table the scan did not store: ignored, not an error
	mock.ExpectQuery(`FROM pg_index i`).WillReturnRows(sqlmock.NewRows([]string{"s", "t", "n", "d", "u", "m"}).
		AddRow("orm", "quote", "idx_quote_sec_time", "CREATE INDEX idx_quote_sec_time ON ONLY orm.quote USING btree (security_id, quote_time DESC)", false, "btree"))
	mock.ExpectQuery(`pg_get_partkeydef`).WillReturnRows(sqlmock.NewRows([]string{"s", "t", "k", "b", "p"}).
		AddRow("orm", "quote", "RANGE (quote_time)", "", "").
		AddRow("orm", "quote_default", "", "DEFAULT", "orm.quote"))
	mock.ExpectQuery(`FROM pg_trigger t`).WillReturnRows(sqlmock.NewRows([]string{"s", "t", "n", "d", "f"}).
		AddRow("orm", "plain", "trg_plain", "CREATE TRIGGER trg_plain AFTER INSERT ON orm.plain FOR EACH ROW EXECUTE FUNCTION orm.f()", "orm.f"))
	mock.ExpectQuery(`FROM pg_proc p`).WillReturnRows(sqlmock.NewRows([]string{"s", "n", "a", "k", "l", "d"}).
		AddRow("orm", "f", "", "f", "plpgsql", "CREATE OR REPLACE FUNCTION orm.f() RETURNS trigger LANGUAGE plpgsql AS $$ begin return new; end $$"))
}

func TestProcessDefinitions_RecordsEachClassOnTheRightNode(t *testing.T) {
	s, mock, n := defsScanner(t)
	expectAll(mock)
	require.NoError(t, s.processDefinitions())

	q := props(t, n["/orm/quote"])
	require.Equal(t, true, q["is_core"], "existing properties are kept")
	require.Equal(t, []interface{}{map[string]interface{}{"name": "chk_quote_price", "definition": "CHECK ((bid_price >= (0)::numeric))"}}, q["check_constraints"])
	idx := q["indexes"].([]interface{})[0].(map[string]interface{})
	require.Equal(t, "idx_quote_sec_time", idx["name"])
	require.Equal(t, "btree", idx["method"])
	require.Equal(t, false, idx["unique"])
	require.Equal(t, map[string]interface{}{"key": "RANGE (quote_time)"}, q["partition"])

	pd := props(t, n["/orm/quote_default"])
	require.Equal(t, map[string]interface{}{"parent": "orm.quote", "bound": "DEFAULT"}, pd["partition"])

	pl := props(t, n["/orm/plain"])
	require.Equal(t, "trg_plain", pl["triggers"].([]interface{})[0].(map[string]interface{})["name"])
	require.NotContains(t, pl, "indexes", "a table with nothing to record keeps its properties as they were")
	require.NotContains(t, pl, "check_constraints")
	require.NotContains(t, pl, "partition")

	sc := props(t, n["/orm"])
	require.Equal(t, true, sc["definitions_captured"])
	require.EqualValues(t, DefinitionsVersion, sc["definitions_version"])
	r := sc["routines"].([]interface{})[0].(map[string]interface{})
	require.Equal(t, "f", r["name"])
	require.Equal(t, "plpgsql", r["language"])
	require.NoError(t, mock.ExpectationsWereMet())
}

// A scan that could not read them must say so, not record nothing silently: a deploy built from it has to refuse.
func TestProcessDefinitions_AFailedQueryMarksTheScanAsNotCapturing(t *testing.T) {
	s, mock, n := defsScanner(t)
	mock.ExpectQuery(`pg_constraint k`).WillReturnRows(sqlmock.NewRows([]string{"s", "t", "n", "d"}))
	mock.ExpectQuery(`FROM pg_index i`).WillReturnError(errors.New("permission denied for relation pg_index"))
	mock.ExpectQuery(`pg_get_partkeydef`).WillReturnRows(sqlmock.NewRows([]string{"s", "t", "k", "b", "p"}))
	mock.ExpectQuery(`FROM pg_trigger t`).WillReturnRows(sqlmock.NewRows([]string{"s", "t", "n", "d", "f"}))
	mock.ExpectQuery(`FROM pg_proc p`).WillReturnRows(sqlmock.NewRows([]string{"s", "n", "a", "k", "l", "d"}))
	err := s.processDefinitions()
	require.Error(t, err)
	sc := props(t, n["/orm"])
	require.Equal(t, false, sc["definitions_captured"])
	require.Contains(t, sc["definitions_error"], "indexes")
	require.NoError(t, mock.ExpectationsWereMet(), "the other classes were still attempted")
}

func TestProcessDefinitions_ARowThatCannotBeReadIsAFailureNotASkip(t *testing.T) {
	s, mock, n := defsScanner(t)
	mock.ExpectQuery(`pg_constraint k`).WillReturnRows(sqlmock.NewRows([]string{"s", "t", "n", "d"}).AddRow("orm", "quote", "c", nil))
	mock.ExpectQuery(`FROM pg_index i`).WillReturnRows(sqlmock.NewRows([]string{"s", "t", "n", "d", "u", "m"}))
	mock.ExpectQuery(`pg_get_partkeydef`).WillReturnRows(sqlmock.NewRows([]string{"s", "t", "k", "b", "p"}))
	mock.ExpectQuery(`FROM pg_trigger t`).WillReturnRows(sqlmock.NewRows([]string{"s", "t", "n", "d", "f"}))
	mock.ExpectQuery(`FROM pg_proc p`).WillReturnRows(sqlmock.NewRows([]string{"s", "n", "a", "k", "l", "d"}))
	require.Error(t, s.processDefinitions())
	require.Equal(t, false, props(t, n["/orm"])["definitions_captured"])
}

// The queries are what decide what counts as "a key index" and "a clone", so the clauses are pinned.
func TestProcessDefinitions_QueriesExcludeWhatTheKeyAndPartitionPropertiesAlreadyCover(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer db.Close()
	s := &AnsiScanner{sourceDB: db, schemaWhitelist: []string{"orm", "mdm"}}
	empty := func(cols ...string) *sqlmock.Rows { return sqlmock.NewRows(cols) }
	mock.ExpectQuery(`k\.contype = 'c' AND k\.conislocal AND n\.nspname IN \(\$1, \$2\)`).WithArgs("orm", "mdm").WillReturnRows(empty("s", "t", "n", "d"))
	mock.ExpectQuery(`NOT EXISTS \(SELECT 1 FROM pg_constraint k WHERE k\.conindid = i\.indexrelid AND k\.contype IN \('p', 'u', 'x'\)\)\s+AND NOT EXISTS \(SELECT 1 FROM pg_inherits h WHERE h\.inhrelid = i\.indexrelid\)`).WithArgs("orm", "mdm").WillReturnRows(empty("s", "t", "n", "d", "u", "m"))
	mock.ExpectQuery(`c\.relkind IN \('r', 'p'\) AND \(c\.relkind = 'p' OR c\.relispartition\)`).WithArgs("orm", "mdm").WillReturnRows(empty("s", "t", "k", "b", "p"))
	mock.ExpectQuery(`NOT t\.tgisinternal AND t\.tgparentid = 0`).WithArgs("orm", "mdm").WillReturnRows(empty("s", "t", "n", "d", "f"))
	mock.ExpectQuery(`p\.prokind IN \('f', 'p'\).*d\.deptype = 'e'`).WithArgs("orm", "mdm").WillReturnRows(empty("s", "n", "a", "k", "l", "d"))
	require.NoError(t, s.processDefinitions())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMergeProps_KeepsExistingKeysAndSurvivesBadProperties(t *testing.T) {
	n := &models.CatalogNode{QualifiedPath: "/a/b", Properties: json.RawMessage(`{"a":1}`)}
	mergeProps(n, map[string]interface{}{"b": 2})
	require.JSONEq(t, `{"a":1,"b":2}`, string(n.Properties))
	bad := &models.CatalogNode{QualifiedPath: "/x", Properties: json.RawMessage(`{nope`)}
	mergeProps(bad, map[string]interface{}{"b": 2})
	require.Equal(t, `{nope`, string(bad.Properties), "unreadable properties are left alone, not overwritten")
	empty := &models.CatalogNode{QualifiedPath: "/y"}
	mergeProps(empty, map[string]interface{}{"b": 2})
	require.JSONEq(t, `{"b":2}`, string(empty.Properties))
}
