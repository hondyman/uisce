package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

func main() {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable"
	}

	var workflowID string
	flag.StringVar(&workflowID, "workflow", "test-workflow-123", "workflow id to use for the test audit row")
	flag.Parse()

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	// Prepare input JSON
	inputMap := map[string]interface{}{"test": true}
	inputBytes, _ := json.Marshal(inputMap)

	id := uuid.New().String()
	tenantID := "00000000-0000-0000-0000-000000000000"
	actorID := "tester@example.com"
	action := "signal"
	reason := "integration test"
	status := "success"
	timestamp := time.Now()

	ctx := context.Background()

	// Insert directly into the audit table
	_, err = db.ExecContext(ctx, `
        INSERT INTO public.admin_audit_logs (id, tenant_id, actor_id, action, workflow_id, run_id, reason, input, status, error_message, created_at)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
    `, id, tenantID, actorID, action, workflowID, "", reason, inputBytes, status, "", timestamp)
	if err != nil {
		log.Fatalf("failed to insert audit row: %v", err)
	}

	// Verify row exists
	var count int
	err = db.QueryRowContext(ctx, `SELECT count(*) FROM public.admin_audit_logs WHERE workflow_id = $1 AND actor_id = $2 AND id = $3`, workflowID, actorID, id).Scan(&count)
	if err != nil {
		log.Fatalf("failed to query audit table: %v", err)
	}

	fmt.Printf("Inserted audit rows matching workflow_id=%s actor=%s : %d\n", workflowID, actorID, count)
}
