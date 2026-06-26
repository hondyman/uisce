import React, { createContext, useContext, useRef, useState, useEffect, useCallback, ReactNode } from 'react';
import { devWarn } from '../utils/devLogger';

/**
 * MetadataProvider - Client-side deduplication for metadata fetches
 * 
 * This solves the "N+1" problem in metadata-driven architectures where
 * multiple components request the same layout definition simultaneously.
 * 
 * Pattern:
 * - 100 ObjectViewer components mount
 * - Each calls useMetadata('asset_summary')
 * - Without deduplication: 100 network requests
 * - With deduplication: 1 network request, 100 components share the promise
 * 
 * This reduces complexity from O(N) to O(1) for layout fetches.
 */

// Type definitions
export interface LayoutDefinition {
  id: string;
  name: string;
  layout: Record<string, unknown>;
  created_at?: string;
  updated_at?: string;
}

export interface SchemaDefinition {
  id: string;
  slug: string;
  fields: Array<{
    name: string;
    type: string;
    label?: string;
    required?: boolean;
    [key: string]: unknown;
  }>;
  [key: string]: unknown;
}

interface MetadataCache {
  layouts: Record<string, LayoutDefinition>;
  schemas: Record<string, SchemaDefinition>;
}

interface InflightRequests {
  layouts: Record<string, Promise<LayoutDefinition>>;
  schemas: Record<string, Promise<SchemaDefinition>>;
}

interface MetadataContextType {
  // Layout fetching with deduplication
  fetchLayout: (layoutKey: string) => Promise<LayoutDefinition | null>;
  getLayout: (layoutKey: string) => LayoutDefinition | null;
  
  // Schema fetching with deduplication
  fetchSchema: (schemaKey: string) => Promise<SchemaDefinition | null>;
  getSchema: (schemaKey: string) => SchemaDefinition | null;
  
  // Cache management
  invalidateLayout: (layoutKey: string) => void;
  invalidateSchema: (schemaKey: string) => void;
  invalidateAll: () => void;
  
  // Preloading for optimistic UI
  preloadLayouts: (layoutKeys: string[]) => void;
  preloadSchemas: (schemaKeys: string[]) => void;
}

const MetadataContext = createContext<MetadataContextType | null>(null);

interface MetadataProviderProps {
  children: ReactNode;
  /** Base URL for metadata API endpoints */
  baseUrl?: string;
}

/**
 * Gets tenant headers from localStorage for API requests.
 * This mirrors the pattern used in layoutsApi.ts and setupTenantFetch.ts
 */
function getTenantHeaders(): Headers {
  const headers = new Headers();
  headers.set('Content-Type', 'application/json');
  
  try {
    const tenant = localStorage.getItem('selected_tenant');
    const datasource = localStorage.getItem('selected_datasource');
    
    if (tenant) {
      const parsed = JSON.parse(tenant);
      if (parsed?.id) {
        headers.set('X-Tenant-ID', parsed.id);
      }
    }
    
    if (datasource) {
      const parsed = JSON.parse(datasource);
      if (parsed?.id) {
        headers.set('X-Tenant-Datasource-ID', parsed.id);
      }
    }
  } catch (e) {
    devWarn('[MetadataProvider] Failed to parse tenant headers from localStorage:', e);
  }
  
  return headers;
}

/**
 * Builds the query string for tenant-scoped requests
 */
function _getTenantQueryParams(): string {
  try {
    const tenant = localStorage.getItem('selected_tenant');
    const datasource = localStorage.getItem('selected_datasource');
    
    const tenantId = tenant ? JSON.parse(tenant)?.id : '';
    const datasourceId = datasource ? JSON.parse(datasource)?.id : '';
    
    if (tenantId && datasourceId) {
      return `?tenant_id=${encodeURIComponent(tenantId)}&tenant_instance_id=${encodeURIComponent(datasourceId)}`;
    }
  } catch (e) {
    // Silently fail - headers will handle auth
  }
  return '';
}

