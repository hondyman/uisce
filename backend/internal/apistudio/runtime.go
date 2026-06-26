package apistudio

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/analytics"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/hondyman/semlayer/backend/internal/region"
	"github.com/jmoiron/sqlx"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// APIRuntime handles dynamic API requests by
// mapping endpoints to the semantic resolver.
type APIRuntime struct {
	repo        *Repository
	resolver    *analytics.BOContextResolver
	db          *sqlx.DB // for execution
	planCache   *GraphQLPlanCache
	rateLimiter *RateLimiter
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

// ServeHTTP implements the dynamic REST dispatcher
func (rt *APIRuntime) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	env := r.Header.Get("X-Env")
	if env == "" {
		env = "production"
	}
	tenantIDStr := jwtmiddleware.GetClaimsFromContext(r).TenantID

	// Match path + method to endpoint
	ep, err := rt.repo.FindByPath(r.Context(), r.Method, r.URL.Path, env, tenantIDStr)
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

	req := analytics.BOSQLRequest{
		Env:           env,
		TenantID:      &tenantUUID,
		BOName:        ep.BOName,
		EndpointID:    &ep.ID,
		Measures:      fields,
		Filters:       filters,
		CurrentUserID: r.Header.Get("X-User-ID"),
		Region:        reg,
	}

	// 2. Resolve to SQL (with Plan Caching)
	start := time.Now()

	// Generate Cache Key
	var filterKeys []string
	for k := range filters {
		filterKeys = append(filterKeys, k)
	}
	planKey := GeneratePlanKey(ep.TenantID, ep.ID.String(), ep.Version, fields, filterKeys)

	var sql string
	cachedSQL, err := rt.planCache.GetPlan(r.Context(), planKey)
	if err == nil && cachedSQL != "" {
		sql = cachedSQL
	} else {
		// Cache Miss
		resolvedSQL, _, err := rt.resolver.ResolveQuery(r.Context(), req)
		if err != nil {
			http.Error(w, fmt.Sprintf("Resolution error: %v", err), http.StatusInternalServerError)
			return
		}
		// Cache Plan
		_ = rt.planCache.SetPlan(r.Context(), planKey, resolvedSQL)
		sql = resolvedSQL
	}

	// 3. Execute
	rows, err := rt.db.QueryxContext(r.Context(), sql)
	duration := time.Since(start)
	statusCode := http.StatusOK
	if err != nil {
		statusCode = http.StatusInternalServerError
	}

	// Log Telemetry
	clientType := r.Header.Get("X-Client-Type")
	if clientType == "" {
		clientType = "external"
	}
	var errMsg *string
	if err != nil {
		m := err.Error()
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

	if err != nil {
		http.Error(w, fmt.Sprintf("Execution error: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var result []map[string]interface{}
	for rows.Next() {
		row := make(map[string]interface{})
		if err := rows.MapScan(row); err == nil {
			result = append(result, row)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
