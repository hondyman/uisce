package pagestudio

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/analytics"
	"github.com/hondyman/semlayer/backend/internal/apistudio"
	"github.com/jmoiron/sqlx"
)

// PageDataBundleDefinition defines the data requirements for a page
type PageDataBundleDefinition struct {
	ID              uuid.UUID      `json:"id" db:"id"`
	PageID          uuid.UUID      `json:"page_id" db:"page_id"`
	Version         int            `json:"version" db:"version"`
	Sources         []BundleSource `json:"sources" db:"sources"`
	CacheTTLSeconds int            `json:"cache_ttl_seconds" db:"cache_ttl_seconds"`
}

// BundleSource defines a single data source within a bundle
type BundleSource struct {
	ID            string                 `json:"id"`              // e.g., "positions"
	APIEndpointID uuid.UUID              `json:"api_endpoint_id"` // Link to API Studio endpoint
	ArgsTemplate  map[string]interface{} `json:"args_template"`   // e.g., {"account_id": "{{route.accountId}}"}
}

// PageBundleService handles the resolution and execution of page data bundles
type PageBundleService struct {
	repo         *Repository
	apiRepo      *apistudio.Repository
	resolver     *analytics.BOContextResolver
	db           *sqlx.DB
	redisClient  *redis.Client
	fusionEngine *apistudio.QueryFusionEngine
	rateLimiter  *apistudio.RateLimiter
}

// NewPageBundleService creates a new service instance
func NewPageBundleService(
	repo *Repository,
	apiRepo *apistudio.Repository,
	resolver *analytics.BOContextResolver,
	db *sqlx.DB,
	redisClient *redis.Client,
) *PageBundleService {
	return &PageBundleService{
		repo:         repo,
		apiRepo:      apiRepo,
		resolver:     resolver,
		db:           db,
		redisClient:  redisClient,
		fusionEngine: apistudio.NewQueryFusionEngine(),
		rateLimiter:  apistudio.NewRateLimiter(redisClient),
	}
}

// generateCacheKey creates a unique key for the bundle request
func (s *PageBundleService) generateCacheKey(tenantID, pageSlug string, routeParams map[string]string) string {
	// Simple hash of params
	jsonParams, _ := json.Marshal(routeParams)
	return fmt.Sprintf("bundle:%s:%s:%s", tenantID, pageSlug, string(jsonParams))
}

// GetBundleDefinitionForPage retrieves or derives the bundle definition for a page
func (s *PageBundleService) GetBundleDefinitionForPage(ctx context.Context, pageID uuid.UUID) (*PageDataBundleDefinition, error) {
	// TODO: distinct table for bundles? For now, we might derive it or mock it.
	// In a real implementation effectively backed by DB, we'd query semantic.page_bundles
	// For this Epic, let's implement a deterministic derivation/mock if not found,
	// or assume we only support pages that have one defined.

	// Placeholder: Return a mock or derived bundle
	// Ideally, we'd look up `semantic.page_bundles` where page_id = ?
	return nil, nil // To be implemented with DB persistence if needed
}

