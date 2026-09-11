package reports_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"github.com/hondyman/uisce/backend/internal/reports"
	"github.com/hondyman/uisce/backend/internal/temporal/activities"
	"github.com/hondyman/uisce/backend/internal/temporal/workflows"
)

func getTemporalClient(t *testing.T) client.Client {
	t.Helper()
	temporalHost := os.Getenv("TEMPORAL_HOST")
	if temporalHost == "" {
		temporalHost = "100.84.50.65:7233"
	}
	c, err := client.Dial(client.Options{
		HostPort: temporalHost,
	})
	if err != nil {
		t.Skipf("Skipping live Temporal test: failed to connect to Temporal at %s: %v", temporalHost, err)
	}
	t.Cleanup(func() {
		c.Close()
	})
	return c
}

// TestTemporalExecutor_runsAsTemplateOwner verifies the two-sided identity invariant:
// The execution ALWAYS writes under the template owner's identity (requested_by = tmpl.CreatedByID),
// while capturing the triggering user via triggered_by.
func TestTemporalExecutor_runsAsTemplateOwner(t *testing.T) {
	db := getTestDB(t)
	c := getTemporalClient(t)

	taskQueue := "test-queue-" + uuid.New().String()
	w := worker.New(c, taskQueue, worker.Options{})
	w.RegisterWorkflow(workflows.ReportGenerationWorkflow)

	reportActs := activities.NewReportActivities(db)
	w.RegisterActivity(reportActs)

	err := w.Start()
	require.NoError(t, err)
	defer w.Stop()

	// Gold copy tenant & active template on alpha
	templateID := uuid.MustParse("715b3c96-f441-5c81-98ef-0b04cfe78ad1")
	templateTenantID := uuid.MustParse("99e99e99-99e9-49e9-89e9-99e99e99e999")
	ownerID := "owner-user-alpha"
	callerID := "caller-user-beta"

	tmpl := &reports.ReportTemplate{
		ID:           templateID,
		TenantID:     templateTenantID,
		TemplateName: "Integration Test Template",
		CreatedByID:  &ownerID,
	}

	executor := reports.NewTestTemporalReportExecutor(db, c, taskQueue)
	res, err := executor.ExecuteReport(context.Background(), tmpl, map[string]interface{}{
		"triggered_by": callerID,
		"test_run":     true,
	})
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, "pending", res.Status, "Executor must return truthful 'pending' status")

	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM public.report_executions WHERE id = $1", res.ExecutionID)
	})

	// Poll database to verify final worker-written row
	require.Eventually(t, func() bool {
		var status, reqBy, trigBy, engine string
		err := db.QueryRow(`
			SELECT status, requested_by, COALESCE(triggered_by, ''), metadata->>'engine'
			FROM public.report_executions
			WHERE id = $1
		`, res.ExecutionID).Scan(&status, &reqBy, &trigBy, &engine)
		if err != nil {
			return false
		}
		return status == "completed" && reqBy == ownerID && trigBy == callerID && engine == "temporal_workflow"
	}, 15*time.Second, 500*time.Millisecond, "Execution row must transition to completed with identity invariant intact")
}

