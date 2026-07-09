import { useEffect, useState, useMemo } from 'react';
import { apiFetch } from '../lib/apiClient';

export interface RegionOption {
  /** Stable code (e.g. "us-east-1") used for the API and storage. */
  value: string;
  /** Display name (e.g. "US East"). */
  label: string;
  /** True when sourced from the global `regions` lookup table. */
  fromLookup: boolean;
}

/**
 * Hardcoded regions used when the global `regions` lookup is missing or
 * returns no rows.  Values are the AWS-style codes; labels are the
 * human-readable names shown in the UI.
 */
export const DEFAULT_REGIONS: RegionOption[] = [
  { value: 'us-east-1', label: 'US East', fromLookup: false },
  { value: 'us-west-2', label: 'US West', fromLookup: false },
  { value: 'eu-central-1', label: 'EUR Central', fromLookup: false },
  { value: 'ap-south-1', label: 'APAC South', fromLookup: false },
];

/** The system workspace tenant id (gold copy). Used to read the global
 *  lookups since regions are platform-wide, not tenant-scoped. */
const SYSTEM_TENANT_ID = '00000000-0000-0000-0000-000000000000';
const REGIONS_LOOKUP_ID = 'regions';

/**
 * Resolves the list of available regions for the platform.
 *
 * 1. Tries to read values from the global `regions` lookup table via
 *    `/api/lookups/regions/values?tenant_id=<system>`.
 * 2. If the lookup is missing, returns no rows, or the request fails,
 *    falls back to {@link DEFAULT_REGIONS}.
 *
 * The lookup is fetched once per mount; results are memoized by lookup id.
 */
export function useRegions(): { regions: RegionOption[]; loading: boolean } {
  const [lookupRegions, setLookupRegions] = useState<RegionOption[] | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let mounted = true;
    setLoading(true);

    (async () => {
      try {
        const response = await apiFetch(
          `/api/lookups/${REGIONS_LOOKUP_ID}/values?tenant_id=${SYSTEM_TENANT_ID}`,
          { credentials: 'include' },
        );
        if (!response.ok) {
          if (mounted) setLookupRegions(null);
          return;
        }
        const raw = await response.json();
        const items = Array.isArray(raw?.items) ? raw.items : Array.isArray(raw) ? raw : [];
        const mapped: RegionOption[] = items
          .map((row: any) => {
            const value = String(row?.value ?? row?.id ?? '').trim();
            const label = String(row?.label ?? row?.name ?? value).trim();
            if (!value || !label) return null;
            return { value, label, fromLookup: true };
          })
          .filter((r: RegionOption | null): r is RegionOption => r !== null);

        if (mounted) {
          setLookupRegions(mapped.length > 0 ? mapped : null);
        }
      } catch {
        if (mounted) setLookupRegions(null);
      } finally {
        if (mounted) setLoading(false);
      }
    })();

    return () => {
      mounted = false;
    };
  }, []);

  const regions = useMemo<RegionOption[]>(() => {
    if (lookupRegions && lookupRegions.length > 0) {
      return lookupRegions;
    }
    return DEFAULT_REGIONS;
  }, [lookupRegions]);

  return { regions, loading };
}
