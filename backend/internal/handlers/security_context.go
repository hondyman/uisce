package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hondyman/uisce/backend/internal/logging"
	"github.com/hondyman/uisce/backend/internal/security"
)

type SecurityContextDeps struct {
	Resolver security.DatasourceResolver
}

func SecurityContextFromRequest(r *http.Request, bodyDatasourceID string, bodyRegion string, deps SecurityContextDeps) (*security.Context, context.Context, error) {
	// Try multiple header names for datasource ID (support legacy and new naming)
	datasourceID := strings.TrimSpace(bodyDatasourceID)
	if datasourceID == "" {
		datasourceID = strings.TrimSpace(r.Header.Get("X-Datasource-Id"))
	}
	if datasourceID == "" {
		datasourceID = strings.TrimSpace(r.Header.Get("X-Tenant-Datasource-ID"))
	}
	if datasourceID == "" {
		datasourceID = strings.TrimSpace(r.Header.Get("X-Tenant-Instance-ID"))
	}

	// Try multiple header names for region
	region := strings.TrimSpace(bodyRegion)
	if region == "" {
		region = strings.TrimSpace(r.Header.Get("X-Region"))
	}
	if region == "" {
		region = strings.TrimSpace(r.Header.Get("X-Tenant-Region"))
	}
	if region == "" {
		region = "us-east-1"
	}
	if deps.Resolver == nil {
		err := fmt.Errorf("datasource resolver not configured (internal error)")
		logging.GetLogger().Sugar().Errorf("[SecurityContextFromRequest] %v", err)
		return nil, r.Context(), err
	}

	// Extract auth info from context (set by AuthContextMiddleware)
	auth, ok := security.AuthInfoFromContext(r.Context())
	if !ok {
		err := fmt.Errorf("authentication required: missing or invalid JWT token")
		logging.GetLogger().Sugar().Warnf("[SecurityContextFromRequest] %v", err)
		return nil, r.Context(), err
	}

	isGlobalAdmin := false
	for _, role := range auth.Roles {
		if role == "global_admin" || role == "global_ops" {
			isGlobalAdmin = true
			break
		}
	}
	auth.IsGlobalAdmin = auth.IsGlobalAdmin || isGlobalAdmin

	// HOTFIX 2026-09-07: this previously accepted a client-supplied
	// X-Tenant-ID header or tenant_id query param unconditionally, prepending
	// it as the primary scoped tenant with no check that it belonged to the
	// caller. Because this function is called from 67 sites across the
	// backend, that meant any authenticated user for any tenant could pivot
	// to any other tenant by setting the header — the single highest-blast-
	// radius finding in backend/docs/INCIDENT_REPORT_20260906.md's
	// tenant-resolution sweep. security.ResolveTenantID is the canonical
	// rule: the requested tenant is honored only if it matches the caller's
	// own tenant list or the caller is a verified global admin/ops.
	targetTenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	if targetTenantID == "" {
		targetTenantID = strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	}
	resolvedTenantID, resolveOK := security.ResolveTenantID(auth, targetTenantID)
	switch {
	case resolveOK:
		if len(auth.TenantIDs) == 0 || auth.TenantIDs[0] != resolvedTenantID {
			auth.TenantIDs = append([]string{resolvedTenantID}, auth.TenantIDs...)
		}
	case targetTenantID != "":
		// A tenant was explicitly requested (header or query param) and did
		// not match the caller's own tenants or admin status — reject
		// outright rather than silently falling back to the caller's own
		// tenant, which would mask the mismatch instead of surfacing it.
		err := fmt.Errorf("forbidden: requested tenant does not match caller's tenant")
		logging.GetLogger().Sugar().Warnf("[SecurityContextFromRequest] user=%s requested=%s ownTenantIDs=%v isGlobalAdmin=%v: %v", auth.UserID, targetTenantID, auth.TenantIDs, isGlobalAdmin, err)
		return nil, r.Context(), err
	case isGlobalAdmin:
		// No tenant requested, caller has no tenant claims of their own, but
		// is a verified global admin — preserves the original behavior of
		// allowing a global-scope request through with empty TenantIDs.
	default:
		err := fmt.Errorf("no tenants assigned to user: JWT token must include tenant_id or tenant_ids claim")
		logging.GetLogger().Sugar().Warnf("[SecurityContextFromRequest] user=%s roles=%v tenantIDs=%v isGlobalAdmin=%v: %v", auth.UserID, auth.Roles, auth.TenantIDs, isGlobalAdmin, err)
		return nil, r.Context(), err
	}

	// Build and validate security context
	secCtx, err := security.BuildContext(r.Context(), auth, security.BuildContextRequest{
		DatasourceID: datasourceID,
		Region:       region,
	}, deps.Resolver)
	if err != nil {
		logging.GetLogger().Sugar().Warnf("[SecurityContextFromRequest] BuildContext failed for user=%s tenantIDs=%v datasource=%s region=%s: %v", auth.UserID, auth.TenantIDs, datasourceID, region, err)
		return nil, r.Context(), err
	}

	// Inject security context into request context for downstream use
	ctx := security.WithContext(r.Context(), secCtx)
	return secCtx, ctx, nil
}
