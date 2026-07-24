package api

// GATEWAY INTEGRATION PATCH
// =======================
// This file demonstrates how to integrate the three-layer semantic query cache
// into the existing LLMGateway.ProcessQuery() method in llm_gateway.go.
//
// THREE INTEGRATION POINTS:
// 1. Layer 1 (NL → SemanticQuery): Cache before/after callPlannerLLM()
// 2. Layer 2 (SemanticQuery → SQL): Cache before/after callExecutorLLM()
// 3. Layer 3 (SQL → Results): Cache before/after executeSQL()
//
// USAGE:
// Apply these patches to internal/api/llm_gateway.go using find-and-replace
// or manual integration. The structure shows exact code locations and changes.

// ============================================================================
// PATCH 1: Add cache member to LLMGateway struct
// ============================================================================
// LOCATION: llm_gateway.go, line ~11 (in the LLMGateway struct definition)
//
// BEFORE:
// type LLMGateway struct {
//     server *Server
// }
//
// AFTER:
// type LLMGateway struct {
//     server *Server
//     cache  *cache.SemanticQueryCache  // Add this line
// }

// ============================================================================
// PATCH 2: Update NewLLMGateway constructor
// ============================================================================
// LOCATION: llm_gateway.go, line ~17 (in NewLLMGateway function)
//
// BEFORE:
// func NewLLMGateway(srv *Server) *LLMGateway {
//     return &LLMGateway{server: srv}
// }
//
// AFTER:
// func NewLLMGateway(srv *Server, queryCache *cache.SemanticQueryCache) *LLMGateway {
//     return &LLMGateway{
//         server: srv,
//         cache:  queryCache,
//     }
// }

// ============================================================================
// PATCH 3: Integrate Layer 1 cache into ProcessQuery
// ============================================================================
// LOCATION: llm_gateway.go, line ~45-60 (in ProcessQuery method, right before callPlannerLLM)
//
// BEFORE:
// // Step 2: Call Planner LLM: NL → SemanticQuery JSON
// semQuery, err := gw.callPlannerLLM(ctx, bundle, req.Prompt, req.Mode)
// if err != nil {
//     baseResp.Error = fmt.Sprintf("planner LLM error: %v", err)
//     return baseResp, err
// }
//
// AFTER:
// // Step 2: Call Planner LLM: NL → SemanticQuery JSON
// // Try Layer 1 cache first: NL → SemanticQuery
// var semQuery *SemanticQuery
// var err error
//
// if gw.cache != nil {
//     plannerCached, cacheErr := gw.cache.GetNLQueryCache(
//         ctx,
//         req.Prompt,
//         req.Datasource,
//         req.Mode,
//         tenantID,
//     )
//     if cacheErr == nil && plannerCached != nil {
//         // Cache hit: deserialize semantic query from cache
//         if err := json.Unmarshal([]byte(plannerCached.SemanticQuery), &semQuery); err != nil {
//             log.Printf("Failed to deserialize cached semantic query: %v", err)
//         } else {
//             log.Printf("Cache HIT (Layer 1): NL → SemanticQuery, saved ~%dms", plannerCached.GenerationTime)
//             // Use cached query, skip LLM call
//             goto validateSemanticQuery
//         }
//     }
// }
//
// // Cache miss: call Planner LLM
// start := time.Now()
// semQuery, err = gw.callPlannerLLM(ctx, bundle, req.Prompt, req.Mode)
// elapsed := time.Since(start)
//
// if err != nil {
//     baseResp.Error = fmt.Sprintf("planner LLM error: %v", err)
//     return baseResp, err
// }
//
// // Cache the result for future use
// if gw.cache != nil {
//     semQueryJSON, _ := json.Marshal(semQuery)
//     cacheEntry := &cache.NLQueryCacheEntry{
//         NLPrompt:       req.Prompt,
//         Datasource:     req.Datasource,
//         Mode:           req.Mode,
//         SemanticQuery:  string(semQueryJSON),
//         GeneratedAt:    time.Now(),
//         LLMModel:       "gemini-pro", // or configured LLM
//         GenerationTime: elapsed.Milliseconds(),
//         TenantID:       tenantID,
//     }
//     if cacheErr := gw.cache.SetNLQueryCache(ctx, req.Prompt, req.Datasource, req.Mode, tenantID, cacheEntry); cacheErr != nil {
//         log.Printf("Warning: failed to cache NL query: %v", cacheErr)
//     }
// }
//
// validateSemanticQuery:
// // Continue with validation...

