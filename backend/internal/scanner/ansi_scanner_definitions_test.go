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
	add := func(path, name string, typ uuid.UUID, props string) {
		n := &models.CatalogNode{QualifiedPath: path, NodeName: name, NodeTypeID: typ, Properties: json.RawMessage(props)}
		nodes[path] = n
		s.nodes = append(s.nodes, n)
	}
	add("/orm", "orm", NODE_TYPE_SCHEMA, `{"is_core":true}`)
	add("/orm/quote", "quote", NODE_TYPE_TABLE, `{"schema":"orm","is_core":true}`)
	add("/orm/quote_default", "quote_default", NODE_TYPE_TABLE, `{"schema":"orm"}`)
	add("/orm/plain", "plain", NODE_TYPE_TABLE, `{"schema":"orm"}`)
	s.columnMap = map[uuid.UUID]*models.CatalogNode{}
	col := &models.CatalogNode{QualifiedPath: "/orm/quote/bid", Properties: json.RawMessage(`{"data_type":"numeric","precision":18}`)}
	s.columnMap[generateID(s.tenantDatasourceId.String(), "crims", NODE_TYPE_COLUMN.String(), col.QualifiedPath)] = col
	nodes["/orm/quote/bid"] = col
	return s, mock, nodes
}

func props(t *testing.T, n *models.CatalogNode) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(n.Properties, &m))
	return m
}

// The queries run in this order; a test that expects them says so once.
const (
	qConstraints = `pg_constraint k`
	qIndexes     = `FROM pg_index i`
	qPartitions  = `pg_get_partkeydef`
	qTriggers    = `FROM pg_trigger t`
	qRoutines    = `FROM pg_proc p`
	qColumns     = `format_type\(a\.atttypid`
	qOptions     = `c\.relpersistence <> 'p'`
	qExtensions  = `FROM pg_extension e`
)

var (
	colsConstraints = []string{"s", "t", "n", "ty", "d"}
	colsIndexes     = []string{"s", "t", "n", "d", "u", "m"}
	colsPartitions  = []string{"s", "t", "k", "b", "p"}
	colsTriggers    = []string{"s", "t", "n", "d", "f"}
	colsRoutines    = []string{"s", "n", "a", "k", "l", "d"}
	colsColumns     = []string{"s", "t", "c", "ft", "g", "i", "co"}
	colsOptions     = []string{"s", "t", "p", "o"}
	colsExtensions  = []string{"n", "v", "s"}
)

func expectEmpty(mock sqlmock.Sqlmock, from int) {
	all := []struct {
		q    string
		cols []string
	}{{qConstraints, colsConstraints}, {qIndexes, colsIndexes}, {qPartitions, colsPartitions}, {qTriggers, colsTriggers},
		{qRoutines, colsRoutines}, {qColumns, colsColumns}, {qOptions, colsOptions}, {qExtensions, colsExtensions}}
	for _, e := range all[from:] {
		mock.ExpectQuery(e.q).WillReturnRows(sqlmock.NewRows(e.cols))
	}
}

