package querybuilder

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFederationExtractEnabled(t *testing.T) {
	t.Setenv("CUBE_FEDERATION_EXTRACT", "")
	assert.False(t, federationExtractEnabled())
	t.Setenv("CUBE_FEDERATION_EXTRACT", "1")
	assert.True(t, federationExtractEnabled())
	t.Setenv("CUBE_FEDERATION_EXTRACT", "true")
	assert.True(t, federationExtractEnabled())
	t.Setenv("CUBE_FEDERATION_EXTRACT", "no")
	assert.False(t, federationExtractEnabled())
}

func TestCubeExtractStagingTableName(t *testing.T) {
	got := CubeExtractStagingTableName("abcdef0123456789", "11111111-2222-3333-4444-555555555555", "pos")
	assert.Equal(t, "cube_ext_abcdef01_11111111_pos", got)
}

func TestRewriteFederationFromSQLForStaging(t *testing.T) {
	sources := []FederationSourcePlan{
		{Alias: "pos", DrivingTable: "pg_alpha.oms.position"},
		{Alias: "acct", DrivingTable: "pg_alpha.oms.account"},
	}
	from := "pg_alpha.oms.position AS pos\nINNER JOIN pg_alpha.oms.account AS acct ON pos.account_number = acct.account_number"
	staging := map[string]string{
		"pos":  "`tenant_t`.`cube_ext_a_b_pos`",
		"acct": "`tenant_t`.`cube_ext_a_b_acct`",
	}
	got, err := RewriteFederationFromSQLForStaging(from, sources, staging)
	require.NoError(t, err)
	assert.Contains(t, got, "`tenant_t`.`cube_ext_a_b_pos` AS pos")
	assert.Contains(t, got, "INNER JOIN `tenant_t`.`cube_ext_a_b_acct` AS acct ON")
	assert.NotContains(t, got, "pg_alpha.oms.position")
	assert.NotContains(t, got, "pg_alpha.oms.account")
}

func TestRewriteCubeDDLFromClause(t *testing.T) {
	oldFrom := "pg_alpha.oms.position AS pos\nINNER JOIN pg_alpha.oms.account AS acct ON pos.a = acct.a"
	newFrom := "`t`.`s_pos` AS pos\nINNER JOIN `t`.`s_acct` AS acct ON pos.a = acct.a"
	ddl := "CREATE MATERIALIZED VIEW cube_x\nAS SELECT\n  pos.a\nFROM " + oldFrom + "\nGROUP BY pos.a;"
	got, err := RewriteCubeDDLFromClause(ddl, oldFrom, newFrom)
	require.NoError(t, err)
	assert.Contains(t, got, "FROM "+newFrom)
	assert.NotContains(t, got, "pg_alpha.oms.position")
}

func TestExtractSources_CTASAndRewrite(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer db.Close()

	from := "pg_alpha.oms.position AS pos\nINNER JOIN pg_alpha.oms.account AS acct ON pos.account_number = acct.account_number"
	ddl := "CREATE MATERIALIZED VIEW cube_x\nREFRESH ASYNC EVERY(INTERVAL 1 HOUR)\nAS SELECT\n  pos.account_id\nFROM " + from + "\nGROUP BY pos.account_id;"
	plan := &CubeMaterializePlan{
		TenantID:       "t1",
		AttemptID:      "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		GrainHash:      "deadbeefcafebabe",
		TargetDatabase: "tenant_t1",
		SourceTable:    from,
		DDL:            ddl,
		ExtractEnabled: true,
		FederationSources: []FederationSourcePlan{
			{Alias: "pos", DrivingTable: "pg_alpha.oms.position"},
			{Alias: "acct", DrivingTable: "pg_alpha.oms.account"},
		},
	}

	mock.ExpectExec(`CREATE DATABASE IF NOT EXISTS .+tenant_t1.+`).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`DROP TABLE IF EXISTS .+cube_ext_deadbeef_aaaaaaaa_pos.+`).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`CREATE TABLE .+cube_ext_deadbeef_aaaaaaaa_pos.+ AS SELECT \* FROM pg_alpha\.oms\.position`).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM .+cube_ext_deadbeef_aaaaaaaa_pos.+`).
		WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(10))
	mock.ExpectExec(`DROP TABLE IF EXISTS .+cube_ext_deadbeef_aaaaaaaa_acct.+`).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`CREATE TABLE .+cube_ext_deadbeef_aaaaaaaa_acct.+ AS SELECT \* FROM pg_alpha\.oms\.account`).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM .+cube_ext_deadbeef_aaaaaaaa_acct.+`).
		WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(3))

	m := &CubeMaterializer{starrocksDB: db}
	res, err := m.ExtractSources(context.Background(), plan)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.NotNil(t, res.Plan)
	assert.True(t, res.Plan.ExtractApplied)
	assert.Len(t, res.StagingTables, 2)
	assert.Equal(t, int64(10), res.RowCounts["pos"])
	assert.Equal(t, int64(3), res.RowCounts["acct"])
	assert.NotContains(t, res.Plan.SourceTable, "pg_alpha.oms.position")
	assert.Contains(t, res.Plan.DDL, res.Plan.SourceTable)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDropStagingTables(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer db.Close()
	mock.ExpectExec(`DROP TABLE IF EXISTS .+a.+`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`DROP TABLE IF EXISTS .+b.+`).WillReturnResult(sqlmock.NewResult(0, 0))
	m := &CubeMaterializer{starrocksDB: db}
	require.NoError(t, m.DropStagingTables(context.Background(), []string{"`t`.`a`", "`t`.`b`"}))
	require.NoError(t, mock.ExpectationsWereMet())
}