// ============================================================================
// PATCH 4: Integrate Layer 2 cache before callExecutorLLM
// ============================================================================
// LOCATION: llm_gateway.go, line ~65-75 (after semantic query validation, before callExecutorLLM)
//
// BEFORE:
// // Step 4: Call Executor (LLM): SemanticQuery + bundle → SQL
// sql, err := gw.callExecutorLLM(ctx, bundle, semQuery)
// if err != nil {
//     baseResp.Error = fmt.Sprintf("executor LLM error: %v", err)
//     return baseResp, err
// }
//
// baseResp.GeneratedSQL = sql
//
// AFTER:
// // Step 4: Call Executor (LLM): SemanticQuery + bundle → SQL
// // Try Layer 2 cache first: SemanticQuery → SQL
// var sql string
// var err error
//
// semQueryJSON, _ := json.Marshal(semQuery)
// if gw.cache != nil {
//     executorCached, cacheErr := gw.cache.GetSQLQueryCache(
//         ctx,
//         string(semQueryJSON),
//         "postgres", // or detect from database
//         tenantID,
//     )
//     if cacheErr == nil && executorCached != nil {
//         sql = executorCached.GeneratedSQL
//         log.Printf("Cache HIT (Layer 2): SemanticQuery → SQL, saved ~%dms", executorCached.GenerationTime)
//         goto executionSQL
//     }
// }
//
// // Cache miss: call Executor LLM
// start := time.Now()
// sql, err = gw.callExecutorLLM(ctx, bundle, semQuery)
// elapsed := time.Since(start)
//
// if err != nil {
//     baseResp.Error = fmt.Sprintf("executor LLM error: %v", err)
//     return baseResp, err
// }
//
// // Cache the result
// if gw.cache != nil {
//     cacheEntry := &cache.SQLQueryCacheEntry{
//         SemanticQuery:  string(semQueryJSON),
//         DatabaseType:   "postgres",
//         GeneratedSQL:   sql,
//         GeneratedAt:    time.Now(),
//         LLMModel:       "gemini-pro",
//         GenerationTime: elapsed.Milliseconds(),
//         TenantID:       tenantID,
//         Validated:      true,
//     }
//     if cacheErr := gw.cache.SetSQLQueryCache(ctx, string(semQueryJSON), "postgres", tenantID, cacheEntry); cacheErr != nil {
//         log.Printf("Warning: failed to cache SQL query: %v", cacheErr)
//     }
// }
//
// executionSQL:
// baseResp.GeneratedSQL = sql

// ============================================================================
// PATCH 5: Integrate Layer 3 cache before executeSQL
// ============================================================================
// LOCATION: llm_gateway.go, line ~80-95 (before executeSQL call)
//
// BEFORE:
// // Step 5: Execute SQL against database
// rows, err := gw.executeSQL(ctx, sql)
// if err != nil {
//     baseResp.Error = fmt.Sprintf("SQL execution failed: %v", err)
//     return baseResp, err
// }
//
// baseResp.Rows = rows
// baseResp.Count = len(rows)
//
// AFTER:
// // Step 5: Execute SQL against database
// // Try Layer 3 cache first: SQL → Results
// var rows []interface{}
// var err error
//
// if gw.cache != nil {
//     resultsCached, cacheErr := gw.cache.GetResultsCache(
//         ctx,
//         sql,
//         tenantID,
//         bundle.DrivingTable, // or database name
//     )
//     if cacheErr == nil && resultsCached != nil {
//         // Cache hit: deserialize results
//         if err := json.Unmarshal([]byte(resultsCached.Results), &rows); err != nil {
//             log.Printf("Failed to deserialize cached results: %v", err)
//         } else {
//             log.Printf("Cache HIT (Layer 3): SQL → Results, saved ~%dms", resultsCached.ExecutionTime)
//             baseResp.Rows = rows
//             baseResp.Count = resultsCached.RowCount
//             return baseResp, nil
//         }
//     }
// }
//
// // Cache miss: execute SQL
// start := time.Now()
// rows, err = gw.executeSQL(ctx, sql)
// elapsed := time.Since(start)
//
// if err != nil {
//     baseResp.Error = fmt.Sprintf("SQL execution failed: %v", err)
//     return baseResp, err
// }
//
// // Cache the results
// if gw.cache != nil {
//     rowsJSON, _ := json.Marshal(rows)
//     cacheEntry := &cache.ResultsCacheEntry{
//         SQL:           sql,
//         RowCount:      len(rows),
//         Results:       string(rowsJSON),
//         ExecutedAt:    time.Now(),
//         ExecutionTime: elapsed.Milliseconds(),
//         TenantID:      tenantID,
//         QueryHash:     cache.HashSQL(sql, tenantID, bundle.DrivingTable),
//         DatabaseName:  bundle.DrivingTable,
//     }
//     if cacheErr := gw.cache.SetResultsCache(ctx, sql, tenantID, bundle.DrivingTable, cacheEntry); cacheErr != nil {
//         log.Printf("Warning: failed to cache results: %v", cacheErr)
//     }
// }
//
// baseResp.Rows = rows
// baseResp.Count = len(rows)

