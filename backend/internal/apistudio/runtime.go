package apistudio

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/hondyman/uisce/backend/internal/cbo"
	"github.com/hondyman/uisce/backend/internal/logging"
	"github.com/hondyman/uisce/backend/internal/region"
	"github.com/jmoiron/sqlx"
	"github.com/hondyman/uisce/libs/jwt-middleware"
)

// applyFieldMasking strips fields from each result row when the caller
// holds none of the roles the field's entitlement policy requires.
// maskedFields maps fieldName -> roles allowed to see it.
func applyFieldMasking(rows []map[string]interface{}, maskedFields map[string][]string, callerRoles []string) {
	if len(maskedFields) == 0 || len(rows) == 0 {
		return
	}

	roleSet := make(map[string]bool, len(callerRoles))
	for _, r := range callerRoles {
		roleSet[r] = true
	}

	for field, allowedRoles := range maskedFields {
		allowed := false
		for _, role := range allowedRoles {
			if roleSet[role] {
				allowed = true
				break
			}
		}
		if allowed {
			continue
		}
		for _, row := range rows {
			delete(row, field)
		}
	}
}

// APIRuntime handles dynamic API requests by
// mapping endpoints to the semantic resolver.
type APIRuntime struct {
	repo        *Repository
	resolver    *analytics.BOContextResolver
	db          *sqlx.DB // for execution
	planCache   *GraphQLPlanCache
	rateLimiter *RateLimiter
	mountPrefix string
}

// NewAPIRuntime creates a new runtime
func NewAPIRuntime(repo *Repository, resolver *analytics.BOContextResolver, db *sqlx.DB, redisClient *redis.Client) *APIRuntime {
	return &APIRuntime{
		repo:        repo,
		resolver:    resolver,
		db:          db,
		planCache:   NewGraphQLPlanCache(redisClient),
		rateLimiter: NewRateLimiter(redisClient),
	}
}

// SetMountPrefix records the path prefix this runtime is mounted under
// (e.g. "/api/runtime"), which is stripped from the incoming request path
// before matching against APIEndpoint.Path — endpoint paths are stored
// relative to that mount point (see sdk.go's generated client baseUrl).
func (rt *APIRuntime) SetMountPrefix(prefix string) {
	rt.mountPrefix = prefix
}

