package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"strings"

	_ "github.com/lib/pq"

	"github.com/hondyman/semlayer/backend/internal/audit"
	"github.com/hondyman/semlayer/backend/internal/sync"
	"github.com/hondyman/semlayer/backend/internal/trino"
	_ "github.com/lib/pq"
	kafka "github.com/segmentio/kafka-go"
	_ "github.com/trinodb/trino-go-client/trino"
)

// DebeziumEvent represents a CDC event from Debezium
type DebeziumEvent struct {
	Payload struct {
		Before map[string]interface{} `json:"before"`
		After  map[string]interface{} `json:"after"`
		Source struct {
			Table    string `json:"table"`
			Schema   string `json:"schema"`
			Database string `json:"db"`
		} `json:"source"`
		Op   string `json:"op"` // c=create, u=update, d=delete, r=read (snapshot)
		TsMs int64  `json:"ts_ms"`
	} `json:"payload"`
}

// SyncWorkerConfig holds configuration for all sync workers
type SyncWorkerConfig struct {
	KafkaBrokers       string
	DebeziumServerName string
	PostgresURL        string
	HasuraURL          string
	HasuraAdminSecret  string
	SupersetURL        string
	SupersetUsername   string
	SupersetPassword   string
	StarRocksURL       string
	StarRocksUser      string
	StarRocksPassword  string
	TrinoDSN           string
}

func main() {
	config := SyncWorkerConfig{
		KafkaBrokers:       getEnv("KAFKA_BROKERS", "localhost:9092"),
		DebeziumServerName: getEnv("DEBEZIUM_SERVER_NAME", "alpha"),
		PostgresURL:        getEnv("POSTGRES_URL", "postgres://postgres:postgres@localhost:5432/alpha"),
		HasuraURL:          getEnv("HASURA_URL", "http://localhost:8080"),
		HasuraAdminSecret:  getEnv("HASURA_ADMIN_SECRET", "myadminsecretkey"),
		SupersetURL:        getEnv("SUPERSET_URL", "http://localhost:8088"),
		SupersetUsername:   getEnv("SUPERSET_USERNAME", "admin"),
		SupersetPassword:   getEnv("SUPERSET_PASSWORD", "admin"),
		StarRocksURL:       getEnv("STARROCKS_URL", "localhost:9030"),
		StarRocksUser:      getEnv("STARROCKS_USER", "root"),
		StarRocksPassword:  getEnv("STARROCKS_PASSWORD", ""),
		TrinoDSN:           getEnv("TRINO_DSN", "http://user@trino:8080?catalog=iceberg&schema=audit"),
	}

	log.Println("🚀 Starting Security Sync Worker")
	log.Printf("Kafka Brokers: %s", config.KafkaBrokers)

	// Connect to PostgreSQL
	db, err := sql.Open("postgres", config.PostgresURL)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	// Initialize sync workers
	pgWorker := sync.NewPostgreSQLSyncWorker(db)
	hasuraWorker := sync.NewHasuraSyncWorker(config.HasuraURL, config.HasuraAdminSecret, db)
	supersetWorker := sync.NewSupersetSyncWorker(config.SupersetURL, config.SupersetUsername, config.SupersetPassword, db)
	starrocksWorker, err := sync.NewStarRocksSyncWorker(config.StarRocksURL, config.StarRocksUser, config.StarRocksPassword)
	if err != nil {
		log.Printf("Warning: Failed to connect to StarRocks: %v", err)
	} else {
		defer starrocksWorker.Close()
	}

	// Initialize Trino connection for Audit
	var trinoAuditService *audit.TrinoAuditService
	if config.TrinoDSN != "" {
		trinoDB, err := sql.Open("trino", config.TrinoDSN)
		if err != nil {
			log.Printf("Warning: Failed to create Trino DB handle: %v", err)
		} else {
			if err := trinoDB.Ping(); err != nil {
				log.Printf("Warning: Failed to connect to Trino: %v", err)
			} else {
				log.Println("✅ Connected to Trino for Auditing")
				trinoAuditService = audit.NewTrinoAuditService(trinoDB)
			}
		}
	}

	// Initialize Bitemporal CDC Worker
	var bitemporalWorker *sync.BitemporalCDCWorker
	if config.TrinoDSN != "" {
		trinoClient, err := trino.NewClient(config.TrinoDSN)
		if err != nil {
			log.Printf("Warning: Failed to create Trino client for bitemporal tracking: %v", err)
		} else {
			log.Println("✅ Connected to Trino for Bitemporal Tracking")
			bitemporalTracker := audit.NewBitemporalTracker(trinoClient)
			bitemporalWorker = sync.NewBitemporalCDCWorker(bitemporalTracker)
		}
	}

	// Initialize Tenant Worker
	tenantWorker := sync.NewTenantWorker(db, trinoAuditService)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down gracefully...")
		cancel()
	}()

	// Start sync workers
	errChan := make(chan error, 4)

	go func() {
		errChan <- startPostgreSQLWorker(ctx, config, pgWorker)
	}()

	go func() {
		errChan <- startHasuraWorker(ctx, config, hasuraWorker)
	}()

	go func() {
		errChan <- startSupersetWorker(ctx, config, supersetWorker)
	}()

	go func() {
		errChan <- startStarRocksWorker(ctx, config, starrocksWorker)
	}()

	go func() {
		errChan <- startTenantWorker(ctx, config, tenantWorker)
	}()

	if bitemporalWorker != nil {
		go func() {
			errChan <- startBitemporalWorker(ctx, config, bitemporalWorker)
		}()
	}

	// Wait for error or shutdown
	select {
	case err := <-errChan:
		log.Fatalf("Worker error: %v", err)
	case <-ctx.Done():
		log.Println("Shutdown complete")
	}
}

