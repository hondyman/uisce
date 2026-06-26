package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/lineage"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/segmentio/kafka-go"
)

type CatalogSyncEvent struct {
	BOID          string   `json:"bo_id"`
	BOKey         string   `json:"bo_key"`
	Name          string   `json:"name"`
	DisplayName   string   `json:"display_name"`
	DriverTableID string   `json:"driver_table_id"`
	SelectedTerms []string `json:"selected_terms"`
	TenantID      string   `json:"tenant_id"`
	DatasourceID  string   `json:"datasource_id"`
}

func main() {
	log.Println("Starting Catalog Sync Worker...")

	// Connect to PostgreSQL
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	log.Println("Connected to PostgreSQL")

	// Kafka configuration
	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	if kafkaBrokers == "" {
		kafkaBrokers = "localhost:9092"
	}
	topic := "BusinessObject.CatalogSync"
	groupID := "catalog-sync-worker"

	log.Printf("Connecting to Kafka at %s, topic: %s", kafkaBrokers, topic)

	// Create Kafka reader (consumer)
	// segmentio/kafka-go automatically handles connection establishment and re-connection
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{kafkaBrokers},
		GroupID:  groupID,
		Topic:    topic,
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})
	defer reader.Close()

	log.Println("Kafka reader created")

	// Handle shutdown gracefully
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	// Context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize lineage repo for catalog sync
	lineageRepo := lineage.NewDBLineageRepository(db)
	// Note: No graph initialization needed for relational storage

	// Handle signals in a separate goroutine
	go func() {
		sig := <-sigchan
		log.Printf("Caught signal %v: terminating", sig)
		cancel()
		reader.Close()
	}()

	log.Println("Starting consumption loop...")

	for {
		m, err := reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				// Context cancelled, clean shutdown
				break
			}
			log.Printf("Error reading message: %v", err)
			// Small backoff on error
			time.Sleep(time.Second)
			continue
		}

		log.Printf("Received message from topic %s partition %d offset %d", m.Topic, m.Partition, m.Offset)

		if err := processCatalogSyncEvent(db, lineageRepo, m.Value); err != nil {
			log.Printf("Error processing event: %v", err)
			// In a real production system, you might want to NACK or send to DLQ here
			// For now, we log and continue (at-most-once semantics effectively if we don't retry locally)
		} else {
			log.Printf("Successfully processed catalog sync for BO")
			// No explicit commit needed if using auto-commit (default),
			// but if using explicit commits: reader.CommitMessages(ctx, m)
		}
	}

	log.Println("Catalog Sync Worker shutdown complete")
}

func processCatalogSyncEvent(db *sqlx.DB, lineageRepo lineage.LineageRepository, data []byte) error {
	var event CatalogSyncEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return fmt.Errorf("unmarshal event: %w", err)
	}

	log.Printf("Processing catalog sync for BO: %s (key: %s)", event.BOID, event.BOKey)

	ctx := context.Background()
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get BO node type ID
	var boNodeTypeID string
	err = tx.GetContext(ctx, &boNodeTypeID, `
		SELECT id FROM catalog_node_type 
		WHERE catalog_type_name = 'business_object'
	`)
	if err != nil {
		return fmt.Errorf("get node type: %w", err)
	}

	// Upsert catalog_node
	_, err = tx.ExecContext(ctx, `
		INSERT INTO catalog_node (
			id, node_name, node_type_id, tenant_id, tenant_datasource_id,
			properties, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			jsonb_build_object(
				'bo_key', $6,
				'display_name', $7,
				'driver_table_id', $8
			),
			NOW(), NOW()
		)
		ON CONFLICT (id) DO UPDATE SET
			node_name = EXCLUDED.node_name,
			properties = EXCLUDED.properties,
			updated_at = NOW()
	`, event.BOID, event.BOKey, boNodeTypeID, event.TenantID, event.DatasourceID,
		event.BOKey, event.DisplayName, event.DriverTableID)
	if err != nil {
		return fmt.Errorf("upsert node: %w", err)
	}

	// Sync node to lineage repository (using relational storage)
	if lineageRepo != nil {
		meta := map[string]interface{}{
			"bo_key":          event.BOKey,
			"display_name":    event.DisplayName,
			"driver_table_id": event.DriverTableID,
		}
		metaBytes, _ := json.Marshal(meta)
		node := lineage.LineageNode{
			ID:       event.BOID,
			Type:     "business_object",
			Name:     event.BOKey,
			TenantID: &event.TenantID,
			Metadata: metaBytes,
			Env:      "dev",
		}
		if err := lineageRepo.UpsertNode(ctx, node); err != nil {
			log.Printf("Warning: Failed to sync BO node %s to lineage repository: %v", event.BOID, err)
		}
	}

	log.Printf("Upserted catalog node for BO: %s", event.BOID)

	// Delete old edges
	_, err = tx.ExecContext(ctx, `
		DELETE FROM catalog_edge
		WHERE target_node_id = $1 AND edge_type_name = 'member_of'
	`, event.BOID)
	if err != nil {
		return fmt.Errorf("delete edges: %w", err)
	}

	// Create new edges
	for _, termID := range event.SelectedTerms {
		edgeID := uuid.New().String()
		_, err = tx.ExecContext(ctx, `
			INSERT INTO catalog_edge (
				id, source_node_id, target_node_id, edge_type_name,
				tenant_id, tenant_datasource_id, created_at, updated_at
			) VALUES ($1, $3, $2, 'member_of', $4, $5, NOW(), NOW())
		`, edgeID, event.BOID, termID, event.TenantID, event.DatasourceID)
		if err != nil {
			return fmt.Errorf("create edge to term %s: %w", termID, err)
		}

		// Sync edge to lineage repository (using relational storage)
		if lineageRepo != nil {
			edge := lineage.LineageEdge{
				FromID:   termID,
				ToID:     event.BOID,
				Type:     "member_of",
				TenantID: &event.TenantID,
				Env:      "dev",
			}
			if err := lineageRepo.UpsertEdge(ctx, edge); err != nil {
				log.Printf("Warning: Failed to sync edge %s -> %s to lineage repository: %v", event.BOID, termID, err)
			}
		}
	}

	log.Printf("Created %d semantic term edges", len(event.SelectedTerms))

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
