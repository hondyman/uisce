package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http/httptest"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	httpapi "github.com/hondyman/semlayer/backend/internal/api"
	"github.com/hondyman/semlayer/backend/internal/identity"
	"github.com/hondyman/semlayer/backend/internal/services"
	_ "github.com/lib/pq"
)

func main() {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable"
	}

	var workflowID string
	var actorID string
	flag.StringVar(&workflowID, "workflow", "test-run-42", "workflow id to use for the test audit row")
	flag.StringVar(&actorID, "actor", "tester@example.com", "actor id to inject into request context")
	flag.Parse()

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	// Security manager now accepts cache and metrics manager parameters - pass nil for test utility
	sec := services.NewSecurityManager(nil, nil, []byte("dev-jwt-secret"))

	handler := httpapi.NewTemporalAdminHandler(nil, db, sec, nil)

	// Build request body
	body := map[string]interface{}{
		"signal_name": "test-signal",
		"input":       map[string]interface{}{"foo": "bar"},
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/temporal/workflows/"+workflowID+"/signal", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	// Ensure tenant header is set to valid UUID for DB insert
	req.Header.Set("X-Tenant-ID", "00000000-0000-0000-0000-000000000000")

	// Inject identity into context as if AuthContextMiddleware had run
	ctx := identity.WithActorTenant(context.Background(), actorID, "")
	// chi requires a RouteContext on the request context for URLParam to work
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", workflowID)
	ctx = context.WithValue(ctx, chi.RouteCtxKey, rc)
	req = req.WithContext(ctx)

	// Call handler
	rr := httptest.NewRecorder()
	handler.HandleSignalWorkflow(rr, req)

	resp := rr.Result()
	defer resp.Body.Close()
	bodyResp, _ := io.ReadAll(resp.Body)

	fmt.Printf("Handler responded with status %d and body: %s\n", resp.StatusCode, string(bodyResp))

	// Small delay to ensure LogAdminAction completed
	time.Sleep(200 * time.Millisecond)

	// Query DB for audit row
	var id string
	var action string
	var actor string
	var wf string
	err = db.QueryRowContext(context.Background(), `SELECT id, action, actor_id, workflow_id FROM public.admin_audit_logs WHERE workflow_id = $1 ORDER BY created_at DESC LIMIT 1`, workflowID).Scan(&id, &action, &actor, &wf)
	if err != nil {
		log.Fatalf("failed to query audit table: %v", err)
	}

	fmt.Printf("Found audit row id=%s action=%s actor=%s workflow=%s\n", id, action, actor, wf)
}
