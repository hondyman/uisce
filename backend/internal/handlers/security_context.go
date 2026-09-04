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

	targetTenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	if targetTenantID == "" {
		targetTenantID = strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	}
	if targetTenantID != "" {
		// SECURITY: the client-supplied tenant value must be validated against
		// the JWT-issued tenant list (or the caller must be a global admin)
		// BEFORE it is trusted. Never merge an unvalidated client value into
		// auth.TenantIDs first — that would make the "is this tenant allowed"
		// check tautological.
		allowed := isGlobalAdmin
		if !allowed {
			for _, tid := range auth.TenantIDs {
				if tid == targetTenantID {
					allowed = true
					break
				}
			}
		}
		if !allowed {
			err := fmt.Errorf("tenant %s is not authorized for this user", targetTenantID)
			logging.GetLogger().Sugar().Warnf("[SecurityContextFromRequest] user=%s requested tenant=%s not in JWT tenantIDs=%v and not global admin: %v", auth.UserID, targetTenantID, auth.TenantIDs, err)
			return nil, r.Context(), err
		}
		// Value is now validated: put it first so BuildContext treats it as the
		// primary scoped tenant (needed for global admins acting on a tenant
		// outside their own JWT tenant list).
		if len(auth.TenantIDs) == 0 {
			auth.TenantIDs = []string{targetTenantID}
		} else {
			filtered := make([]string, 0, len(auth.TenantIDs))
			for _, tid := range auth.TenantIDs {
				if tid != targetTenantID {
					filtered = append(filtered, tid)
				}
			}
			auth.TenantIDs = append([]string{targetTenantID}, filtered...)
		}
	}

	if len(auth.TenantIDs) == 0 && !isGlobalAdmin {
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
