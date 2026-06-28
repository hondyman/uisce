package security

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Context represents the security context.
type Context struct {
	UserID         string
	Roles          []string
	TenantID       string
	InstanceID     string
	ProductID      string
	DatasourceID   string
	OperatingScope string
	Region         string
	Attributes     map[string]any

	// Impersonation fields — only set when a global admin has assumed a tenant context.
	// ImpersonationActive is true for the entire duration of the impersonation window.
	// All write operations MUST check this flag and require break_glass mode.
	IsGlobalAdmin          bool
	ImpersonationActive    bool
	RealAdminUserID        string // The admin's true identity (immutable)
	ImpersonationSessionID string // Links to platform_admin_audit.session_id
	ImpersonationMode      string // "read_only" | "break_glass"
}

type AuthInfo struct {
	UserID    string
	Roles     []string
	TenantIDs []string

	// IsGlobalAdmin is true when the user holds the global_admin or global_ops role.
	IsGlobalAdmin bool

	// Impersonation fields — populated by AuthContextMiddleware when it detects
	// an impersonation context token in the Authorization header.
	ImpersonationActive    bool
	RealAdminUserID        string
	ImpersonationSessionID string
	ImpersonationMode      string
}

type BuildContextRequest struct {
	DatasourceID string
	Region       string
}

type ResolvedDatasource struct {
	TenantID       string
	InstanceID     string
	ProductID      string
	DatasourceID   string
	AllowedRegions []string
}

type DatasourceResolver interface {
	Resolve(ctx context.Context, datasourceID string) (*ResolvedDatasource, error)
}

var scopePartPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

func ValidateRegion(region string) error {
	value := strings.TrimSpace(region)
	if value == "" {
		return fmt.Errorf("region is required")
	}
	if !scopePartPattern.MatchString(value) {
		return fmt.Errorf("region contains invalid characters")
	}
	return nil
}

func BuildContext(ctx context.Context, auth AuthInfo, req BuildContextRequest, resolver DatasourceResolver) (*Context, error) {
	if resolver == nil {
		return nil, fmt.Errorf("datasource resolver not configured")
	}
	if strings.TrimSpace(req.DatasourceID) == "" {
		return nil, fmt.Errorf("datasource_id is required")
	}
	if err := ValidateRegion(req.Region); err != nil {
		return nil, err
	}
	isGlobalAdmin := containsRole(auth.Roles, "global_admin") || containsRole(auth.Roles, "global_ops")
	if len(auth.TenantIDs) == 0 && !isGlobalAdmin {
		return nil, fmt.Errorf("no allowed tenants configured for user")
	}

	resolved, err := resolver.Resolve(ctx, req.DatasourceID)
	if err != nil {
		return nil, err
	}
	if !isGlobalAdmin && !tenantAllowed(auth.TenantIDs, resolved.TenantID) {
		return nil, fmt.Errorf("datasource not found")
	}
	if len(resolved.AllowedRegions) > 0 && !containsRegion(resolved.AllowedRegions, req.Region) {
		return nil, fmt.Errorf("region '%s' is not configured for datasource", req.Region)
	}

	// Phase 3 enforcement: when impersonation is active and a scope was chosen,
	// verify the resolved datasource is within the chosen scope. This is the
	// defence-in-depth backstop for the audit-recorded scope narrowing.
	if auth.ImpersonationActive && auth.ImpersonationSessionID != "" {
		// Pull the scope from the request context (set by AuthContextMiddleware
		// after validating the impersonation token). Default to tenant-wide.
		scope := ImpersonationScopeFromContext(ctx)
		if err := ValidateScope(scope.Kind, scope.ID, *resolved); err != nil {
			return nil, err
		}
	}

	operatingScope := fmt.Sprintf("%s:%s:%s:%s", resolved.TenantID, resolved.InstanceID, resolved.ProductID, resolved.DatasourceID)
	secCtx := &Context{
		UserID:         auth.UserID,
		Roles:          auth.Roles,
		TenantID:       resolved.TenantID,
		InstanceID:     resolved.InstanceID,
		ProductID:      resolved.ProductID,
		DatasourceID:   resolved.DatasourceID,
		Region:         strings.TrimSpace(req.Region),
		OperatingScope: operatingScope,
		Attributes:     map[string]any{},

		// Propagate impersonation metadata from AuthInfo into the security context.
		IsGlobalAdmin:          auth.IsGlobalAdmin,
		ImpersonationActive:    auth.ImpersonationActive,
		RealAdminUserID:        auth.RealAdminUserID,
		ImpersonationSessionID: auth.ImpersonationSessionID,
		ImpersonationMode:      auth.ImpersonationMode,
	}
	secCtx.Attributes["operating_scope"] = secCtx.OperatingScope
	secCtx.Attributes["region"] = secCtx.Region
	secCtx.Attributes["tenant_id"] = secCtx.TenantID
	secCtx.Attributes["instance_id"] = secCtx.InstanceID
	secCtx.Attributes["product_id"] = secCtx.ProductID
	secCtx.Attributes["datasource_id"] = secCtx.DatasourceID
	secCtx.Attributes["impersonation_active"] = secCtx.ImpersonationActive
	secCtx.Attributes["impersonation_mode"] = secCtx.ImpersonationMode

	return secCtx, nil
}

func parseAllowedRegions(raw sql.NullString) []string {
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return nil
	}
	s := strings.TrimSpace(raw.String)
	if strings.HasPrefix(s, "[") {
		var arr []string
		if err := json.Unmarshal([]byte(s), &arr); err == nil {
			return normalizeRegions(arr)
		}
	}
	parts := strings.Split(s, ",")
	return normalizeRegions(parts)
}

func normalizeRegions(regions []string) []string {
	result := []string{}
	seen := map[string]struct{}{}
	for _, region := range regions {
		value := strings.TrimSpace(strings.Trim(region, `"`))
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	return result
}

func containsRegion(regions []string, region string) bool {
	value := strings.TrimSpace(region)
	for _, candidate := range regions {
		if strings.EqualFold(strings.TrimSpace(candidate), value) {
			return true
		}
	}
	return false
}

func tenantAllowed(allowed []string, tenantID string) bool {
	for _, candidate := range allowed {
		if strings.TrimSpace(candidate) == tenantID {
			return true
		}
	}
	return false
}

func containsRole(roles []string, role string) bool {
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}

type contextKey struct{}

func WithContext(ctx context.Context, secCtx *Context) context.Context {
	return context.WithValue(ctx, contextKey{}, secCtx)
}

func FromContext(ctx context.Context) (*Context, bool) {
	value := ctx.Value(contextKey{})
	secCtx, ok := value.(*Context)
	return secCtx, ok
}

// RLSPolicy is a function that returns predicates for RLS.
type RLSPolicy func(modelName string, ctx Context) []Predicate

// Predicate represents a security predicate.
type Predicate struct {
	Field  string
	Params []any
}

// OrdersPolicy is an example RLS policy.
func OrdersPolicy(modelName string, ctx Context) []Predicate {
	return []Predicate{
		{Field: "user_id", Params: []any{ctx.UserID}},
	}
}
