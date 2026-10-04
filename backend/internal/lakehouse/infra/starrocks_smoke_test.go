package infra

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Smoke test of the audit copy's StarRocks adapter against a REAL StarRocks, Lakekeeper and object
// store (ADR-036, ADR-044). Everything else in this package runs against a mock, so nothing there has
// proved that the DDL, the INSERT or the read-back work on the deployed versions. Run this before
// anything depends on the copy.
//
// It is opt-in and skips without LAKEHOUSE_SMOKE=1. It needs a tenant whose warehouse already exists in
// Lakekeeper (provision one through the lakehouse workflow first). It goes only through the adapter,
// never raw SQL:
//
//	LAKEHOUSE_SMOKE=1
//	LAKEHOUSE_SMOKE_TENANT_ID        the tenant's uuid
//	LAKEHOUSE_SMOKE_WAREHOUSE        its warehouse name (ivy-t-<id without hyphens>)
//	LAKEHOUSE_SMOKE_ACCESS_KEY / _SECRET_KEY   that tenant's own storage credential
//	LAKEHOUSE_STARROCKS_DSN, _CATALOG_URI, _S3_ENDPOINT, (_S3_REGION)   as the worker uses them
//
// The test appends to the tenant's real audit copy table, so use a tenant made for the purpose. Ids
// continue after the table's current highest id, so it is safe to re-run.
func TestSmoke_AuditCopyAgainstRealStarRocks(t *testing.T) {
	if os.Getenv("LAKEHOUSE_SMOKE") != "1" {
		t.Skip("LAKEHOUSE_SMOKE is not 1")
	}
	need := func(k string) string {
		v := os.Getenv(k)
		require.NotEmpty(t, v, "%s is required for the smoke test", k)
		return v
	}
	tenant, err := uuid.Parse(need("LAKEHOUSE_SMOKE_TENANT_ID"))
	require.NoError(t, err)
	spec := AuditDestinationSpec{TenantID: tenant, WarehouseName: need("LAKEHOUSE_SMOKE_WAREHOUSE"),
		AccessKeyID: need("LAKEHOUSE_SMOKE_ACCESS_KEY"), SecretKey: need("LAKEHOUSE_SMOKE_SECRET_KEY")}
	dest := AuditDestinationFromEnv()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	// 1. DDL, twice: the second call must be a no-op (idempotent), not an error.
	require.NoError(t, dest.EnsureAuditDestination(ctx, spec), "catalog, database and table creation")
	require.NoError(t, dest.EnsureAuditDestination(ctx, spec), "ensure must be idempotent")

	// 2. Append a chained batch after whatever is already there.
	start, err := dest.MaxAuditID(ctx, tenant)
	require.NoError(t, err)
	prev := strings.Repeat("0", 64)
	if start > 0 {
		h, ok, err := dest.AuditHash(ctx, tenant, start)
		require.NoError(t, err)
		require.True(t, ok, "the table reports a highest id with no such entry")
		prev = h
	}
	base := time.Now().UTC().Truncate(time.Microsecond)
	var rows []AuditRow
	for i := 1; i <= 5; i++ {
		h := fmt.Sprintf("%064x", start+int64(i)) // a fixed-width stand-in hash; the adapter does not recompute it
		rows = append(rows, AuditRow{ID: start + int64(i), At: base.Add(time.Duration(i) * time.Microsecond), ActorID: "smoke", ActorRole: "system",
			Action: "smoke_test", BeforeJSON: "", AfterJSON: `{"smoke":` + strconv.Itoa(i) + `,"quote":"it's \"quoted\" \\ done"}`, PrevHash: prev, Hash: h})
		prev = h
	}
	require.NoError(t, dest.AppendAudit(ctx, tenant, rows), "one INSERT is one Iceberg commit")

	// 3. Read it back three ways, and compare every field the verifier compares.
	max, err := dest.MaxAuditID(ctx, tenant)
	require.NoError(t, err)
	require.Equal(t, rows[len(rows)-1].ID, max)
	h, ok, err := dest.AuditHash(ctx, tenant, rows[2].ID)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, rows[2].Hash, h)

	got, err := dest.AuditRange(ctx, tenant, start, len(rows))
	require.NoError(t, err)
	require.Len(t, got, len(rows))
	for i, want := range rows {
		require.Equal(t, want.ID, got[i].ID)
		require.True(t, want.At.Equal(got[i].At), "timestamp survived the round trip: want %s got %s", want.At, got[i].At)
		require.Equal(t, want.ActorID, got[i].ActorID)
		require.Equal(t, want.ActorRole, got[i].ActorRole)
		require.Equal(t, want.Action, got[i].Action)
		require.Equal(t, want.AfterJSON, got[i].AfterJSON, "JSON text, including quotes and backslashes, must survive exactly")
		require.Equal(t, want.PrevHash, got[i].PrevHash)
		require.Equal(t, want.Hash, got[i].Hash)
	}
}
