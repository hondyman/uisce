package main

import (
	"context"
	"database/sql"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

type TicketRow struct {
	VendorID    string
	TicketID    string
	Summary     string
	HoursSpent  float64
	EntityDomain string
}

type AggregatedFriction struct {
	VendorID     string
	TicketCount  int
	HoursSpent   float64
	FrictionCost float64
}

func main() {
	filePath := flag.String("file", "", "Path to Jira or ServiceNow CSV export file")
	dsn := flag.String("dsn", "", "PostgreSQL connection string (defaults to POSTGRES_DSN env var)")
	tenantID := flag.String("tenant", "a11c0001-0001-4000-8000-000000000001", "Tenant ID")
	hourlyRate := flag.Float64("rate", 150.00, "Hourly labor cost rate for data stewards ($/hr)")
	dryRun := flag.Bool("dry-run", false, "Validate and display metrics without writing to database")
	flag.Parse()

	if *filePath == "" {
		fmt.Println("Usage: mdm-friction-importer -file <tickets.csv> [-dsn <postgres_dsn>] [-rate 150.0] [-dry-run]")
		os.Exit(1)
	}

	pgDSN := *dsn
	if pgDSN == "" {
		pgDSN = os.Getenv("POSTGRES_DSN")
	}

	file, err := os.Open(*filePath)
	if err != nil {
		log.Fatalf("Failed to open CSV file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	header, err := reader.Read()
	if err != nil {
		log.Fatalf("Failed to read CSV header: %v", err)
	}

	// Normalize header column indices
	vendorIdx := -1
	ticketIdx := -1
	hoursIdx := -1
	domainIdx := -1

	for i, h := range header {
		clean := strings.ToLower(strings.TrimSpace(h))
		switch {
		case clean == "vendor_id" || clean == "vendor" || strings.Contains(clean, "vendor"):
			if vendorIdx == -1 {
				vendorIdx = i
			}
		case clean == "ticket_id" || clean == "issue key" || clean == "key" || clean == "number" || clean == "incident":
			if ticketIdx == -1 {
				ticketIdx = i
			}
		case clean == "hours_spent" || clean == "investigation_hours" || clean == "time spent (h)" || clean == "hours":
			if hoursIdx == -1 {
				hoursIdx = i
			}
		case clean == "domain" || clean == "entity_domain" || clean == "category":
			if domainIdx == -1 {
				domainIdx = i
			}
		}
	}

	if vendorIdx == -1 {
		log.Fatalf("CSV header must contain a vendor column (e.g., 'vendor_id', 'Vendor')")
	}

	// Read and aggregate rows
	aggMap := make(map[string]*AggregatedFriction)
	var validRows int

	for {
		rec, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Warning: skipping corrupt CSV line: %v", err)
			continue
		}

		vendor := strings.ToUpper(strings.TrimSpace(rec[vendorIdx]))
		if vendor == "" {
			continue
		}

		// Normalize vendor code alias
		switch {
		case strings.HasPrefix(vendor, "BLOOM"):
			vendor = "BBG"
		case strings.HasPrefix(vendor, "REFIN") || strings.HasPrefix(vendor, "LSEG"):
			vendor = "RFT"
		case strings.HasPrefix(vendor, "FACT"):
			vendor = "FDS"
		case strings.HasPrefix(vendor, "ICE"):
			vendor = "ICE"
		case strings.HasPrefix(vendor, "S&P") || strings.HasPrefix(vendor, "SPG"):
			vendor = "SPG"
		}

		hours := 2.5 // default 2.5 hrs per defect if unspecified
		if hoursIdx != -1 && hoursIdx < len(rec) {
			if hVal, parseErr := strconv.ParseFloat(strings.TrimSpace(rec[hoursIdx]), 64); parseErr == nil && hVal >= 0 {
				hours = hVal
			}
		}

		agg, exists := aggMap[vendor]
		if !exists {
			agg = &AggregatedFriction{VendorID: vendor}
			aggMap[vendor] = agg
		}

		agg.TicketCount++
		agg.HoursSpent += hours
		agg.FrictionCost += hours * (*hourlyRate)
		validRows++
	}

	fmt.Printf("\n=== MDM Operational Friction Importer (Jira / ServiceNow) ===\n")
	fmt.Printf("Parsed %d tickets across %d vendors (Hourly Rate: $%.2f/hr)\n\n", validRows, len(aggMap), *hourlyRate)
	fmt.Printf("%-8s | %-12s | %-16s | %-18s\n", "VENDOR", "TICKETS", "HOURS SPENT", "CALCULATED FRICTION")
	fmt.Printf("---------+--------------+------------------+--------------------\n")

	for _, v := range []string{"BBG", "RFT", "FDS", "ICE", "SPG"} {
		if agg, ok := aggMap[v]; ok {
			fmt.Printf("%-8s | %-12d | %-16.1fh | $%-18.2f\n", agg.VendorID, agg.TicketCount, agg.HoursSpent, agg.FrictionCost)
		}
	}
	// Print any other non-canonical vendors
	for v, agg := range aggMap {
		if v != "BBG" && v != "RFT" && v != "FDS" && v != "ICE" && v != "SPG" {
			fmt.Printf("%-8s | %-12d | %-16.1fh | $%-18.2f\n", agg.VendorID, agg.TicketCount, agg.HoursSpent, agg.FrictionCost)
		}
	}
	fmt.Println()

	if *dryRun {
		fmt.Println("[DRY RUN] Validation successful. No database records modified.")
		return
	}

	if pgDSN == "" {
		log.Fatalf("Error: PostgreSQL DSN not specified (-dsn or POSTGRES_DSN)")
	}

	db, err := sql.Open("postgres", pgDSN)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	windowEnd := time.Now().Truncate(24 * time.Hour)
	windowStart := windowEnd.AddDate(0, 0, -30)

	upsertSQL := `
		INSERT INTO mdm_eval.vendor_operational_friction (
			tenant_id, vendor_id, evaluation_window_start, evaluation_window_end,
			defect_tickets_count, investigation_hours, hourly_labor_rate, calculated_friction_cost, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		ON CONFLICT (tenant_id, vendor_id, evaluation_window_end)
		DO UPDATE SET
			defect_tickets_count = EXCLUDED.defect_tickets_count,
			investigation_hours = EXCLUDED.investigation_hours,
			hourly_labor_rate = EXCLUDED.hourly_labor_rate,
			calculated_friction_cost = EXCLUDED.calculated_friction_cost,
			updated_at = NOW();
	`

	stmt, err := db.PrepareContext(ctx, upsertSQL)
	if err != nil {
		log.Fatalf("Failed to prepare upsert query: %v", err)
	}
	defer stmt.Close()

	var insertedCount int
	for _, agg := range aggMap {
		_, err := stmt.ExecContext(
			ctx,
			*tenantID,
			agg.VendorID,
			windowStart,
			windowEnd,
			agg.TicketCount,
			agg.HoursSpent,
			*hourlyRate,
			agg.FrictionCost,
		)
		if err != nil {
			log.Printf("Failed to upsert friction for vendor %s: %v", agg.VendorID, err)
		} else {
			insertedCount++
		}
	}

	fmt.Printf("Successfully synchronized %d vendor friction records into mdm_eval.vendor_operational_friction.\n\n", insertedCount)
}