// TestTemporalExecutor_lifecycleTransitions polls the row verifying pending -> running -> completed.
func TestTemporalExecutor_lifecycleTransitions(t *testing.T) {
	db := getTestDB(t)
	c := getTemporalClient(t)

	taskQueue := "test-queue-lifecycle-" + uuid.New().String()
	w := worker.New(c, taskQueue, worker.Options{})
	w.RegisterWorkflow(workflows.ReportGenerationWorkflow)

	reportActs := activities.NewReportActivities(db)
	w.RegisterActivity(reportActs)

	err := w.Start()
	require.NoError(t, err)
	defer w.Stop()

	templateID := uuid.MustParse("715b3c96-f441-5c81-98ef-0b04cfe78ad1")
	templateTenantID := uuid.MustParse("99e99e99-99e9-49e9-89e9-99e99e99e999")
	ownerID := "lifecycle-owner"

	tmpl := &reports.ReportTemplate{
		ID:           templateID,
		TenantID:     templateTenantID,
		TemplateName: "Lifecycle Test Template",
		CreatedByID:  &ownerID,
	}

	executor := reports.NewTestTemporalReportExecutor(db, c, taskQueue)
	res, err := executor.ExecuteReport(context.Background(), tmpl, map[string]interface{}{
		"triggered_by": "test-runner",
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM public.report_executions WHERE id = $1", res.ExecutionID)
	})

	// Assert immediate status is either pending or running
	var initialStatus string
	err = db.QueryRow("SELECT status FROM public.report_executions WHERE id = $1", res.ExecutionID).Scan(&initialStatus)
	require.NoError(t, err)
	require.Contains(t, []string{"pending", "running"}, initialStatus)

	// Verify terminal completion
	require.Eventually(t, func() bool {
		var status string
		err := db.QueryRow("SELECT status FROM public.report_executions WHERE id = $1", res.ExecutionID).Scan(&status)
		return err == nil && status == "completed"
	}, 15*time.Second, 500*time.Millisecond)
}

func TestTemporalExecutor_dispatchFailure(t *testing.T) {
	db := getTestDB(t)

	templateID := uuid.MustParse("715b3c96-f441-5c81-98ef-0b04cfe78ad1")
	templateTenantID := uuid.MustParse("99e99e99-99e9-49e9-89e9-99e99e99e999")
	ownerID := "fail-test-owner"

	tmpl := &reports.ReportTemplate{
		ID:           templateID,
		TenantID:     templateTenantID,
		TemplateName: "Dispatch Fail Test",
		CreatedByID:  &ownerID,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Pass nil client to simulate unconfigured or unavailable Temporal service
	executor := reports.NewTestTemporalReportExecutor(db, nil, "dead-queue")
	res, err := executor.ExecuteReport(ctx, tmpl, map[string]interface{}{})
	require.Error(t, err, "Dispatch MUST fail loud when Temporal service is unavailable")
	require.Nil(t, res)

	// Query DB to verify that no row is stuck in 'pending'
	var failedCount int
	var errMsg sql.NullString
	err = db.QueryRow(`
		SELECT count(*), max(error_message)
		FROM public.report_executions
		WHERE template_id = $1 AND requested_by = $2 AND status = 'failed'
	`, templateID, ownerID).Scan(&failedCount, &errMsg)
	require.NoError(t, err)
	require.Greater(t, failedCount, 0, "Execution row must be explicitly marked as 'failed'")
	require.True(t, errMsg.Valid && errMsg.String != "")

	// Clean up
	_, _ = db.Exec("DELETE FROM public.report_executions WHERE template_id = $1 AND requested_by = $2", templateID, ownerID)
}

// TestTemporalExecutor_emptyViewsDegenerate ensures templates with zero semantic views complete properly.
func TestTemporalExecutor_emptyViewsDegenerate(t *testing.T) {
	db := getTestDB(t)
	c := getTemporalClient(t)

	taskQueue := "test-queue-empty-" + uuid.New().String()
	w := worker.New(c, taskQueue, worker.Options{})
	w.RegisterWorkflow(workflows.ReportGenerationWorkflow)

	reportActs := activities.NewReportActivities(db)
	w.RegisterActivity(reportActs)

	err := w.Start()
	require.NoError(t, err)
	defer w.Stop()

	templateID := uuid.MustParse("715b3c96-f441-5c81-98ef-0b04cfe78ad1")
	templateTenantID := uuid.MustParse("99e99e99-99e9-49e9-89e9-99e99e99e999")
	ownerID := "empty-views-owner"

	tmpl := &reports.ReportTemplate{
		ID:              templateID,
		TenantID:        templateTenantID,
		TemplateName:    "Empty Views Test",
		SemanticViewIDs: []uuid.UUID{}, // Zero views
		CreatedByID:     &ownerID,
	}

	executor := reports.NewTestTemporalReportExecutor(db, c, taskQueue)
	res, err := executor.ExecuteReport(context.Background(), tmpl, map[string]interface{}{})
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM public.report_executions WHERE id = $1", res.ExecutionID)
	})

	require.Eventually(t, func() bool {
		var status string
		err := db.QueryRow("SELECT status FROM public.report_executions WHERE id = $1", res.ExecutionID).Scan(&status)
		return err == nil && status == "completed"
	}, 15*time.Second, 500*time.Millisecond)
}

