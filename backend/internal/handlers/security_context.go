package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hondyman/uisce/backend/internal/logging"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/hondyman/uisce/backend/internal/services"
)

type SecurityContextDeps struct {
	Resolver security.DatasourceResolver
	// GroupRoleResolver, when set, merges tenant-scoped role_keys derived from
	// the caller's IdP group claims (security.idp_group_role_mappings) into
	// secCtx.Roles. Optional: nil means group-based entitlements are skipped
	// and only literal role claims on the token apply (prior behavior).
	GroupRoleResolver *security.GroupRoleResolver
	// UserProvisioner, when set, just-in-time creates the app_user row for a
	// first-time authenticated identity. Optional: nil means an unrecognized
	// user simply has no roles/permissions rows to match (fails closed, not
	// an error).
	UserProvisioner *security.UserProvisioner
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
	if datasourceID == "" {
		// Mirrors the ?tenant_id= query fallback below — callers that pass
		// tenant/datasource as query params (not headers) need both or
		// neither; without this, secCtx.DatasourceID silently falls back
		// to BuildContext's "none" sentinel even when the caller clearly
		// specified one, and every downstream uuid.Parse(secCtx.DatasourceID)
		// then fails on that literal string.
		datasourceID = strings.TrimSpace(r.URL.Query().Get("datasource_id"))
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
		// The client-supplied tenant must be validated against the JWT-issued
		// tenant list BEFORE it is merged in. Merging first and validating
		// after (the prior behavior) checks the value against a list that
		// already contains it — a tautology that let any authenticated user
		// impersonate an arbitrary tenant via X-Tenant-ID/?tenant_id=.
		if !isGlobalAdmin && !tenantAllowedForRequest(auth.TenantIDs, targetTenantID) {
			err := fmt.Errorf("tenant %s is not assigned to this user", targetTenantID)
			logging.GetLogger().Sugar().Warnf("[SecurityContextFromRequest] user=%s attempted to access unassigned tenant=%s (assigned=%v)", auth.UserID, targetTenantID, auth.TenantIDs)
			return nil, r.Context(), err
		}
		if len(auth.TenantIDs) == 0 {
			auth.TenantIDs = []string{targetTenantID}
		} else {
			// Prepend targetTenantID so BuildContext treats it as the primary scoped tenant
			auth.TenantIDs = append([]string{targetTenantID}, auth.TenantIDs...)
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

	// Just-in-time provision the app_user row for a first-time identity.
	// Grants nothing by itself — role/entitlement resolution below is
	// unaffected either way — it only ensures a row exists for admin UIs
	// (user lists, role assignment) to attach to without a manual step.
	if deps.UserProvisioner != nil {
		email, name := "", ""
		if claims, ok := auth.RawClaims.(*services.JWTClaims); ok && claims != nil {
			email = claims.Email
		}
		if err := deps.UserProvisioner.EnsureUser(r.Context(), auth.UserID, email, name, secCtx.TenantID); err != nil {
			logging.GetLogger().Sugar().Warnf("[SecurityContextFromRequest] user provisioning failed for user=%s: %v", auth.UserID, err)
		}
	}

	// Merge tenant-scoped, group-derived roles on top of any literal role
	// claims. Must happen after BuildContext resolves the active TenantID,
	// since the same group can grant different role_keys in different
	// tenants (e.g. read-only in one, full CRUD in another).
	if deps.GroupRoleResolver != nil && !secCtx.IsGlobalAdmin {
		if claims, ok := auth.RawClaims.(*services.JWTClaims); ok && claims != nil && len(claims.IdpGroups) > 0 {
			groupRoles, err := deps.GroupRoleResolver.ResolveRoles(r.Context(), auth.UserID, secCtx.TenantID, claims.IdpGroups)
			if err != nil {
				logging.GetLogger().Sugar().Warnf("[SecurityContextFromRequest] group role resolution failed for user=%s tenant=%s: %v", auth.UserID, secCtx.TenantID, err)
			} else if len(groupRoles) > 0 {
				secCtx.Roles = mergeRoles(secCtx.Roles, groupRoles)
			}
		}
	}

	// Inject security context into request context for downstream use
	ctx := security.WithContext(r.Context(), secCtx)
	return secCtx, ctx, nil
}

// tenantAllowedForRequest reports whether targetTenantID is present in the
// JWT-issued tenant list, i.e. whether the caller is actually assigned to
// the tenant they're asking to operate as.
func tenantAllowedForRequest(assignedTenantIDs []string, targetTenantID string) bool {
	for _, t := range assignedTenantIDs {
		if strings.TrimSpace(t) == targetTenantID {
			return true
		}
	}
	return false
}

func mergeRoles(existing, additional []string) []string {
	seen := make(map[string]struct{}, len(existing))
	merged := make([]string, 0, len(existing)+len(additional))
	for _, r := range existing {
		if _, ok := seen[r]; !ok {
			seen[r] = struct{}{}
			merged = append(merged, r)
		}
	}
	for _, r := range additional {
		if _, ok := seen[r]; !ok {
			seen[r] = struct{}{}
			merged = append(merged, r)
		}
	}
	return merged
}
