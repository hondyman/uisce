package platform

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hondyman/uisce/backend/internal/db"
	"github.com/hondyman/uisce/backend/internal/tenantdb"
)

type fakeResolver struct {
	calls  int
	tenant string
	app    string
	err    error
	closed int
}

func (f *fakeResolver) ResolveApp(ctx context.Context, app string) (*tenantdb.Pool, error) {
	f.calls++
	f.tenant, _ = db.GetTenantIDFromCtx(ctx)
	f.app = app
	return nil, f.err
}
func (f *fakeResolver) Close() { f.closed++ }

func TestGetConnection_AsksTheRouterForTheTenantsCoreDatasource(t *testing.T) {
	f := &fakeResolver{err: tenantdb.ErrUnbound}
	m := NewTenantDBManagerWithRouter(f, CoreApp)
	conn, err := m.GetConnection("tenant-1")
	require.Nil(t, conn)
	require.ErrorIs(t, err, tenantdb.ErrUnbound, "the router's reason must reach the caller")
	require.Equal(t, 1, f.calls)
	require.Equal(t, "tenant-1", f.tenant, "the tenant is placed in the context as the caller")
	require.Equal(t, "core", f.app)
}

func TestGetConnection_FailsClosed(t *testing.T) {
	t.Run("no tenant id never reaches the router", func(t *testing.T) {
		f := &fakeResolver{}
		_, err := NewTenantDBManagerWithRouter(f, CoreApp).GetConnection("")
		require.ErrorIs(t, err, tenantdb.ErrNoTenant)
		require.Zero(t, f.calls)
	})
	t.Run("any router failure is an error, never a default connection", func(t *testing.T) {
		for _, e := range []error{tenantdb.ErrTenantMismatch, tenantdb.ErrAmbiguousApp, tenantdb.ErrIncomplete, errors.New("alpha down")} {
			f := &fakeResolver{err: e}
			conn, err := NewTenantDBManagerWithRouter(f, CoreApp).GetConnection("t")
			require.Nil(t, conn)
			require.ErrorIs(t, err, e)
		}
	})
	t.Run("a nil manager is an error, not a panic", func(t *testing.T) {
		var m *TenantDBManager
		_, err := m.GetConnection("t")
		require.Error(t, err)
	})
	t.Run("a manager over a missing database is an error, not a panic", func(t *testing.T) {
		_, err := NewTenantDBManager(nil).GetConnection("11111111-2222-3333-4444-555555555555")
		require.Error(t, err)
	})
}

func TestCloseAll_ReleasesTheRouter(t *testing.T) {
	f := &fakeResolver{}
	NewTenantDBManagerWithRouter(f, CoreApp).CloseAll()
	require.Equal(t, 1, f.closed)
	var m *TenantDBManager
	m.CloseAll() // must not panic
}

// The manager must not grow its own connections or registry again: that was the second pool set
// and the plaintext platform.tenants lookup this change removed (ADR-030).
func TestManagerDoesNotReopenItsOwnPathAroundTheRouter(t *testing.T) {
	src, err := os.ReadFile("tenant_db_manager.go")
	require.NoError(t, err)
	body := string(src)
	for _, banned := range []string{"db_connection_string", "platform.tenants", "sql.Open", "pgxpool.New", "has_dedicated_db"} {
		// Comments may explain history; code may not use it.
		for _, line := range strings.Split(body, "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "//") {
				continue
			}
			require.NotContains(t, line, banned, "tenant_db_manager.go must reach tenant databases only through tenantdb")
		}
	}
}
