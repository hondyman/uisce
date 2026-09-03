// customer-xirr-demo runs the boresolver host-runtime calc engine end to
// end against real Northwind data: CalcGraph -> CompileDeepCalculations
// (cutting the xirr() node out of the SQL) -> SQLRowSource (one batched,
// tenant-scoped query against public.customer_cashflows) ->
// HostRuntimeExecutor -> finlib.XIRR.
//
// customer_cashflows (see the migration SQL alongside this command) is a
// synthetic fixture, not a real financial metric: Northwind is pure sales
// data with no natural "return" leg, so each order line item is modeled as
// a negative cash flow (customer spend) with one synthetic positive
// terminal leg appended per customer so XIRR has the sign change it needs
// to solve. The point of this demo is proving the calc engine works
// against real data volume and real irregular dates, not proving anything
// about Northwind customers' actual returns.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/jmoiron/sqlx"

	_ "github.com/lib/pq"

	"github.com/hondyman/uisce/backend/internal/boresolver"
)

func main() {
	dsn := flag.String("dsn", os.Getenv("NORTHWINDS_DATABASE_URL"), "Postgres DSN for the northwinds database")
	tenantID := flag.String("tenant", "99e99e99-99e9-49e9-89e9-99e99e99e999", "tenant_id to scope the query to (must match customer_cashflows.tenant_id -- the real 'northwind' tenant row in alpha.tenants)")
	customer := flag.String("customer", "", "limit to one customer_id (e.g. SAVEA); empty = all customers")
	flag.Parse()

	if *dsn == "" {
		log.Fatal("NORTHWINDS_DATABASE_URL is required (or pass -dsn)")
	}

	db, err := sqlx.Connect("postgres", *dsn)
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer db.Close()

	resolver := boresolver.NewResolver("customer_cashflows", "customer_cashflows", boresolver.PostgresDialect{})
	resolver.AddMapping("customer_id", "customer_cashflows", "customer_id")
	resolver.AddMapping("cashflow_date", "customer_cashflows", "cashflow_date")
	resolver.AddMapping("cashflow_amount", "customer_cashflows", "cashflow_amount")

	source := &boresolver.SQLRowSource{
		DB:         db,
		Resolver:   resolver,
		EntityTerm: "customer_id",
		OrderTerm:  "cashflow_date",
	}
	executor := &boresolver.HostRuntimeExecutor{Rows: source}

	graph := boresolver.NewCalcGraph()
	graph.AddNode(&boresolver.CalcNode{TermKey: "cashflow_amount", IsBaseField: true})
	graph.AddNode(&boresolver.CalcNode{TermKey: "cashflow_date", IsBaseField: true})
	graph.AddNode(&boresolver.CalcNode{
		TermKey:      "customer_xirr",
		Formula:      "xirr(${cashflow_amount}, ${cashflow_date})",
		Dependencies: []string{"cashflow_amount", "cashflow_date"},
	})

	layers, err := graph.ResolveExecutionLayers()
	if err != nil {
		log.Fatalf("failed to resolve calc graph: %v", err)
	}

	gen := &boresolver.BOSQLGenerator{Dialect: boresolver.PostgresDialect{}}
	_, hostNodes, err := gen.CompileDeepCalculations(layers, "SELECT 1", []string{"customer_xirr"})
	if err != nil {
		log.Fatalf("failed to compile calc graph: %v", err)
	}

	results, err := executor.Execute(context.Background(), *tenantID, hostNodes)
	if err != nil {
		log.Fatalf("failed to execute host-runtime calcs: %v", err)
	}

	sort.Slice(results, func(i, j int) bool { return results[i].EntityID < results[j].EntityID })

	fmt.Printf("%-10s %10s\n", "customer", "xirr")
	for _, r := range results {
		if *customer != "" && r.EntityID != *customer {
			continue
		}
		if r.Err != nil {
			fmt.Printf("%-10s %10s  (%v)\n", r.EntityID, "ERROR", r.Err)
			continue
		}
		fmt.Printf("%-10s %9.2f%%\n", r.EntityID, r.Value*100)
	}
}
