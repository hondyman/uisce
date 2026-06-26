package repository

import (
	"testing"

	"calendar-service/internal/testutil"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func TestTenantCreate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tdb := testutil.NewTestDB(t)
	defer tdb.Close(t)

	ctx := tdb.Context()
	logger := logrus.NewEntry(logrus.New())
	repos := NewPostgresRepositories(tdb.Pool, logger)

	tenant := &Tenant{
		ID:     uuid.New(),
		Name:   "test-tenant-create",
		Region: "us-west-2",
	}

	err := repos.Tenant.Create(ctx, tenant)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}

	if tenant.CreatedAt.IsZero() {
		t.Error("CreatedAt not set")
	}
	if tenant.UpdatedAt.IsZero() {
		t.Error("UpdatedAt not set")
	}
}

func TestTenantGetByID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tdb := testutil.NewTestDB(t)
	defer tdb.Close(t)

	ctx := tdb.Context()
	logger := logrus.NewEntry(logrus.New())
	repos := NewPostgresRepositories(tdb.Pool, logger)

	tenant := &Tenant{
		ID:     uuid.New(),
		Name:   "test-tenant-getbyid",
		Region: "eu-west-1",
	}

	repos.Tenant.Create(ctx, tenant)

	retrieved, err := repos.Tenant.GetByID(ctx, tenant.ID)
	if err != nil {
		t.Fatalf("Failed to get tenant: %v", err)
	}

	if retrieved.ID != tenant.ID {
		t.Errorf("Expected ID %s, got %s", tenant.ID, retrieved.ID)
	}
	if retrieved.Name != tenant.Name {
		t.Errorf("Expected name %s, got %s", tenant.Name, retrieved.Name)
	}
}

func TestTenantGetByName(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tdb := testutil.NewTestDB(t)
	defer tdb.Close(t)

	ctx := tdb.Context()
	logger := logrus.NewEntry(logrus.New())
	repos := NewPostgresRepositories(tdb.Pool, logger)

	name := "test-tenant-getbyname-" + uuid.New().String()[:8]
	tenant := &Tenant{
		ID:     uuid.New(),
		Name:   name,
		Region: "ap-southeast-1",
	}

	repos.Tenant.Create(ctx, tenant)

	retrieved, err := repos.Tenant.GetByName(ctx, name)
	if err != nil {
		t.Fatalf("Failed to get tenant by name: %v", err)
	}

	if retrieved.ID != tenant.ID {
		t.Errorf("Expected ID %s, got %s", tenant.ID, retrieved.ID)
	}
}

func TestTenantList(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tdb := testutil.NewTestDB(t)
	defer tdb.Close(t)

	ctx := tdb.Context()
	logger := logrus.NewEntry(logrus.New())
	repos := NewPostgresRepositories(tdb.Pool, logger)

	// Create multiple tenants
	for i := 0; i < 3; i++ {
		tenant := &Tenant{
			ID:     uuid.New(),
			Name:   "test-list-" + uuid.New().String()[:8],
			Region: "us-east-1",
		}
		repos.Tenant.Create(ctx, tenant)
	}

	tenants, err := repos.Tenant.List(ctx, 10, 0)
	if err != nil {
		t.Fatalf("Failed to list tenants: %v", err)
	}

	if len(tenants) < 3 {
		t.Errorf("Expected at least 3 tenants, got %d", len(tenants))
	}
}

func TestTenantUpdate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tdb := testutil.NewTestDB(t)
	defer tdb.Close(t)

	ctx := tdb.Context()
	logger := logrus.NewEntry(logrus.New())
	repos := NewPostgresRepositories(tdb.Pool, logger)

	tenant := &Tenant{
		ID:     uuid.New(),
		Name:   "test-tenant-update",
		Region: "us-east-1",
	}

	repos.Tenant.Create(ctx, tenant)
	originalCreatedAt := tenant.CreatedAt

	// Update
	tenant.Region = "us-west-2"
	err := repos.Tenant.Update(ctx, tenant)
	if err != nil {
		t.Fatalf("Failed to update tenant: %v", err)
	}

	// Verify update
	retrieved, err := repos.Tenant.GetByID(ctx, tenant.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve updated tenant: %v", err)
	}

	if retrieved.Region != "us-west-2" {
		t.Errorf("Expected region us-west-2, got %s", retrieved.Region)
	}

	// CreatedAt should not change
	if retrieved.CreatedAt != originalCreatedAt {
		t.Error("CreatedAt changed on update")
	}

	// UpdatedAt should change
	if !retrieved.UpdatedAt.After(originalCreatedAt) {
		t.Error("UpdatedAt not updated")
	}
}

func TestTenantDelete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tdb := testutil.NewTestDB(t)
	defer tdb.Close(t)

	ctx := tdb.Context()
	logger := logrus.NewEntry(logrus.New())
	repos := NewPostgresRepositories(tdb.Pool, logger)

	tenant := &Tenant{
		ID:     uuid.New(),
		Name:   "test-tenant-delete",
		Region: "us-east-1",
	}

	repos.Tenant.Create(ctx, tenant)

	// Delete
	err := repos.Tenant.Delete(ctx, tenant.ID)
	if err != nil {
		t.Fatalf("Failed to delete tenant: %v", err)
	}

	// Verify deletion (soft delete - sets deleted_at)
	_, err = repos.Tenant.GetByID(ctx, tenant.ID)
	if err == nil {
		t.Error("Expected error getting deleted tenant, got nil")
	}
}
