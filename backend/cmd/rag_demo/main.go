package main

import (
	"fmt"
)

func main() {
	fmt.Println("RAG Demo is currently disabled due to refactoring.")
	/*
	// 1. Setup Database Connection
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		dsn = "postgres://ws:ws_pass@localhost:5432/wealthstream_dev?sslmode=disable"
	}
	
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping db: %v", err)
	}
	fmt.Println("✅ Connected to database")

	// 2. Initialize Components
	store := rag.NewStore(db)
	sanitizer := &rag.SanitizationClient{BaseURL: "http://localhost:8081"}
	embedder := rag.DemoEmbedder{Dim: 1536}
	tokenizer := rag.WhitespaceTokenizer{}

	tenantID := "demo_tenant"
	clientID := "client_001"
	modelID := "demo-model-v1"
	snapshotID := fmt.Sprintf("snap_%d", time.Now().Unix())

	// 3. Provision Tenant Schema
	ctx := context.Background()
	fmt.Printf("🔄 Provisioning schema for tenant: %s...\n", tenantID)
	if err := store.ProvisionTenant(ctx, tenantID); err != nil {
		log.Fatalf("failed to provision tenant: %v", err)
	}
	fmt.Println("✅ Tenant schema provisioned")

	// 4. Prepare Document
	docID := "doc_ips_2024"
	docText := `
		Investment Policy Statement (IPS) for John Doe.
		Section: Risk Factors.
		The client has a high risk tolerance and is willing to accept significant volatility.
		Concentration risk in technology sector is noted.
		Account number: 123-456-789.
		SSN: 987-65-4321.
		
		Section: Asset Allocation.
		Target allocation is 80% Equities, 20% Fixed Income.
		Rebalancing should occur quarterly.
	`

	// 5. Chunk Document
	fmt.Println("🔄 Chunking document...")
	chunks := rag.ChunkDocument(docID, docText, tokenizer, 50, 10)
	fmt.Printf("✅ Generated %d chunks\n", len(chunks))

	// Add section metadata for hybrid search demo
	for i := range chunks {
		if chunks[i].Metadata == nil {
			chunks[i].Metadata = make(map[string]any)
		}
		if i < 2 {
			chunks[i].Metadata["section"] = "Risk Factors"
		} else {
			chunks[i].Metadata["section"] = "Asset Allocation"
		}
	}

	// 6. Ingest Chunks (Sanitize -> Embed -> Upsert)
	fmt.Println("🔄 Ingesting chunks (Sanitize -> Embed -> Upsert)...")
	if err := rag.IngestChunks(ctx, store, tenantID, clientID, modelID, snapshotID, chunks, sanitizer, embedder); err != nil {
		log.Fatalf("failed to ingest chunks: %v", err)
	}
	fmt.Println("✅ Ingestion complete")

	// 7. Perform Hybrid Search
	query := "What is the client's risk tolerance?"
	fmt.Printf("\n🔎 Searching for: %q\n", query)
	
	hits, err := rag.Search(ctx, store, sanitizer, embedder, rag.SearchRequest{
		TenantID: tenantID,
		ClientID: clientID,
		Query:    query,
		Filters:  map[string]any{"section": "Risk Factors"}, // Hybrid filter
		Limit:    3,
		ModelID:  modelID,
	})
	if err != nil {
		log.Fatalf("search failed: %v", err)
	}

	fmt.Printf("✅ Found %d hits:\n", len(hits))
	for i, hit := range hits {
		fmt.Printf("[%d] Score: %.4f | ID: %s\n", i+1, hit.Similarity, hit.ChunkID)
		fmt.Printf("    Text: %s\n", hit.Text)
		fmt.Printf("    Meta: %v\n", hit.Metadata)
		fmt.Println("    ---")
	}
	*/
}
