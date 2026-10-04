package platform

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/hondyman/uisce/backend/internal/db"
	"github.com/hondyman/uisce/backend/internal/dscreds"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/hondyman/uisce/backend/internal/tenantdb"
	"github.com/jmoiron/sqlx"
)

// CoreApp is the app code of the datasource this manager serves: a tenant's one "core" database.
const CoreApp = "core"

// resolver is the part of tenantdb.Router the manager uses.
type resolver interface {
	ResolveApp(ctx context.Context, app string) (*tenantdb.Pool, error)
	Close()
}

// TenantDBManager hands out a tenant's database connection to code written against *sql.DB (the
// wealth service and activities, business-object instance operations).
//
// It no longer holds connections of its own. Every call resolves through tenantdb (ADR-030): the
// tenant's datasource for the app is found in alpha's datasource model, its owner is checked, its
// binding must be active, its credential comes from the secrets store, and the connection is
// asserted to be on the registered database as the registered user. The older design read a
// plaintext db_connection_string from platform.tenants, a second registry that bypassed all of
// that, and kept a second pool set per tenant.
//
// tenantID is the caller's argument here, as it always was: these callers are workflow activities
// and services that have no request context. It is placed in the context as the tenant the router
// treats as the caller, so the datasource must belong to THAT tenant, but the authenticity of
// tenantID remains the caller's responsibility.
type TenantDBManager struct {
	router  resolver
	app     string
	timeout time.Duration
	initErr error
}

// NewTenantDBManager builds a manager over alpha with the default router limits. A router that
// cannot be built is not a reason to fall back: the manager is returned and every GetConnection
// fails with the reason.
func NewTenantDBManager(centralDB *sql.DB) *TenantDBManager {
	r, err := tenantdb.New(tenantdb.Config{
		Registry: &tenantdb.AlphaRegistry{
			DB:       centralDB,
			Resolver: security.NewDBDatasourceResolver(sqlx.NewDb(centralDB, "pgx")),
			Creds:    dscreds.Default(),
		},
		CallerTenant:    db.GetTenantIDFromCtx,
		MaxPools:        256,
		MaxConnsPerPool: 10,
		IdleTTL:         10 * time.Minute,
		DialTimeout:     10 * time.Second,
		// A few seconds: a suspension or offboarding in alpha bites within this long, and a warm
		// connection no longer costs alpha 18 statements per call (see tenantdb.Config.AuthTTL).
		AuthTTL: tenantdb.DefaultAuthTTL,
	})
	if err != nil {
		return &TenantDBManager{app: CoreApp, timeout: 15 * time.Second, initErr: err}
	}
	return NewTenantDBManagerWithRouter(r, CoreApp)
}

// NewTenantDBManagerWithRouter builds a manager over an existing router and app code.
func NewTenantDBManagerWithRouter(r resolver, app string) *TenantDBManager {
	return &TenantDBManager{router: r, app: app, timeout: 15 * time.Second}
}

// GetConnection returns the tenant's *sql.DB. Any failure, including a tenant with no datasource
// for the app, an inactive binding or an unreachable database, is an error; there is no default
// database and no fallback.
func (m *TenantDBManager) GetConnection(tenantID string) (*sql.DB, error) {
	if m == nil {
		return nil, errors.New("tenant database manager is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()
	return m.GetConnectionContext(ctx, tenantID)
}

// GetConnectionContext is GetConnection with the caller's context (deadline, cancellation).
func (m *TenantDBManager) GetConnectionContext(ctx context.Context, tenantID string) (*sql.DB, error) {
	if m == nil {
		return nil, errors.New("tenant database manager is not configured")
	}
	if m.initErr != nil {
		return nil, fmt.Errorf("tenant database manager unavailable: %w", m.initErr)
	}
	if tenantID == "" {
		return nil, tenantdb.ErrNoTenant
	}
	p, err := m.router.ResolveApp(db.WithTenantContextToCtx(ctx, tenantID), m.app)
	if err != nil {
		return nil, fmt.Errorf("tenant %s: %w", tenantID, err)
	}
	return p.SQLDB(), nil
}

// CloseAll releases every pool the manager's router holds.
func (m *TenantDBManager) CloseAll() {
	if m != nil && m.router != nil {
		m.router.Close()
	}
}
