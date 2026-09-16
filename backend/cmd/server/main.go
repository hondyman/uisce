package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/api"
	fixpkg "github.com/hondyman/uisce/backend/internal/fix"
	"github.com/hondyman/uisce/backend/internal/trading"
	temporalclientlib "github.com/hondyman/uisce/libs/temporal-client"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	sdkclient "go.temporal.io/sdk/client"
)

// withPublicSearchPath forces every physical connection opened from this DSN to
// start with search_path=public. The shared "alpha" database has a
// database-level `ALTER DATABASE alpha SET search_path = 'vend, public'`
// (another service's schema, unrelated to Uisce) - without this, unqualified
// table names like `tenants` silently resolve to that other schema's
// same-named-but-differently-shaped tables instead of public.tenants.
func withPublicSearchPath(dsn string) string {
	sep := "&"
	if !strings.Contains(dsn, "?") {
		sep = "?"
	}
	return dsn + sep + "options=" + url.QueryEscape("-c search_path=public")
}

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" || dbURL == "<VALUE_TO_BE_PROVIDED>" {
		dbURL = os.Getenv("POSTGRES_DSN")
	}
	if dbURL == "" || dbURL == "<VALUE_TO_BE_PROVIDED>" {
		dbURL = "postgresql://postgres:postgres@100.84.50.65:5432/alpha?sslmode=disable"
	}
	dbURL = withPublicSearchPath(dbURL)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Fail-closed startup assertion: unsafe dev-only flags must not be active
	// in production. ENVIRONMENT must be explicitly "development", "local", or "test"
	// to permit ALLOW_CLIENT_TENANT_HEADER_FALLBACK or API_TOKEN_ENCRYPTION_KEY_DEV_FALLBACK.
	// An unset or unknown ENVIRONMENT is treated as production (fail-closed).
	if err := api.AssertProductionConfig(); err != nil {
		log.Fatalf("FATAL: production config assertion failed: %v", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Println("Connected to database successfully")

	sqlxDB := sqlx.NewDb(db, "postgres")
	_ = sqlxDB

	// Context for graceful shutdown of all goroutines (FIX server, etc).
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	var temporalClient sdkclient.Client
	if tc, err := temporalclientlib.NewClientWithRetry(); err != nil {
		log.Printf("WARNING: Temporal unavailable; FIX order-entry commands will 503: %v", err)
	} else {
		temporalClient = tc
		defer temporalClient.Close()
		log.Println("✅ Connected to Temporal")
	}

	// Start the FIX acceptor alongside the HTTP server (HANDOFF_FIX_OVER_PIPELINE.md
	// §13 build step 6). When FIX_ENABLE is unset, this is skipped — legacy mode.
	if os.Getenv("FIX_ENABLE") == "true" {
		startFIXServer(ctx, db, temporalClient)
	}

	// GEMINI_API_KEY is optional: every caller of the Gemini gateway (NL-to-SQL
	// query generation, AI page generation) already falls back to a
	// deterministic path when this is nil, so a missing/invalid key degrades
	// gracefully instead of failing startup.
	var geminiClient *api.GeminiClient
	if apiKey := os.Getenv("GEMINI_API_KEY"); apiKey != "" {
		client, err := api.NewGeminiClient(apiKey)
		if err != nil {
			log.Printf("WARNING: failed to initialize Gemini client, AI features will use their deterministic fallback: %v", err)
		} else {
			geminiClient = client
		}
	}

	router := api.SetupRouter(db, nil, nil, temporalClient, nil, geminiClient, nil, nil, nil)

	addr := fmt.Sprintf(":%s", port)
	log.Printf("Starting main Uisce Unified API server on %s...\n", addr)
	log.Fatal(http.ListenAndServe(addr, router))
}

// startFIXServer boots the FIX acceptor with the Postgres-backed message
// store and the internal admin API on 127.0.0.1:<port>. Configuration via
// env vars:
//
//	FIX_ENABLE=true                  (gated above)
//	FIX_ACCEPTOR_PORT=8980           (TCP port for inbound FIX)
//	FIX_ADMIN_ADDR=127.0.0.1:8981    (admin API listener)
//	FIX_ADMIN_TOKEN=<shared secret>  (required when FIX_ADMIN_ADDR is set)
//	FIX_CONFIG_PATH=                 (optional path to quickfix settings file)
//
// Per HANDOFF §19 "Admin-API network co-location", when running
// containerized, FIX_ADMIN_ADDR must be reachable from the Temporal worker
// process — typically via Docker service name + shared network.
func startFIXServer(ctx context.Context, db *sql.DB, temporalClient sdkclient.Client) {
	// Latency budget for Amendment 2 fallback (HANDOFF §7).
	// Default 30s, configurable via FIX_PIPELINE_LATENCY_BUDGET_MS.
	budget := 30 * time.Second
	if v := os.Getenv("FIX_PIPELINE_LATENCY_BUDGET_MS"); v != "" {
		if ms, err := strconv.Atoi(v); err == nil && ms > 0 {
			budget = time.Duration(ms) * time.Millisecond
		}
	}

	demoTenant := uuid.MustParse("99e99e99-99e9-49e9-89e9-99e99e99e999")
	if v := os.Getenv("FIX_DEMO_TENANT_ID"); v != "" {
		if parsed, err := uuid.Parse(v); err == nil {
			demoTenant = parsed
		}
	}
	resolver := fixpkg.StaticTenantResolver{TenantID: demoTenant, BrokerID: uuid.MustParse("a1000000-0000-4000-8000-000000000020")}
	var sink fixpkg.InboundSink
	if temporalClient != nil {
		tc := temporalClient
		sink = fixpkg.InboundSinkFunc(func(ctx context.Context, rec fixpkg.InboundRecord) error {
			if rec.MsgType != "8" || rec.ClOrdID == "" {
				return nil
			}
			wfID := "fix-order-" + rec.TenantID.String() + "-" + rec.ClOrdID
			lastQty, _ := strconv.ParseFloat(rec.LastQty, 64)
			lastPx, _ := strconv.ParseFloat(rec.LastPx, 64)
			status := rec.OrdStatus
			if rec.ExecType == "2" {
				status = "Filled"
			}
			return tc.SignalWorkflow(ctx, wfID, "", "ExecutionReport", trading.ExecutionReportSignal{
				ClOrdID:   rec.ClOrdID,
				ExecID:    rec.ExecID,
				Status:    status,
				LastQty:   lastQty,
				LastPx:    lastPx,
				OrdStatus: rec.OrdStatus,
			})
		})
	}
	adapter := fixpkg.NewPipelineAdapter(resolver, sink, db, budget)
	adminAddr := os.Getenv("FIX_ADMIN_ADDR")
	adminToken := os.Getenv("FIX_ADMIN_TOKEN")
	if adminToken == "" && os.Getenv("ENVIRONMENT") == "development" {
		adminToken = "dev-fix-admin"
		os.Setenv("FIX_ADMIN_TOKEN", adminToken)
	}
	if adminAddr == "" && os.Getenv("FIX_DEMO_AGENT") == "true" {
		adminAddr = "127.0.0.1:8981"
	}
	configPath := os.Getenv("FIX_CONFIG_PATH")

	srv, err := fixpkg.NewServer(adapter, configPath, db, adminAddr, adminToken)
	if err != nil {
		log.Printf("[FIX] Failed to start FIX server: %v (continuing without it)", err)
		return
	}
	if err := srv.Start(ctx); err != nil {
		log.Printf("[FIX] FIX server start failed: %v (continuing without it)", err)
		return
	}
	log.Println("✅ FIX acceptor started")

	if os.Getenv("FIX_DEMO_AGENT") == "true" {
		port := os.Getenv("FIX_ACCEPTOR_PORT")
		if port == "" {
			port = "8980"
		}
		if _, err := fixpkg.StartDemoAgent(ctx, "127.0.0.1", port); err != nil {
			log.Printf("[FIX] demo agent failed to start: %v", err)
		} else {
			log.Println("✅ FIX demo agent started (sample broker)")
		}
	}
}
