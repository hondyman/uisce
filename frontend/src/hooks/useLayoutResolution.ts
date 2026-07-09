import { useEffect, useState } from 'react';
import { resolveLayoutApi } from '@/api/layoutResolver';
import type { PageResolution } from '@/types/mutability';

export interface UseLayoutResolutionOptions {
  pageKey: string;
  bindingId?: string;
  tenantId?: string;
  /** Skip the fetch entirely when false. Default: true. */
  enabled?: boolean;
}

export interface UseLayoutResolutionResult {
  data: PageResolution | null;
  loading: boolean;
  error: Error | null;
  refresh: () => Promise<void>;
}

/**
 * React hook wrapping GET /api/v1/layout/resolve. Returns the resolved
 * hydration payload + loading / error state.
 *
 * Cardinal Rule 1.3: pageKey is required; tenantId falls back to the
 * X-Tenant-ID header that apiClient injects when omitted.
 */
export function useLayoutResolution(
  opts: UseLayoutResolutionOptions,
): UseLayoutResolutionResult {
  const { pageKey, bindingId, tenantId, enabled = true } = opts;
  const [data, setData] = useState<PageResolution | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const fetchOnce = async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await resolveLayoutApi(pageKey, { bindingId, tenantId });
      setData(result);
    } catch (e) {
      setError(e instanceof Error ? e : new Error(String(e)));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (!enabled || !pageKey) return;
    void fetchOnce();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pageKey, bindingId, tenantId, enabled]);

  return { data, loading, error, refresh: fetchOnce };
}