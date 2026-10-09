package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/hondyman/uisce/backend/internal/logging"
	"github.com/hondyman/uisce/backend/internal/msgcat"
	"github.com/hondyman/uisce/backend/internal/security"
)

type SecurityContextDeps struct {
	Resolver security.DatasourceResolver
}

// Reasons a security context is refused. These strings are the stable contract a
// client may read; callers must not match on any other error text.
const (
	ReasonAuthRequired           = "authentication required: missing or invalid JWT token"
	ReasonRegionRequired         = "region is required: send X-Region or X-Tenant-Region"
	ReasonTenantNotPermitted     = "forbidden: requested tenant does not match caller's tenant"
	ReasonNoTenantGranted        = "no tenants assigned to user: JWT token must include tenant_id or tenant_ids claim"
	ReasonDatasourceNotAvailable = "datasource not found" // unchanged text: clients already see it
	ReasonInvalidRegion          = "invalid region"
	ReasonRegionNotConfigured    = "region is not configured for this datasource"
	ReasonScopeNotEstablished    = "request scope could not be established"
	ReasonResolverMissing        = "internal error: datasource resolver not configured"
)

// SecurityError is a refused security context. Status says what kind of refusal
// it is: 401 when the caller has no valid identity; 403 when the identity is valid
// but the tenant, datasource or region is not one the caller may use; 400 when the
// request itself is incomplete; 500 when the scope could not be established for a
// reason that is not the caller's to fix.
type SecurityError struct {
	Status int
	Reason string
	cause  error
}

func (e *SecurityError) Error() string { return e.Reason }
func (e *SecurityError) Unwrap() error { return e.cause }

func securityRefusal(status int, reason string, cause error) error {
	return &SecurityError{Status: status, Reason: reason, cause: cause}
}

// SecurityErrorStatus returns the HTTP status for an error from
// SecurityContextFromRequest. Anything that is not a SecurityError is an internal failure.
func SecurityErrorStatus(err error) int {
	var se *SecurityError
	if errors.As(err, &se) {
		return se.Status
	}
	return http.StatusInternalServerError
}

// WriteSecurityError answers a refused security context with the status it earns and
// its stable reason. Handlers use it wherever SecurityContextFromRequest fails, so a
// caller who names a tenant they may not use gets 403, not a 401 that reads as a bad token.
func WriteSecurityError(w http.ResponseWriter, err error) {
	var se *SecurityError
	if errors.As(err, &se) {
		http.Error(w, se.Reason, se.Status)
		return
	}
	http.Error(w, "internal error", http.StatusInternalServerError)
}

// SecurityContextFromRequest establishes who is calling and for which tenant. The
// tenant is taken from verified authentication only: a tenant named in a header or
// query parameter is honored only if the caller belongs to it (or is a global admin),
// and is refused otherwise.
func SecurityContextFromRequest(r *http.Request, bodyDatasourceID string, bodyRegion string, deps SecurityContextDeps) (*security.Context, context.Context, error) {
	// Identity first: a request with no verified caller is 401, whatever else it lacks.
	// Extract auth info from context (set by AuthContextMiddleware)
	auth, ok := security.AuthInfoFromContext(r.Context())
	if !ok {
		err := securityRefusal(http.StatusUnauthorized, ReasonAuthRequired, nil)
		logging.GetLogger().Sugar().Warnf("[SecurityContextFromRequest] %v", err)
		return nil, r.Context(), err
	}

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
		// No default region: the caller must state it.
		err := securityRefusal(http.StatusBadRequest, ReasonRegionRequired, nil)
		logging.GetLogger().Sugar().Warnf("[SecurityContextFromRequest] %v", err)
		return nil, r.Context(), err
	}
	if deps.Resolver == nil {
		err := securityRefusal(http.StatusInternalServerError, ReasonResolverMissing, nil)
		logging.GetLogger().Sugar().Errorf("[SecurityContextFromRequest] %v", err)
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
		auth.ActiveTenantID = resolvedTenantID
		if len(auth.TenantIDs) == 0 || auth.TenantIDs[0] != resolvedTenantID {
			auth.TenantIDs = append([]string{resolvedTenantID}, auth.TenantIDs...)
		}
	case targetTenantID != "":
		// Authenticated, but the tenant they named is not one they may act for.
		// Reject outright rather than falling back to their own tenant, which
		// would hide the mismatch.
		err := securityRefusal(http.StatusForbidden, ReasonTenantNotPermitted, nil)
		logging.GetLogger().Sugar().Warnf("[SecurityContextFromRequest] user=%s requested=%s ownTenantIDs=%v isGlobalAdmin=%v: %v", auth.UserID, targetTenantID, auth.TenantIDs, isGlobalAdmin, err)
		return nil, r.Context(), err
	case isGlobalAdmin:
		// No tenant requested, caller has no tenant claims of their own, but
		// is a verified global admin — preserves the original behavior of
		// allowing a global-scope request through with empty TenantIDs.
	default:
		err := securityRefusal(http.StatusForbidden, ReasonNoTenantGranted, nil)
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
		return nil, r.Context(), classifyScopeError(err)
	}

	// Inject security context into request context for downstream use
	ctx := security.WithContext(r.Context(), secCtx)
	return secCtx, ctx, nil
}

// classifyScopeError maps a BuildContext refusal to its status. A scope the caller
// may not use is 403; a region the request got wrong is 400; anything else is an
// internal failure, not an authentication one.
func classifyScopeError(err error) error {
	switch {
	case errors.Is(err, security.ErrDatasourceNotAvailable):
		return securityRefusal(http.StatusForbidden, ReasonDatasourceNotAvailable, err)
	case errors.Is(err, security.ErrNoActiveTenant), errors.Is(err, security.ErrNoTenantGranted):
		return securityRefusal(http.StatusForbidden, ReasonNoTenantGranted, err)
	case errors.Is(err, security.ErrRegionNotConfigured):
		return securityRefusal(http.StatusForbidden, ReasonRegionNotConfigured, err)
	case errors.Is(err, security.ErrInvalidRegion):
		return securityRefusal(http.StatusBadRequest, ReasonInvalidRegion, err)
	default:
		return securityRefusal(http.StatusInternalServerError, ReasonScopeNotEstablished, err)
	}
}

// SecurityContextError is the catalog message for a SecurityContextFromRequest
// failure. A tenant or datasource the caller may not use is a scope problem (403),
// not missing credentials; everything else stays "authentication is required" (401).
// The tenant itself is never taken from this mapping: it is still resolved from the token alone.
func SecurityContextError(err error) *msgcat.Error {
	if errors.Is(err, security.ErrDatasourceNotAvailable) {
		return msgcat.DatasourceNotAvailable().Wrap(err)
	}
	var se *SecurityError
	if errors.As(err, &se) && se.Status == http.StatusForbidden {
		return msgcat.NotPermitted("act for the requested tenant").Wrap(err)
	}
	return msgcat.Unauthenticated().Wrap(err)
}