func expectAll(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(qConstraints).WillReturnRows(sqlmock.NewRows(colsConstraints).
		AddRow("orm", "quote", "quote_pkey", "p", "PRIMARY KEY (id, quote_time)").
		AddRow("orm", "quote", "chk_quote_price", "c", "CHECK ((bid_price >= (0)::numeric))").
		AddRow("orm", "ghost", "chk_ghost", "c", "CHECK (true)")) // a table the scan did not store: ignored, not an error
	mock.ExpectQuery(qIndexes).WillReturnRows(sqlmock.NewRows(colsIndexes).
		AddRow("orm", "quote", "idx_quote_sec_time", "CREATE INDEX idx_quote_sec_time ON ONLY orm.quote USING btree (security_id, quote_time DESC)", false, "btree"))
	mock.ExpectQuery(qPartitions).WillReturnRows(sqlmock.NewRows(colsPartitions).
		AddRow("orm", "quote", "RANGE (quote_time)", "", "").
		AddRow("orm", "quote_default", "", "DEFAULT", "orm.quote"))
	mock.ExpectQuery(qTriggers).WillReturnRows(sqlmock.NewRows(colsTriggers).
		AddRow("orm", "plain", "trg_plain", "CREATE TRIGGER trg_plain AFTER INSERT ON orm.plain FOR EACH ROW EXECUTE FUNCTION orm.f()", "orm.f"))
	mock.ExpectQuery(qRoutines).WillReturnRows(sqlmock.NewRows(colsRoutines).
		AddRow("orm", "f", "", "f", "plpgsql", "CREATE OR REPLACE FUNCTION orm.f() RETURNS trigger LANGUAGE plpgsql AS $$ begin return new; end $$"))
	mock.ExpectQuery(qColumns).WillReturnRows(sqlmock.NewRows(colsColumns).
		AddRow("orm", "quote", "bid", "numeric(18,9)", "", "", "").
		AddRow("orm", "quote", "ghost", "text", "", "", "")) // a column the scan did not store: ignored
	mock.ExpectQuery(qOptions).WillReturnRows(sqlmock.NewRows(colsOptions).AddRow("orm", "plain", "u", "fillfactor=70"))
	mock.ExpectQuery(qExtensions).WillReturnRows(sqlmock.NewRows(colsExtensions).AddRow("uuid-ossp", "1.1", "public"))
}

func TestProcessDefinitions_RecordsEachClassOnTheRightNode(t *testing.T) {
	s, mock, n := defsScanner(t)
	expectAll(mock)
	require.NoError(t, s.processDefinitions())

	q := props(t, n["/orm/quote"])
	require.Equal(t, true, q["is_core"], "existing properties are kept")
	require.Equal(t, []interface{}{
		map[string]interface{}{"name": "quote_pkey", "type": "p", "definition": "PRIMARY KEY (id, quote_time)"},
		map[string]interface{}{"name": "chk_quote_price", "type": "c", "definition": "CHECK ((bid_price >= (0)::numeric))"},
	}, q["constraints"], "every local constraint, in the server's own words and in key order")
	idx := q["indexes"].([]interface{})[0].(map[string]interface{})
	require.Equal(t, "idx_quote_sec_time", idx["name"])
	require.Equal(t, "btree", idx["method"])
	require.Equal(t, false, idx["unique"])
	require.Equal(t, map[string]interface{}{"key": "RANGE (quote_time)"}, q["partition"])
	require.Contains(t, q, "persistence", "every structural key is written, null when the table has none, so a stale value in alpha is overwritten")
	require.Nil(t, q["persistence"])
	require.Nil(t, q["options"])

	col := props(t, n["/orm/quote/bid"])
	require.Equal(t, "numeric(18,9)", col["format_type"])
	require.EqualValues(t, 18, col["precision"], "existing column properties are kept")
	for _, k := range []string{"generated", "identity", "collation"} {
		require.Contains(t, col, k, "written as null so a column that stopped being %s is not still flagged in alpha", k)
		require.Nil(t, col[k], k)
	}

	pd := props(t, n["/orm/quote_default"])
	require.Equal(t, map[string]interface{}{"parent": "orm.quote", "bound": "DEFAULT"}, pd["partition"])

	pl := props(t, n["/orm/plain"])
	require.Equal(t, "trg_plain", pl["triggers"].([]interface{})[0].(map[string]interface{})["name"])
	require.Equal(t, []interface{}{}, pl["indexes"], "a table with none records an EMPTY list: a missing key would leave a dropped index in alpha")
	require.Equal(t, []interface{}{}, pl["constraints"])
	require.Contains(t, pl, "partition")
	require.Nil(t, pl["partition"])
	require.Equal(t, "u", pl["persistence"], "an unlogged table is recorded so a compiler can refuse it")
	require.Equal(t, "fillfactor=70", pl["options"])

	sc := props(t, n["/orm"])
	require.Equal(t, true, sc["definitions_captured"])
	require.Contains(t, sc, "definitions_error")
	require.Nil(t, sc["definitions_error"], "a clean scan clears an earlier error")
	require.EqualValues(t, DefinitionsVersion, sc["definitions_version"])
	require.Equal(t, []interface{}{map[string]interface{}{"name": "uuid-ossp", "version": "1.1", "schema": "public"}}, sc["extensions"])
	r := sc["routines"].([]interface{})[0].(map[string]interface{})
	require.Equal(t, "f", r["name"])
	require.Equal(t, "plpgsql", r["language"])
	require.NoError(t, mock.ExpectationsWereMet())
}

