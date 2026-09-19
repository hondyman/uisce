import { devError } from '../../utils/devLogger';

export interface InstrumentSearchResultItem {
  symbol: string;
  name: string;
  isin?: string;
  asset_class: string;
  exchange?: string;
  internal_id: string;
}

export interface InstrumentSearchResponse {
  results: InstrumentSearchResultItem[];
}

function getAuthHeaders(): Record<string, string> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  };
  const token = typeof localStorage !== 'undefined' ? localStorage.getItem('auth_token') : null;
  if (token && !token.includes('demo')) {
    headers['Authorization'] = `Bearer ${token}`;
  }
  const tenantId = typeof localStorage !== 'undefined' ? localStorage.getItem('tenant_id') : null;
  if (tenantId) {
    headers['X-Tenant-ID'] = tenantId;
  }
  return headers;
}

/**
 * Searches instruments via GET /api/instruments/search?q=<query>&limit=<limit>&asset_class=<assetClass>
 * Supports cancellation via AbortSignal.
 */
export async function searchInstruments(
  query: string,
  options?: {
    limit?: number;
    assetClass?: string;
    signal?: AbortSignal;
  }
): Promise<InstrumentSearchResultItem[]> {
  const q = query.trim();
  if (!q) {
    return [];
  }

  const params = new URLSearchParams();
  params.set('q', q);
  if (options?.limit) {
    params.set('limit', String(options.limit));
  }
  if (options?.assetClass) {
    params.set('asset_class', options.assetClass);
  }

  const url = `/api/instruments/search?${params.toString()}`;

  try {
    const res = await fetch(url, {
      method: 'GET',
      headers: getAuthHeaders(),
      signal: options?.signal,
    });

    if (!res.ok) {
      throw new Error(`Instrument search failed: ${res.status} ${res.statusText}`);
    }

    const data: InstrumentSearchResponse = await res.json();
    return data.results || [];
  } catch (err: unknown) {
    if (err instanceof DOMException && err.name === 'AbortError') {
      throw err;
    }
    devError('[instrumentsApi] Failed to search instruments:', err);
    throw err;
  }
}
