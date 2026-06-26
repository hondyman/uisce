import { useQuery, useQueryClient, UseQueryOptions } from '@tanstack/react-query';
import { useCallback } from 'react';

export interface SemanticField {
    field_name: string;
    field_type: string;
    description?: string;
    required: boolean;
}

export interface SemanticViewSchema {
    view_id: string;
    view_name: string;
    tenant_id: string;
    fields: SemanticField[];
    metadata?: Record<string, any>;
    version: number;
    published_at: string;
}

interface SemanticViewsResponse {
    views: SemanticViewSchema[];
}

// Query keys for React Query
const semanticViewKeys = {
    all: ['semanticViews'] as const,
    tenant: (tenantId: string) => [...semanticViewKeys.all, tenantId] as const,
    view: (tenantId: string, viewId: string) => [...semanticViewKeys.tenant(tenantId), viewId] as const,
    multiple: (tenantId: string, viewIds: string[]) => [...semanticViewKeys.tenant(tenantId), 'batch', viewIds.join(',')] as const,
};

/**
 * Fetch a single semantic view from the backend
 */
async function fetchSemanticView(tenantId: string, viewId: string): Promise<SemanticViewSchema> {
    const response = await fetch(`/api/semantic-views/${viewId}?tenant_id=${tenantId}`, {
        headers: {
            'X-Tenant-ID': tenantId,
        },
    });

    if (!response.ok) {
        throw new Error(`Failed to fetch semantic view: ${response.statusText}`);
    }

    return response.json();
}

/**
 * Fetch multiple semantic views in a single request
 */
async function fetchMultipleSemanticViews(tenantId: string, viewIds: string[]): Promise<SemanticViewSchema[]> {
    if (viewIds.length === 0) {
        return [];
    }

    const queryParams = new URLSearchParams({
        tenant_id: tenantId,
        view_ids: viewIds.join(','),
    });

    const response = await fetch(`/api/semantic-views?${queryParams}`, {
        headers: {
            'X-Tenant-ID': tenantId,
        },
    });

    if (!response.ok) {
        throw new Error(`Failed to fetch semantic views: ${response.statusText}`);
    }

    const data: SemanticViewsResponse = await response.json();
    return data.views;
}

/**
 * Hook to fetch a single semantic view with caching
 * 
 * Features:
 * - Automatic caching with 24-hour stale time
 * - Background refetching
 * - Deduplication of concurrent requests
 */
export function useSemanticView(
    tenantId: string,
    viewId: string,
    options?: Omit<UseQueryOptions<SemanticViewSchema, Error>, 'queryKey' | 'queryFn'>
) {
    return useQuery<SemanticViewSchema, Error>({
        queryKey: semanticViewKeys.view(tenantId, viewId),
        queryFn: () => fetchSemanticView(tenantId, viewId),
        staleTime: 24 * 60 * 60 * 1000, // 24 hours
        gcTime: 24 * 60 * 60 * 1000, // Keep in cache for 24 hours
        refetchOnWindowFocus: false, // Don't refetch on every focus
        refetchOnReconnect: true,
        retry: 2,
        ...options,
    });
}

/**
 * Hook to fetch multiple semantic views with caching
 * 
 * Optimizes network requests by batching multiple view fetches
 */
export function useSemanticViews(
    tenantId: string,
    viewIds: string[],
    options?: Omit<UseQueryOptions<SemanticViewSchema[], Error>, 'queryKey' | 'queryFn'>
) {
    return useQuery<SemanticViewSchema[], Error>({
        queryKey: semanticViewKeys.multiple(tenantId, viewIds),
        queryFn: () => fetchMultipleSemanticViews(tenantId, viewIds),
        staleTime: 24 * 60 * 60 * 1000, // 24 hours
        gcTime: 24 * 60 * 60 * 1000,
        refetchOnWindowFocus: false,
        refetchOnReconnect: true,
        retry: 2,
        enabled: viewIds.length > 0,
        ...options,
    });
}

/**
 * Hook to manage semantic view cache invalidation
 * 
 * Provides utilities to invalidate cached views when they're published or updated
 */
export function useSemanticViewCache(tenantId: string) {
    const queryClient = useQueryClient();

    const invalidateView = useCallback(
        async (viewId: string) => {
            await queryClient.invalidateQueries({
                queryKey: semanticViewKeys.view(tenantId, viewId),
            });
        },
        [queryClient, tenantId]
    );

    const invalidateAllViews = useCallback(async () => {
        await queryClient.invalidateQueries({
            queryKey: semanticViewKeys.tenant(tenantId),
        });
    }, [queryClient, tenantId]);

    const prefetchView = useCallback(
        async (viewId: string) => {
            await queryClient.prefetchQuery({
                queryKey: semanticViewKeys.view(tenantId, viewId),
                queryFn: () => fetchSemanticView(tenantId, viewId),
                staleTime: 24 * 60 * 60 * 1000,
            });
        },
        [queryClient, tenantId]
    );

    const getCachedView = useCallback(
        (viewId: string): SemanticViewSchema | undefined => {
            return queryClient.getQueryData(semanticViewKeys.view(tenantId, viewId));
        },
        [queryClient, tenantId]
    );

    const setCachedView = useCallback(
        (viewId: string, data: SemanticViewSchema) => {
            queryClient.setQueryData(semanticViewKeys.view(tenantId, viewId), data);
        },
        [queryClient, tenantId]
    );

    return {
        invalidateView,
        invalidateAllViews,
        prefetchView,
        getCachedView,
        setCachedView,
    };
}

/**
 * Hook to get cache statistics (for debugging/monitoring)
 */
export function useSemanticViewCacheStats(tenantId: string) {
    const queryClient = useQueryClient();

    const getStats = useCallback(() => {
        const queryCache = queryClient.getQueryCache();
        const allQueries = queryCache.getAll();

        const semanticViewQueries = allQueries.filter((query) => {
            const key = query.queryKey;
            return Array.isArray(key) && key[0] === 'semanticViews' && key[1] === tenantId;
        });

        return {
            total_cached_views: semanticViewQueries.length,
            fresh_views: semanticViewQueries.filter((q) => q.state.dataUpdateCount > 0 && !q.isStale()).length,
            stale_views: semanticViewQueries.filter((q) => q.isStale()).length,
            fetching_views: semanticViewQueries.filter((q) => q.state.fetchStatus === 'fetching').length,
        };
    }, [queryClient, tenantId]);

    return { getStats };
}
