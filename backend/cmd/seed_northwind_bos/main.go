package main

import (
	"context"
	"log"
	"os"

	metadata "github.com/hondyman/semlayer/backend/internal/metadata"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/internal/security"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	// Connect to database
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@100.84.126.19:5432/alpha?sslmode=disable"
	}

	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Get uisce tenant specifically
	tenantQuery := `SELECT id FROM tenants WHERE name = 'uisce' LIMIT 1`
	var tenantID string
	err = db.GetContext(ctx, &tenantID, tenantQuery)
	if err != nil {
		log.Fatalf("Uisce tenant not found in database: %v", err)
	}

	log.Printf("Seeding Northwind BOs for tenant: %s\n", tenantID)

	boService := metadata.NewBusinessObjectService(db, nil, nil, nil)

	// Define all Northwind BOs
	northwindBOs := []models.CreateBusinessObjectRequest{
		{
			Name:        "Customer",
			DisplayName: "Customers",
			Description: "Customers and their demographics, linking to orders and demographics",
			Icon:        "users",
			Category:    "Sales",
		},
		{
			Name:        "Employee",
			DisplayName: "Employees",
			Description: "Staff hierarchy and territories, supporting order assignments",
			Icon:        "users",
			Category:    "HR",
		},
		{
			Name:        "Supplier",
			DisplayName: "Suppliers",
			Description: "Tracks vendors for product sourcing",
			Icon:        "truck",
			Category:    "Procurement",
		},
		{
			Name:        "Product",
			DisplayName: "Products",
			Description: "Manages inventory items, categories, and pricing",
			Icon:        "box",
			Category:    "Inventory",
		},
		{
			Name:        "Order",
			DisplayName: "Orders",
			Description: "Captures sales transactions, shipping, and customer details",
			Icon:        "shopping-cart",
			Category:    "Sales",
		},
		{
			Name:        "Order Detail",
			DisplayName: "Order Details",
			Description: "Line items within orders",
			Icon:        "list",
			Category:    "Sales",
		},
		{
			Name:        "Shipper",
			DisplayName: "Shippers",
			Description: "Logistics providers for order fulfillment",
			Icon:        "truck",
			Category:    "Logistics",
		},
		{
			Name:        "Territory",
			DisplayName: "Territories",
			Description: "Geographic segmentation for employees and sales",
			Icon:        "map",
			Category:    "Geography",
		},
	}

	for _, boReq := range northwindBOs {
		// Check if already exists
		checkQuery := `SELECT COUNT(*) FROM business_objects WHERE tenant_id = $1 AND name = $2`
		var count int
		_ = db.GetContext(ctx, &count, checkQuery, tenantID, boReq.Name)

		if count > 0 {
			log.Printf("✓ %s already exists, skipping\n", boReq.Name)
			continue
		}

		// Create BO
		secCtx := &security.Context{TenantID: tenantID}
		bo, err := boService.CreateBusinessObject(ctx, secCtx, boReq, "00000000-0000-0000-0000-000000000000")
		if err != nil {
			log.Printf("✗ Failed to create %s: %v\n", boReq.Name, err)
			continue
		}

		log.Printf("✓ Created %s (ID: %s)\n", boReq.Name, bo.ID)
	}

	log.Println("\n✓ Northwind BO seed complete!")
}