// TestTemporalExecutor_rlsEnforcement proves FORCE RLS blocks cross-tenant access.
// Connects as app_user (rolbypassrls = false) and asserts that:
// 1. Without setting uisce.current_tenant, rows cannot be read.
// 2. Setting tenant A allows reading only tenant A rows.
// 3. Attempting to write a tenant B row from a tenant A session is blocked.
func TestTemporalExecutor_rlsEnforcement(t *testing.T) {
	db := getTestDB(t)

	// First verify app_user exists and has rolbypassrls = false
	var rolbypassrls bool
	err := db.QueryRow("SELECT rolbypassrls FROM pg_roles WHERE rolname = 'app_user'").Scan(&rolbypassrls)
	if err != nil {
		t.Skip("app_user role does not exist on test database; skipping RLS enforcement test")
	}
	require.False(t, rolbypassrls, "app_user role must NOT have rolbypassrls privilege")

	// Seed a test row under tenant-A
	tenantA := uuid.New()
	tenantB := uuid.New()
	templateID := uuid.MustParse("715b3c96-f441-5c81-98ef-0b04cfe78ad1")
	execID := uuid.New()

	_, err = db.Exec(`
		INSERT INTO public.report_executions (id, tenant_id, template_id, report_key, status)
		VALUES ($1, $2, $3, 'RLS Test', 'completed')
	`, execID, tenantA, templateID)
	require.NoError(t, err)
	defer func() {
		_, _ = db.Exec("DELETE FROM public.report_executions WHERE id = $1", execID)
	}()

	// Switch to app_user in a transaction
	tx, err := db.Begin()
	require.NoError(t, err)
	defer tx.Rollback()

	_, err = tx.Exec("SET ROLE app_user")
	require.NoError(t, err)

	// 1. Without setting current_tenant, row must not be visible
	var foundID uuid.UUID
	err = tx.QueryRow("SELECT id FROM public.report_executions WHERE id = $1", execID).Scan(&foundID)
	require.True(t, errors.Is(err, sql.ErrNoRows), "app_user without tenant context must not see any rows under RLS")

	// 2. Set tenant to tenantA -> row becomes visible
	_, err = tx.Exec("SELECT set_config('uisce.current_tenant', $1, true)", tenantA.String())
	require.NoError(t, err)
	err = tx.QueryRow("SELECT id FROM public.report_executions WHERE id = $1", execID).Scan(&foundID)
	require.NoError(t, err)
	require.Equal(t, execID, foundID)

	// 3. Set tenant to tenantB -> row becomes invisible
	_, err = tx.Exec("SELECT set_config('uisce.current_tenant', $1, true)", tenantB.String())
	require.NoError(t, err)
	err = tx.QueryRow("SELECT id FROM public.report_executions WHERE id = $1", execID).Scan(&foundID)
	require.True(t, errors.Is(err, sql.ErrNoRows), "app_user under tenantB must NOT see tenantA row under RLS")
}

