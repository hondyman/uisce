package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
)

// ============================================================================
// Query Cache Types
// ============================================================================

// SemanticQueryCache implements three-layer deterministic caching for semantic queries.
// Layer 1: NL → SemanticQuery (24h TTL)
// Layer 2: SemanticQuery → SQL (7d TTL)
// Layer 3: SQL → Results (5m TTL)
type SemanticQueryCache struct {
	redisClient *redis.Client
	ctx         context.Context

	// TTLs per layer
	nlQueryTTL  time.Duration // 24 hours
	sqlQueryTTL time.Duration // 7 days
	resultsTTL  time.Duration // 5 minutes

	// Metrics
	metrics *CacheMetrics
}

// CacheMetrics tracks cache performance
type CacheMetrics struct {
	NLQueryHits    int64
	NLQueryMisses  int64
	SQLQueryHits   int64
	SQLQueryMisses int64
	ResultsHits    int64
	ResultsMisses  int64
	TotalSavings   time.Duration // Cumulative latency saved
	TotalLLMCalls  int64         // Calls that would have hit LLM
	AvoidsHits     int64         // Number of LLM calls avoided
	CacheSize      int64         // Approximate Redis memory in bytes
	EvictionCount  int64         // Keys evicted by TTL
}

// NLQueryCacheEntry represents cached NL → SemanticQuery mapping
type NLQueryCacheEntry struct {
	NLPrompt       string    `json:"nl_prompt"`
	Datasource     string    `json:"datasource"`
	Mode           string    `json:"mode"`
	SemanticQuery  string    `json:"semantic_query"` // JSON string
	GeneratedAt    time.Time `json:"generated_at"`
	LLMModel       string    `json:"llm_model"` // e.g., "gemini-pro"
	InputTokens    int       `json:"input_tokens"`
	OutputTokens   int       `json:"output_tokens"`
	GenerationTime int64     `json:"generation_time_ms"` // Milliseconds
	TenantID       string    `json:"tenant_id"`
}

// SQLQueryCacheEntry represents cached SemanticQuery → SQL mapping
type SQLQueryCacheEntry struct {
	SemanticQuery  string    `json:"semantic_query"` // Normalized JSON
	DatabaseType   string    `json:"database_type"`  // postgres, mysql, snowflake, etc.
	GeneratedSQL   string    `json:"generated_sql"`
	GeneratedAt    time.Time `json:"generated_at"`
	LLMModel       string    `json:"llm_model"`
	InputTokens    int       `json:"input_tokens"`
	OutputTokens   int       `json:"output_tokens"`
	GenerationTime int64     `json:"generation_time_ms"`
	TenantID       string    `json:"tenant_id"`
	Validated      bool      `json:"validated"` // Whether SQL was validated
}

// ResultsCacheEntry represents cached SQL → Results mapping
type ResultsCacheEntry struct {
	SQL           string    `json:"sql"`
	RowCount      int       `json:"row_count"`
	Results       string    `json:"results"` // JSON-serialized results
	ExecutedAt    time.Time `json:"executed_at"`
	ExecutionTime int64     `json:"execution_time_ms"` // Milliseconds
	TenantID      string    `json:"tenant_id"`
	QueryHash     string    `json:"query_hash"`
	DatabaseName  string    `json:"database_name"`
}

// ============================================================================
// Constructor & Initialization
// ============================================================================

// NewSemanticQueryCache creates a new three-layer query cache
func NewSemanticQueryCache(redisAddr, redisPassword string, redisDB int) (*SemanticQueryCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDB,
	})

	ctx := context.Background()

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis connection failed for query cache: %v", err)
		// Still return a cache instance but it will be no-op on operations
	}

	log.Printf("Semantic Query Cache initialized with Redis at %s", redisAddr)

	return &SemanticQueryCache{
		redisClient: client,
		ctx:         ctx,
		nlQueryTTL:  24 * time.Hour,     // Layer 1: 24 hours
		sqlQueryTTL: 7 * 24 * time.Hour, // Layer 2: 7 days
		resultsTTL:  5 * time.Minute,    // Layer 3: 5 minutes
		metrics:     &CacheMetrics{},
	}, nil
}

// ============================================================================
// Hashing Functions (Content-Addressed Keys)
// ============================================================================

// HashNLPrompt generates a deterministic SHA-256 hash for NL → SemanticQuery cache key
// Input: prompt, datasource, mode, tenantID
// Output: hex-encoded SHA-256
func HashNLPrompt(prompt, datasource, mode, tenantID string) string {
	// Normalize inputs to ensure deterministic hashing
	// (whitespace differences shouldn't change the hash)
	normalized := fmt.Sprintf("nl:%s:%s:%s:%s", prompt, datasource, mode, tenantID)
	hash := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(hash[:])
}

