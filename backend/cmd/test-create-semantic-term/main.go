package main

import (
	"context"
	"fmt"
	"log"

	"github.com/hondyman/semlayer/backend/internal/analytics"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	connStr := "postgresql://postgres:postgres@localhost:5432/alpha?sslmode=disable"
	db, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	service := analytics.NewSemanticMappingService(db, nil, nil, nil, nil, nil)

	tenantID := "910638ba-a459-4a3f-bb2d-78391b0595f6"
	datasourceID := "a2b1c3d4-e5f6-4a5b-9c8d-7e6f5a4b3c2d"
	columnID := "f8697762-6d9e-520e-ad74-a7eb0b2bc10b" // geographic_region

	req := &analytics.ApplyEnrichmentRequest{
		TenantID:     tenantID,
		DatasourceID: datasourceID,
		ColumnID:     columnID,
		Proposal: &analytics.EnrichmentProposal{
			SemanticTermName: "ZIP_CODE_MASKED",
			SemanticTermType: "Dimension",
			BusinessTermName: "Zip Code Masked",
			DomainHierarchy:  []string{"Enterprise", "Security"},
			Confidence:       0.99,
			Reasoning:        "Manual test creation",
			ColumnName:       "geographic_region",
		},
	}

	fmt.Printf("Attempting to apply enrichment: %s\n", req.Proposal.SemanticTermName)
	ids, err := service.ApplyEnrichment(context.Background(), req)
	if err != nil {
		log.Fatalf("Failed to apply enrichment: %v", err)
	}

	fmt.Printf("Successfully applied enrichment!\n")
	for k, v := range ids {
		fmt.Printf("  %s: %s\n", k, v)
	}
}
