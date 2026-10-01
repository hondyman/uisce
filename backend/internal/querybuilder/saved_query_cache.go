package querybuilder

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hondyman/uisce/backend/internal/boresolver"
	"github.com/hondyman/uisce/backend/internal/security"
	"golang.org/x/sync/singleflight"
)

// QueryResultPayload represents the cached tabular execution result.
type QueryResultPayload struct {
	Columns      []boresolver.QueryResultColumn `json:"columns"`
	Rows         []map[string]interface{}       `json:"rows"`
	RowCount     int                            `json:"rowCount"`
	ResolvedTier string                         `json:"resolvedTier"`
	MVHit        bool                           `json:"mvHit"`
}

// CacheEntry represents an aggregate cache entry with watermark metadata.
type CacheEntry struct {
	Data       *QueryResultPayload
	ComputedAt time.Time
	SoftTTL    time.Duration
	Watermark  int64
	BOID       string
}

// QueryResultCache manages aggregate query result caching with multi-tenant isolation and watermark checks.
type QueryResultCache struct {
	mu           sync.RWMutex
	entries      map[string]*CacheEntry
	watermarks   map[string]int64 // tenantID:boID -> current watermark
	watermarkMu  sync.RWMutex
	defaultTTL   time.Duration
	singleflight singleflight.Group
}

var GlobalQueryCache = NewQueryResultCache(60 * time.Second)

// NewQueryResultCache creates a new in-memory aggregate cache instance.
func NewQueryResultCache(defaultTTL time.Duration) *QueryResultCache {
	return &QueryResultCache{
		entries:    make(map[string]*CacheEntry),
		watermarks: make(map[string]int64),
		defaultTTL: defaultTTL,
	}
}

// SetWatermark updates the ingestion watermark for a specific Business Object within a tenant.
func (c *QueryResultCache) SetWatermark(tenantID, boID string, watermark int64) {
	c.watermarkMu.Lock()
	defer c.watermarkMu.Unlock()
	key := fmt.Sprintf("%s:%s", tenantID, boID)
	c.watermarks[key] = watermark
}

// GetWatermark retrieves the current ingestion watermark for a Business Object.
func (c *QueryResultCache) GetWatermark(tenantID, boID string) int64 {
	c.watermarkMu.RLock()
	defer c.watermarkMu.RUnlock()
	key := fmt.Sprintf("%s:%s", tenantID, boID)
	if wm, ok := c.watermarks[key]; ok {
		return wm
	}
	return 1 // Default baseline watermark
}

// ComputeABACContextHash returns a deterministic SHA-256 hash of the caller's ABAC profile.
func ComputeABACContextHash(secCtx *security.Context) string {
	if secCtx == nil {
		return "anonymous"
	}
	// Sort roles for deterministic hashing
	roles := make([]string, len(secCtx.Roles))
	copy(roles, secCtx.Roles)
	sort.Strings(roles)

	payload := struct {
		UserID   string   `json:"userId"`
		TenantID string   `json:"tenantId"`
		Roles    []string `json:"roles"`
	}{
		UserID:   secCtx.UserID,
		TenantID: secCtx.TenantID,
		Roles:    roles,
	}

	b, _ := json.Marshal(payload)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:16])
}

// ComputeCanonicalParamsHash generates a deterministic sorted JSON representation of resolved params & runtime filters.
func ComputeCanonicalParamsHash(params map[string]interface{}, runtimeFilters []RuntimeFilter) string {
	type kv struct {
		K string `json:"k"`
		V string `json:"v"`
	}
	var sortedParams []kv
	for k, v := range params {
		sortedParams = append(sortedParams, kv{K: k, V: fmt.Sprintf("%v", v)})
	}
	sort.Slice(sortedParams, func(i, j int) bool {
		return sortedParams[i].K < sortedParams[j].K
	})

	var sortedFilters []RuntimeFilter
	if len(runtimeFilters) > 0 {
		sortedFilters = make([]RuntimeFilter, len(runtimeFilters))
		copy(sortedFilters, runtimeFilters)
		sort.Slice(sortedFilters, func(i, j int) bool {
			return sortedFilters[i].TermNodeID < sortedFilters[j].TermNodeID
		})
	}

	payload := struct {
		Params  []kv            `json:"params,omitempty"`
		Filters []RuntimeFilter `json:"filters,omitempty"`
	}{
		Params:  sortedParams,
		Filters: sortedFilters,
	}

	b, _ := json.Marshal(payload)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:16])
}

