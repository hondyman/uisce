package reports_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"

	"github.com/hondyman/uisce/backend/internal/reports"
)

// createScheduleTestTenant creates an ephemeral tenant in public.tenants with automatic cleanup.
func createScheduleTestTenant(t *testing.T, db *sql.DB, name string) uuid.UUID {
	t.Helper()
	tenantID := uuid.New()
	displayName := name + " " + tenantID.String()[:8]
	_, err := db.Exec(`
		INSERT INTO public.tenants (id, name, display_name, status, plan, is_active, is_deleted, created_at, updated_at)
		VALUES ($1, $2, $3, 'active', 'standard', true, false, NOW(), NOW())
	`, tenantID, name+"-"+tenantID.String()[:8], displayName)
	require.NoError(t, err, "failed to insert test tenant")

	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM public.tenants WHERE id = $1`, tenantID)
	})

	return tenantID
}

// createScheduleTestUser creates an ephemeral app_user record and registers cleanup.
func createScheduleTestUser(t *testing.T, db *sql.DB, tenantID uuid.UUID) string {
	t.Helper()
	userID := "test-sched-user-" + uuid.New().String()
	email := fmt.Sprintf("%s@integration-test.internal", userID)

	_, err := db.Exec(`
		INSERT INTO app_user (id, email, tenant_id, username, is_active)
		VALUES ($1, $2, $3, $4, true)
	`, userID, email, tenantID.String(), userID)
	require.NoError(t, err, "failed to insert test app_user")

	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM app_user WHERE id = $1`, userID)
	})

	return userID
}

// createScheduleTestTemplate inserts a template directly and registers cleanup via t.Cleanup.
func createScheduleTestTemplate(t *testing.T, db *sql.DB, tenantID uuid.UUID, name string, isPersonal bool, createdByID *string) *reports.ReportTemplate {
	t.Helper()
	repo := reports.NewRepository(db)
	tmpl := &reports.ReportTemplate{
		ID:           uuid.New(),
		TenantID:     tenantID,
		TemplateName: name + "_" + uuid.New().String()[:8],
		Category:     "test",
		IsActive:     true,
		IsPersonal:   isPersonal,
		CreatedByID:  createdByID,
	}
	if err := repo.CreateTemplate(context.Background(), tmpl); err != nil {
		t.Fatalf("failed to create test template: %v", err)
	}

	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM public.report_templates WHERE id = $1", tmpl.ID)
	})

	return tmpl
}

// 1. TestCreateSchedule_linksToTemplate — verifies report_definition_id FK is populated
func TestCreateSchedule_linksToTemplate(t *testing.T) {
	db := getTestDB(t)
	repo := reports.NewRepository(db)
	ctx := context.Background()

	tenantID := createScheduleTestTenant(t, db, "SchedLinkTenant")
	ownerID := createScheduleTestUser(t, db, tenantID)
	tmpl := createScheduleTestTemplate(t, db, tenantID, "SchedLinkReport", false, &ownerID)

	sched, err := repo.CreateSchedule(ctx, tenantID, ownerID, reports.CreateScheduleInput{
		TemplateID:     tmpl.ID,
		ScheduleName:   "Daily Valuation",
		CronExpression: "0 8 * * 1-5",
		ExportFormat:   "PDF",
		NotifyInApp:    true,
	})
	if err != nil {
		t.Fatalf("CreateSchedule failed: %v", err)
	}
	defer func() {
		_, _ = db.Exec("DELETE FROM public.report_schedules WHERE id = $1", sched.ID)
	}()

	if sched.ReportDefinitionID != tmpl.ID {
		t.Errorf("expected ReportDefinitionID %s, got %s", tmpl.ID, sched.ReportDefinitionID)
	}
	if sched.OwnerID != ownerID {
		t.Errorf("expected OwnerID %s, got %s", ownerID, sched.OwnerID)
	}
	if !sched.IsActive {
		t.Errorf("expected IsActive to be true")
	}
}

