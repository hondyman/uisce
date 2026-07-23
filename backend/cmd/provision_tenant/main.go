package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

func main() {
	tenantName := flag.String("name", "", "Tenant Name")
	dbHost := flag.String("db-host", "localhost", "Database Host")
	dbPort := flag.String("db-port", "5432", "Database Port")
	dbUser := flag.String("db-user", "postgres", "Database User")
	dbPass := flag.String("db-pass", "postgres", "Database Password")
	centralDBName := flag.String("central-db", "alpha", "Central Database Name")
	flag.Parse()

	if *tenantName == "" {
		log.Fatal("Tenant name is required")
	}

	// Connect to Central DB
	centralConnStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", *dbUser, *dbPass, *dbHost, *dbPort, *centralDBName)
	centralDB, err := sql.Open("postgres", centralConnStr)
	if err != nil {
		log.Fatalf("Failed to connect to central DB: %v", err)
	}
	defer centralDB.Close()

	// Generate Tenant ID
	tenantID := uuid.New().String()
	safeName := strings.ToLower(strings.ReplaceAll(*tenantName, " ", "_"))
	tenantDBName := fmt.Sprintf("wealth_tenant_%s", safeName)

	log.Printf("Provisioning tenant: %s (ID: %s)", *tenantName, tenantID)
	log.Printf("Target Database: %s", tenantDBName)

	// 1. Create Tenant Database
	// We need to connect to 'postgres' database to create a new database
	adminConnStr := fmt.Sprintf("postgres://%s:%s@%s:%s/postgres?sslmode=disable", *dbUser, *dbPass, *dbHost, *dbPort)
	adminDB, err := sql.Open("postgres", adminConnStr)
	if err != nil {
		log.Fatalf("Failed to connect to admin DB: %v", err)
	}
	defer adminDB.Close()

	_, err = adminDB.Exec(fmt.Sprintf("CREATE DATABASE %s", tenantDBName))
	if err != nil {
		// Ignore if already exists for idempotency, but warn
		if strings.Contains(err.Error(), "already exists") {
			log.Printf("Database %s already exists, proceeding...", tenantDBName)
		} else {
			log.Fatalf("Failed to create database: %v", err)
		}
	} else {
		log.Println("✅ Created tenant database")
	}

	// 2. Apply Schema to Tenant Database
	// Connect to the new tenant DB
	tenantConnStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", *dbUser, *dbPass, *dbHost, *dbPort, tenantDBName)
	tenantDB, err := sql.Open("postgres", tenantConnStr)
	if err != nil {
		log.Fatalf("Failed to connect to tenant DB: %v", err)
	}
	defer tenantDB.Close()

	// Read schema file
	// Assuming running from backend root
	schemaPath := "internal/api/migrations/wealth_app_schema.sql"
	schemaContent, err := os.ReadFile(schemaPath)
	if err != nil {
		// Try absolute path or relative to cmd
		schemaContent, err = os.ReadFile("../../internal/api/migrations/wealth_app_schema.sql")
		if err != nil {
			log.Fatalf("Failed to read schema file: %v", err)
		}
	}

	_, err = tenantDB.Exec(string(schemaContent))
	if err != nil {
		log.Fatalf("Failed to apply schema: %v", err)
	}
	log.Println("✅ Applied wealth schema to tenant database")

	// 3. Register Tenant in Central DB
	insertQuery := `
		INSERT INTO platform.tenants (tenant_id, name, status, db_connection_string, has_dedicated_db)
		VALUES ($1, $2, 'active', $3, true)
		ON CONFLICT (tenant_id) DO NOTHING
	`
	_, err = centralDB.Exec(insertQuery, tenantID, *tenantName, tenantConnStr)
	if err != nil {
		log.Fatalf("Failed to register tenant: %v", err)
	}
	log.Println("✅ Registered tenant in platform.tenants")

	fmt.Printf("\nProvisioning Complete!\nTenant ID: %s\nDatabase: %s\n", tenantID, tenantDBName)
}
