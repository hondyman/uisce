import { DynamicQueryResponse, QueryCacheEntry, DynamicQueryRequest } from '../types/dynamic';

export class QueryCacheService {
  private static instance: QueryCacheService;
  private cache: Map<string, QueryCacheEntry> = new Map();
  private maxCacheSize: number = 1000;
  private defaultTTL: number = 5 * 60 * 1000; // 5 minutes

  static getInstance(): QueryCacheService {
    if (!QueryCacheService.instance) {
      QueryCacheService.instance = new QueryCacheService();
    }
    return QueryCacheService.instance;
  }

  /**
   * Get cached result for a query
   */
  get(queryKey: string): DynamicQueryResponse | null {
    const entry = this.cache.get(queryKey);

    if (!entry) {
      return null;
    }

    // Check if entry has expired
    if (Date.now() > entry.timestamp + entry.ttl) {
      this.cache.delete(queryKey);
      return null;
    }

    // Update hits for LFU
    entry.hits++;

    return entry.result;
  }

  /**
   * Store result in cache
   */
  set(queryKey: string, result: DynamicQueryResponse, ttl?: number): void {
    // Implement size-based eviction if cache is full
    if (this.cache.size >= this.maxCacheSize) {
      this.evictOldest();
    }

    const entry: QueryCacheEntry = {
      key: queryKey,
      query: {} as DynamicQueryRequest, // Placeholder until we have the actual query payload
      result,
      timestamp: Date.now(),
      ttl: ttl || this.defaultTTL,
      hits: 0
    };

    this.cache.set(queryKey, entry);
  }

  /**
   * Invalidate cache entries matching a pattern
   */
  invalidate(pattern: string): void {
    const keysToDelete: string[] = [];

    for (const [key] of this.cache) {
      if (this.matchesPattern(key, pattern)) {
        keysToDelete.push(key);
      }
    }

    keysToDelete.forEach(key => this.cache.delete(key));
  }

  /**
   * Clear all cache entries
   */
  clear(): void {
    this.cache.clear();
  }

  /**
   * Get cache statistics
   */
  getStats(): {
    totalEntries: number;
    totalSize: number;
    hitRate: number;
    averageAccessCount: number;
  } {
    const entries = Array.from(this.cache.values());
    const totalEntries = entries.length;
    const totalSize = entries.reduce((sum, entry) => sum + this.calculateResultSize(entry.result), 0);
    const totalAccessCount = entries.reduce((sum, entry) => sum + entry.hits, 0);
    const averageAccessCount = totalEntries > 0 ? totalAccessCount / totalEntries : 0;

    // Calculate hit rate (this is a simple approximation)
    const recentEntries = entries.filter(entry => Date.now() - entry.timestamp < 60 * 60 * 1000); // Last hour
    const hitRate = recentEntries.length > 0 ?
      recentEntries.reduce((sum, entry) => sum + (entry.hits > 1 ? 1 : 0), 0) / recentEntries.length : 0;

    return {
      totalEntries,
      totalSize,
      hitRate,
      averageAccessCount
    };
  }

  /**
   * Generate cache key for a query
   */
  generateCacheKey(
    metricId: string,
    parameters: Record<string, any>,
    filters?: Record<string, any>
  ): string {
    const sortedParams = Object.keys(parameters)
      .sort()
      .map(key => `${key}:${JSON.stringify(parameters[key])}`)
      .join('|');

    const sortedFilters = filters ?
      Object.keys(filters)
        .sort()
        .map(key => `${key}:${JSON.stringify(filters[key])}`)
        .join('|') : '';

    return `${metricId}|${sortedParams}|${sortedFilters}`;
  }

  /**
   * Pre-warm cache with frequently accessed queries
   */
  async prewarmCache(
    queries: Array<{
      metricId: string;
      parameters: Record<string, any>;
      filters?: Record<string, any>;
    }>,
    fetchFunction: (metricId: string, params: Record<string, any>, filters?: Record<string, any>) => Promise<DynamicQueryResponse>
  ): Promise<void> {
    const promises = queries.map(async ({ metricId, parameters, filters }) => {
      const cacheKey = this.generateCacheKey(metricId, parameters, filters);

      // Only pre-warm if not already cached
      if (!this.get(cacheKey)) {
        try {
          const result = await fetchFunction(metricId, parameters, filters);
          this.set(cacheKey, result, this.defaultTTL * 2); // Longer TTL for pre-warmed entries
        } catch (error) {
          // Failed to pre-warm cache - non-critical
        }
      }
    });

    await Promise.allSettled(promises);
  }