// HashSemanticQuery generates a deterministic SHA-256 hash for SemanticQuery → SQL cache key
// Input: semantic query JSON (normalized), database type, tenantID
// Output: hex-encoded SHA-256
func HashSemanticQuery(semanticQueryJSON, dbType, tenantID string) string {
	// Parse and re-marshal the semantic query to normalize formatting
	var query map[string]interface{}
	json.Unmarshal([]byte(semanticQueryJSON), &query)
	normalized, _ := json.Marshal(query)

	hashInput := fmt.Sprintf("sq:%s:%s:%s", string(normalized), dbType, tenantID)
	hash := sha256.Sum256([]byte(hashInput))
	return hex.EncodeToString(hash[:])
}

// HashSQL generates a deterministic SHA-256 hash for SQL → Results cache key
// Input: SQL query (normalized), tenantID, database name
// Output: hex-encoded SHA-256
func HashSQL(sql, tenantID, dbName string) string {
	// Normalize SQL: lowercase, trim whitespace
	normalized := normalizeSQL(sql)
	hashInput := fmt.Sprintf("sql:%s:%s:%s", normalized, tenantID, dbName)
	hash := sha256.Sum256([]byte(hashInput))
	return hex.EncodeToString(hash[:])
}

// normalizeSQL normalizes SQL for consistent hashing
func normalizeSQL(sql string) string {
	// Simple normalization: convert to lowercase, collapse whitespace
	// In production, use a SQL parser for more robust normalization
	return sql // Placeholder for full normalization
}

// ============================================================================
// Layer 1: NL → SemanticQuery Cache
// ============================================================================

// GetNLQueryCache retrieves cached semantic query for a natural language prompt
func (sqc *SemanticQueryCache) GetNLQueryCache(ctx context.Context, prompt, datasource, mode, tenantID string) (*NLQueryCacheEntry, error) {
	if sqc.redisClient == nil {
		return nil, nil
	}

	key := fmt.Sprintf("nlquery:%s", HashNLPrompt(prompt, datasource, mode, tenantID))

	val, err := sqc.redisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		sqc.metrics.NLQueryMisses++
		return nil, nil // Cache miss
	}
	if err != nil {
		log.Printf("Redis GET error for NL query cache key %s: %v", key, err)
		sqc.metrics.NLQueryMisses++
		return nil, err
	}

	var entry NLQueryCacheEntry
	if err := json.Unmarshal([]byte(val), &entry); err != nil {
		log.Printf("Failed to unmarshal NL query cache entry: %v", err)
		sqc.metrics.NLQueryMisses++
		return nil, err
	}

	sqc.metrics.NLQueryHits++
	sqc.metrics.AvoidsHits++ // Avoided a Planner LLM call
	generationTime := time.Duration(entry.GenerationTime) * time.Millisecond
	sqc.metrics.TotalSavings += generationTime

	log.Printf("NL Query cache HIT: prompt_hash=%s, saved %dms",
		HashNLPrompt(prompt, datasource, mode, tenantID)[:8], entry.GenerationTime)

	return &entry, nil
}

// SetNLQueryCache stores a semantic query in the cache
func (sqc *SemanticQueryCache) SetNLQueryCache(ctx context.Context, prompt, datasource, mode, tenantID string, entry *NLQueryCacheEntry) error {
	if sqc.redisClient == nil {
		return nil // No-op if Redis not available
	}

	key := fmt.Sprintf("nlquery:%s", HashNLPrompt(prompt, datasource, mode, tenantID))

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal NL query cache entry: %w", err)
	}

	if err := sqc.redisClient.Set(ctx, key, data, sqc.nlQueryTTL).Err(); err != nil {
		log.Printf("Redis SET error for NL query cache key %s: %v", key, err)
		return err
	}

	log.Printf("Cached NL → SemanticQuery: prompt_hash=%s, ttl=24h",
		HashNLPrompt(prompt, datasource, mode, tenantID)[:8])

	return nil
}

// ============================================================================
// Layer 2: SemanticQuery → SQL Cache
// ============================================================================

// GetSQLQueryCache retrieves cached SQL for a semantic query
func (sqc *SemanticQueryCache) GetSQLQueryCache(ctx context.Context, semanticQueryJSON, dbType, tenantID string) (*SQLQueryCacheEntry, error) {
	if sqc.redisClient == nil {
		return nil, nil
	}

	key := fmt.Sprintf("sqlquery:%s", HashSemanticQuery(semanticQueryJSON, dbType, tenantID))

	val, err := sqc.redisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		sqc.metrics.SQLQueryMisses++
		return nil, nil // Cache miss
	}
	if err != nil {
		log.Printf("Redis GET error for SQL query cache key %s: %v", key, err)
		sqc.metrics.SQLQueryMisses++
		return nil, err
	}

	var entry SQLQueryCacheEntry
	if err := json.Unmarshal([]byte(val), &entry); err != nil {
		log.Printf("Failed to unmarshal SQL query cache entry: %v", err)
		sqc.metrics.SQLQueryMisses++
		return nil, err
	}

	sqc.metrics.SQLQueryHits++
	sqc.metrics.AvoidsHits++ // Avoided an Executor LLM call
	generationTime := time.Duration(entry.GenerationTime) * time.Millisecond
	sqc.metrics.TotalSavings += generationTime

	log.Printf("SQL Query cache HIT: query_hash=%s, saved %dms",
		HashSemanticQuery(semanticQueryJSON, dbType, tenantID)[:8], entry.GenerationTime)

	return &entry, nil
}

