package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/altinv"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://postgres:postgres@100.84.50.65:5432/alpha?sslmode=disable"
	}

	db, err := sqlx.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}

	invID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	var svc altinv.Service = altinv.NewService(db)
	_ = svc

	ctx := context.Background()
	log.Printf("Executing on-the-fly XIRR calculation for Investment %s...", invID)

	res, err := svc.CalculateInvestmentMetrics(ctx, invID)
	if err != nil {
		log.Fatalf("Calculation failed: %v", err)
	}

	fmt.Println("\n=======================================================")
	fmt.Println("       ON-THE-FLY XIRR & PERFORMANCE RESULTS           ")
	fmt.Println("=======================================================")
	fmt.Printf("Investment ID:         %s\n", res.InvestmentID)
	fmt.Printf("Total Capital Called:  $%.2f\n", res.TotalCapitalCalled)
	fmt.Printf("Total Distributed:     $%.2f\n", res.TotalDistributions)
	fmt.Printf("Current NAV:           $%.2f\n", res.CurrentNAV)
	fmt.Printf("TVPI Multiple:         %.2fx\n", res.TVPI)
	fmt.Printf("DPI Multiple:          %.2fx\n", res.DPI)
	fmt.Printf("RVPI Multiple:         %.2fx\n", res.RVPI)
	fmt.Printf("Calculated XIRR:       %.2f%% (%.4f)\n", res.XIRR*100, res.XIRR)
	fmt.Println("-------------------------------------------------------")
	fmt.Printf("Cash Flows Considered (%d events):\n", res.CashFlowCount)
	for i, f := range res.CashFlows {
		fmt.Printf("  [%d] %s | %-14s | $%12.2f\n", i+1, f.Date.Format("2006-01-02"), f.Type, f.Amount)
	}
	fmt.Println("=======================================================\n")
}