// ============================================================================
// PATCH 6: Update llm_handlers.go - handlePlannerOnly endpoint
// ============================================================================
// LOCATION: internal/api/llm_handlers.go, line ~90 (in handlePlannerOnly function)
//
// BEFORE:
// gateway := NewLLMGateway(srv)
//
// AFTER:
// // Get query cache from server if available
// var queryCache *cache.SemanticQueryCache
// if srv.QueryCache != nil {
//     queryCache = srv.QueryCache
// }
// gateway := NewLLMGateway(srv, queryCache)

// ============================================================================
// PATCH 7: Update api.go - register query cache with server
// ============================================================================
// LOCATION: internal/api/api.go, in Server struct (around line 50)
//
// BEFORE:
// type Server struct {
//     // ... existing fields
//     GeminiClient *GeminiClient
// }
//
// AFTER:
// type Server struct {
//     // ... existing fields
//     GeminiClient *GeminiClient
//     QueryCache   *cache.SemanticQueryCache  // Add this
// }

// ============================================================================
// PATCH 8: server.go - Initialize query cache on startup
// ============================================================================
// LOCATION: internal/api/server.go, in InitializeServer() or main setup
//
// ADD THESE LINES (after Redis client initialization):
//
// // Initialize semantic query cache
// queryCache, err := cache.NewSemanticQueryCache(
//     cfg.RedisAddr,      // e.g., "localhost:6379"
//     cfg.RedisPassword,  // from config
//     1,                  // Redis DB 1 for query cache (0 for views)
// )
// if err != nil {
//     log.Printf("Warning: failed to initialize query cache: %v", err)
// }
// srv.QueryCache = queryCache
// log.Printf("Semantic Query Cache initialized")

// ============================================================================
// INTEGRATION CHECKLIST
// ============================================================================
// 
// [ ] Add cache member to LLMGateway struct (Patch 1)
// [ ] Update NewLLMGateway constructor (Patch 2)
// [ ] Integrate Layer 1 NL→Query cache (Patch 3)
// [ ] Integrate Layer 2 SemanticQuery→SQL cache (Patch 4)
// [ ] Integrate Layer 3 SQL→Results cache (Patch 5)
// [ ] Update llm_handlers.go references (Patch 6)
// [ ] Add QueryCache field to Server struct (Patch 7)
// [ ] Initialize cache in server startup (Patch 8)
// [ ] Add "encoding/json" import in llm_gateway.go if not present
// [ ] Add "github.com/eganpj/GitHub/semlayer/backend/internal/cache" import
// [ ] Test each cache layer individually
// [ ] Verify metrics collection works
// [ ] Monitor cache hit rates in production
//
// EXPECTED PERFORMANCE GAINS:
// - Layer 1 misses ~1-50ms (Gemini planning time)
// - Layer 2 misses ~500-1500ms (Gemini SQL generation)
// - Layer 3 misses ~50-500ms (DB query execution)
// - Total potential savings: 90% reduction in LLM costs
// - Latency improvement: 10x faster on cache hits

type QueryCacheIntegrationPatch struct {
	Description string
	Location    string
	PatchNumber int
}

// DocumentedPatches lists all required integration patches
var DocumentedPatches = []QueryCacheIntegrationPatch{
	{
		Description: "Add cache member to LLMGateway struct",
		Location:    "internal/api/llm_gateway.go:11",
		PatchNumber: 1,
	},
	{
		Description: "Update NewLLMGateway constructor",
		Location:    "internal/api/llm_gateway.go:17",
		PatchNumber: 2,
	},
	{
		Description: "Integrate Layer 1 NL→Query cache in ProcessQuery",
		Location:    "internal/api/llm_gateway.go:45",
		PatchNumber: 3,
	},
	{
		Description: "Integrate Layer 2 SemanticQuery→SQL cache in ProcessQuery",
		Location:    "internal/api/llm_gateway.go:65",
		PatchNumber: 4,
	},
	{
		Description: "Integrate Layer 3 SQL→Results cache in ProcessQuery",
		Location:    "internal/api/llm_gateway.go:80",
		PatchNumber: 5,
	},
	{
		Description: "Update llm_handlers.go handlePlannerOnly endpoint",
		Location:    "internal/api/llm_handlers.go:90",
		PatchNumber: 6,
	},
	{
		Description: "Add QueryCache field to Server struct",
		Location:    "internal/api/api.go:50",
		PatchNumber: 7,
	},
	{
		Description: "Initialize query cache on server startup",
		Location:    "internal/api/server.go:startup",
		PatchNumber: 8,
	},
}
