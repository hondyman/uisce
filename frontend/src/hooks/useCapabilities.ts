import { useState, useEffect, useRef } from 'react';
import { apiFetch } from '../lib/apiClient';
import { useAccess } from '../contexts/AccessContext';

const CACHE_TTL_MS = 5 * 60 * 1000; // 5 minutes

interface CacheEntry {
  data: Record<string, boolean>;
  expiresAt: number;
}

const capabilitiesCache = new Map<string, CacheEntry>();

/**
 * Fetches and caches the effective ABAC capability map for the current
 * user+tenant from GET /api/capabilities.
 *
 * Returns undefined while loading or when the user is a platform operator
 * (operators bypass all capability gates in filterNavigationByCapabilities).
 */
export function useCapabilities(): Record<string, boolean> | undefined {
  const { scope, isPlatformOperator } = useAccess();
  const tenantId = scope.tenantId ?? 'global';
  const [caps, setCaps] = useState<Record<string, boolean> | undefined>(undefined);
  const abortRef = useRef<AbortController | null>(null);

  useEffect(() => {
    // Platform operators bypass the capability gate entirely.
    if (isPlatformOperator) {
      setCaps(undefined);
      return;
    }

    const cacheKey = tenantId;
    const cached = capabilitiesCache.get(cacheKey);
    if (cached && cached.expiresAt > Date.now()) {
      setCaps(cached.data);
      return;
    }

    abortRef.current?.abort();
    const ctrl = new AbortController();
    abortRef.current = ctrl;

    apiFetch('/api/capabilities', { signal: ctrl.signal })
      .then((r) => r.json())
      .then((data: Record<string, boolean>) => {
        capabilitiesCache.set(cacheKey, { data, expiresAt: Date.now() + CACHE_TTL_MS });
        if (!ctrl.signal.aborted) setCaps(data);
      })
      .catch((err: unknown) => {
        if ((err as Error)?.name !== 'AbortError') {
          // On error fall back to empty map — nothing capability-gated is shown.
          if (!ctrl.signal.aborted) setCaps({});
        }
      });

    return () => ctrl.abort();
  }, [tenantId, isPlatformOperator]);

  return caps;
}
