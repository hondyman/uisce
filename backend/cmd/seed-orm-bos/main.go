package main

import (
	"context"
	"fmt"
	"log"
	"os"

	metadata "github.com/hondyman/uisce/backend/internal/metadata"
	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@100.84.50.65:5432/alpha?sslmode=disable"
	}

	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to alpha DB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	var tenantID string
	err = db.GetContext(ctx, &tenantID, `SELECT id FROM tenants WHERE gold_copy = true LIMIT 1`)
	if err != nil || tenantID == "" {
		err = db.GetContext(ctx, &tenantID, `SELECT id FROM tenants LIMIT 1`)
		if err != nil {
			log.Fatalf("Failed to find tenant: %v", err)
		}
	}

	log.Printf("Seeding ORM Financial BOs for tenant %s\n", tenantID)

	boService := metadata.NewBusinessObjectService(db, nil, nil, nil)
	secCtx := &security.Context{
		TenantID:      tenantID,
		Region:        "us-west",
		IsGlobalAdmin: true,
	}

	ormBOs := []models.CreateBusinessObjectRequest{
		{
			Name:        "Trade",
			DisplayName: "Trade / Execution",
			Description: "Front-office executed trades, fills, and allocations from CRIMS ORM",
			Icon:        "trending-up",
			Category:    "Trading",
		},
		{
			Name:        "Account",
			DisplayName: "Investment Account",
			Description: "Client investment account master, custody details, and cash balances from CRIMS ORM",
			Icon:        "credit-card",
			Category:    "Account Management",
		},
		{
			Name:        "Position",
			DisplayName: "Holdings / Position",
			Description: "Real-time portfolio security positions and holdings from CRIMS ORM",
			Icon:        "pie-chart",
			Category:    "Portfolio Management",
		},
		{
			Name:        "Security",
			DisplayName: "Security Master",
			Description: "Financial instruments, equities, fixed income, and derivatives catalog from CRIMS ORM",
			Icon:        "shield",
			Category:    "Market Data",
		},
		{
			Name:        "Cash Flow",
			DisplayName: "Cash Flow & Settlement",
			Description: "Dividend payments, coupon settlements, and cash movements from CRIMS ORM",
			Icon:        "dollar-sign",
			Category:    "Treasury",
		},
	}

	seederUUID := "113d0169-4819-42ff-968b-778f72af79e9"

	for _, req := range ormBOs {
		var existsCount int
		err := db.QueryRow("SELECT COUNT(*) FROM business_objects WHERE tenant_id = $1 AND name = $2", tenantID, req.Name).Scan(&existsCount)
		if err == nil && existsCount > 0 {
			log.Printf("BO '%s' already exists for tenant %s, skipping.\n", req.Name, tenantID)
			continue
		}

		bo, err := boService.CreateBusinessObject(ctx, secCtx, req, seederUUID)
		if err != nil {
			log.Printf("Failed to create BO '%s': %v\n", req.Name, err)
			continue
		}
		log.Printf("✅ Created BO '%s' (%s) with ID %s\n", bo.DisplayName, bo.Name, bo.ID)
	}

	fmt.Println("ORM Business Objects seeding finished successfully.")
}