  /**
   * Set cache configuration
   */
  setConfig(maxSize: number, defaultTTL: number): void {
    this.maxCacheSize = maxSize;
    this.defaultTTL = defaultTTL;

    // Evict entries if new max size is smaller
    while (this.cache.size > this.maxCacheSize) {
      this.evictOldest();
    }
  }

  /**
   * Check if cache key matches a pattern
   */
  private matchesPattern(key: string, pattern: string): boolean {
    // Simple wildcard matching
    const regex = new RegExp(pattern.replace(/\*/g, '.*').replace(/\?/g, '.'));
    return regex.test(key);
  }

  /**
   * Evict oldest entries (FIFO)
   */
  private evictOldest(): void {
    if (this.cache.size === 0) return;

    let oldestKey: string | null = null;
    let oldestTime = Date.now();

    for (const [key, entry] of this.cache) {
      if (entry.timestamp < oldestTime) {
        oldestTime = entry.timestamp;
        oldestKey = key;
      }
    }

    if (oldestKey) {
      this.cache.delete(oldestKey);
    }
  }

  /**
   * Calculate approximate size of result in bytes
   */
  private calculateResultSize(result: DynamicQueryResponse): number {
    // Rough estimation based on JSON string length
    return JSON.stringify(result).length * 2; // UTF-16 characters
  }
}

export class ResultMemoizationService {
  private static instance: ResultMemoizationService;
  private memoizedResults: Map<string, {
    result: any;
    timestamp: number;
    ttl: number;
    dependencies: string[];
  }> = new Map();

  static getInstance(): ResultMemoizationService {
    if (!ResultMemoizationService.instance) {
      ResultMemoizationService.instance = new ResultMemoizationService();
    }
    return ResultMemoizationService.instance;
  }

  /**
   * Memoize a computation result
   */
  memoize<T>(
    key: string,
    computation: () => T,
    ttl: number = 5 * 60 * 1000, // 5 minutes
    dependencies: string[] = []
  ): T {
    const existing = this.memoizedResults.get(key);

    if (existing && Date.now() < existing.timestamp + existing.ttl) {
      // Check if dependencies have changed
      if (this.checkDependencies(dependencies, existing.dependencies)) {
        return existing.result;
      }
    }

    // Compute new result
    const result = computation();
    this.memoizedResults.set(key, {
      result,
      timestamp: Date.now(),
      ttl,
      dependencies: [...dependencies]
    });

    return result;
  }

  /**
   * Invalidate memoized results based on dependencies
   */
  invalidateByDependency(dependency: string): void {
    const keysToDelete: string[] = [];

    for (const [key, entry] of this.memoizedResults) {
      if (entry.dependencies.includes(dependency)) {
        keysToDelete.push(key);
      }
    }

    keysToDelete.forEach(key => this.memoizedResults.delete(key));
  }

  /**
   * Clear all memoized results
   */
  clear(): void {
    this.memoizedResults.clear();
  }

  /**
   * Get memoization statistics
   */
  getStats(): {
    totalEntries: number;
    hitRate: number;
    averageAge: number;
  } {
    const entries = Array.from(this.memoizedResults.values());
    const totalEntries = entries.length;

    if (totalEntries === 0) {
      return { totalEntries: 0, hitRate: 0, averageAge: 0 };
    }

    const now = Date.now();
    const averageAge = entries.reduce((sum, entry) => sum + (now - entry.timestamp), 0) / totalEntries;

    // This is a simplified hit rate calculation
    const recentEntries = entries.filter(entry => now - entry.timestamp < 60 * 60 * 1000); // Last hour
    const hitRate = recentEntries.length / totalEntries;

    return {
      totalEntries,
      hitRate,
      averageAge
    };
  }

  /**
   * Check if dependencies have changed
   */
  private checkDependencies(newDeps: string[], oldDeps: string[]): boolean {
    if (newDeps.length !== oldDeps.length) return false;

    // Simple comparison - in a real implementation, you'd want more sophisticated dependency tracking
    return newDeps.every(dep => oldDeps.includes(dep));
  }
}

export const queryCache = QueryCacheService.getInstance();
export const resultMemo = ResultMemoizationService.getInstance();