// 2. TestCreateSchedule_visibilityPredicate — own personal passes; foreign/other user personal gets ErrNotFound
func TestCreateSchedule_visibilityPredicate(t *testing.T) {
	db := getTestDB(t)
	repo := reports.NewRepository(db)
	ctx := context.Background()

	tenantA := createScheduleTestTenant(t, db, "TenantA")
	tenantB := createScheduleTestTenant(t, db, "TenantB")
	userAlice := createScheduleTestUser(t, db, tenantA)
	userBob := createScheduleTestUser(t, db, tenantA)
	userForeign := createScheduleTestUser(t, db, tenantB)

	// Alice's personal report in Tenant A
	tmplPersonalAlice := createScheduleTestTemplate(t, db, tenantA, "AlicePersonal", true, &userAlice)

	// Bob attempting to schedule Alice's personal report -> must return ErrNotFound
	_, err := repo.CreateSchedule(ctx, tenantA, userBob, reports.CreateScheduleInput{
		TemplateID:     tmplPersonalAlice.ID,
		ScheduleName:   "Bob Steals Schedule",
		CronExpression: "0 8 * * 1-5",
	})
	if !errors.Is(err, reports.ErrNotFound) {
		t.Errorf("expected ErrNotFound when non-owner schedules personal report, got: %v", err)
	}

	// Foreign tenant user attempting to schedule Alice's report -> must return ErrNotFound
	_, err = repo.CreateSchedule(ctx, tenantB, userForeign, reports.CreateScheduleInput{
		TemplateID:     tmplPersonalAlice.ID,
		ScheduleName:   "Cross-tenant attempt",
		CronExpression: "0 8 * * 1-5",
	})
	if !errors.Is(err, reports.ErrNotFound) {
		t.Errorf("expected ErrNotFound for cross-tenant schedule attempt, got: %v", err)
	}

	// Alice scheduling her own personal report -> succeeds
	schedAlice, err := repo.CreateSchedule(ctx, tenantA, userAlice, reports.CreateScheduleInput{
		TemplateID:     tmplPersonalAlice.ID,
		ScheduleName:   "Alice Own Schedule",
		CronExpression: "0 8 * * 1-5",
	})
	if err != nil {
		t.Fatalf("expected Alice to schedule own personal report, got err: %v", err)
	}
	defer func() {
		_, _ = db.Exec("DELETE FROM public.report_schedules WHERE id = $1", schedAlice.ID)
	}()
}

// 3. TestDeleteSchedule_ownerOrAdmin and TestDeleteSchedule_isSoftDelete
func TestDeleteSchedule_ownerOrAdminAndSoftDelete(t *testing.T) {
	db := getTestDB(t)
	repo := reports.NewRepository(db)
	ctx := context.Background()

	tenantID := createScheduleTestTenant(t, db, "DelTenant")
	ownerID := createScheduleTestUser(t, db, tenantID)
	nonOwner := createScheduleTestUser(t, db, tenantID)
	tmpl := createScheduleTestTemplate(t, db, tenantID, "DeleteTestReport", false, &ownerID)

	sched, err := repo.CreateSchedule(ctx, tenantID, ownerID, reports.CreateScheduleInput{
		TemplateID:     tmpl.ID,
		ScheduleName:   "ToDeleteSched",
		CronExpression: "0 8 * * 1-5",
	})
	if err != nil {
		t.Fatalf("CreateSchedule failed: %v", err)
	}
	defer func() {
		_, _ = db.Exec("DELETE FROM public.report_schedules WHERE id = $1", sched.ID)
	}()

	// Non-owner non-admin attempting delete -> must return ErrForbidden
	err = repo.DeleteSchedule(ctx, tenantID, nonOwner, false, sched.ID)
	if !errors.Is(err, reports.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for non-owner non-admin delete, got: %v", err)
	}

	// Trigger an execution prior to deletion to verify audit linkage persists
	res, err := repo.TriggerScheduleRun(ctx, tenantID, ownerID, false, sched.ID, nil)
	if err != nil {
		t.Fatalf("TriggerScheduleRun before delete failed: %v", err)
	}

	// Owner deleting -> succeeds (soft delete)
	err = repo.DeleteSchedule(ctx, tenantID, ownerID, false, sched.ID)
	if err != nil {
		t.Fatalf("owner DeleteSchedule failed: %v", err)
	}

	// Verify soft delete in DB: row still exists, is_active = false, deleted_at IS NOT NULL
	var isActive bool
	var deletedAt sql.NullTime
	err = db.QueryRow("SELECT is_active, deleted_at FROM public.report_schedules WHERE id = $1", sched.ID).Scan(&isActive, &deletedAt)
	if err != nil {
		t.Fatalf("query schedule after delete failed: %v", err)
	}
	if isActive {
		t.Errorf("expected is_active to be false after soft delete")
	}
	if !deletedAt.Valid {
		t.Errorf("expected deleted_at to be populated after soft delete")
	}

	// Verify invisible to GetSchedule
	_, err = repo.GetSchedule(ctx, tenantID, sched.ID)
	if !errors.Is(err, reports.ErrNotFound) {
		t.Errorf("expected soft-deleted schedule to be invisible to GetSchedule, got: %v", err)
	}

	// Assert audit-link preservation: execution record still exists after schedule soft delete
	var execStatus string
	err = db.QueryRow("SELECT status FROM public.report_executions WHERE id = $1", res.ExecutionID).Scan(&execStatus)
	if err != nil {
		t.Fatalf("query report_executions after schedule soft delete failed: %v", err)
	}
	if execStatus != "synthetic" {
		t.Errorf("expected execution row to survive schedule soft delete with status synthetic, got: %s", execStatus)
	}
}