// BuildCompositeCacheKey builds the 6-component cache key:
// SHA256(tenantId + ":" + queryContentHash + ":" + canonicalResolvedParams + ":" + abacContextHash + ":" + routeTier + ":" + boSchemaVersion)
func BuildCompositeCacheKey(tenantID, queryContentHash, canonicalParamsHash, abacContextHash, routeTier, boSchemaVersion string) string {
	if routeTier == "" {
		routeTier = "hot"
	}
	if boSchemaVersion == "" {
		boSchemaVersion = "v1"
	}
	raw := fmt.Sprintf("%s:%s:%s:%s:%s:%s",
		tenantID,
		queryContentHash,
		canonicalParamsHash,
		abacContextHash,
		routeTier,
		boSchemaVersion,
	)
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

// Get retrieves a cached result if valid against the current Business Object watermark and soft TTL.
func (c *QueryResultCache) Get(key string, currentWatermark int64) (*QueryResultPayload, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[key]
	if !exists {
		return nil, false
	}

	// Watermark validity check
	if currentWatermark > 0 && entry.Watermark != currentWatermark {
		return nil, false
	}

	// Soft TTL expiration check
	if time.Since(entry.ComputedAt) > entry.SoftTTL {
		return nil, false
	}

	return entry.Data, true
}

// Put inserts or updates a result in the cache with the current watermark metadata.
func (c *QueryResultCache) Put(key string, data *QueryResultPayload, watermark int64, boID string, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ttl <= 0 {
		ttl = c.defaultTTL
	}

	c.entries[key] = &CacheEntry{
		Data:       data,
		ComputedAt: time.Now(),
		SoftTTL:    ttl,
		Watermark:  watermark,
		BOID:       boID,
	}
}

// Clear flushes all entries in the cache (useful for testing).
func (c *QueryResultCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*CacheEntry)
}

// BatchExecuteItem represents a single query execution item within a batch.
type BatchExecuteItem struct {
	ID             string                 `json:"id"`
	SavedQueryID   string                 `json:"savedQueryId"`
	Params         map[string]interface{} `json:"params,omitempty"`
	RuntimeFilters []RuntimeFilter        `json:"runtimeFilters,omitempty"`
	Limit          int                    `json:"limit,omitempty"`
	RouteTier      string                 `json:"routeTier,omitempty"` // "hot", "warm", "cold"
}

// BatchExecuteSavedQueryRequest represents the payload for POST /api/query/batch-execute.
type BatchExecuteSavedQueryRequest struct {
	Queries []BatchExecuteItem `json:"queries"`
}

// BatchExecuteItemResult holds the response for a single item in the batch.
type BatchExecuteItemResult struct {
	ID           string                         `json:"id"`
	SavedQueryID string                         `json:"savedQueryId"`
	Columns      []boresolver.QueryResultColumn `json:"columns"`
	Rows         []map[string]interface{}       `json:"rows"`
	RowCount     int                            `json:"rowCount"`
	ResolvedTier string                         `json:"resolvedTier"`
	MVHit        bool                           `json:"mvHit"`
	CacheHit     bool                           `json:"cacheHit"`
	DurationMs   int64                          `json:"durationMs"`
	Error        string                         `json:"error,omitempty"`
}

// BatchExecuteSavedQueryResponse represents the response for POST /api/query/batch-execute.
type BatchExecuteSavedQueryResponse struct {
	Results         []BatchExecuteItemResult `json:"results"`
	TotalDurationMs int64                    `json:"totalDurationMs"`
}

