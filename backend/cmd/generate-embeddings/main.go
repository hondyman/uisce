package main

import (

	"flag"
	"fmt"
	"log"
	"os"


	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	tenantID := flag.String("tenant", "", "Tenant ID (UUID)")
	datasourceID := flag.String("datasource", "", "Datasource ID (UUID)")
	dbURL := flag.String("db", "", "Database connection string (optional, defaults to env var DATABASE_URL)")
	apiKey := flag.String("api-key", "", "Gemini API key (optional, defaults to env var GEMINI_API_KEY)")

	flag.Parse()

	if *tenantID == "" || *datasourceID == "" {
		fmt.Println("Usage: generate-embeddings --tenant=<TENANT_ID> --datasource=<DATASOURCE_ID>")
		flag.PrintDefaults()
		os.Exit(1)
	}

	databaseURL := *dbURL
	if databaseURL == "" {
		databaseURL = os.Getenv("DATABASE_URL")
		if databaseURL == "" {
			log.Fatal("Database URL not provided. Set DATABASE_URL env var or use --db flag")
		}
	}

	db, err := sqlx.Connect("postgres", databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	apiKeyValue := *apiKey
	if apiKeyValue == "" {
		apiKeyValue = os.Getenv("GEMINI_API_KEY")
	}

	// llmProvider := llm.NewGeminiProvider(apiKeyValue, "")
	// embeddingService := services.NewCatalogEmbeddingService(db, llmProvider)

	fmt.Printf("Starting embedding generation for tenant %s, datasource %s\n", *tenantID, *datasourceID)


	// err = embeddingService.GenerateEmbeddingsForTenant(ctx, *tenantID, *datasourceID)
	// if err != nil {
	// 	log.Fatalf("Failed to generate embeddings: %v", err)
	// }

	fmt.Println("✅ Embedding generation complete!")
}
