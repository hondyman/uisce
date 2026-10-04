package registry_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/lakehouse/registry"
	"github.com/stretchr/testify/require"
)

// platformDB opens the test database and resets the one-row table. It needs migration 20261216_001.
func platformDB(t *testing.T) (*sql.DB, *registry.PlatformStore) {
	t.Helper()
	db := openDB(t)
	// The guard refuses to delete a provisioned row, so the reset lifts it for the test only.
	_, err := db.Exec(`ALTER TABLE public.platform_lakehouse DISABLE TRIGGER trg_platform_lakehouse_guard`)
	if err != nil {
		t.Skipf("public.platform_lakehouse is not available to this role: %v", err)
	}
	_, err = db.Exec(`DELETE FROM public.platform_lakehouse`)
	require.NoError(t, err)
	_, err = db.Exec(`ALTER TABLE public.platform_lakehouse ENABLE TRIGGER trg_platform_lakehouse_guard`)
	require.NoError(t, err)
	return db, registry.NewPlatformStore(db)
}

func TestPlatform_NothingIsConfiguredUntilARetentionIsChosen(t *testing.T) {
	_, s := platformDB(t)
	_, err := s.Get(context.Background())
	require.ErrorIs(t, err, registry.ErrPlatformNotConfigured)
	require.ErrorIs(t, s.MarkCredentialIssued(context.Background()), registry.ErrPlatformNotConfigured)
	require.ErrorIs(t, s.MarkProvisioned(context.Background(), uuid.New(), "k", 365), registry.ErrPlatformNotConfigured)
}

func TestPlatform_RetentionHasNoDefaultAndOnlyRises(t *testing.T) {
	_, s := platformDB(t)
	ctx := context.Background()
	require.ErrorIs(t, s.ConfigureRetention(ctx, 0), registry.ErrInvalidRetention)
	require.ErrorIs(t, s.ConfigureRetention(ctx, registry.MaxRetentionDays+1), registry.ErrInvalidRetention)

	require.NoError(t, s.ConfigureRetention(ctx, 365))
	require.NoError(t, s.ConfigureRetention(ctx, 365), "the same value is a no-op")
	require.NoError(t, s.ConfigureRetention(ctx, 730))
	require.ErrorIs(t, s.ConfigureRetention(ctx, 365), registry.ErrRetentionLowered, "never silently ignored")
	c, err := s.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, 730, c.AuditRetentionDays)
	require.False(t, c.Provisioned)
	require.False(t, c.CredentialIssued)
}

func TestPlatform_CredentialIssuedIsRecordedOnce(t *testing.T) {
	db, s := platformDB(t)
	ctx := context.Background()
	require.NoError(t, s.ConfigureRetention(ctx, 365))
	require.NoError(t, s.MarkCredentialIssued(ctx))
	var first sql.NullTime
	require.NoError(t, db.QueryRow(`SELECT credential_issued_at FROM public.platform_lakehouse`).Scan(&first))
	require.NoError(t, s.MarkCredentialIssued(ctx), "a retry is a no-op")
	var second sql.NullTime
	require.NoError(t, db.QueryRow(`SELECT credential_issued_at FROM public.platform_lakehouse`).Scan(&second))
	require.True(t, first.Time.Equal(second.Time), "the first issue time stands")
	c, _ := s.Get(ctx)
	require.True(t, c.CredentialIssued)
}

func TestPlatform_ProvisioningRecordsTheWarehouseOnceAndRefusesAnother(t *testing.T) {
	db, s := platformDB(t)
	ctx := context.Background()
	require.NoError(t, s.ConfigureRetention(ctx, 365))
	wh := uuid.New()
	require.NoError(t, s.MarkProvisioned(ctx, wh, "ivy-control", 365))
	require.NoError(t, s.MarkProvisioned(ctx, wh, "ivy-control", 365), "idempotent for the same warehouse")
	c, err := s.Get(ctx)
	require.NoError(t, err)
	require.True(t, c.Provisioned)
	require.Equal(t, wh, *c.WarehouseID)
	require.Equal(t, 365, *c.RetentionAppliedDays)

	require.ErrorIs(t, s.MarkProvisioned(ctx, uuid.New(), "ivy-control", 365), registry.ErrWarehouseMismatch)
	c, _ = s.Get(ctx)
	require.Equal(t, wh, *c.WarehouseID, "the recorded warehouse is untouched")

	require.Error(t, s.MarkProvisioned(ctx, uuid.Nil, "k", 365))
	require.Error(t, s.MarkProvisioned(ctx, wh, "", 365))

	// The database is the backstop for a writer that skips the Go checks.
	_, err = db.Exec(`UPDATE public.platform_lakehouse SET audit_retention_days = 1`)
	require.Error(t, err, "retention cannot be lowered")
	_, err = db.Exec(`UPDATE public.platform_lakehouse SET lakekeeper_warehouse_id = gen_random_uuid()`)
	require.Error(t, err, "the warehouse id is set once")
	_, err = db.Exec(`DELETE FROM public.platform_lakehouse`)
	require.Error(t, err, "a provisioned record cannot be deleted")
	_, err = db.Exec(`UPDATE public.platform_lakehouse SET bucket = 'ivy-t-x'`)
	require.Error(t, err, "the bucket is immutable")
}

func TestPlatform_OnlyOneRowCanExist(t *testing.T) {
	db, s := platformDB(t)
	require.NoError(t, s.ConfigureRetention(context.Background(), 365))
	_, err := db.Exec(`INSERT INTO public.platform_lakehouse (name, bucket, audit_retention_days) VALUES ('ivy-other', 'ivy-other', 365)`)
	require.Error(t, err, "the name is the only legal value")
	_, err = db.Exec(`INSERT INTO public.platform_lakehouse (name, bucket, audit_retention_days) VALUES ('ivy-control', 'ivy-control', 365)`)
	require.Error(t, err, "and it is the primary key")
}