// ExecuteBundle resolves and fetches all data for a page bundle
func (s *PageBundleService) ExecuteBundle(
	ctx context.Context,
	tenantID string,
	pageSlug string,
	routeParams map[string]string,
	env string,
	user *string,
	region string,
) (map[string]interface{}, error) {
	// 0. Rate Limit Check (approximate)
	// We don't know exact cost yet, but assume at least 1 unit if requesting anything
	allowed, err := s.rateLimiter.Allow(ctx, tenantID, 1)
	if err != nil {
		// log error
	}
	if !allowed {
		return nil, fmt.Errorf("rate limit exceeded")
	}

	// 0. Check Cache
	cacheKey := s.generateCacheKey(tenantID, pageSlug, routeParams)
	if s.redisClient != nil {
		val, err := s.redisClient.Get(ctx, cacheKey).Result()
		if err == nil {
			var cachedResult map[string]interface{}
			if err := json.Unmarshal([]byte(val), &cachedResult); err == nil {
				return cachedResult, nil
			}
		}
	}

	// 1. Resolve Effective Page
	page, err := s.repo.GetPageBySlug(ctx, pageSlug, env)
	if err != nil {
		return nil, fmt.Errorf("page not found: %w", err)
	}

	// 2. Load Bundle Sources
	var bindings map[string]interface{}
	if len(page.DataBindings) > 0 {
		if err := json.Unmarshal(page.DataBindings, &bindings); err != nil {
			return nil, fmt.Errorf("invalid data bindings: %w", err)
		}
	}

	var sources []BundleSource
	for key, binding := range bindings {
		bMap, ok := binding.(map[string]interface{})
		if !ok {
			continue
		}
		epIDStr, _ := bMap["endpoint_id"].(string)
		if epIDStr == "" {
			continue
		}
		epID, err := uuid.Parse(epIDStr)
		if err != nil {
			continue
		}

		params, _ := bMap["params"].(map[string]interface{})
		sources = append(sources, BundleSource{
			ID:            key,
			APIEndpointID: epID,
			ArgsTemplate:  params,
		})
	}

	if len(sources) == 0 {
		return map[string]interface{}{}, nil
	}

	// 3. Prepare Requests
	requests := make(map[string]analytics.BOSQLRequest)
	// We need to fetch endpoints to build requests. In a future optimization, we could cache this check.
	// We also need to capture any errors during preparation.

	prepErrChan := make(chan error, 1)

	for _, source := range sources {
		// Resolve Args
		args := make(map[string]interface{})
		for k, v := range source.ArgsTemplate {
			if sVal, ok := v.(string); ok && strings.HasPrefix(sVal, "{{route.") {
				paramName := strings.TrimSuffix(strings.TrimPrefix(sVal, "{{route."), "}}")
				if val, ok := routeParams[paramName]; ok {
					args[k] = val
				}
			} else {
				args[k] = v
			}
		}

		// Fetch Endpoint
		ep, err := s.apiRepo.GetEndpoint(ctx, source.APIEndpointID)
		if err != nil {
			prepErrChan <- fmt.Errorf("source %s: endpoint not found", source.ID)
			break
		}

		var fields []string
		if err := json.Unmarshal(ep.Fields, &fields); err != nil {
			prepErrChan <- fmt.Errorf("source %s: invalid fields", source.ID)
			break
		}

		tenantUUID, _ := uuid.Parse(tenantID)
		userID := ""
		if user != nil {
			userID = *user
		}

		req := analytics.BOSQLRequest{
			Env:           env,
			TenantID:      &tenantUUID,
			BOName:        ep.BOName,
			EndpointID:    &ep.ID,
			Measures:      fields,
			Filters:       args,
			CurrentUserID: userID,
			Region:        region,
		}
		requests[source.ID] = req
	}

	if len(prepErrChan) > 0 {
		return nil, <-prepErrChan
	}

	// 4. Try Fusion
	fused, remaining := s.fusionEngine.TryFuse(requests)

	// 5. Execution
	results := make(map[string]interface{})
	var mu sync.Mutex
	var wg sync.WaitGroup
	errChan := make(chan error, len(sources)) // pessimistic size

	// Execute Unfused
	for id, req := range remaining {
		wg.Add(1)
		go func(sid string, r analytics.BOSQLRequest) {
			defer wg.Done()
			res, err := s.executeRequest(ctx, r)
			if err != nil {
				errChan <- fmt.Errorf("source %s: %w", sid, err)
				return
			}
			mu.Lock()
			results[sid] = res
			mu.Unlock()
		}(id, req)
	}

	// Execute Fused
	for _, fq := range fused {
		wg.Add(1)
		go func(q apistudio.FusedQuery) {
			defer wg.Done()

			// Execute Composite
			rows, err := s.executeRequest(ctx, q.CompositeRequest)
			if err != nil {
				// Fail all sources in this bundle
				for _, sid := range q.SourceIDs {
					errChan <- fmt.Errorf("source %s (fused): %w", sid, err)
				}
				return
			}

			// Split Results
			for _, sid := range q.SourceIDs {
				originalReq := requests[sid] // Safe lookup

				// Filter rows for this source
				var sourceRows []map[string]interface{}
				for _, row := range rows {
					filteredRow := make(map[string]interface{})
					for _, m := range originalReq.Measures {
						if val, ok := row[m]; ok {
							filteredRow[m] = val
						}
					}
					sourceRows = append(sourceRows, filteredRow)
				}

				mu.Lock()
				results[sid] = sourceRows
				mu.Unlock()
			}
		}(fq)
	}

	wg.Wait()
	close(errChan)

	if len(errChan) > 0 {
		return nil, <-errChan
	}

	// Cache Result
	if s.redisClient != nil {
		go func() {
			data, _ := json.Marshal(results)
			s.redisClient.Set(context.Background(), cacheKey, data, 5*time.Minute)
		}()
	}

	return results, nil
}

// executeRequest handles resolution and DB execution for a single request
func (s *PageBundleService) executeRequest(ctx context.Context, req analytics.BOSQLRequest) ([]map[string]interface{}, error) {
	// Resolve
	start := time.Now()
	sql, _, err := s.resolver.ResolveQuery(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("resolution error: %w", err)
	}

	// Execute
	rows, err := s.db.QueryxContext(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("execution error: %w", err)
	}
	defer rows.Close()

	var rowsData []map[string]interface{}
	for rows.Next() {
		row := make(map[string]interface{})
		if err := rows.MapScan(row); err == nil {
			rowsData = append(rowsData, row)
		}
	}

	// Telemetry
	// We use the EndpointID from the request if available, but for Fused queries checking just one endpoint might be misleading?
	// For fused queries, we attribute cost to the primary endpoint ID if preserved, or log as "fused".
	// The current fusion logic preserves EndpointID from the first request.
	if req.EndpointID != nil {
		_ = s.apiRepo.LogTelemetry(ctx, &apistudio.APITelemetry{
			APIID:       *req.EndpointID,
			Env:         req.Env,
			TenantID:    req.TenantID,
			ClientType:  "page-bundle-fused",
			StatusCode:  200,
			LatencyMs:   int(time.Since(start).Milliseconds()),
			RequestedAt: time.Now(),
		})
	}

	return rowsData, nil
}