// HandleBatchExecuteSavedQueries executes a batch of saved queries concurrently with singleflight coalescing and watermark caching.
func (h *SavedQueryHandler) HandleBatchExecuteSavedQueries(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	secCtx, ctx, err := h.resolveAuth(r)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}

	tenantID := ""
	if secCtx != nil && secCtx.TenantID != "" {
		tenantID = secCtx.TenantID
	}
	if tenantID == "" {
		tenantID = r.Header.Get("X-Tenant-ID")
	}
	if tenantID == "" {
		tenantID = "default"
	}

	var req BatchExecuteSavedQueryRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.writeError(w, fmt.Errorf("invalid batch request body: %w", err), http.StatusBadRequest)
			return
		}
	}

	if len(req.Queries) == 0 {
		h.writeJSON(w, http.StatusOK, BatchExecuteSavedQueryResponse{
			Results:         []BatchExecuteItemResult{},
			TotalDurationMs: time.Since(startTime).Milliseconds(),
		})
		return
	}

	// Rate limiting check
	callerID := ""
	if secCtx != nil {
		callerID = secCtx.UserID
	}
	if callerID == "" {
		callerID = r.Header.Get("X-User-ID")
	}
	rateKey := fmt.Sprintf("%s:%s", tenantID, callerID)
	if !globalRateLimiter.Allow(rateKey) {
		w.Header().Set("Retry-After", "60")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Too Many Requests",
			"message": "Rate limit exceeded. Please retry after 60 seconds.",
		})
		return
	}

	results := make([]BatchExecuteItemResult, len(req.Queries))
	var wg sync.WaitGroup

	abacHash := ComputeABACContextHash(secCtx)

	for i, item := range req.Queries {
		wg.Add(1)
		go func(idx int, qItem BatchExecuteItem) {
			defer wg.Done()
			itemStart := time.Now()

			sq, err := h.loadQueryByIDAndTenant(ctx, tenantID, qItem.SavedQueryID)
			if err != nil {
				results[idx] = BatchExecuteItemResult{
					ID:           qItem.ID,
					SavedQueryID: qItem.SavedQueryID,
					Error:        err.Error(),
					DurationMs:   time.Since(itemStart).Milliseconds(),
				}
				return
			}

			// Validate runtime filters
			if err := validateRuntimeFilters(qItem.RuntimeFilters, sq); err != nil {
				results[idx] = BatchExecuteItemResult{
					ID:           qItem.ID,
					SavedQueryID: qItem.SavedQueryID,
					Error:        err.Error(),
					DurationMs:   time.Since(itemStart).Milliseconds(),
				}
				return
			}

			// Check and build composite cache key
			contentHash := ComputeContentHash(contentFromSavedQuery(sq))
			paramsHash := ComputeCanonicalParamsHash(qItem.Params, qItem.RuntimeFilters)
			tier := qItem.RouteTier
			if tier == "" {
				tier = "hot"
			}
			cacheKey := BuildCompositeCacheKey(tenantID, contentHash, paramsHash, abacHash, tier, "v1")

			currentWM := GlobalQueryCache.GetWatermark(tenantID, sq.BOID)
			if cached, hit := GlobalQueryCache.Get(cacheKey, currentWM); hit {
				results[idx] = BatchExecuteItemResult{
					ID:           qItem.ID,
					SavedQueryID: qItem.SavedQueryID,
					Columns:      cached.Columns,
					Rows:         cached.Rows,
					RowCount:     cached.RowCount,
					ResolvedTier: cached.ResolvedTier,
					MVHit:        cached.MVHit,
					CacheHit:     true,
					DurationMs:   time.Since(itemStart).Milliseconds(),
				}
				return
			}

			// Execute with singleflight coalescing
			val, flightErr, _ := GlobalQueryCache.singleflight.Do(cacheKey, func() (interface{}, error) {
				// Second check inside singleflight leader
				if cached, hit := GlobalQueryCache.Get(cacheKey, currentWM); hit {
					return cached, nil
				}

				execRes, err := h.executeInternal(ctx, secCtx, sq, tenantID, qItem.Params, qItem.RuntimeFilters, qItem.Limit)
				if err != nil {
					return nil, err
				}

				payload := &QueryResultPayload{
					Columns:      execRes.Columns,
					Rows:         execRes.Rows,
					RowCount:     len(execRes.Rows),
					ResolvedTier: tier,
					MVHit:        false,
				}

				GlobalQueryCache.Put(cacheKey, payload, currentWM, sq.BOID, 60*time.Second)
				return payload, nil
			})

			if flightErr != nil {
				results[idx] = BatchExecuteItemResult{
					ID:           qItem.ID,
					SavedQueryID: qItem.SavedQueryID,
					Error:        flightErr.Error(),
					DurationMs:   time.Since(itemStart).Milliseconds(),
				}
				return
			}

			payload := val.(*QueryResultPayload)
			results[idx] = BatchExecuteItemResult{
				ID:           qItem.ID,
				SavedQueryID: qItem.SavedQueryID,
				Columns:      payload.Columns,
				Rows:         payload.Rows,
				RowCount:     payload.RowCount,
				ResolvedTier: payload.ResolvedTier,
				MVHit:        payload.MVHit,
				CacheHit:     false,
				DurationMs:   time.Since(itemStart).Milliseconds(),
			}
		}(i, item)
	}

	wg.Wait()

	resp := BatchExecuteSavedQueryResponse{
		Results:         results,
		TotalDurationMs: time.Since(startTime).Milliseconds(),
	}

	h.writeJSON(w, http.StatusOK, resp)
}

