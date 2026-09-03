package boresolver_test

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	br "github.com/hondyman/uisce/backend/internal/boresolver"
)

// TestSQLRowSource_HostRuntimeExecutor_FullChain proves the whole pipeline
// end to end against the Microsoft-documented XIRR fixture: a CalcGraph
// with a host-runtime xirr() field gets compiled (cutting the node out of
// the SQL, per calc_compiler.go), the cut node is fetched via ONE batched,
// tenant-scoped SQL query (SQLRowSource), and evaluated by finlib through
// HostRuntimeExecutor — landing on the correct 37.34% XIRR.
func TestSQLRowSource_HostRuntimeExecutor_FullChain(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	resolver := br.NewResolver("cashflows", "cashflows", br.PostgresDialect{})
	resolver.AddMapping("fund_id", "cashflows", "fund_id")
	resolver.AddMapping("cashflow_date", "cashflows", "cashflow_date")
	resolver.AddMapping("cashflow_amount", "cashflows", "cashflow_amount")

	source := &br.SQLRowSource{
		DB:         sqlx.NewDb(db, "postgres"),
		Resolver:   resolver,
		EntityTerm: "fund_id",
		OrderTerm:  "cashflow_date",
	}
	executor := &br.HostRuntimeExecutor{Rows: source}

	graph := br.NewCalcGraph()
	graph.AddNode(&br.CalcNode{TermKey: "cashflow_amount", IsBaseField: true})
	graph.AddNode(&br.CalcNode{TermKey: "cashflow_date", IsBaseField: true})
	graph.AddNode(&br.CalcNode{
		TermKey:      "fund_xirr",
		Formula:      "xirr(${cashflow_amount}, ${cashflow_date})",
		Dependencies: []string{"cashflow_amount", "cashflow_date"},
	})

	layers, err := graph.ResolveExecutionLayers()
	require.NoError(t, err)

	gen := &br.BOSQLGenerator{}
	_, hostNodes, err := gen.CompileDeepCalculations(layers, "SELECT 1", []string{"fund_xirr"})
	require.NoError(t, err)
	require.Len(t, hostNodes, 1)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT "fund_id" AS entity_id, json_agg("cashflow_amount" ORDER BY "cashflow_date")::text AS "cashflow_amount", json_agg("cashflow_date" ORDER BY "cashflow_date")::text AS "cashflow_date" FROM "cashflows" WHERE "tenant_id" = $1 GROUP BY "fund_id"`,
	)).WithArgs("tenant-alpha").WillReturnRows(
		sqlmock.NewRows([]string{"entity_id", "cashflow_amount", "cashflow_date"}).
			AddRow("fund-1", `[-10000,2750,4250,3250,2750]`, `["2008-01-01","2008-03-01","2008-10-30","2009-02-15","2009-04-01"]`),
	)

	results, err := executor.Execute(context.Background(), "tenant-alpha", hostNodes)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
	require.Len(t, results, 1)

	assert.NoError(t, results[0].Err)
	assert.Equal(t, "fund-1", results[0].EntityID)
	assert.InDelta(t, 0.373362535, results[0].Value, 1e-6)
}
