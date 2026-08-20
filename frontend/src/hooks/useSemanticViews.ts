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
    view: (viewId: string) => [...semanticViewKeys.all, viewId] as const,
    multiple: (viewIds: string[]) => [...semanticViewKeys.all, 'batch', viewIds.join(',')] as const,
};

/**
 * Fetch a single semantic view from the backend
 */
async function fetchSemanticView(viewId: string): Promise<SemanticViewSchema> {
    const response = await fetch(`/api/semantic-views/${viewId}`);

    if (!response.ok) {
        throw new Error(`Failed to fetch semantic view: ${response.statusText}`);
    }

    return response.json();
}

/**
 * Fetch multiple semantic views in a single request
 */
async function fetchMultipleSemanticViews(viewIds: string[]): Promise<SemanticViewSchema[]> {
    if (viewIds.length === 0) {
        return [];
    }

    const queryParams = new URLSearchParams({
        view_ids: viewIds.join(','),
    });

    const response = await fetch(`/api/semantic-views?${queryParams}`);

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
    viewId: string,
    options?: Omit<UseQueryOptions<SemanticViewSchema, Error>, 'queryKey' | 'queryFn'>
) {
    return useQuery<SemanticViewSchema, Error>({
        queryKey: semanticViewKeys.view(viewId),
        queryFn: () => fetchSemanticView(viewId),
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
    viewIds: string[],
    options?: Omit<UseQueryOptions<SemanticViewSchema[], Error>, 'queryKey' | 'queryFn'>
) {
    return useQuery<SemanticViewSchema[], Error>({
        queryKey: semanticViewKeys.multiple(viewIds),
        queryFn: () => fetchMultipleSemanticViews(viewIds),
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
export function useSemanticViewCache() {
    const queryClient = useQueryClient();

    const invalidateView = useCallback(
        async (viewId: string) => {
            await queryClient.invalidateQueries({
                queryKey: semanticViewKeys.view(viewId),
            });
        },
        [queryClient]
    );

    const invalidateAllViews = useCallback(async () => {
        await queryClient.invalidateQueries({
            queryKey: semanticViewKeys.all,
        });
    }, [queryClient]);

    const prefetchView = useCallback(
        async (viewId: string) => {
            await queryClient.prefetchQuery({
                queryKey: semanticViewKeys.view(viewId),
                queryFn: () => fetchSemanticView(viewId),
                staleTime: 24 * 60 * 60 * 1000,
            });
        },
        [queryClient]
    );

    const getCachedView = useCallback(
        (viewId: string): SemanticViewSchema | undefined => {
            return queryClient.getQueryData(semanticViewKeys.view(viewId));
        },
        [queryClient]
    );

    const setCachedView = useCallback(
        (viewId: string, data: SemanticViewSchema) => {
            queryClient.setQueryData(semanticViewKeys.view(viewId), data);
        },
        [queryClient]
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
export function useSemanticViewCacheStats() {
    const queryClient = useQueryClient();

    const getStats = useCallback(() => {
        const queryCache = queryClient.getQueryCache();
        const allQueries = queryCache.getAll();

        const semanticViewQueries = allQueries.filter((query) => {
            const key = query.queryKey;
            return Array.isArray(key) && key[0] === 'semanticViews';
        });

        return {
            total_cached_views: semanticViewQueries.length,
            fresh_views: semanticViewQueries.filter((q) => q.state.dataUpdateCount > 0 && !q.isStale()).length,
            stale_views: semanticViewQueries.filter((q) => q.isStale()).length,
            fetching_views: semanticViewQueries.filter((q) => q.state.fetchStatus === 'fetching').length,
        };
    }, [queryClient]);

    return { getStats };
}