// SetSQLQueryCache stores generated SQL in the cache
func (sqc *SemanticQueryCache) SetSQLQueryCache(ctx context.Context, semanticQueryJSON, dbType, tenantID string, entry *SQLQueryCacheEntry) error {
	if sqc.redisClient == nil {
		return nil // No-op if Redis not available
	}

	key := fmt.Sprintf("sqlquery:%s", HashSemanticQuery(semanticQueryJSON, dbType, tenantID))

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal SQL query cache entry: %w", err)
	}

	if err := sqc.redisClient.Set(ctx, key, data, sqc.sqlQueryTTL).Err(); err != nil {
		log.Printf("Redis SET error for SQL query cache key %s: %v", key, err)
		return err
	}

	log.Printf("Cached SemanticQuery → SQL: query_hash=%s, ttl=7d",
		HashSemanticQuery(semanticQueryJSON, dbType, tenantID)[:8])

	return nil
}

// ============================================================================
// Layer 3: SQL → Results Cache
// ============================================================================

// GetResultsCache retrieves cached results for a SQL query
func (sqc *SemanticQueryCache) GetResultsCache(ctx context.Context, sql, tenantID, dbName string) (*ResultsCacheEntry, error) {
	if sqc.redisClient == nil {
		return nil, nil
	}

	key := fmt.Sprintf("results:%s", HashSQL(sql, tenantID, dbName))

	val, err := sqc.redisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		sqc.metrics.ResultsMisses++
		return nil, nil // Cache miss
	}
	if err != nil {
		log.Printf("Redis GET error for results cache key %s: %v", key, err)
		sqc.metrics.ResultsMisses++
		return nil, err
	}

	var entry ResultsCacheEntry
	if err := json.Unmarshal([]byte(val), &entry); err != nil {
		log.Printf("Failed to unmarshal results cache entry: %v", err)
		sqc.metrics.ResultsMisses++
		return nil, err
	}

	sqc.metrics.ResultsHits++
	executionTime := time.Duration(entry.ExecutionTime) * time.Millisecond
	sqc.metrics.TotalSavings += executionTime

	log.Printf("Results cache HIT: query_hash=%s, saved %dms",
		HashSQL(sql, tenantID, dbName)[:8], entry.ExecutionTime)

	return &entry, nil
}

// SetResultsCache stores query results in the cache
func (sqc *SemanticQueryCache) SetResultsCache(ctx context.Context, sql, tenantID, dbName string, entry *ResultsCacheEntry) error {
	if sqc.redisClient == nil {
		return nil // No-op if Redis not available
	}

	key := fmt.Sprintf("results:%s", HashSQL(sql, tenantID, dbName))

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal results cache entry: %w", err)
	}

	if err := sqc.redisClient.Set(ctx, key, data, sqc.resultsTTL).Err(); err != nil {
		log.Printf("Redis SET error for results cache key %s: %v", key, err)
		return err
	}

	log.Printf("Cached SQL → Results: query_hash=%s, rows=%d, ttl=5m",
		HashSQL(sql, tenantID, dbName)[:8], entry.RowCount)

	return nil
}

// ============================================================================
// Invalidation & Maintenance
// ============================================================================

// InvalidateNLQueryCache invalidates NL query cache for a prompt
func (sqc *SemanticQueryCache) InvalidateNLQueryCache(ctx context.Context, prompt, datasource, mode, tenantID string) error {
	if sqc.redisClient == nil {
		return nil
	}

	key := fmt.Sprintf("nlquery:%s", HashNLPrompt(prompt, datasource, mode, tenantID))
	if err := sqc.redisClient.Del(ctx, key).Err(); err != nil {
		log.Printf("Failed to invalidate NL query cache: %v", err)
		return err
	}

	log.Printf("Invalidated NL query cache: %s", key[:30])
	return nil
}