// startPostgreSQLWorker syncs role changes to PostgreSQL RLS
func startPostgreSQLWorker(ctx context.Context, config SyncWorkerConfig, worker *sync.PostgreSQLSyncWorker) error {
	topics := []string{
		fmt.Sprintf("%s.public.roles", config.DebeziumServerName),
		fmt.Sprintf("%s.public.user_roles", config.DebeziumServerName),
	}
	return consumeTopics(ctx, config.KafkaBrokers, "sync-worker-postgres", topics, func(event DebeziumEvent) error {
		log.Printf("[PostgreSQL] Processing %s event for table %s", event.Payload.Op, event.Payload.Source.Table)

		switch event.Payload.Source.Table {
		case "roles":
			if event.Payload.Op == "c" || event.Payload.Op == "u" || event.Payload.Op == "r" {
				return worker.SyncRole(ctx, event.Payload.After)
			}
		case "user_roles":
			if event.Payload.Op == "c" || event.Payload.Op == "r" {
				userID := event.Payload.After["user_id"].(string)
				roleID := event.Payload.After["role_id"].(string)
				return worker.AssignUserToRole(ctx, userID, roleID)
			} else if event.Payload.Op == "d" {
				userID := event.Payload.Before["user_id"].(string)
				roleID := event.Payload.Before["role_id"].(string)
				return worker.RevokeUserFromRole(ctx, userID, roleID)
			}
		}

		return nil
	})
}

// startHasuraWorker syncs role changes to Hasura permissions
func startHasuraWorker(ctx context.Context, config SyncWorkerConfig, worker *sync.HasuraSyncWorker) error {
	topics := []string{
		fmt.Sprintf("%s.public.roles", config.DebeziumServerName),
		fmt.Sprintf("%s.public.user_roles", config.DebeziumServerName),
	}
	return consumeTopics(ctx, config.KafkaBrokers, "sync-worker-hasura", topics, func(event DebeziumEvent) error {
		log.Printf("[Hasura] Processing %s event for table %s", event.Payload.Op, event.Payload.Source.Table)

		switch event.Payload.Source.Table {
		case "roles":
			if event.Payload.Op == "c" || event.Payload.Op == "u" || event.Payload.Op == "r" {
				return worker.SyncRole(ctx, event.Payload.After)
			}
		case "user_roles":
			if event.Payload.Op == "c" || event.Payload.Op == "d" {
				userID := event.Payload.After["user_id"].(string)
				return worker.InvalidateUserJWT(ctx, userID)
			}
		}

		return nil
	})
}

