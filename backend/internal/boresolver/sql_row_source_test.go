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

func newResolver() *br.Resolver {
	r := br.NewResolver("cashflows", "cashflows", br.PostgresDialect{})
	r.AddMapping("customer_id", "cashflows", "customer_id")
	r.AddMapping("cashflow_date", "cashflows", "cashflow_date")
	r.AddMapping("cashflow_amount", "cashflows", "cashflow_amount")
	return r
}

// TestSQLRowSource_BatchedSingleQuery is the throughput proof: fetching
// rows for N entities must issue exactly ONE query (GROUP BY entity,
// aggregate into a JSON array per entity) — never one query per entity.
// That single-query-per-node property is what makes this viable at
// multi-tenant, millions-of-positions scale.
func TestSQLRowSource_BatchedSingleQuery(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	source := &br.SQLRowSource{
		DB:         sqlxDB,
		Resolver:   newResolver(),
		EntityTerm: "customer_id",
		OrderTerm:  "cashflow_date",
	}

	rows := sqlmock.NewRows([]string{"entity_id", "cashflow_amount", "cashflow_date"}).
		AddRow("cust-alpha", `[-10000,2750,4250,3250,2750]`, `["2008-01-01","2008-03-01","2008-10-30","2009-02-15","2009-04-01"]`).
		AddRow("cust-beta", `[-20000,5500]`, `["2020-01-01","2020-06-01"]`)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT "customer_id" AS entity_id, json_agg("cashflow_amount" ORDER BY "cashflow_date")::text AS "cashflow_amount", json_agg("cashflow_date" ORDER BY "cashflow_date")::text AS "cashflow_date" FROM "cashflows" WHERE "tenant_id" = $1 GROUP BY "customer_id"`,
	)).WithArgs("tenant-a").WillReturnRows(rows)

	result, err := source.FetchRows(context.Background(), "tenant-a", []string{"cashflow_amount", "cashflow_date"})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet()) // proves exactly one query was issued

	require.Len(t, result, 2)
	require.Len(t, result["cust-alpha"], 5)
	assert.Equal(t, float64(-10000), result["cust-alpha"][0]["cashflow_amount"])
	assert.Equal(t, "2008-01-01", result["cust-alpha"][0]["cashflow_date"])
	require.Len(t, result["cust-beta"], 2)
	assert.Equal(t, float64(5500), result["cust-beta"][1]["cashflow_amount"])
}

// TestSQLRowSource_TenantScoped proves the WHERE clause carries the tenant
// filter with the requesting tenant's ID as a bound parameter (not
// interpolated into the query text), matching the Rule 7 convention used
// elsewhere in boresolver.
func TestSQLRowSource_TenantScoped(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	source := &br.SQLRowSource{
		DB:         sqlxDB,
		Resolver:   newResolver(),
		EntityTerm: "customer_id",
		OrderTerm:  "cashflow_date",
	}

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT "customer_id" AS entity_id, json_agg("cashflow_amount" ORDER BY "cashflow_date")::text AS "cashflow_amount" FROM "cashflows" WHERE "tenant_id" = $1 GROUP BY "customer_id"`,
	)).WithArgs("tenant-b").WillReturnRows(sqlmock.NewRows([]string{"entity_id", "cashflow_amount"}))

	_, err = source.FetchRows(context.Background(), "tenant-b", []string{"cashflow_amount"})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSQLRowSource_RequiresTenantID(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	source := &br.SQLRowSource{DB: sqlx.NewDb(db, "postgres"), Resolver: newResolver(), EntityTerm: "customer_id", OrderTerm: "cashflow_date"}
	_, err = source.FetchRows(context.Background(), "", []string{"cashflow_amount"})
	assert.Error(t, err)
}

// TestSQLRowSource_RejectsCrossTableTerms proves the (documented) single-
// table limitation fails loudly at compile time rather than silently
// joining or dropping data.
func TestSQLRowSource_RejectsCrossTableTerms(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	resolver := br.NewResolver("cashflows", "cashflows", br.PostgresDialect{})
	resolver.AddMapping("customer_id", "cashflows", "customer_id")
	resolver.AddMapping("cashflow_date", "cashflows", "cashflow_date")
	resolver.AddMapping("cashflow_amount", "other_table", "amount")

	source := &br.SQLRowSource{DB: sqlx.NewDb(db, "postgres"), Resolver: resolver, EntityTerm: "customer_id", OrderTerm: "cashflow_date"}
	_, err = source.FetchRows(context.Background(), "tenant-a", []string{"cashflow_amount"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cashflow_amount")
}
