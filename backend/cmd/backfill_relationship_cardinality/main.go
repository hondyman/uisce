// Command backfill_relationship_cardinality re-resolves cardinality on
// existing entity_relationship rows through the real PK/unique-constraint
// based CardinalityResolver (backend/internal/api/cardinality_resolver.go),
// and detects existing junction/associative tables (e.g. order_items
// between orders and products) to add the MANY_TO_MANY relationship that
// direction-only FK discovery never produced.
//
// The actual logic lives in backend/internal/api/cardinality_backfill.go
// (ReresolveExistingRelationships, BackfillJunctionRelationships) so it can
// be exercised directly by tests — see
// backend/internal/api/cardinality_backfill_integration_test.go, which
// proves idempotency by running BackfillJunctionRelationships twice against
// a real database (env-gated on TEST_DATABASE_URL; skipped otherwise).
package main

import (
	"context"
	"database/sql"
	"log"
	"os"

	"github.com/hondyman/uisce/backend/internal/api"
	_ "github.com/lib/pq"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	resolver := api.NewCardinalityResolver(db)

	reresolved, err := api.ReresolveExistingRelationships(ctx, db, resolver)
	if err != nil {
		log.Fatalf("failed to re-resolve existing relationships: %v", err)
	}
	log.Printf("re-resolved cardinality on %d existing entity_relationship rows", reresolved)

	junctions, err := api.BackfillJunctionRelationships(ctx, db, resolver)
	if err != nil {
		log.Fatalf("failed to backfill junction relationships: %v", err)
	}
	log.Printf("added/updated %d synthesized MANY_TO_MANY junction relationships", junctions)
}
