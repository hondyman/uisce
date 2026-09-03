package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/api"
	"github.com/hondyman/uisce/backend/internal/observability"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/hondyman/uisce/backend/internal/semantic_bridge"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" || dbURL == "<VALUE_TO_BE_PROVIDED>" {
		dbURL = os.Getenv("POSTGRES_DSN")
	}
	if dbURL == "" || dbURL == "<VALUE_TO_BE_PROVIDED>" {
		dbURL = "postgresql://postgres:postgres@100.84.50.65:5432/alpha?sslmode=disable"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Println("Connected to database successfully")

	sqlxDB := sqlx.NewDb(db, "postgres")

	startLedgerMonitor(sqlxDB)

	router := api.SetupRouter(db, nil, nil, nil, nil, nil, nil, nil, nil)

	addr := fmt.Sprintf(":%s", port)
	log.Printf("Starting main Uisce Unified API server on %s...\n", addr)
	log.Fatal(http.ListenAndServe(addr, router))
}

// startLedgerMonitor periodically re-verifies the AI Semantic Bridge's
// tamper-evident sync-log hash chain (internal/semantic_bridge/ledger.go)
// and reports any breach — a chain nobody watches is just a chain. Skips
// starting (rather than crashing the whole server) if the shared server key
// isn't configured, matching how the semantic-bridge HTTP routes themselves
// fail closed in that case.
func startLedgerMonitor(sqlxDB *sqlx.DB) {
	hmacKey, err := security.LoadKeyFromEnv(semantic_bridge.CredentialVaultKeyEnv, semantic_bridge.CredentialVaultDevFallbackEnv)
	if err != nil {
		log.Printf("[WARN] AI Semantic Bridge ledger monitor disabled: %v", err)
		return
	}

	dispatcher := observability.NewAlertDispatcher()
	channelsRegistered := false
	if webhookURL := os.Getenv("SLO_SLACK_WEBHOOK_URL"); webhookURL != "" {
		dispatcher.RegisterChannel("slack", observability.NewSlackChannel(webhookURL, "#ops-alerts", "Uisce Ledger Monitor"))
		channelsRegistered = true
	}
	if routingKey := os.Getenv("PAGERDUTY_ROUTING_KEY"); routingKey != "" {
		dispatcher.RegisterChannel("pagerduty", observability.NewPagerDutyChannel(routingKey))
		channelsRegistered = true
	}
	if !channelsRegistered {
		log.Println("[INFO] AI Semantic Bridge ledger monitor: no SLO_SLACK_WEBHOOK_URL or PAGERDUTY_ROUTING_KEY configured — breaches will only be logged, not paged.")
	}

	ledger := semantic_bridge.NewLedger(sqlxDB, hmacKey)
	monitor := semantic_bridge.NewLedgerMonitor(sqlxDB, ledger, func(ctx context.Context, breach semantic_bridge.LedgerBreach) {
		dispatcher.DispatchToAll(ctx, &observability.Alert{
			ID:       uuid.New(),
			TenantID: breach.TenantID,
			Severity: string(observability.SeverityCritical),
			Message:  fmt.Sprintf("AI Semantic Bridge audit ledger tamper detected for tenant %s (first broken log: %s)", breach.TenantID, breach.FirstBadLog),
			Status:   string(observability.AlertFiring),
			FiredAt:  breach.DetectedAt,
		})
	})

	interval := 15 * time.Minute
	go monitor.Start(context.Background(), interval)
	log.Printf("AI Semantic Bridge ledger monitor started (checking every %s)", interval)
}
