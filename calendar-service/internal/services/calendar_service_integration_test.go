package services

import (
	"context"
	"testing"

	"calendar-service/internal/repository"

	"github.com/sirupsen/logrus"
)

// ============================================================================
// Phase 3 Integration Tests: Tenant Context Flow
// ============================================================================

func TestPhase3CalendarCreateWithTenant(t *testing.T) {
	// Setup
	repo := repository.NewInMemoryCalendarRepository(logrus.NewEntry(logrus.New()))
	repoAdapter := NewRepositoryAdapter(repo, logrus.NewEntry(logrus.New()))
	service := NewCalendarServiceImpl(repoAdapter, logrus.NewEntry(logrus.New()))
	ctx := context.Background()

	tenantID := "tenant-123"
	userID := "user-456"
	name := "Q1 2026 Calendar"

	// Act
	calendar, err := service.Create(ctx, tenantID, userID, name, "Fiscal calendar", "UTC")

	// Assert
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if calendar.TenantID != tenantID {
		t.Errorf("Expected tenant_id %s, got %s", tenantID, calendar.TenantID)
	}

	if calendar.CreatedBy != userID {
		t.Errorf("Expected created_by %s, got %s", userID, calendar.CreatedBy)
	}

	if calendar.Name != name {
		t.Errorf("Expected name %s, got %s", name, calendar.Name)
	}

	t.Log("✓ Calendar created with correct tenant context")
}

func TestPhase3CalendarGetByTenant(t *testing.T) {
	// Setup
	repo := repository.NewInMemoryCalendarRepository(logrus.NewEntry(logrus.New()))
	repoAdapter := NewRepositoryAdapter(repo, logrus.NewEntry(logrus.New()))
	service := NewCalendarServiceImpl(repoAdapter, logrus.NewEntry(logrus.New()))
	ctx := context.Background()

	// Create calendar for tenant-123
	cal1, _ := service.Create(ctx, "tenant-123", "user-456", "Calendar 1", "", "UTC")

	// Act & Assert: Same tenant CAN retrieve
	retrieved, err := service.GetByID(ctx, "tenant-123", cal1.ID)
	if err != nil {
		t.Fatalf("GetByID failed for same tenant: %v", err)
	}
	if retrieved.ID != cal1.ID {
		t.Errorf("Retrieved wrong calendar")
	}
	t.Log("✓ Same tenant can retrieve calendar")

	// Act & Assert: Different tenant CANNOT retrieve
	_, err = service.GetByID(ctx, "tenant-789", cal1.ID)
	if err == nil {
		t.Fatal("GetByID should fail for different tenant")
	}
	t.Log("✓ Different tenant blocked from accessing calendar")
}

func TestPhase3CrossTenantAccessDenied(t *testing.T) {
	// Setup
	repo := repository.NewInMemoryCalendarRepository(logrus.NewEntry(logrus.New()))
	repoAdapter := NewRepositoryAdapter(repo, logrus.NewEntry(logrus.New()))
	service := NewCalendarServiceImpl(repoAdapter, logrus.NewEntry(logrus.New()))
	ctx := context.Background()

	// Create calendar for Tenant A
	calendarID, _ := service.Create(ctx, "tenant-a", "user-1", "Calendar A", "", "UTC")

	// Tenant B tries to access Tenant A's calendar
	_, err := service.GetByID(ctx, "tenant-b", calendarID.ID)

	// Assert: Access denied
	if err == nil {
		t.Fatal("Expected error for cross-tenant access, got nil")
	}

	t.Logf("✓ Cross-tenant access blocked: %v", err)
}

func TestPhase3ListByTenantIsolation(t *testing.T) {
	// Setup
	repo := repository.NewInMemoryCalendarRepository(logrus.NewEntry(logrus.New()))
	repoAdapter := NewRepositoryAdapter(repo, logrus.NewEntry(logrus.New()))
	service := NewCalendarServiceImpl(repoAdapter, logrus.NewEntry(logrus.New()))
	ctx := context.Background()

	// Create 3 calendars for Tenant A
	service.Create(ctx, "tenant-a", "user-1", "Calendar A1", "", "UTC")
	service.Create(ctx, "tenant-a", "user-1", "Calendar A2", "", "UTC")
	service.Create(ctx, "tenant-a", "user-1", "Calendar A3", "", "UTC")

	// Create 2 calendars for Tenant B
	service.Create(ctx, "tenant-b", "user-2", "Calendar B1", "", "UTC")
	service.Create(ctx, "tenant-b", "user-2", "Calendar B2", "", "UTC")

	// Act: List calendars for Tenant A
	calsA, _ := service.ListByTenant(ctx, "tenant-a", 10, 0)
	calsB, _ := service.ListByTenant(ctx, "tenant-b", 10, 0)

	// Assert: Each tenant only sees their own calendars
	if len(calsA) != 3 {
		t.Errorf("Expected 3 calendars for tenant-a, got %d", len(calsA))
	}
	if len(calsB) != 2 {
		t.Errorf("Expected 2 calendars for tenant-b, got %d", len(calsB))
	}

	// Verify all calendars have correct tenant_id
	for _, cal := range calsA {
		if cal.TenantID != "tenant-a" {
			t.Errorf("Found calendar from wrong tenant in tenant-a list: %s", cal.TenantID)
		}
	}

	for _, cal := range calsB {
		if cal.TenantID != "tenant-b" {
			t.Errorf("Found calendar from wrong tenant in tenant-b list: %s", cal.TenantID)
		}
	}

	t.Log("✓ Tenant isolation verified: each tenant sees only their calendars")
}

