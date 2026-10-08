package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/lakehouse"
	"github.com/hondyman/uisce/backend/internal/security"
)

// LakehouseStatusHandler is the read-only HTTP surface for Platform > Lakehouse status and
// the tenant-scoped "My data" view. Two endpoints, one service, two authz scopes.
//
//	GET /api/admin/lakehouse/status    platform admins only; full payload (cluster + RGs + every tenant)
//	GET /api/tenant/lakehouse/status   any signed-in user; payload is the session tenant's row only
//
// The tenant name for the second endpoint is ALWAYS read from the verified session, never
// from a query parameter. Taking it from a query parameter is an IDOR that hands every
// tenant admin the platform view — the exact failure the runbook gap 4 calls out.
type LakehouseStatusHandler struct {
	Svc *lakehouse.StatusService
}

// NewLakehouseStatusHandler wires the service. The caller owns lifecycle.
func NewLakehouseStatusHandler(svc *lakehouse.StatusService) *LakehouseStatusHandler {
	return &LakehouseStatusHandler{Svc: svc}
}

// RegisterRoutes mounts the two endpoints on a chi router. Both endpoints are mounted
// inside the /api group with the existing auth middleware in front of them; that middleware
// fills the auth context this handler reads from.
func (h *LakehouseStatusHandler) RegisterRoutes(r chi.Router) {
	r.Get("/admin/lakehouse/status", h.platformStatus)
	r.Get("/tenant/lakehouse/status", h.tenantStatus)
}

// platformStatus is the global view. Requires a global admin.
func (h *LakehouseStatusHandler) platformStatus(w http.ResponseWriter, r *http.Request) {
	auth, ok := security.RequireAuth(w, r)
	if !ok {
		return
	}
	if !auth.IsGlobalAdmin {
		writeLakehouseStatusError(w, http.StatusForbidden, "forbidden", "global admin role required")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	out, err := h.Svc.PlatformStatus(ctx)
	if err != nil {
		slog.Error("lakehouse platform status failed", "error", err)
		writeLakehouseStatusError(w, http.StatusInternalServerError, "internal_error", "internal error")
		return
	}
	writeLakehouseStatusJSON(w, http.StatusOK, out)
}

// tenantStatus is the per-session view. The tenant name comes from the verified session;
// the auth context fills it after AuthContextMiddleware runs. There is no fallback to a
// query parameter.
func (h *LakehouseStatusHandler) tenantStatus(w http.ResponseWriter, r *http.Request) {
	if _, ok := security.RequireAuth(w, r); !ok {
		return
	}
	tenantID, ok := security.RequireTenant(w, r)
	if !ok {
		return
	}
	// The auth context carries the tenant ID (a UUID). The service looks up the
	// registry row by UUID to get the friendly name. The tenant name is NEVER read
	// from a query parameter — that would be an IDOR that exposes the platform view
	// to any tenant admin who knows a name.
	id, err := uuid.Parse(tenantID)
	if err != nil {
		writeLakehouseStatusError(w, http.StatusBadRequest, "invalid_tenant_id", "tenant id must be a UUID")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	out, err := h.Svc.TenantStatus(ctx, id)
	if err != nil {
		var notFound *lakehouse.TenantNotFoundError
		if errors.As(err, &notFound) {
			writeLakehouseStatusError(w, http.StatusNotFound, "tenant_not_found", "no such tenant")
			return
		}
		slog.Error("lakehouse tenant status failed", "tenant_id", tenantID, "error", err)
		writeLakehouseStatusError(w, http.StatusInternalServerError, "internal_error", "internal error")
		return
	}
	writeLakehouseStatusJSON(w, http.StatusOK, out)
}

func writeLakehouseStatusJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeLakehouseStatusError(w http.ResponseWriter, status int, code, msg string) {
	writeLakehouseStatusJSON(w, status, map[string]string{"code": code, "error": msg})
}