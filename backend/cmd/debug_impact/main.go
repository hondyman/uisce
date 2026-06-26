package main

import (
	"context"
	"fmt"
	"log"

	"github.com/hondyman/semlayer/backend/internal/analytics"
	"github.com/hondyman/semlayer/backend/internal/config"
	"github.com/hondyman/semlayer/backend/internal/lineage"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"

	// Drivers
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/lib/pq"
)

func main() {
	// Load .env
	if err := godotenv.Load("backend/.env"); err != nil {
		log.Println("No .env file found, using defaults")
	}

	// Load config
	cfg, err := config.LoadConfig("backend/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to DB
	db, err := sqlx.Connect("pgx", cfg.DSN)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer db.Close()

	// Initialize DBLineageRepository
	lineage.NewDBLineageRepository(db)

	// 1. Get a valid Semantic Term ID that has edges
	var termID string
	err = db.QueryRow(`
		SELECT n.id 
		FROM catalog_node n
		JOIN catalog_edge e ON n.id = e.source_node_id OR n.id = e.target_node_id
		WHERE n.node_type_id = '820b942a-9c9e-4abc-acdc-84616db33098' 
		LIMIT 1
	`).Scan(&termID)
	if err != nil {
		fmt.Printf("Warning: Failed to find a semantic term WITH edges: %v. Falling back to any term.\n", err)
		err = db.QueryRow("SELECT id FROM catalog_node WHERE node_type_id = '820b942a-9c9e-4abc-acdc-84616db33098' LIMIT 1").Scan(&termID)
		if err != nil {
			log.Fatalf("Failed to get any semantic term from catalog_node: %v", err)
		}
	}
	fmt.Printf("Using Semantic Term ID: %s\n", termID)

	// 1b. Check relational counts
	var nodeCount, edgeCount int
	db.Get(&nodeCount, "SELECT count(*) FROM public.catalog_node")
	db.Get(&edgeCount, "SELECT count(*) FROM public.catalog_edge")
	fmt.Printf("Public Schema: %d catalog_nodes, %d catalog_edges\n", nodeCount, edgeCount)

	// 1c. Check edges for this specific term in both
	var relevantEdgesPublic int
	db.Get(&relevantEdgesPublic, "SELECT count(*) FROM public.catalog_edge WHERE source_node_id = $1 OR target_node_id = $1", termID)
	fmt.Printf("Edges for %s: Public=%d\n", termID, relevantEdgesPublic)

	// 2. Test the lineage repository
	fmt.Println("\nTesting Lineage Repository...")
	// Note: No graph initialization needed for relational storage

	// 3. Query Impact Service
	impactService := analytics.NewImpactService(db)
	impactGraph, err := impactService.GetImpactGraph(context.Background(), termID, "semantic_term", 1)
	if err != nil {
		log.Fatalf("Failed to get impact graph: %v", err)
	}

	fmt.Println("\n--- Impact Graph ---")
	fmt.Printf("Nodes: %d\n", len(impactGraph.Nodes))
	fmt.Printf("Edges: %d\n", len(impactGraph.Edges))

	summary, err := impactService.GetImpactSummary(context.Background(), impactGraph, termID)
	if err != nil {
		log.Fatalf("Failed to get impact summary: %v", err)
	}

	fmt.Println("\n--- Impact Summary ---")
	fmt.Printf("Total Impacted: %d\n", summary.TotalNodes)
	fmt.Printf("Explanation: %s\n", summary.Explanation)
	for typeName, count := range summary.NodesByType {
		fmt.Printf(" - Impacted Type: %s (Count: %d)\n", typeName, count)
	}
}

func printGraph(g *lineage.Graph) {
	if g == nil {
		fmt.Println("Graph is nil")
		return
	}
	fmt.Printf("Nodes: %d\n", len(g.Nodes))
	for _, n := range g.Nodes {
		fmt.Printf(" - [%s] %s (%s)\n", n.Type, n.Name, n.ID)
	}
	fmt.Printf("Edges: %d\n", len(g.Edges))
	for _, e := range g.Edges {
		fmt.Printf(" - %s -> %s (%s)\n", e.FromID, e.ToID, e.Type)
	}
	if len(g.Nodes) == 0 {
		fmt.Println("WARNING: Graph is empty! This explains the blank UI.")
	}
}
