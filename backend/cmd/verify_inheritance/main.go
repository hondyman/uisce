package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/hondyman/semlayer/backend/pkg/semantic"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL required")
	}

	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}

	service := semantic.NewService(db)
	ctx := context.Background()
	tenantID := "123e4567-e89b-12d3-a456-426614174000" // Valid UUID

	// Cleanup first
	db.Exec("DELETE FROM semantic_cubes_v2 WHERE tenant_id = $1", tenantID)

	// 1. Create Core Cube
	fmt.Println("Creating Core Cube...")
	coreCube := &semantic.Cube{
		TenantID:    tenantID,
		Name:        "core_employee",
		DisplayName: "Core Employee",
		Description: "System Core Employee Model",
		SQL:         "SELECT * FROM employees",
		Status:      "active",
		IsSystem:    true,
		CreatedBy:   tenantID, // Use valid UUID
	}
	if err := service.CreateCube(ctx, coreCube); err != nil {
		log.Fatalf("Failed to create core cube: %v", err)
	}

	// Add Core Dimension
	coreDim := &semantic.Dimension{
		CubeID:      coreCube.ID,
		Name:        "email",
		DisplayName: "Email Address",
		Type:        "string",
		SQL:         "email",
		Shown:       true,
	}
	service.CreateDimension(ctx, coreDim)

	// 2. Create Custom Cube (extending Core)
	fmt.Println("Creating Custom Cube...")
	customCube := &semantic.Cube{
		TenantID:     tenantID,
		Name:         "my_employee",
		DisplayName:  "My Employee",
		Status:       "active",
		SourceCubeID: &coreCube.ID, // Extend core
		CreatedBy:    tenantID,     // Use valid UUID
	}
	if err := service.CreateCube(ctx, customCube); err != nil {
		log.Fatalf("Failed to create custom cube: %v", err)
	}

	// Add Custom Dimension (new)
	customDim := &semantic.Dimension{
		CubeID:      customCube.ID,
		Name:        "department",
		DisplayName: "Department",
		Type:        "string",
		SQL:         "department",
		Shown:       true,
	}
	if err := service.CreateDimension(ctx, customDim); err != nil {
		log.Fatalf("Failed to create custom dimension: %v", err)
	}

	// Add Custom Dimension (override) - Change display name of email
	overrideDim := &semantic.Dimension{
		CubeID:      customCube.ID,
		Name:        "email",
		DisplayName: "Work Email", // Changed from "Email Address"
		Type:        "string",
		SQL:         "company_email", // Changed SQL
		Shown:       true,
	}
	if err := service.CreateDimension(ctx, overrideDim); err != nil {
		log.Fatalf("Failed to create override dimension: %v", err)
	}

	// 3. Verify Merging
	fmt.Println("Verifying Merged Cube...")
	mergedCube, err := service.GetCube(ctx, tenantID, "my_employee")
	if err != nil {
		log.Fatalf("Failed to get merged cube: %v", err)
	}

	// Check Basic Props
	if mergedCube.DisplayName != "My Employee" {
		log.Fatalf("Expected DisplayName 'My Employee', got '%s'", mergedCube.DisplayName)
	}
	if mergedCube.SQL != "SELECT * FROM employees" { // Should inherit from Core since not overridden
		log.Fatalf("Expected inherited SQL 'SELECT * FROM employees', got '%s'", mergedCube.SQL)
	}

	// Check Dimensions
	fmt.Printf("Found dimensions: %d\n", len(mergedCube.Dimensions))
	dims := make(map[string]semantic.Dimension)
	for _, d := range mergedCube.Dimensions {
		fmt.Printf(" - %s (%s)\n", d.Name, d.DisplayName)
		dims[d.Name] = d
	}

	// "department" (Custom)
	if _, ok := dims["department"]; !ok {
		log.Fatal("Missing custom dimension 'department'")
	}

	// "email" (Overridden)
	emailDim, ok := dims["email"]
	if !ok {
		log.Fatal("Missing core dimension 'email'")
	}
	if emailDim.DisplayName != "Work Email" {
		log.Fatalf("Expected overridden display name 'Work Email', got '%s'", emailDim.DisplayName)
	}
	if emailDim.SQL != "company_email" {
		log.Fatalf("Expected overridden SQL 'company_email', got '%s'", emailDim.SQL)
	}

	fmt.Println("SUCCESS: Core vs Custom inheritance verified!")

	// Cleanup
	db.Exec("DELETE FROM semantic_cubes_v2 WHERE tenant_id = $1", tenantID)
}