// 4. TestDeleteTemplate_cascadesSchedule — verifies ON DELETE CASCADE
func TestDeleteTemplate_cascadesSchedule(t *testing.T) {
	db := getTestDB(t)
	repo := reports.NewRepository(db)
	ctx := context.Background()

	tenantID := createScheduleTestTenant(t, db, "CascadeTenant")
	ownerID := createScheduleTestUser(t, db, tenantID)
	tmpl := createScheduleTestTemplate(t, db, tenantID, "CascadeTestReport", false, &ownerID)

	sched, err := repo.CreateSchedule(ctx, tenantID, ownerID, reports.CreateScheduleInput{
		TemplateID:     tmpl.ID,
		ScheduleName:   "CascadeSched",
		CronExpression: "0 8 * * 1-5",
	})
	if err != nil {
		t.Fatalf("CreateSchedule failed: %v", err)
	}

	// Hard delete the template
	_, err = db.Exec("DELETE FROM public.report_templates WHERE id = $1", tmpl.ID)
	if err != nil {
		t.Fatalf("delete template failed: %v", err)
	}

	// Verify schedule row is completely gone (ON DELETE CASCADE)
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM public.report_schedules WHERE id = $1", sched.ID).Scan(&count)
	if err != nil {
		t.Fatalf("query schedule count failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected schedule row to be cascade deleted when template is deleted, found count %d", count)
	}
}

// RecordingMockExecutor records the exact template passed during TriggerScheduleRun
type RecordingMockExecutor struct {
	CapturedTenantID    uuid.UUID
	CapturedCreatedByID *string
	CapturedTemplateID  uuid.UUID
}

func (m *RecordingMockExecutor) ExecuteReport(ctx context.Context, tmpl *reports.ReportTemplate, params map[string]interface{}) (*reports.ScheduleExecutionResult, error) {
	m.CapturedTenantID = tmpl.TenantID
	m.CapturedCreatedByID = tmpl.CreatedByID
	m.CapturedTemplateID = tmpl.ID
	reqBy := ""
	if tmpl.CreatedByID != nil {
		reqBy = *tmpl.CreatedByID
	}
	return &reports.ScheduleExecutionResult{
		ExecutionID: uuid.New(),
		OutputURL:   "s3://mock/snapshot.pdf",
		Status:      "completed",
		RequestedBy: reqBy,
		TenantID:    tmpl.TenantID,
		TemplateID:  tmpl.ID,
	}, nil
}

