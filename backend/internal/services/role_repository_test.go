package services

import (
	"context"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func TestSQLRoleRepository_CRUDLifecycle(t *testing.T) {
	db := connectAlphaDB(t)
	ensureSemanticRoleSchema(t, db)
	truncateSemanticRoles(t, db)

	ctx := context.Background()
	repo := &sqlRoleRepository{db: db}

	role := newSeedRole("AlphaPortfolioManager", "Integration test role for sqlRoleRepository", models.RoleStatusActive, models.RoleTypeBusiness, []string{"integration", "alpha"}, []string{"front_office_performance_v1.3"})
	role.Owner = "tester@alpha"
	role.Attributes = map[string]string{"region": "global"}
	role.Tags = append(role.Tags, "lifecycle")
	role.CreatedAt = time.Now().UTC().Truncate(time.Second)
	role.UpdatedAt = role.CreatedAt
	role.AuditMetadata.CreatedAt = role.CreatedAt
	role.AuditMetadata.CreatedBy = role.Owner
	role.AuditMetadata.LastModifiedBy = role.Owner
	role.AuditMetadata.LastModifiedAt = timePtr(role.CreatedAt)

	if err := repo.CreateRole(ctx, role); err != nil {
		t.Fatalf("CreateRole failed: %v", err)
	}

	normalized := normalizeRoleKey(role.Name)
	fetched, err := repo.GetRoleByName(ctx, nil, normalized)
	if err != nil {
		t.Fatalf("GetRoleByName failed: %v", err)
	}
	if fetched == nil {
		t.Fatalf("expected role to be fetched")
	}
	if fetched.Name != role.Name {
		t.Fatalf("expected name %s, got %s", role.Name, fetched.Name)
	}
	if fetched.Owner != role.Owner {
		t.Fatalf("expected owner %s, got %s", role.Owner, fetched.Owner)
	}

	// Update metadata and ensure persistence works round-trip.
	role.Description = "Updated integration role description"
	role.Tags = append(role.Tags, "updated")
	role.BundleIDs = append(role.BundleIDs, "alpha-risk")
	role.Attributes["desk"] = "equities"
	role.Status = models.RoleStatusSuspended
	role.Version = "1.1.0"
	role.UpdatedAt = time.Now().UTC().Truncate(time.Second)
	role.Lifecycle.LastAction = "status_change"
	role.Lifecycle.LastActor = "tester@alpha"
	role.Lifecycle.LastNotes = "Suspended for regression"
	role.Lifecycle.SuspendedAt = timePtr(role.UpdatedAt)
	role.AuditTrail = append(role.AuditTrail, models.RoleChangeRecord{
		Version:   role.Version,
		State:     string(role.Status),
		Action:    "status_change",
		Actor:     "tester@alpha",
		Timestamp: role.UpdatedAt,
		Notes:     "Automated suspension during integration test",
	})

	if err := repo.SaveRole(ctx, role); err != nil {
		t.Fatalf("SaveRole failed: %v", err)
	}

	updated, err := repo.GetRoleByName(ctx, nil, normalized)
	if err != nil {
		t.Fatalf("GetRoleByName after update failed: %v", err)
	}
	if updated == nil {
		t.Fatalf("expected updated role to be fetched")
	}
	if updated.Description != role.Description {
		t.Fatalf("expected description %q, got %q", role.Description, updated.Description)
	}
	if updated.Status != models.RoleStatusSuspended {
		t.Fatalf("expected status Suspended, got %s", updated.Status)
	}
	if len(updated.BundleIDs) != len(role.BundleIDs) {
		t.Fatalf("expected %d bundle ids, got %d", len(role.BundleIDs), len(updated.BundleIDs))
	}
	if _, ok := updated.Attributes["desk"]; !ok {
		t.Fatalf("expected desk attribute to be persisted")
	}

	exists, err := repo.RoleExists(ctx, nil, normalized)
	if err != nil {
		t.Fatalf("RoleExists failed: %v", err)
	}
	if !exists {
		t.Fatalf("expected RoleExists to return true")
	}

	roles, err := repo.ListRoles(ctx, nil)
	if err != nil {
		t.Fatalf("ListRoles failed: %v", err)
	}
	if len(roles) == 0 {
		t.Fatalf("expected at least one role, got 0")
	}
}

func connectAlphaDB(t *testing.T) *sqlx.DB {
	t.Helper()

	dsn := alphaDatabaseURL()
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Skipf("skipping SQL role repository tests; unable to connect to %s: %v", sanitizeDSN(dsn), err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

func ensureSemanticRoleSchema(t *testing.T, db *sqlx.DB) {
	t.Helper()

	_, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS tenants (
            id uuid PRIMARY KEY,
            name text,
            created_at timestamptz DEFAULT now()
        );
    `)
	if err != nil {
		t.Fatalf("failed to ensure tenants table: %v", err)
	}

	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS semantic_roles (
            id uuid PRIMARY KEY,
            tenant_id uuid REFERENCES tenants(id) ON DELETE CASCADE,
            name text NOT NULL,
            normalized_name text NOT NULL,
            display_name text NOT NULL,
            description text,
            version text NOT NULL,
            status text NOT NULL,
            role_type text NOT NULL,
            owner text NOT NULL,
            scope text NOT NULL,
            tags jsonb NOT NULL DEFAULT '[]'::jsonb,
            attributes jsonb NOT NULL DEFAULT '{}'::jsonb,
            policies jsonb NOT NULL DEFAULT '[]'::jsonb,
            permissions jsonb NOT NULL DEFAULT '[]'::jsonb,
            attribute_constraints jsonb NOT NULL DEFAULT '[]'::jsonb,
            members jsonb NOT NULL DEFAULT '[]'::jsonb,
            bundle_ids jsonb NOT NULL DEFAULT '[]'::jsonb,
            audit_trail jsonb NOT NULL DEFAULT '[]'::jsonb,
            audit_metadata jsonb NOT NULL,
            lifecycle jsonb NOT NULL,
            created_at timestamptz NOT NULL DEFAULT now(),
            updated_at timestamptz NOT NULL DEFAULT now(),
            UNIQUE (tenant_id, normalized_name)
        );
    `)
	if err != nil {
		t.Fatalf("failed to ensure semantic_roles table: %v", err)
	}
}

func truncateSemanticRoles(t *testing.T, db *sqlx.DB) {
	t.Helper()
	if _, err := db.Exec(`TRUNCATE TABLE semantic_roles RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("failed to truncate semantic_roles: %v", err)
	}
}

func alphaDatabaseURL() string {
	if value := os.Getenv("ALPHA_DATABASE_URL"); value != "" {
		return value
	}
	if value := os.Getenv("ALPHA_PG_DSN"); value != "" {
		return value
	}
	if value := os.Getenv("POLICY_DB_URL"); value != "" {
		return value
	}
	if value := os.Getenv("DATABASE_URL"); value != "" {
		return value
	}
	return "postgres://postgres@localhost:5432/alpha?sslmode=disable"
}

// sanitizeDSN mirrors the server-side helper for logging DSNs without credentials.
func sanitizeDSN(dsn string) string {
	parsed, err := url.Parse(dsn)
	if err != nil {
		return "unknown"
	}
	parsed.User = nil
	if parsed.RawQuery != "" {
		parsed.RawQuery = ""
	}
	if parsed.Fragment != "" {
		parsed.Fragment = ""
	}
	return parsed.String()
}
