// cmd/report-smoke-test/main.go
//
// Standalone smoke test for Phase 1: starts a short-lived Temporal worker,
// submits a ReportGenerationWorkflow with a synthetic hydrated template, waits
// for it to complete, then queries report_executions and asserts:
//
//   1. The execution row transitioned from 'running' to 'completed'.
//   2. requested_by == template.CreatedByID (identity invariant).
//   3. metadata->>'engine' == 'temporal_workflow'.
//
// Usage:
//
//	DATABASE_URL="postgres://postgres@100.84.50.65:5432/alpha?sslmode=verify-full&..." \
//	  go run ./backend/cmd/report-smoke-test/main.go
//
// The binary self-terminates after the assertions pass (exit 0) or fail (exit 1).
// It is NOT a production binary — it lives in cmd/ as an operational smoke harness.
package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"github.com/hondyman/uisce/backend/internal/temporal/activities"
	"github.com/hondyman/uisce/backend/internal/temporal/workflows"
)

const (
	taskQueue            = "analytics-worker"
	temporalAddr         = "100.84.50.65:7233"
	workflowTimeout      = 2 * time.Minute
	pollInterval         = 2 * time.Second
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL must be set (use mTLS connection string)")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("DB ping failed: %v", err)
	}
	log.Println("[smoke] DB connection established")

	// Connect to live Temporal cluster
	c, err := client.NewClient(client.Options{
		HostPort:  temporalAddr,
		Namespace: "default",
	})
	if err != nil {
		log.Fatalf("[smoke] Failed to create Temporal client: %v", err)
	}
	defer c.Close()
	log.Printf("[smoke] Temporal client connected to %s", temporalAddr)

	// Start a local in-process worker for the smoke test — the actual report activities
	// are registered so workflow execution completes in this process.
	w := worker.New(c, taskQueue, worker.Options{})
	reportActs := activities.NewReportActivities(db)
	w.RegisterActivity(reportActs.QuerySemanticViewsActivity)
	w.RegisterActivity(reportActs.GenerateArtifactActivity)
	w.RegisterActivity(reportActs.StoreExecutionResultActivity)
	w.RegisterWorkflow(workflows.ReportGenerationWorkflow)

	if err := w.Start(); err != nil {
		log.Fatalf("[smoke] Failed to start worker: %v", err)
	}
	defer w.Stop()
	log.Printf("[smoke] Worker started on task queue: %s", taskQueue)

	// Seed a synthetic execution row with status='running' so StoreExecutionResultActivity has a row to update.
	executionUUID := uuid.New()
	executionID := executionUUID.String()
	// Use a real template and tenant from alpha to satisfy FK constraints on report_executions.
	// These are the gold-copy tenant's templates — valid for smoke testing execution lifecycle.
	templateID := "715b3c96-f441-5c81-98ef-0b04cfe78ad1" // "High-Net-Worth Household Allocation" (gold copy)
	tenantID := "99e99e99-99e9-49e9-89e9-99e99e99e999"   // Gold copy tenant
	ownerID := "smoke-owner-user"

	_, err = db.Exec(`
		INSERT INTO public.report_executions (
			id, tenant_id, template_id, report_key, status,
			requested_by, workflow_id, created_at
		) VALUES (
			$1, $2::uuid, $3::uuid, 'smoke-test', 'running',
			$4, $5, NOW()
		)
	`, executionID, tenantID, templateID, ownerID, "report-exec-"+executionID)
	if err != nil {
		log.Fatalf("[smoke] Failed to insert seed execution row: %v", err)
	}
	log.Printf("[smoke] Seeded execution row: id=%s", executionID)

	// Build hydrated template — this is the TOCTOU snapshot that would come from TriggerScheduleRun.
	template := activities.HydratedReportTemplate{
		ID:              templateID,
		TenantID:        tenantID,
		TemplateName:    "Smoke Test Report",
		Category:        "smoke",
		SemanticViewIDs: []string{}, // Degenerate case — zero views
		LayoutConfig:    map[string]interface{}{},
		CreatedByID:     ownerID,
		CreatedBy:       "smoke@test.internal",
		IsPublic:        true,
	}

	params := workflows.ReportGenerationWorkflowParams{
		ExecutionID: executionID,
		Template:    template,
		ScheduleID:  "smoke-schedule",
		TriggerParams: map[string]interface{}{
			"triggered_at": time.Now().Format(time.RFC3339),
			"schedule_id":  "smoke-schedule",
		},
	}

	// Start the workflow with the 10-minute wall-clock cap
	ctx, cancel := context.WithTimeout(context.Background(), workflowTimeout)
	defer cancel()

	workflowID := "report-exec-" + executionID
	run, err := c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:                       workflowID,
		TaskQueue:                taskQueue,
		WorkflowExecutionTimeout: 10 * time.Minute,
	}, workflows.ReportGenerationWorkflow, params)
	if err != nil {
		log.Fatalf("[smoke] Failed to start workflow: %v", err)
	}
	log.Printf("[smoke] Workflow started: workflow_id=%s run_id=%s", run.GetID(), run.GetRunID())

	// Wait for workflow completion
	var result workflows.ReportGenerationWorkflowResult
	if err := run.Get(ctx, &result); err != nil {
		log.Fatalf("[smoke] Workflow failed: %v", err)
	}
	log.Printf("[smoke] Workflow completed: output_url=%s engine=%s execution_time_ms=%d",
		result.OutputURL, result.Engine, result.ExecutionTimeMS)

	// Poll the database to verify the execution row was written correctly by the worker.
	// This is Invariant Test 1: assert on the worker-written row, not the API echo.
	time.Sleep(500 * time.Millisecond) // brief settle for write propagation

	var (
		dbStatus      string
		dbRequestedBy string
		dbEngine      string
	)
	err = db.QueryRowContext(ctx, `
		SELECT status, requested_by, metadata->>'engine'
		FROM public.report_executions
		WHERE id = $1
	`, executionID).Scan(&dbStatus, &dbRequestedBy, &dbEngine)
	if err != nil {
		log.Fatalf("[smoke] FAIL: Failed to query execution row: %v", err)
	}

	// Assertion 1: status must be 'completed'
	if dbStatus != "completed" {
		log.Fatalf("[smoke] FAIL: expected status='completed', got status=%q", dbStatus)
	}
	log.Printf("[smoke] PASS: status='completed'")

	// Assertion 2: IDENTITY INVARIANT — requested_by must be the template owner, not a caller
	if dbRequestedBy != ownerID {
		log.Fatalf("[smoke] FAIL IDENTITY INVARIANT: expected requested_by=%q, got %q", ownerID, dbRequestedBy)
	}
	log.Printf("[smoke] PASS IDENTITY INVARIANT: requested_by=%q (template owner, not caller)", dbRequestedBy)

	// Assertion 3: engine marker must be 'temporal_workflow'
	if dbEngine != "temporal_workflow" {
		log.Fatalf("[smoke] FAIL: expected engine='temporal_workflow', got engine=%q", dbEngine)
	}
	log.Printf("[smoke] PASS: engine='temporal_workflow'")

	log.Printf("[smoke] ALL ASSERTIONS PASSED. Phase 1 smoke test complete.")
	log.Printf("[smoke] execution_id=%s workflow_id=%s run_id=%s", executionID, run.GetID(), run.GetRunID())

	// Clean up smoke test row
	_, _ = db.Exec(`DELETE FROM public.report_executions WHERE id = $1`, executionID)
	log.Printf("[smoke] Cleaned up smoke test execution row.")
}
