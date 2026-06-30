package region

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// Header name for region
const RegionHeader = "X-Tenant-Region"

// context key type to avoid collisions
type ctxKey string

const RegionCtxKey ctxKey = "region"

// AllowedRegionsProvider abstracts how we obtain allowed regions for a tenant.
type AllowedRegionsProvider interface {
	GetAllowedRegions(tenantID string) ([]string, error)
}

// DBAllowedRegionsProvider reads tenant allowed regions from the control DB (tenants.metadata)
type DBAllowedRegionsProvider struct {
	DB *sql.DB
}

func NewDBAllowedRegionsProvider(db *sql.DB) *DBAllowedRegionsProvider {
	return &DBAllowedRegionsProvider{DB: db}
}

func (p *DBAllowedRegionsProvider) GetAllowedRegions(tenantID string) ([]string, error) {
	if tenantID == "" {
		return nil, nil
	}
	// Prefer explicit column allowed_regions (jsonb), fallback to metadata fields
	var raw sql.NullString
	query := `SELECT COALESCE(allowed_regions::text, metadata->>'allowed_regions', metadata->>'regions') FROM public.tenants WHERE id = $1`
	err := p.DB.QueryRow(query, tenantID).Scan(&raw)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return nil, nil
	}
	// raw may be a JSON array like ["eu-west","us-east"] or a comma-separated string
	s := strings.TrimSpace(raw.String)
	if strings.HasPrefix(s, "[") {
		// JSON array
		var arr []string
		if err := json.Unmarshal([]byte(s), &arr); err == nil {
			return arr, nil
		}
		// fallthrough to try comma-split if not valid JSON
	}
	// comma separated
	parts := strings.Split(s, ",")
	out := []string{}
	for _, part := range parts {
		if t := strings.TrimSpace(part); t != "" {
			// Trim quotes if the DB returned a JSON string representation like "eu-west"
			t = strings.Trim(t, `"`)
			out = append(out, t)
		}
	}
	return out, nil
}

// RegionValidationMiddleware enforces region validation and Gold Copy bypass logic
// It accepts either a TenantRegionResolver or legacy AllowedRegionsProvider
func RegionValidationMiddleware(provider interface{}) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Allow CORS preflight and other non-mutating requests to proceed
			if r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			// Exempt health, auth, admin, and RBAC routes from region validation (JWT/API key auth sufficient)
			if r.URL.Path == "/health" ||
				strings.HasPrefix(r.URL.Path, "/api/admin/") ||
				strings.HasPrefix(r.URL.Path, "/api/auth/") ||
				strings.HasPrefix(r.URL.Path, "/api/rbac/") {
				next.ServeHTTP(w, r)
				return
			}

			// Exempt catalog metadata endpoints (glossary, node-types, edge-types) - these are tenant-level metadata
			// Exempt tenant management endpoints - these are global operations for platform operators
			// Semantic operations on DATA require region (via /api/semantic/*), but metadata doesn't
			path := r.URL.Path
			// Debug: log path matching for region middleware (temporary)
			fmt.Fprintf(os.Stderr, "[REGION-MW] path=%s, authPrefix=%v\n", path, strings.HasPrefix(path, "/api/auth/"))

			if strings.HasPrefix(path, "/api/glossary/") ||
				strings.HasPrefix(path, "/api/node-types") ||
				strings.HasPrefix(path, "/api/edge-types") ||
				strings.HasPrefix(path, "/api/catalog/") ||
				strings.HasPrefix(path, "/api/bp-notifications/") ||
				strings.HasPrefix(path, "/api/tenants") ||
				strings.HasPrefix(path, "/api/datasources") ||
				strings.HasPrefix(path, "/api/products") ||
				strings.HasPrefix(path, "/api/business-objects") ||
				strings.HasPrefix(path, "/api/roles") ||
				strings.HasPrefix(path, "/api/views") ||
				strings.HasPrefix(path, "/api/users") ||
				strings.HasPrefix(path, "/api/audit") ||
				strings.HasPrefix(path, "/api/ws/token") ||
				strings.HasPrefix(path, "/api/auth/") ||
				strings.HasPrefix(path, "/api/semantic/bundles/") {
				next.ServeHTTP(w, r)
				return
			}

			// Get tenant and region from headers
			var tenantID string
			if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil {
				tenantID = strings.TrimSpace(claims.TenantID)
			}
			if tenantID == "" {
				tenantID = strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
			}
			region := strings.TrimSpace(r.Header.Get("X-Tenant-Region"))

			// Gold Copy bypass — always allow, no region required
			if tenantID == GoldCopyTenantID {
				ctx := context.WithValue(r.Context(), RegionCtxKey, "global")
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// For regular tenants: region is required
			if region == "" {
				// Try to infer region from tenant (optional, if using TenantRegionResolver)
				if resolver, ok := provider.(*TenantRegionResolver); ok {
					if inferred, ok := resolver.InferRegionForTenant(tenantID); ok {
						region = inferred
					}
				}

				if region == "" {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					_ = json.NewEncoder(w).Encode(map[string]string{"error": "region is required for all semantic operations."})
					return
				}
			}

			// Validate region is allowed for tenant
			var isAllowed bool
			if resolver, ok := provider.(*TenantRegionResolver); ok {
				isAllowed = resolver.IsRegionAllowedForTenant(tenantID, region)
			} else if legacyProvider, ok := provider.(AllowedRegionsProvider); ok {
				// Fallback to legacy AllowedRegionsProvider
				allowed, err := legacyProvider.GetAllowedRegions(tenantID)
				if err == nil && len(allowed) > 0 {
					isAllowed = false
					for _, a := range allowed {
						if strings.EqualFold(strings.TrimSpace(a), region) {
							isAllowed = true
							break
						}
					}
				} else if err == nil && len(allowed) == 0 {
					// No allowed regions means single-region tenant, infer match
					isAllowed = true
				}
			}

			if !isAllowed {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("region '%s' is not allowed for tenant '%s'", region, tenantID)})
				return
			}

			// Inject region into context for downstream handlers
			ctx := context.WithValue(r.Context(), RegionCtxKey, region)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetRegionFromContext retrieves the region string injected by RegionMiddleware.
// Returns (region, true) if present.
func GetRegionFromContext(ctx context.Context) (string, bool) {
	v := ctx.Value(RegionCtxKey)
	s, ok := v.(string)
	return s, ok
}