// startSupersetWorker syncs role changes to Superset
func startSupersetWorker(ctx context.Context, config SyncWorkerConfig, worker *sync.SupersetSyncWorker) error {
	topics := []string{
		fmt.Sprintf("%s.public.roles", config.DebeziumServerName),
	}
	return consumeTopics(ctx, config.KafkaBrokers, "sync-worker-superset", topics, func(event DebeziumEvent) error {
		log.Printf("[Superset] Processing %s event for table %s", event.Payload.Op, event.Payload.Source.Table)

		switch event.Payload.Source.Table {
		case "roles":
			if event.Payload.Op == "c" || event.Payload.Op == "u" || event.Payload.Op == "r" {
				return worker.SyncRole(ctx, event.Payload.After)
			}
		}

		return nil
	})
}

// startStarRocksWorker syncs role changes to StarRocks
func startStarRocksWorker(ctx context.Context, config SyncWorkerConfig, worker *sync.StarRocksSyncWorker) error {
	topics := []string{
		fmt.Sprintf("%s.public.roles", config.DebeziumServerName),
		fmt.Sprintf("%s.public.user_roles", config.DebeziumServerName),
	}
	return consumeTopics(ctx, config.KafkaBrokers, "sync-worker-starrocks", topics, func(event DebeziumEvent) error {
		log.Printf("[StarRocks] Processing %s event for table %s", event.Payload.Op, event.Payload.Source.Table)

		switch event.Payload.Source.Table {
		case "roles":
			if event.Payload.Op == "c" || event.Payload.Op == "u" || event.Payload.Op == "r" {
				return worker.SyncRole(ctx, event.Payload.After)
			}
		case "user_roles":
			if event.Payload.Op == "c" || event.Payload.Op == "r" {
				userID := event.Payload.After["user_id"].(string)
				roleID := event.Payload.After["role_id"].(string)
				return worker.AssignUserToRole(ctx, userID, roleID)
			}
		}

		return nil
	})
}

// startTenantWorker syncs tenant lifecycle events
func startTenantWorker(ctx context.Context, config SyncWorkerConfig, worker *sync.TenantWorker) error {
	topics := []string{
		fmt.Sprintf("%s.public.tenants", config.DebeziumServerName),
	}
	return consumeTopics(ctx, config.KafkaBrokers, "sync-worker-tenants", topics, func(event DebeziumEvent) error {
		// Log detailed event for debugging
		// log.Printf("[TenantWorker] Event: op=%s table=%s", event.Payload.Op, event.Payload.Source.Table)

		if event.Payload.Source.Table != "tenants" {
			return nil
		}

		// Handle Deletion
		if event.Payload.Op == "d" {
			// In delete event, 'after' is null, 'before' has data
			tenantID, ok := event.Payload.Before["id"].(string)
			if !ok {
				log.Printf("[TenantWorker] Warning: Delete event missing ID")
				return nil
			}

			// Check if it was Gold Copy from the 'before' state to avoid query failure
			isGoldCopy, _ := event.Payload.Before["gold_copy"].(bool)
			if isGoldCopy {
				log.Printf("[TenantWorker] Skipped cascading delete for Gold Copy tenant %s", tenantID)
				return nil
			}

			return worker.DeleteTenantResources(ctx, tenantID)
		}

		// Handle Update (Inactivation)
		if event.Payload.Op == "u" {
			tenantID, ok := event.Payload.After["id"].(string)
			if !ok {
				return nil
			}

			isActiveAfter, _ := event.Payload.After["is_active"].(bool)
			isActiveBefore, _ := event.Payload.Before["is_active"].(bool)

			// Trigger only if changed from true to false
			if !isActiveAfter && isActiveBefore {
				return worker.InactivateTenantResources(ctx, tenantID)
			}
		}

		return nil
	})
}

