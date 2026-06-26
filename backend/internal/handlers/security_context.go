package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hondyman/semlayer/backend/internal/security"
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

	// Validate required parameters with helpful error messages
	if datasourceID == "" {
		return nil, r.Context(), fmt.Errorf("datasource_id is required: provide via X-Datasource-Id, X-Tenant-Datasource-ID, or X-Tenant-Instance-ID header")
	}
	if region == "" {
		return nil, r.Context(), fmt.Errorf("region is required: provide via X-Region or X-Tenant-Region header")
	}
	if deps.Resolver == nil {
		return nil, r.Context(), fmt.Errorf("datasource resolver not configured (internal error)")
	}

	// Extract auth info from context (set by AuthContextMiddleware)
	auth, ok := security.AuthInfoFromContext(r.Context())
	if !ok {
		return nil, r.Context(), fmt.Errorf("authentication required: missing or invalid JWT token")
	}

	isGlobalAdmin := false
	for _, role := range auth.Roles {
		if role == "global_admin" || role == "global_ops" {
			isGlobalAdmin = true
			break
		}
	}

	if len(auth.TenantIDs) == 0 && !isGlobalAdmin {
		return nil, r.Context(), fmt.Errorf("no tenants assigned to user: JWT token must include tenant_id or tenant_ids claim")
	}

	// Build and validate security context
	secCtx, err := security.BuildContext(r.Context(), auth, security.BuildContextRequest{
		DatasourceID: datasourceID,
		Region:       region,
	}, deps.Resolver)
	if err != nil {
		return nil, r.Context(), err
	}

	// Inject security context into request context for downstream use
	ctx := security.WithContext(r.Context(), secCtx)
	return secCtx, ctx, nil
}
