import { useState, useEffect, useRef } from 'react';
import { apiFetch } from '../lib/apiClient';
import { useAccess } from '../contexts/AccessContext';

const CACHE_TTL_MS = 5 * 60 * 1000; // 5 minutes

interface CacheEntry {
  data: Record<string, boolean>;
  /** Server-resolved target_profile_key, from the X-Resolved-Profile header. */
  profile?: string;
  expiresAt: number;
}

const capabilitiesCache = new Map<string, CacheEntry>();

export interface CapabilitiesResult {
  /** action_attribute -> allowed. Undefined while loading or on error. */
  capabilities: Record<string, boolean> | undefined;
  /**
   * The caller's target_profile_key as resolved server-side (IAM assignment,
   * else verified JWT role claims, else BASE_USER). Used to enforce Menu
   * Designer required_entitlement. Undefined while loading, and for platform
   * operators, who bypass the gate.
   */
  profile: string | undefined;
}

/**
 * Fetches and caches the effective ABAC capability map for the current
 * user+tenant from GET /api/capabilities.
 *
 * Returns undefined capabilities while loading or when the user is a platform
 * operator (operators bypass all capability gates in filterNavigationByCapabilities).
 */
export function useCapabilities(): CapabilitiesResult {
  const { scope, isPlatformOperator } = useAccess();
  const tenantId = scope.tenantId ?? 'global';
  const [caps, setCaps] = useState<Record<string, boolean> | undefined>(undefined);
  const [profile, setProfile] = useState<string | undefined>(undefined);
  const abortRef = useRef<AbortController | null>(null);

  useEffect(() => {
    // Platform operators bypass the capability gate entirely.
    if (isPlatformOperator) {
      setCaps(undefined);
      setProfile(undefined);
      return;
    }

    const cacheKey = tenantId;
    const cached = capabilitiesCache.get(cacheKey);
    if (cached && cached.expiresAt > Date.now()) {
      setCaps(cached.data);
      setProfile(cached.profile);
      return;
    }

    abortRef.current?.abort();
    const ctrl = new AbortController();
    abortRef.current = ctrl;

    apiFetch('/api/capabilities', { signal: ctrl.signal })
      .then((r) => {
        // Read the profile before parsing the body: both come from the same
        // server-resolved identity, so there is one source of truth.
        const resolved = r.headers.get('X-Resolved-Profile') ?? undefined;
        return r.json().then((data: Record<string, boolean>) => ({ data, resolved }));
      })
      .then(({ data, resolved }) => {
        capabilitiesCache.set(cacheKey, {
          data,
          profile: resolved,
          expiresAt: Date.now() + CACHE_TTL_MS,
        });
        if (!ctrl.signal.aborted) {
          setCaps(data);
          setProfile(resolved);
        }
      })
      .catch((err: unknown) => {
        if ((err as Error)?.name !== 'AbortError') {
          // On error fall back to empty map — nothing capability-gated is shown.
          if (!ctrl.signal.aborted) {
            setCaps({});
            setProfile(undefined);
          }
        }
      });

    return () => ctrl.abort();
  }, [tenantId, isPlatformOperator]);

  return { capabilities: caps, profile };
}