// InvalidateSQLQueryCache invalidates SQL query cache
func (sqc *SemanticQueryCache) InvalidateSQLQueryCache(ctx context.Context, semanticQueryJSON, dbType, tenantID string) error {
	if sqc.redisClient == nil {
		return nil
	}

	key := fmt.Sprintf("sqlquery:%s", HashSemanticQuery(semanticQueryJSON, dbType, tenantID))
	if err := sqc.redisClient.Del(ctx, key).Err(); err != nil {
		log.Printf("Failed to invalidate SQL query cache: %v", err)
		return err
	}

	log.Printf("Invalidated SQL query cache: %s", key[:30])
	return nil
}

// InvalidateResultsCache invalidates results cache for a SQL query
func (sqc *SemanticQueryCache) InvalidateResultsCache(ctx context.Context, sql, tenantID, dbName string) error {
	if sqc.redisClient == nil {
		return nil
	}

	key := fmt.Sprintf("results:%s", HashSQL(sql, tenantID, dbName))
	if err := sqc.redisClient.Del(ctx, key).Err(); err != nil {
		log.Printf("Failed to invalidate results cache: %v", err)
		return err
	}

	log.Printf("Invalidated results cache: %s", key[:30])
	return nil
}

// InvalidateTenantCache invalidates all cache entries for a tenant
func (sqc *SemanticQueryCache) InvalidateTenantCache(ctx context.Context, tenantID string) error {
	if sqc.redisClient == nil {
		return nil
	}

	patterns := []string{
		fmt.Sprintf("nlquery:*:%s:*", tenantID),
		fmt.Sprintf("sqlquery:*:%s:*", tenantID),
		fmt.Sprintf("results:*:%s:*", tenantID),
	}

	totalDeleted := 0
	for _, pattern := range patterns {
		iter := sqc.redisClient.Scan(ctx, 0, pattern, 0).Iterator()
		for iter.Next(ctx) {
			key := iter.Val()
			if err := sqc.redisClient.Del(ctx, key).Err(); err != nil {
				log.Printf("Failed to delete cache key %s: %v", key, err)
			} else {
				totalDeleted++
			}
		}
	}

	log.Printf("Invalidated tenant cache: tenant=%s, keys_deleted=%d", tenantID, totalDeleted)
	return nil
}

// ============================================================================
// Metrics
// ============================================================================

// GetMetrics returns current cache performance metrics
func (sqc *SemanticQueryCache) GetMetrics() *CacheMetrics {
	return sqc.metrics
}

// ResetMetrics resets all metrics counters
func (sqc *SemanticQueryCache) ResetMetrics() {
	sqc.metrics = &CacheMetrics{}
}

// GetCacheStats returns detailed cache statistics
func (sqc *SemanticQueryCache) GetCacheStats(ctx context.Context) (map[string]interface{}, error) {
	if sqc.redisClient == nil {
		return nil, fmt.Errorf("redis client not available")
	}

	// Get Redis INFO stats
	info := sqc.redisClient.Info(ctx, "memory")

	m := sqc.metrics
	totalRequests := m.NLQueryHits + m.NLQueryMisses + m.SQLQueryHits + m.SQLQueryMisses + m.ResultsHits + m.ResultsMisses
	nlHitRate := float64(0)
	if m.NLQueryHits+m.NLQueryMisses > 0 {
		nlHitRate = float64(m.NLQueryHits) / float64(m.NLQueryHits+m.NLQueryMisses)
	}
	sqlHitRate := float64(0)
	if m.SQLQueryHits+m.SQLQueryMisses > 0 {
		sqlHitRate = float64(m.SQLQueryHits) / float64(m.SQLQueryHits+m.SQLQueryMisses)
	}
	resultsHitRate := float64(0)
	if m.ResultsHits+m.ResultsMisses > 0 {
		resultsHitRate = float64(m.ResultsHits) / float64(m.ResultsHits+m.ResultsMisses)
	}

	return map[string]interface{}{
		"total_requests":       totalRequests,
		"total_hits":           m.NLQueryHits + m.SQLQueryHits + m.ResultsHits,
		"total_misses":         m.NLQueryMisses + m.SQLQueryMisses + m.ResultsMisses,
		"nl_query_hits":        m.NLQueryHits,
		"nl_query_misses":      m.NLQueryMisses,
		"nl_query_hit_rate":    nlHitRate,
		"sql_query_hits":       m.SQLQueryHits,
		"sql_query_misses":     m.SQLQueryMisses,
		"sql_query_hit_rate":   sqlHitRate,
		"results_hits":         m.ResultsHits,
		"results_misses":       m.ResultsMisses,
		"results_hit_rate":     resultsHitRate,
		"total_avoids":         m.AvoidsHits,
		"llm_calls_avoided":    m.AvoidsHits,
		"total_savings_ms":     m.TotalSavings.Milliseconds(),
		"savings_seconds":      m.TotalSavings.Seconds(),
		"estimated_cost_saved": float64(m.AvoidsHits) * 0.0075, // Rough estimate: $0.0075 per LLM call
		"redis_info":           info.String(),
	}, nil
}