// ServeHTTP implements the dynamic REST dispatcher
func (rt *APIRuntime) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	env := r.Header.Get("X-Env")
	if env == "" {
		env = "production"
	}
	tenantIDStr := jwtmiddleware.GetClaimsFromContext(r).TenantID

	// Match path + method to endpoint. Endpoint paths are stored relative
	// to the runtime's mount point, so strip it before matching.
	matchPath := strings.TrimPrefix(r.URL.Path, rt.mountPrefix)
	if matchPath == "" {
		matchPath = "/"
	}
	ep, err := rt.repo.FindByPath(r.Context(), r.Method, matchPath, env, tenantIDStr)
	if err != nil {
		http.Error(w, "Endpoint not found", http.StatusNotFound)
		return
	}

	// 1. Rate Check
	allowed, err := rt.rateLimiter.Allow(r.Context(), tenantIDStr, 1)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Rate limiter error: %v", err)
		// Fail open or closed? Closed for safety, but log error.
	}
	if !allowed {
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	// 2. Build BOSQLRequest
	var fields []string
	if err := json.Unmarshal(ep.Fields, &fields); err != nil {
		http.Error(w, "invalid endpoint definition (fields)", http.StatusInternalServerError)
		return
	}

	tenantUUID, _ := uuid.Parse(tenantIDStr)

	// Simple query param to filter mapping
	filters := make(map[string]interface{})
	for k, v := range r.URL.Query() {
		if len(v) > 0 {
			filters[k] = v[0]
		}
	}

	reg := ""
	if rg, ok := region.GetRegionFromContext(r.Context()); ok {
		reg = rg
	}

	claims := jwtmiddleware.GetClaimsFromContext(r)

	req := analytics.BOSQLRequest{
		Env:                  env,
		TenantID:             &tenantUUID,
		BOName:               ep.BOName,
		EndpointID:           &ep.ID,
		Measures:             fields,
		Filters:              filters,
		CurrentUserID:        r.Header.Get("X-User-ID"),
		Region:               reg,
		CallerRoles:          claims.Roles,
		CallerOrganizationID: claims.OrganizationID,
	}

	// 2. Resolve to SQL (with Plan Caching). The plan key includes the
	// caller's roles, so a cache hit implies entitlement evaluation already
	// ran and passed for this exact role set — see graphql_plan_cache.go.
	start := time.Now()

	planKey := GeneratePlanKey(ep.TenantID, ep.ID.String(), ep.Version, fields, filters, claims.Roles)

	plan, cacheErr := rt.planCache.GetPlan(r.Context(), planKey)
	if cacheErr != nil || plan == nil {
		// Cache Miss — plan and evaluate entitlements
		resolvedSQL, meta, err := rt.resolver.ResolveQuery(r.Context(), req)
		if err != nil {
			if errors.Is(err, cbo.ErrEntitlementDenied) {
				http.Error(w, "Forbidden: caller does not satisfy entitlement policy", http.StatusForbidden)
				return
			}
			http.Error(w, fmt.Sprintf("Resolution error: %v", err), http.StatusInternalServerError)
			return
		}
		plan = &CachedPlan{SQL: resolvedSQL, MaskedFields: meta.MaskedFields}
		_ = rt.planCache.SetPlan(r.Context(), planKey, *plan)
	}
	sql := plan.SQL

	// 3. Execute — tenant-scoped: RLS policies key off uisce.current_tenant,
	// which only exists inside a transaction that sets it (see db.WithTenantTransaction).
	// A bare pooled query here would run with no tenant GUC set, defeating RLS entirely.
	var result []map[string]interface{}
	execErr := withTenantScopedQuery(r.Context(), rt.db, tenantIDStr, sql, func(rows *sqlx.Rows) error {
		for rows.Next() {
			row := make(map[string]interface{})
			if err := rows.MapScan(row); err != nil {
				return err
			}
			result = append(result, row)
		}
		return rows.Err()
	})
	duration := time.Since(start)
	statusCode := http.StatusOK
	if execErr != nil {
		statusCode = http.StatusInternalServerError
	}

	// Log Telemetry
	clientType := r.Header.Get("X-Client-Type")
	if clientType == "" {
		clientType = "external"
	}
	var errMsg *string
	if execErr != nil {
		m := execErr.Error()
		errMsg = &m
	}

	_ = rt.repo.LogTelemetry(r.Context(), &APITelemetry{
		APIID:        ep.ID,
		Env:          env,
		TenantID:     &tenantUUID,
		ClientType:   clientType,
		StatusCode:   statusCode,
		LatencyMs:    int(duration.Milliseconds()),
		ErrorMessage: errMsg,
		RequestedAt:  time.Now(),
	})

	if execErr != nil {
		http.Error(w, fmt.Sprintf("Execution error: %v", execErr), http.StatusInternalServerError)
		return
	}

	applyFieldMasking(result, plan.MaskedFields, claims.Roles)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// withTenantScopedQuery runs sql inside a transaction with the RLS tenant GUC
// (uisce.current_tenant) set, mirroring db.WithTenantTransaction. The dynamic
// API Studio runtime executes ad-hoc SQL built from endpoint definitions and
// must never run it on a bare pooled connection, or Postgres RLS policies
// have no tenant to scope against.
func withTenantScopedQuery(ctx context.Context, db *sqlx.DB, tenantID string, sql string, scan func(*sqlx.Rows) error) error {
	if tenantID == "" {
		return fmt.Errorf("withTenantScopedQuery: tenantID cannot be empty")
	}

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("withTenantScopedQuery: BeginTxx failed: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if _, err := tx.ExecContext(ctx, "SELECT set_config('uisce.current_tenant', $1, true)", tenantID); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("withTenantScopedQuery: set_config failed: %w", err)
	}

	rows, err := tx.QueryxContext(ctx, sql)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	scanErr := scan(rows)
	_ = rows.Close()
	if scanErr != nil {
		_ = tx.Rollback()
		return scanErr
	}

	return tx.Commit()
}