func TestPhase3UpdateWithTenantVerification(t *testing.T) {
	// Setup
	repo := repository.NewInMemoryCalendarRepository(logrus.NewEntry(logrus.New()))
	repoAdapter := NewRepositoryAdapter(repo, logrus.NewEntry(logrus.New()))
	service := NewCalendarServiceImpl(repoAdapter, logrus.NewEntry(logrus.New()))
	ctx := context.Background()

	// Create calendar for Tenant A
	cal, _ := service.Create(ctx, "tenant-a", "user-1", "Original Name", "", "UTC")

	// Act: Tenant A updates (should succeed)
	updated, err := service.Update(ctx, "tenant-a", cal.ID, "user-1", map[string]interface{}{
		"name": "Updated Name",
	})

	// Assert: Success
	if err != nil {
		t.Fatalf("Update failed for same tenant: %v", err)
	}
	if updated.Name != "Updated Name" {
		t.Errorf("Expected updated name, got %s", updated.Name)
	}
	t.Log("✓ Tenant A updated their calendar")

	// Act: Tenant B tries to update (should fail)
	_, err = service.Update(ctx, "tenant-b", cal.ID, "user-2", map[string]interface{}{
		"name": "Hacked Name",
	})

	// Assert: Access denied
	if err == nil {
		t.Fatal("Update should fail for different tenant")
	}

	// Verify calendar wasn't modified
	current, _ := service.GetByID(ctx, "tenant-a", cal.ID)
	if current.Name != "Updated Name" {
		t.Errorf("Calendar was modified by cross-tenant update attempt")
	}

	t.Log("✓ Cross-tenant update blocked, calendar unchanged")
}

func TestPhase3DeleteWithTenantVerification(t *testing.T) {
	// Setup
	repo := repository.NewInMemoryCalendarRepository(logrus.NewEntry(logrus.New()))
	repoAdapter := NewRepositoryAdapter(repo, logrus.NewEntry(logrus.New()))
	service := NewCalendarServiceImpl(repoAdapter, logrus.NewEntry(logrus.New()))
	ctx := context.Background()

	// Create calendar for Tenant A
	cal, _ := service.Create(ctx, "tenant-a", "user-1", "Calendar to Delete", "", "UTC")

	// Act: Tenant B tries to delete (should fail)
	err := service.Delete(ctx, "tenant-b", cal.ID, "user-2")

	// Assert: Access denied
	if err == nil {
		t.Fatal("Delete should fail for different tenant")
	}

	// Verify calendar still exists
	retrieved, _ := service.GetByID(ctx, "tenant-a", cal.ID)
	if retrieved == nil {
		t.Fatal("Calendar should still exist after failed cross-tenant delete")
	}

	t.Log("✓ Cross-tenant delete blocked, calendar still exists")

	// Act: Tenant A deletes (should succeed)
	err = service.Delete(ctx, "tenant-a", cal.ID, "user-1")

	// Assert: Success
	if err != nil {
		t.Fatalf("Delete failed for same tenant: %v", err)
	}

	// Verify calendar is deleted
	_, err = service.GetByID(ctx, "tenant-a", cal.ID)
	if err == nil {
		t.Fatal("Calendar should not exist after delete")
	}

	t.Log("✓ Tenant A successfully deleted their calendar")
}

func TestPhase3MissingTenantRejected(t *testing.T) {
	// Setup
	repo := repository.NewInMemoryCalendarRepository(logrus.NewEntry(logrus.New()))
	repoAdapter := NewRepositoryAdapter(repo, logrus.NewEntry(logrus.New()))
	service := NewCalendarServiceImpl(repoAdapter, logrus.NewEntry(logrus.New()))
	ctx := context.Background()

	// Act: Try to create without tenant
	_, err := service.Create(ctx, "", "user-1", "Calendar", "", "UTC")

	// Assert: Rejected
	if err == nil {
		t.Fatal("Create should fail without tenant_id")
	}

	t.Log("✓ Create rejected: missing tenant_id")
}