export const MetadataProvider: React.FC<MetadataProviderProps> = ({ 
  children,
  baseUrl = ''
}) => {
  // Store the cached data
  const cache = useRef<MetadataCache>({
    layouts: {},
    schemas: {},
  });
  
  // Store pending requests (Promise Singleton pattern)
  // This is the key to deduplication - multiple callers get the same promise
  const inflight = useRef<InflightRequests>({
    layouts: {},
    schemas: {},
  });
  
  // Force re-render trigger for consumers using getLayout/getSchema synchronously
  const [, setVersion] = useState(0);
  const incrementVersion = useCallback(() => setVersion(v => v + 1), []);

  /**
   * Fetch a layout definition with request deduplication.
   * 
   * If 100 components call this with the same key in the same tick:
   * - First call creates the fetch promise and stores it in inflight
   * - Remaining 99 calls return the same promise
   * - When resolved, all 100 components receive the same data
   */
  const fetchLayout = useCallback(async (layoutKey: string): Promise<LayoutDefinition | null> => {
    // 1. Return immediately if data exists in cache
    if (cache.current.layouts[layoutKey]) {
      return cache.current.layouts[layoutKey];
    }

    // 2. Deduplication: Return existing promise if request is already in flight
    if (layoutKey in inflight.current.layouts) {
      return inflight.current.layouts[layoutKey];
    }

    // 3. Create new fetch request
    const fetchPromise = (async () => {
      try {
        const headers = getTenantHeaders();
        const response = await fetch(`${baseUrl}/api/layouts/${layoutKey}`, { headers });
        
        if (!response.ok) {
          if (response.status === 404) {
            devWarn(`[MetadataProvider] Layout not found: ${layoutKey}`);
            return null;
          }
          throw new Error(`Failed to fetch layout: ${response.status}`);
        }
        
        const data = await response.json();
        
        // Store in cache
        cache.current.layouts[layoutKey] = data;
        
        // Trigger re-render for sync consumers
        incrementVersion();
        
        return data;
      } catch (error) {
        console.error(`[MetadataProvider] Error fetching layout ${layoutKey}:`, error);
        return null;
      } finally {
        // Cleanup: Remove from inflight after resolution
        delete inflight.current.layouts[layoutKey];
      }
    })();

    // Store the promise for deduplication
    inflight.current.layouts[layoutKey] = fetchPromise;
    
    return fetchPromise;
  }, [baseUrl, incrementVersion]);

  /**
   * Synchronously get a layout from cache (returns null if not cached)
   * Useful for optimistic rendering where you want to show cached data immediately
   */
  const getLayout = useCallback((layoutKey: string): LayoutDefinition | null => {
    return cache.current.layouts[layoutKey] || null;
  }, []);

  /**
   * Fetch a schema definition with request deduplication
   */
  const fetchSchema = useCallback(async (schemaKey: string): Promise<SchemaDefinition | null> => {
    if (cache.current.schemas[schemaKey]) {
      return cache.current.schemas[schemaKey];
    }

    if (schemaKey in inflight.current.schemas) {
      return inflight.current.schemas[schemaKey];
    }

    const fetchPromise = (async () => {
      try {
        const headers = getTenantHeaders();
        const response = await fetch(`${baseUrl}/api/schemas/${schemaKey}`, { headers });
        
        if (!response.ok) {
          if (response.status === 404) {
            devWarn(`[MetadataProvider] Schema not found: ${schemaKey}`);
            return null;
          }
          throw new Error(`Failed to fetch schema: ${response.status}`);
        }
        
        const data = await response.json();
        cache.current.schemas[schemaKey] = data;
        incrementVersion();
        return data;
      } catch (error) {
        console.error(`[MetadataProvider] Error fetching schema ${schemaKey}:`, error);
        return null;
      } finally {
        delete inflight.current.schemas[schemaKey];
      }
    })();

    inflight.current.schemas[schemaKey] = fetchPromise;
    return fetchPromise;
  }, [baseUrl, incrementVersion]);

  const getSchema = useCallback((schemaKey: string): SchemaDefinition | null => {
    return cache.current.schemas[schemaKey] || null;
  }, []);

  /**
   * Invalidate a specific layout (force re-fetch on next request)
   */
  const invalidateLayout = useCallback((layoutKey: string) => {
    delete cache.current.layouts[layoutKey];
    incrementVersion();
  }, [incrementVersion]);

  /**
   * Invalidate a specific schema
   */
  const invalidateSchema = useCallback((schemaKey: string) => {
    delete cache.current.schemas[schemaKey];
    incrementVersion();
  }, [incrementVersion]);

  /**
   * Invalidate all cached metadata
   * Call this when tenant context changes
   */
  const invalidateAll = useCallback(() => {
    cache.current = { layouts: {}, schemas: {} };
    inflight.current = { layouts: {}, schemas: {} };
    incrementVersion();
  }, [incrementVersion]);

  /**
   * Preload layouts in parallel for optimistic UI
   * Call this when you know what layouts a page will need
   */
  const preloadLayouts = useCallback((layoutKeys: string[]) => {
    layoutKeys.forEach(key => {
      if (!cache.current.layouts[key] && !inflight.current.layouts[key]) {
        fetchLayout(key); // Fire and forget - results will be cached
      }
    });
  }, [fetchLayout]);

  /**
   * Preload schemas in parallel
   */
  const preloadSchemas = useCallback((schemaKeys: string[]) => {
    schemaKeys.forEach(key => {
      if (!cache.current.schemas[key] && !inflight.current.schemas[key]) {
        fetchSchema(key);
      }
    });
  }, [fetchSchema]);

  const value: MetadataContextType = {
    fetchLayout,
    getLayout,
    fetchSchema,
    getSchema,
    invalidateLayout,
    invalidateSchema,
    invalidateAll,
    preloadLayouts,
    preloadSchemas,
  };

  return (
    <MetadataContext.Provider value={value}>
      {children}
    </MetadataContext.Provider>
  );
};