// 5. TestTriggerRun_twoSidedIdentity — admin trigger executes as template owner; non-owner non-admin gets 403 and zero DB rows
func TestTriggerRun_twoSidedIdentity(t *testing.T) {
	db := getTestDB(t)
	repo := reports.NewRepository(db)
	ctx := context.Background()

	tenantID := createScheduleTestTenant(t, db, "TwoSidedTenant")
	templateOwner := createScheduleTestUser(t, db, tenantID)
	tmpl := createScheduleTestTemplate(t, db, tenantID, "TwoSidedReport", false, &templateOwner)

	scheduleOwner := createScheduleTestUser(t, db, tenantID)
	sched, err := repo.CreateSchedule(ctx, tenantID, scheduleOwner, reports.CreateScheduleInput{
		TemplateID:     tmpl.ID,
		ScheduleName:   "TwoSidedSched",
		CronExpression: "0 8 * * 1-5",
	})
	if err != nil {
		t.Fatalf("CreateSchedule failed: %v", err)
	}
	defer func() {
		_, _ = db.Exec("DELETE FROM public.report_schedules WHERE id = $1", sched.ID)
	}()

	mockExec := &RecordingMockExecutor{}

	// Side A: Non-owner non-admin caller -> ErrForbidden, ZERO rows in report_executions
	var countBefore int
	_ = db.QueryRow("SELECT COUNT(*) FROM public.report_executions WHERE template_id = $1", tmpl.ID).Scan(&countBefore)

	_, err = repo.TriggerScheduleRun(ctx, tenantID, "user-intruder", false, sched.ID, mockExec)
	if !errors.Is(err, reports.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for non-owner non-admin, got: %v", err)
	}

	var countAfter int
	_ = db.QueryRow("SELECT COUNT(*) FROM public.report_executions WHERE template_id = $1", tmpl.ID).Scan(&countAfter)
	if countAfter != countBefore {
		t.Fatalf("expected 0 execution rows written after rejected trigger, got count delta %d", countAfter-countBefore)
	}

	// Side B: Admin caller (who is NOT the owner) triggers run -> succeeds, and executes as templateOwner
	adminUser := createScheduleTestUser(t, db, tenantID)
	res, err := repo.TriggerScheduleRun(ctx, tenantID, adminUser, true, sched.ID, mockExec)
	if err != nil {
		t.Fatalf("expected admin to successfully trigger run, got: %v", err)
	}

	// Assert on the mock executor: received the template's owner identity, NOT the admin's identity
	if mockExec.CapturedCreatedByID == nil || *mockExec.CapturedCreatedByID != templateOwner {
		t.Errorf("expected executor to receive template owner %s, got %v", templateOwner, mockExec.CapturedCreatedByID)
	}
	if res.RequestedBy != templateOwner {
		t.Errorf("expected execution result requested_by %s, got %s", templateOwner, res.RequestedBy)
	}
}

// 6. TestTriggerRun_writesCacheMetadata — verifies 24h TTL record in report_cache_metadata
func TestTriggerRun_writesCacheMetadata(t *testing.T) {
	db := getTestDB(t)
	repo := reports.NewRepository(db)
	ctx := context.Background()

	tenantID := createScheduleTestTenant(t, db, "CacheMetaTenant")
	ownerID := createScheduleTestUser(t, db, tenantID)
	tmpl := createScheduleTestTemplate(t, db, tenantID, "CacheMetaReport", false, &ownerID)

	sched, err := repo.CreateSchedule(ctx, tenantID, ownerID, reports.CreateScheduleInput{
		TemplateID:     tmpl.ID,
		ScheduleName:   "CacheMetaSched",
		CronExpression: "0 8 * * 1-5",
	})
	if err != nil {
		t.Fatalf("CreateSchedule failed: %v", err)
	}
	defer func() {
		_, _ = db.Exec("DELETE FROM public.report_schedules WHERE id = $1", sched.ID)
		_, _ = db.Exec("DELETE FROM public.report_cache_metadata WHERE template_id = $1", tmpl.ID)
		_, _ = db.Exec("DELETE FROM public.report_executions WHERE template_id = $1", tmpl.ID)
	}()

	// Trigger using default executor
	res, err := repo.TriggerScheduleRun(ctx, tenantID, ownerID, false, sched.ID, nil)
	if err != nil {
		t.Fatalf("TriggerScheduleRun failed: %v", err)
	}

	// Assert in report_executions DB row
	var reqByInDB string
	err = db.QueryRow("SELECT requested_by FROM public.report_executions WHERE id = $1", res.ExecutionID).Scan(&reqByInDB)
	if err != nil {
		t.Fatalf("query report_executions failed: %v", err)
	}
	if reqByInDB != ownerID {
		t.Errorf("expected requested_by in DB to be %s, got %s", ownerID, reqByInDB)
	}

	// Assert in report_cache_metadata
	var expiresAt time.Time
	var hitCount int
	cacheKey := "sched_" + sched.ID.String()
	err = db.QueryRow(`
		SELECT expires_at, hit_count
		FROM public.report_cache_metadata
		WHERE tenant_id = $1 AND template_id = $2 AND cache_key = $3
	`, tenantID, tmpl.ID, cacheKey).Scan(&expiresAt, &hitCount)
	if err != nil {
		t.Fatalf("query report_cache_metadata failed: %v", err)
	}

	// Verify TTL is approximately 24 hours from now (+/- 5 minutes)
	expectedExpiry := time.Now().Add(24 * time.Hour)
	diff := expiresAt.Sub(expectedExpiry)
	if diff < -5*time.Minute || diff > 5*time.Minute {
		t.Errorf("expected expires_at approx %v, got %v (diff %v)", expectedExpiry, expiresAt, diff)
	}
}