// A scan that could not read them must say so, not record nothing silently: a deploy built from it has to refuse.
func TestProcessDefinitions_AFailedQueryMarksTheScanAsNotCapturing(t *testing.T) {
	s, mock, n := defsScanner(t)
	mock.ExpectQuery(qConstraints).WillReturnRows(sqlmock.NewRows(colsConstraints))
	mock.ExpectQuery(qIndexes).WillReturnError(errors.New("permission denied for relation pg_index"))
	expectEmpty(mock, 2)
	err := s.processDefinitions()
	require.Error(t, err)
	sc := props(t, n["/orm"])
	require.Equal(t, false, sc["definitions_captured"])
	require.Contains(t, sc["definitions_error"], "indexes")
	require.NoError(t, mock.ExpectationsWereMet(), "the other classes were still attempted")
}

func TestProcessDefinitions_ARowThatCannotBeReadIsAFailureNotASkip(t *testing.T) {
	s, mock, n := defsScanner(t)
	mock.ExpectQuery(qConstraints).WillReturnRows(sqlmock.NewRows(colsConstraints).AddRow("orm", "quote", "c", "c", nil))
	expectEmpty(mock, 1)
	require.Error(t, s.processDefinitions())
	require.Equal(t, false, props(t, n["/orm"])["definitions_captured"])
}

// Each of these queries is what decides what counts as "a key", "a clone", "inherited" or "the extension's own", so the
// clauses are pinned.
func TestProcessDefinitions_QueriesExcludeWhatIsInheritedClonedOrOwnedByAnExtension(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer db.Close()
	s := &AnsiScanner{sourceDB: db, schemaWhitelist: []string{"orm", "mdm"}}
	empty := func(cols []string) *sqlmock.Rows { return sqlmock.NewRows(cols) }
	mock.ExpectQuery(`k\.contype IN \('p', 'u', 'c', 'f', 'x'\) AND k\.conislocal AND k\.conparentid = 0 AND c\.relkind IN \('r', 'p'\) AND n\.nspname IN \(\$1, \$2\)`).WithArgs("orm", "mdm").WillReturnRows(empty(colsConstraints))
	mock.ExpectQuery(`NOT EXISTS \(SELECT 1 FROM pg_constraint k WHERE k\.conindid = i\.indexrelid AND k\.contype IN \('p', 'u', 'x'\)\)\s+AND NOT EXISTS \(SELECT 1 FROM pg_inherits h WHERE h\.inhrelid = i\.indexrelid\)`).WithArgs("orm", "mdm").WillReturnRows(empty(colsIndexes))
	mock.ExpectQuery(`c\.relkind IN \('r', 'p'\) AND \(c\.relkind = 'p' OR c\.relispartition\)`).WithArgs("orm", "mdm").WillReturnRows(empty(colsPartitions))
	mock.ExpectQuery(`NOT t\.tgisinternal AND t\.tgparentid = 0`).WithArgs("orm", "mdm").WillReturnRows(empty(colsTriggers))
	mock.ExpectQuery(`p\.prokind IN \('f', 'p'\).*d\.deptype = 'e'`).WithArgs("orm", "mdm").WillReturnRows(empty(colsRoutines))
	mock.ExpectQuery(`a\.attnum > 0 AND NOT a\.attisdropped AND c\.relkind IN \('r', 'p'\) AND n\.nspname IN \(\$1, \$2\)`).WithArgs("orm", "mdm").WillReturnRows(empty(colsColumns))
	mock.ExpectQuery(`c\.relkind IN \('r', 'p'\) AND \(c\.relpersistence <> 'p' OR c\.reloptions IS NOT NULL\)`).WithArgs("orm", "mdm").WillReturnRows(empty(colsOptions))
	mock.ExpectQuery(`e\.extname <> 'plpgsql'`).WillReturnRows(empty(colsExtensions))
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