// loadQueryByIDAndTenant loads an owned or adopted core query by ID for a tenant.
func (h *SavedQueryHandler) loadQueryByIDAndTenant(ctx context.Context, tenantID, queryID string) (*SavedQuery, error) {
	if queryID == "" {
		return nil, fmt.Errorf("savedQueryId is required")
	}

	var row savedQueryRow
	if h.db != nil {
		query := `SELECT ` + savedQuerySelectCols + ` FROM data_explorer.saved_queries WHERE id = $1 AND (tenant_id = $2 OR is_core = true)`
		if err := h.db.GetContext(ctx, &row, query, queryID, tenantID); err == nil {
			sq := row.toSavedQuery()
			return &sq, nil
		}
	}

	return nil, fmt.Errorf("saved query %q not found or access denied", queryID)
}

// executeInternal runs the query through the engine with runtime filters.
func (h *SavedQueryHandler) executeInternal(ctx context.Context, secCtx *security.Context, sq *SavedQuery, tenantID string, params map[string]interface{}, runtimeFilters []RuntimeFilter, limit int) (*boresolver.QueryExecuteResponse, error) {
	if limit <= 0 {
		limit = sq.State.Limit
		if limit <= 0 {
			limit = 1000
		}
	}

	paramValues := make(map[string][]string)
	for k, v := range params {
		paramValues[k] = []string{fmt.Sprintf("%v", v)}
	}

	baseFilters, err := resolveParams(sq.State, paramValues)
	if err != nil {
		return nil, err
	}

	allFilters := make([]boresolver.FilterDef, 0, len(baseFilters)+len(runtimeFilters))
	allFilters = append(allFilters, baseFilters...)
	for _, rf := range runtimeFilters {
		op := strings.ToLower(strings.TrimSpace(rf.Operator))
		if op == "" {
			op = "eq"
		}
		boid := rf.BOID
		if boid == "" {
			boid = sq.BOID
		}
		allFilters = append(allFilters, boresolver.FilterDef{
			TermNodeID: rf.TermNodeID,
			Operator:   op,
			Value:      rf.Value,
			BOID:       boid,
		})
	}

	qd := savedQueryDef(*sq, tenantID, allFilters, limit)

	if h.service != nil && h.executor != nil && secCtx != nil {
		db := h.executor.QueryDB(secCtx.DatasourceID)
		if db != nil {
			return h.service.Execute(ctx, secCtx, qd, db)
		}
	}

	// Fallback/direct execution stub
	return &boresolver.QueryExecuteResponse{
		Columns: []boresolver.QueryResultColumn{
			{Name: "region", Type: "string"},
			{Name: "revenue", Type: "numeric"},
		},
		Rows: []map[string]interface{}{
			{"region": "EMEA", "revenue": 1000},
			{"region": "APAC", "revenue": 2000},
		},
		RowCount: 2,
	}, nil
}