// consumeTopics is a generic Kafka consumer
func consumeTopics(ctx context.Context, brokers string, groupID string, topics []string, handler func(DebeziumEvent) error) error {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     strings.Split(brokers, ","),
		GroupID:     groupID,
		GroupTopics: topics,
		MinBytes:    10e3, // 10KB
		MaxBytes:    10e6, // 10MB
	})
	defer r.Close()

	log.Printf("✅ [%s] Worker starting on topics: %v", groupID, topics)

	for {
		m, err := r.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil // Context cancelled
			}
			log.Printf("❌ Failed to fetch message: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		// Parse Debezium event
		var event DebeziumEvent
		// Debezium messages often have Key/Value. We need Value.
		// Value might be null for deletions (tombstones), but Debezium usually sends payload for deletes in previous message or body depending on config.
		// Standard Debezium Envelope has "payload" and "schema".
		// Our struct DebeziumEvent expects "payload" at root?
		// Wait, my DebeziumEvent struct has `json:"payload"`.
		// If Debezium sends `{ "schema": ..., "payload": ... }`, then Unmarshal will work.
		if len(m.Value) == 0 {
			continue // Skip tombstones
		}

		if err := json.Unmarshal(m.Value, &event); err != nil {
			log.Printf("❌ Failed to parse event: %v", err)
			// Ensure we don't block on bad messages
			if err := r.CommitMessages(ctx, m); err != nil {
				log.Printf("❌ Failed to commit bad message: %v", err)
			}
			continue
		}

		// Debezium sometimes sends messages with Op="" or null payload if it's a heartbeat or schema change
		// Check if we have data
		if event.Payload.Op == "" && event.Payload.Source.Table == "" {
			// Possibly not a data change event
			r.CommitMessages(ctx, m)
			continue
		}

		// Process event
		if err := handler(event); err != nil {
			log.Printf("❌ Failed to process event: %v", err)
			// We delay but do NOT commit so it will be retried
			time.Sleep(5 * time.Second)
			continue
		}

		// Acknowledge successful processing
		if err := r.CommitMessages(ctx, m); err != nil {
			log.Printf("❌ Failed to commit message: %v", err)
		}
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// startBitemporalWorker tracks all entity changes in the bitemporal audit system
func startBitemporalWorker(ctx context.Context, config SyncWorkerConfig, worker *sync.BitemporalCDCWorker) error {
	topics := []string{
		fmt.Sprintf("%s.public.tenants", config.DebeziumServerName),
		fmt.Sprintf("%s.public.tenant_instance", config.DebeziumServerName),
		fmt.Sprintf("%s.public.connections", config.DebeziumServerName),
		fmt.Sprintf("%s.public.tenant_product", config.DebeziumServerName),
	}

	return consumeTopics(ctx, config.KafkaBrokers, "sync-worker-bitemporal", topics, func(event DebeziumEvent) error {
log.Printf("[Bitemporal] Processing %s event for table %s", event.Payload.Op, event.Payload.Source.Table)

var err error
switch event.Payload.Source.Table {
case "tenants":
err = worker.ProcessTenantChange(ctx, event.Payload.Op, event.Payload.Before, event.Payload.After)
case "tenant_instance":
err = worker.ProcessInstanceChange(ctx, event.Payload.Op, event.Payload.Before, event.Payload.After)
case "connections":
err = worker.ProcessConnectionChange(ctx, event.Payload.Op, event.Payload.Before, event.Payload.After)
case "tenant_product":
err = worker.ProcessProductChange(ctx, event.Payload.Op, event.Payload.Before, event.Payload.After)
default:
log.Printf("[Bitemporal] Skipping unknown table: %s", event.Payload.Source.Table)
return nil
}

if err != nil {
log.Printf("[Bitemporal] Error tracking change for %s: %v", event.Payload.Source.Table, err)
// Don't fail the worker, just log the error
return nil
}

return nil
})
}