// 7. TestListSchedules_tenantIsolation — confirms cross-tenant bleed does not occur
func TestListSchedules_tenantIsolation(t *testing.T) {
	db := getTestDB(t)
	repo := reports.NewRepository(db)
	ctx := context.Background()

	tenantA := createScheduleTestTenant(t, db, "TenantA")
	tenantB := createScheduleTestTenant(t, db, "TenantB")
	userA := createScheduleTestUser(t, db, tenantA)
	userB := createScheduleTestUser(t, db, tenantB)
	tmplA := createScheduleTestTemplate(t, db, tenantA, "TenantAReport", false, &userA)

	schedA, err := repo.CreateSchedule(ctx, tenantA, userA, reports.CreateScheduleInput{
		TemplateID:     tmplA.ID,
		ScheduleName:   "SchedTenantA",
		CronExpression: "0 8 * * 1-5",
	})
	if err != nil {
		t.Fatalf("CreateSchedule tenant A failed: %v", err)
	}
	defer func() {
		_, _ = db.Exec("DELETE FROM public.report_schedules WHERE id = $1", schedA.ID)
	}()

	// Querying schedules from Tenant B for Tenant A's template ID -> must return ErrNotFound
	_, err = repo.ListSchedulesForTemplate(ctx, tenantB, userB, tmplA.ID)
	if !errors.Is(err, reports.ErrNotFound) {
		t.Errorf("expected ErrNotFound when listing schedules across tenants, got: %v", err)
	}

	// Querying schedules from Tenant A -> succeeds and includes schedA
	listA, err := repo.ListSchedulesForTemplate(ctx, tenantA, userA, tmplA.ID)
	if err != nil {
		t.Fatalf("ListSchedulesForTemplate tenant A failed: %v", err)
	}
	if len(listA) != 1 || listA[0].ID != schedA.ID {
		t.Errorf("expected 1 schedule with ID %s, got %d results", schedA.ID, len(listA))
	}
}

// 8. TestCreateSchedule_goldCopyCoreReport_Allowed — tenant user can schedule gold-copy core reports
func TestCreateSchedule_goldCopyCoreReport_Allowed(t *testing.T) {
	db := getTestDB(t)
	repo := reports.NewRepository(db)
	ctx := context.Background()

	goldCopyID, err := repo.ResolveGoldCopyTenantID(ctx)
	if err != nil || goldCopyID == uuid.Nil {
		t.Skip("No gold copy tenant configured on alpha; skipping gold copy test")
	}

	// Create a core report in gold copy tenant
	goldOwnerID := createScheduleTestUser(t, db, goldCopyID)
	tmplCore := createScheduleTestTemplate(t, db, goldCopyID, "GoldCopyCoreSched", false, &goldOwnerID)

	// User from a standard client tenant schedules this core report
	clientTenantID := createScheduleTestTenant(t, db, "ClientTenant")
	clientUserID := createScheduleTestUser(t, db, clientTenantID)

	sched, err := repo.CreateSchedule(ctx, clientTenantID, clientUserID, reports.CreateScheduleInput{
		TemplateID:     tmplCore.ID,
		ScheduleName:   "Client Scheduled Core",
		CronExpression: "0 8 * * 1-5",
	})
	if err != nil {
		t.Fatalf("expected client tenant user to schedule gold copy core report, got: %v", err)
	}
	defer func() {
		_, _ = db.Exec("DELETE FROM public.report_schedules WHERE id = $1", sched.ID)
	}()

	if sched.TenantID != clientTenantID {
		t.Errorf("expected schedule tenant_id %s, got %s", clientTenantID, sched.TenantID)
	}
	if sched.ReportDefinitionID != tmplCore.ID {
		t.Errorf("expected schedule report_definition_id %s, got %s", tmplCore.ID, sched.ReportDefinitionID)
	}
}
