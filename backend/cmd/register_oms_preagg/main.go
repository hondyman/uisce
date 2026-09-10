// One-off verification binary: registers a real pre-aggregation catalog node
// for the Execution BO's "Excel NPV" calculated term, using the same
// PreAggregationService the new /api/preaggregations HTTP handler calls.
// Confirms UpsertPreAggregation + GenerateDDL work end-to-end against the
// live catalog schema.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

const tenantID = "99e99e99-99e9-49e9-89e9-99e99e99e999"

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required")
	}
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer db.Close()

	graph := analytics.NewSemanticGraphService(db)
	if err := graph.Initialize(); err != nil {
		// Matches production behavior in api.go: Initialize() is
		// best-effort there too (schema drift on catalog_node_type;
		// tracked separately, not blocking for UpsertPreAggregation
		// which writes catalog_node directly via SQL).
		log.Printf("graph init warning (non-fatal, matches prod): %v", err)
	}
	resolver := analytics.NewBOContextResolver(db, graph)
	svc := analytics.NewPreAggregationService(db, resolver, graph)

	ctx := context.Background()
	desc, err := svc.UpsertPreAggregation(ctx, models.UpsertPreAggRequest{
		TenantID:    tenantID,
		BOName:      "Execution",
		Name:        "execution_npv_rollup",
		Description: "Hot-tier StarRocks rollup of Excel NPV for Execution, fed via CDC (cdc_service -> Kafka -> stream_loader)",
		Terms:       []string{"PlacementID", "BrokerID"},
		Calculations: []string{"Excel NPV"},
		GroupBy:      []string{"PlacementID"},
		Materialization: models.MaterializationConfig{
			Type:       "materialized_view",
			TargetName: "mv_execution_npv",
		},
		RefreshStrategy:        "interval",
		RefreshIntervalMinutes: 15,
	})
	if err != nil {
		log.Fatalf("upsert failed: %v", err)
	}
	fmt.Printf("Upserted pre-aggregation: id=%s name=%s bo=%s target=%s\n", desc.ID, desc.Name, desc.BOName, desc.TargetName)

	ddl, err := svc.GenerateDDL(ctx, desc.ID, "starrocks")
	if err != nil {
		log.Fatalf("generate DDL failed: %v", err)
	}
	fmt.Println("--- Generated StarRocks DDL ---")
	fmt.Println(ddl)
}
