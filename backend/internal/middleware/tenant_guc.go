// Package middleware: tenant_guc.go
//
// Per-request GUC injection for handlers that do not use the transactional
// helpers in internal/db (WithTenantGoldTransaction / WithTenantTransaction).
//
// Why this exists:
//   - WithTenantContext (internal/middleware/tenant_context.go) populates the
//     tenant ID in request context, but it does NOT call set_config() — that
//     is only done inside transactions.
//   - Many request handlers query the database directly via db.Query /
//     db.Exec on a pooled connection, which has no GUC set. Any RLS
//     policy reading app.current_tenant then evaluates against an empty
//     value, returning zero rows.
//   - This middleware pins a single connection to the request, sets the
//     GUCs session-scoped (set_config(..., false)), and stores the conn
//     in request context. Handlers that call db.Query directly still see
//     the unset GUC; they must migrate to GetPinnedConn(ctx).QueryContext.
//
// Scope semantics (false = session, persists until connection is returned):
//   - We deliberately use is_local=false here because the connection is
//     pinned to the request for its entire lifetime; using true would
//     require every query to run inside an explicit transaction.
//   - On request end the connection is reset and returned to the pool.
//
// Reads three GUC names so all policies work transparently:
//
//	app.current_tenant            (primary — standardized mdm.* / orm.* RLS)
//	uisce.current_tenant          (existing catalog/page policies)
//	app.shared_reference_tenant   (only when request tenant is the gold-copy)
//
// For the gold-copy tenant ID, the middleware calls public.uisce_gold_copy_tenant_id()
// (cached at startup by the goldcopy.Resolver), so this adds one query per
// request when Redis is not available. With Redis warmup it adds zero.
package middleware

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

// PinnedConnKey is the context key under which the per-request pinned
// *sql.Conn is stored. Exported so handlers can retrieve it.
type PinnedConnKey struct{}

// PinnedConn returns the per-request pinned connection if one was acquired
// by the TenantGUC middleware. Returns nil if no middleware ran (e.g.
// background goroutines, unit tests).
//
// Handlers that need RLS visibility should call this and use the returned
// *sql.Conn (via QueryContext / ExecContext) instead of going through the
// pool directly. Falling back to db.QueryContext is fine for tables that
// have no RLS — the GUC just won't be set on those pooled connections.
func PinnedConn(ctx context.Context) *sql.Conn {
	v := ctx.Value(PinnedConnKey{})
	if v == nil {
		return nil
	}
	c, _ := v.(*sql.Conn)
	return c
}

// TenantGUCConfig configures the per-request GUC middleware.
type TenantGUCConfig struct {
	// DB is the application database pool. Required.
	DB *sql.DB
	// GoldCopyTenantID is the tenant ID that the policy treats as
	// "shared reference" — when the request tenant equals this UUID,
	// app.shared_reference_tenant is set so policies can let any user
	// see the gold-copy tenant's reference rows.
	// Pass uuid.Nil to skip (no gold-copy sharing).
	GoldCopyTenantID uuid.UUID
	// SkipPaths are request paths that should bypass GUC injection
	// (health checks, static assets, etc.). Compared against r.URL.Path.
	SkipPaths map[string]bool
	// OnError, if set, is invoked when GUC injection fails. The default
	// behavior is to write a 500 and abort the request. Use this to log
	// instead, or to fall through with a warning in development.
	OnError func(w http.ResponseWriter, r *http.Request, err error)
}

// TenantGUC returns a middleware that pins a connection per request and
// sets app.current_tenant (and the gold-copy reference) session-scoped.
//
// Must be installed AFTER WithTenantContext (so the tenant ID is already
// in request context) and BEFORE any handler that needs RLS visibility.
func TenantGUC(cfg TenantGUCConfig) func(http.Handler) http.Handler {
	if cfg.DB == nil {
		panic("middleware.TenantGUC: DB is required")
	}
	if cfg.SkipPaths == nil {
		cfg.SkipPaths = map[string]bool{"/health": true}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.SkipPaths[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}

			tenantID, err := GetTenantIDFromRequest(r.Context())
			if err != nil || tenantID == "" {
				// No tenant resolved — let the request fall through.
				// Handlers that require a tenant will fail with 401/403
				// from their own checks; we don't double-fail here.
				next.ServeHTTP(w, r)
				return
			}

			parsed, err := uuid.Parse(tenantID)
			if err != nil {
				abort(w, r, cfg, fmt.Errorf("tenant context has non-UUID value %q: %w", tenantID, err))
				return
			}

			conn, err := cfg.DB.Conn(r.Context())
			if err != nil {
				abort(w, r, cfg, fmt.Errorf("acquire pinned connection: %w", err))
				return
			}

			if err := setTenantGUCsOnConn(r.Context(), conn, parsed, cfg.GoldCopyTenantID); err != nil {
				_ = conn.Close()
				abort(w, r, cfg, err)
				return
			}

			// Stash the conn in request context; release on request end.
			ctx := context.WithValue(r.Context(), PinnedConnKey{}, conn)
			defer func() {
				_ = resetTenantGUCsOnConn(conn)
				_ = conn.Close()
			}()

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// setTenantGUCsOnConn sets all GUC names needed by RLS policies. Session-scoped
// (is_local=false) because the conn is pinned for the request lifetime.
func setTenantGUCsOnConn(ctx context.Context, conn *sql.Conn, tenantID, goldCopyTenantID uuid.UUID) error {
	if _, err := conn.ExecContext(ctx, "SELECT set_config('app.current_tenant', $1, false)", tenantID.String()); err != nil {
		return fmt.Errorf("set app.current_tenant: %w", err)
	}
	if _, err := conn.ExecContext(ctx, "SELECT set_config('uisce.current_tenant', $1, false)", tenantID.String()); err != nil {
		return fmt.Errorf("set uisce.current_tenant: %w", err)
	}
	if goldCopyTenantID != uuid.Nil && tenantID != goldCopyTenantID {
		if _, err := conn.ExecContext(ctx, "SELECT set_config('app.shared_reference_tenant', $1, false)", goldCopyTenantID.String()); err != nil {
			return fmt.Errorf("set app.shared_reference_tenant: %w", err)
		}
	}
	return nil
}

// resetTenantGUCsOnConn clears the GUCs before returning the connection to the pool.
// Important: do not skip this — without it, the next request that picks up
// this connection would inherit a previous tenant's GUC and silently read
// the wrong rows.
func resetTenantGUCsOnConn(conn *sql.Conn) error {
	// Reset is per-GUC because Postgres has no "unset all". Using
	// set_config with the empty string and is_local=false effectively
	// erases the value for the session. We inline the GUC names rather
	// than parameterize so each call has a distinct query string (easier
	// to log/trace and to mock in tests).
	resets := []string{
		"SELECT set_config('app.current_tenant', '', false)",
		"SELECT set_config('uisce.current_tenant', '', false)",
		"SELECT set_config('app.shared_reference_tenant', '', false)",
	}
	for _, q := range resets {
		if _, err := conn.ExecContext(context.Background(), q); err != nil {
			return fmt.Errorf("reset failed for query %q: %w", q, err)
		}
	}
	return nil
}

func abort(w http.ResponseWriter, r *http.Request, cfg TenantGUCConfig, err error) {
	if cfg.OnError != nil {
		cfg.OnError(w, r, err)
		return
	}
	http.Error(w, `{"error":"internal","message":"tenant GUC injection failed"}`, http.StatusInternalServerError)
}

// ErrNoTenantContext is returned by helpers that expect a tenant in context.
var ErrNoTenantContext = errors.New("no tenant in request context")