func TestPhase3MissingUserRejected(t *testing.T) {
	// Setup
	repo := repository.NewInMemoryCalendarRepository(logrus.NewEntry(logrus.New()))
	repoAdapter := NewRepositoryAdapter(repo, logrus.NewEntry(logrus.New()))
	service := NewCalendarServiceImpl(repoAdapter, logrus.NewEntry(logrus.New()))
	ctx := context.Background()

	// Act: Try to create without user
	_, err := service.Create(ctx, "tenant-a", "", "Calendar", "", "UTC")

	// Assert: Rejected
	if err == nil {
		t.Fatal("Create should fail without user_id")
	}

	t.Log("✓ Create rejected: missing user_id")
}

func TestPhase3AuditContextCarriedThrough(t *testing.T) {
	// Setup - we'll verify this by checking that operations don't panic without tenant context
	repo := repository.NewInMemoryCalendarRepository(logrus.NewEntry(logrus.New()))
	repoAdapter := NewRepositoryAdapter(repo, logrus.NewEntry(logrus.New()))
	service := NewCalendarServiceImpl(repoAdapter, logrus.NewEntry(logrus.New()))
	ctx := context.Background()

	tenantID := "tenant-123"
	userID := "user-456"

	// Create
	cal, _ := service.Create(ctx, tenantID, userID, "Calendar", "", "UTC")

	// Verify creation metadata
	if cal.CreatedBy != userID {
		t.Fatalf("CreatedBy not set correctly: expected %s, got %s", userID, cal.CreatedBy)
	}
	if cal.TenantID != tenantID {
		t.Fatalf("TenantID not set correctly: expected %s, got %s", tenantID, cal.TenantID)
	}
	if cal.CreatedAt.IsZero() {
		t.Fatal("CreatedAt not set")
	}

	// Update
	service.Update(ctx, tenantID, cal.ID, userID, map[string]interface{}{
		"name": "Updated",
	})

	updated, _ := service.GetByID(ctx, tenantID, cal.ID)

	// Verify update metadata
	if updated.UpdatedBy != userID {
		t.Fatalf("UpdatedBy not set correctly: expected %s, got %s", userID, updated.UpdatedBy)
	}
	if updated.UpdatedAt.Before(updated.CreatedAt) {
		t.Fatal("UpdatedAt should be after CreatedAt")
	}

	t.Log("✓ Audit context (user_id, tenant_id, timestamps) carried through all operations")
}

func TestPhase3MultiTenantConcurrency(t *testing.T) {
	// Test that multiple tenants can operate independently
	repo := repository.NewInMemoryCalendarRepository(logrus.NewEntry(logrus.New()))
	repoAdapter := NewRepositoryAdapter(repo, logrus.NewEntry(logrus.New()))
	service := NewCalendarServiceImpl(repoAdapter, logrus.NewEntry(logrus.New()))
	ctx := context.Background()

	// Run concurrent operations from different tenants
	results := make(chan error, 3)

	go func() {
		_, err := service.Create(ctx, "tenant-x", "user-x", "Cal X1", "", "UTC")
		results <- err
	}()

	go func() {
		_, err := service.Create(ctx, "tenant-y", "user-y", "Cal Y1", "", "UTC")
		results <- err
	}()

	go func() {
		_, err := service.Create(ctx, "tenant-z", "user-z", "Cal Z1", "", "UTC")
		results <- err
	}()

	// Collect results
	for i := 0; i < 3; i++ {
		if err := <-results; err != nil {
			t.Errorf("Concurrent operation failed: %v", err)
		}
	}

	// Verify isolation
	xCals, _ := service.ListByTenant(ctx, "tenant-x", 10, 0)
	yCals, _ := service.ListByTenant(ctx, "tenant-y", 10, 0)
	zCals, _ := service.ListByTenant(ctx, "tenant-z", 10, 0)

	if len(xCals) != 1 || len(yCals) != 1 || len(zCals) != 1 {
		t.Errorf("Expected 1 calendar per tenant, got %d, %d, %d", len(xCals), len(yCals), len(zCals))
	}

	t.Log("✓ Multiple tenants can operate concurrently with full isolation")
}

// ============================================================================
// Test Helpers
// ============================================================================

// newTenantContext creates a test tenant context
func newTenantContext(tenantID, userID string) TenantContext {
	return TenantContext{
		TenantID: tenantID,
		UserID:   userID,
		Roles:    []string{"user"},
		Email:    "user@example.com",
	}
}

// eof