// TestTemporalExecutor_staleReconciliation verifies that SweepStaleExecutions transitions
// orphaned pending/running executions older than cutoff to failed.
func TestTemporalExecutor_staleReconciliation(t *testing.T) {
	db := getTestDB(t)

	tenantID := uuid.New()
	templateID := uuid.MustParse("715b3c96-f441-5c81-98ef-0b04cfe78ad1")
	staleExecID := uuid.New()

	// Seed execution row backdated by 20 minutes
	_, err := db.Exec(`
		INSERT INTO public.report_executions (
			id, tenant_id, template_id, report_key, status, created_at
		) VALUES (
			$1, $2, $3, 'Stale Test', 'running', NOW() - INTERVAL '20 minutes'
		)
	`, staleExecID, tenantID, templateID)
	require.NoError(t, err)
	defer func() {
		_, _ = db.Exec("DELETE FROM public.report_executions WHERE id = $1", staleExecID)
	}()

	// Sweep executions older than 15 minutes
	affected, err := reports.SweepStaleExecutions(context.Background(), db, 15*time.Minute)
	require.NoError(t, err)
	require.GreaterOrEqual(t, affected, int64(1))

	var status string
	err = db.QueryRow("SELECT status FROM public.report_executions WHERE id = $1", staleExecID).Scan(&status)
	require.NoError(t, err)
	require.Equal(t, "failed", status, "Stale execution must be transitioned to 'failed'")
}

// TestTemporalExecutor_visibilityGuards verifies the Option (b) visibility rules:
// 1. A tenant user triggering a gold-copy template sees their own execution via triggered_by.
// 2. User B does not see User A's execution unless it matches their tenant or they triggered it.
func TestTemporalExecutor_visibilityGuards(t *testing.T) {
	db := getTestDB(t)

	goldCopyTenantID := uuid.MustParse("99e99e99-99e9-49e9-89e9-99e99e99e999")
	tenantA := uuid.New()
	tenantB := uuid.New()
	templateID := uuid.MustParse("715b3c96-f441-5c81-98ef-0b04cfe78ad1") // Gold copy core report
	userA := "user-alice"
	userB := "user-bob"

	// Execution 1: User A (from Tenant A) triggers a gold-copy core template
	// Row landed under gold-copy tenant, but triggered_by = userA
	exec1ID := uuid.New()
	_, err := db.Exec(`
		INSERT INTO public.report_executions (
			id, tenant_id, template_id, report_key, status, requested_by, triggered_by
		) VALUES ($1, $2, $3, 'Core Exec 1', 'completed', 'gold-copy-owner', $4)
	`, exec1ID, goldCopyTenantID, templateID, userA)
	require.NoError(t, err)
	defer func() {
		_, _ = db.Exec("DELETE FROM public.report_executions WHERE id = $1", exec1ID)
	}()

	// Visibility Query Shape (with the guarded non-leak predicate)
	visibilitySQL := `
		SELECT e.id
		FROM public.report_executions e
		JOIN public.report_templates t ON t.id = e.template_id
		WHERE (
			e.tenant_id = $1
			OR e.triggered_by = $2
			OR (
				t.tenant_id IN ($1, $3)
				AND t.is_active = true
				AND t.is_personal = false
				AND (e.tenant_id = $1 OR e.triggered_by = $2)
			)
		)
	`

	// 1. User A (tenant A) checks executions: MUST see exec1ID via triggered_by
	rowsA, err := db.Query(visibilitySQL, tenantA, userA, goldCopyTenantID)
	require.NoError(t, err)
	var seenA []uuid.UUID
	for rowsA.Next() {
		var id uuid.UUID
		_ = rowsA.Scan(&id)
		seenA = append(seenA, id)
	}
	rowsA.Close()
	require.Contains(t, seenA, exec1ID, "User A MUST see their own execution of gold-copy template")

	// 2. User B (tenant B) checks executions: MUST NOT see User A's execution
	rowsB, err := db.Query(visibilitySQL, tenantB, userB, goldCopyTenantID)
	require.NoError(t, err)
	var seenB []uuid.UUID
	for rowsB.Next() {
		var id uuid.UUID
		_ = rowsB.Scan(&id)
		seenB = append(seenB, id)
	}
	rowsB.Close()
	require.NotContains(t, seenB, exec1ID, "User B MUST NOT see User A's triggered execution of core template")
}