/**
 * Hook to access the metadata context
 */
export const useMetadataContext = (): MetadataContextType => {
  const context = useContext(MetadataContext);
  if (!context) {
    throw new Error('useMetadataContext must be used within a MetadataProvider');
  }
  return context;
};

/**
 * Hook to fetch and subscribe to a layout definition
 * 
 * Usage:
 * ```tsx
 * const { layout, loading, error } = useLayout('asset_summary');
 * if (loading) return <Skeleton />;
 * if (error) return <ErrorBoundary />;
 * return <DynamicForm layout={layout} />;
 * ```
 */
export const useLayout = (layoutKey: string | null) => {
  const { fetchLayout, getLayout } = useMetadataContext();
  const [layout, setLayout] = useState<LayoutDefinition | null>(
    layoutKey ? getLayout(layoutKey) : null
  );
  const [loading, setLoading] = useState(layoutKey ? !getLayout(layoutKey) : false);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    if (!layoutKey) {
      setLayout(null);
      setLoading(false);
      return;
    }

    let mounted = true;
    
    // Check cache first for instant render
    const cached = getLayout(layoutKey);
    if (cached) {
      setLayout(cached);
      setLoading(false);
      return;
    }

    setLoading(true);
    fetchLayout(layoutKey)
      .then((data) => {
        if (mounted) {
          setLayout(data);
          setLoading(false);
        }
      })
      .catch((err) => {
        if (mounted) {
          setError(err);
          setLoading(false);
        }
      });

    return () => {
      mounted = false;
    };
  }, [layoutKey, fetchLayout, getLayout]);

  return { layout, loading, error };
};

/**
 * Hook to fetch and subscribe to a schema definition
 */
export const useSchema = (schemaKey: string | null) => {
  const { fetchSchema, getSchema } = useMetadataContext();
  const [schema, setSchema] = useState<SchemaDefinition | null>(
    schemaKey ? getSchema(schemaKey) : null
  );
  const [loading, setLoading] = useState(schemaKey ? !getSchema(schemaKey) : false);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    if (!schemaKey) {
      setSchema(null);
      setLoading(false);
      return;
    }

    let mounted = true;
    
    const cached = getSchema(schemaKey);
    if (cached) {
      setSchema(cached);
      setLoading(false);
      return;
    }

    setLoading(true);
    fetchSchema(schemaKey)
      .then((data) => {
        if (mounted) {
          setSchema(data);
          setLoading(false);
        }
      })
      .catch((err) => {
        if (mounted) {
          setError(err);
          setLoading(false);
        }
      });

    return () => {
      mounted = false;
    };
  }, [schemaKey, fetchSchema, getSchema]);

  return { schema, loading, error };
};

export default MetadataProvider;
