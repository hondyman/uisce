import { QueryClient } from '@tanstack/react-query';

/**
 * React Query Configuration for Risk & Compliance Console
 * 
 * This configures:
 * - Cache durations (stale times)
 * - Garbage collection
 * - Query retry logic
 * - Default error handling
 */

export const createQueryClient = () => {
  return new QueryClient({
    defaultOptions: {
      queries: {
        /**
         * Cache Durations
         * - Dashboard KPIs: 5 minutes (relatively static, update with ETL runs)
         * - Sparklines: 1 minute (frequent updates)
         * - Data tables: 2 minutes (user might paginate/sort)
         * - Detail views: 5 minutes (rarely changes during session)
         */
        staleTime: 1000 * 60 * 5, // 5 minutes default
        gcTime: 1000 * 60 * 10, // 10 minutes (formerly cacheTime)

        /**
         * Retry Configuration
         * - Retry up to 2 times for network failures
         * - Don't retry on 4xx errors (client error, won't help)
         * - Exponential backoff: 1s, 2s
         */
        retry: (failureCount, error: any) => {
          // Don't retry on 4xx errors
          if (error?.status >= 400 && error?.status < 500) {
            return false;
          }
          // Retry up to 2 times on network/5xx errors
          return failureCount < 2;
        },
        retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 30000),

        /**
         * Refetch behavior
         * - Don't refetch on window regain (user might be AFK)
         * - Don't refetch on reconnect (manual refresh available)
         */
        refetchOnWindowFocus: false,
        refetchOnReconnect: false,
        refetchOnMount: false,
      },

      mutations: {
        /**
         * Mutation defaults
         * - Retry once on network failure
         * - Don't retry on validation errors
         */
        retry: (failureCount, error: any) => {
          if (error?.status >= 400 && error?.status < 500) {
            return false;
          }
          return failureCount < 1;
        },
      },
    },
  });
};

/**
 * Query Key Factory
 * 
 * Centralized query key generation to prevent typos and ensure consistency.
 * See: https://tkdodo.eu/blog/effective-react-query-keys
 */

export const queryKeys = {
  // Dashboard domain
  dashboard: {
    all: ['dashboard'],
    compliance: (tenantId: string, valuationDate: string) => [
      ...queryKeys.dashboard.all,
      'compliance',
      tenantId,
      valuationDate,
    ],
    risk: (tenantId: string, valuationDate: string) => [
      ...queryKeys.dashboard.all,
      'risk',
      tenantId,
      valuationDate,
    ],
    sparklines: (tenantId: string) => [
      ...queryKeys.dashboard.all,
      'sparklines',
      tenantId,
    ],
    etlHealth: (tenantId: string) => [
      ...queryKeys.dashboard.all,
      'etl-health',
      tenantId,
    ],
    alerts: (tenantId: string, valuationDate: string) => [
      ...queryKeys.dashboard.all,
      'alerts',
      tenantId,
      valuationDate,
    ],
  },

  // ETL domain
  etl: {
    all: ['etl-runs'],
    list: (filters?: { tenant_id?: string; status?: string; limit?: number }) => [
      ...queryKeys.etl.all,
      'list',
      filters,
    ],
    detail: (id: string) => [...queryKeys.etl.all, 'detail', id],
  },

  // WASM domain
  wasm: {
    all: ['wasm-versions'],
    list: (moduleName: string) => [...queryKeys.wasm.all, moduleName],
    detail: (id: string) => [...queryKeys.wasm.all, 'detail', id],
  },

  // Rule Lineage domain
  ruleLineage: {
    all: ['rule-lineage'],
    detail: (ruleId: string, filters?: { date_from?: string; date_to?: string; portfolio_id?: string }) => [
      ...queryKeys.ruleLineage.all,
      ruleId,
      filters,
    ],
  },

  // Scenario Lineage domain
  scenarioLineage: {
    all: ['scenario-lineage'],
    detail: (scenarioId: string, filters?: { date_from?: string; date_to?: string; portfolio_id?: string }) => [
      ...queryKeys.scenarioLineage.all,
      scenarioId,
      filters,
    ],
  },
};

/**
 * Query Invalidation Patterns
 * 
 * After mutations, invalidate related queries:
 * - Activate WASM version → invalidate wasm list
 * - ETL run completes → invalidate dashboard, etl list, sparklines
 * - Rule breach updated → invalidate dashboard alerts, compliance summary
 * 
 * Example in mutation:
 * 
 *   const mutation = useMutation({
 *     mutationFn: async (id) => {
 *       const res = await fetch(`/api/wasm-versions/${id}/activate`, { method: 'POST' });
 *       return res.json();
 *     },
 *     onSuccess: () => {
 *       queryClient.invalidateQueries({ queryKey: queryKeys.wasm.all });
 *     },
 *   });
 */

/**
 * Cache Strategy
 * 
 * DASHBOARD (5 min stale):
 * - ComplianceSummary: Query key includes tenantId + valuationDate
 * - RiskSummary: Query key includes tenantId + valuationDate
 * - Sparklines: Query key includes tenantId only (7-day window)
 * - ETLHealth: Query key includes tenantId only
 * - Alerts: Query key includes tenantId + valuationDate
 * 
 * TABLES (2 min stale):
 * - ETLRuns: Paginated, includes status filter
 * - WASMVersions: By module name
 * - RuleLineage: By rule ID, date range optional
 * - ScenarioLineage: By scenario ID, date range optional
 * 
 * INVALIDATION:
 * - When tenant changes: invalidate dashboard.all
 * - When valuation date changes: invalidate dashboard.all
 * - After ETL completes: invalidate etl.all, dashboard.all
 * - After WASM activation: invalidate wasm.all
 */
