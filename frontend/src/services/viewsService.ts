export type FetchViewsParams = {
  tenantId?: string;
  datasourceId?: string;
  pageSize?: number;
  status?: string;
  q?: string;
};

export type ViewModel = Record<string, unknown>;

export async function fetchViews(
  params: FetchViewsParams = {},
  options: { fetchFn?: typeof fetch; signal?: AbortSignal } = {}
) {
  const { tenantId, datasourceId, pageSize = 100, status, q } = params;
  const fetchFn = options.fetchFn ?? fetch;

  const query = new URLSearchParams();
  if (tenantId) query.set('tenant_id', tenantId);
  if (datasourceId) query.set('datasource_id', datasourceId);
  if (pageSize) query.set('page_size', String(pageSize));
  if (status) query.set('status', status);
  if (q) query.set('q', q);

  const url = `/api/views?${query.toString()}`;
  const reqInit: RequestInit = { signal: options.signal, cache: 'no-store' };
  const res = await fetchFn(url, reqInit);
  if (!res.ok) {
    // Let caller handle non-ok statuses
    const text = await res.text().catch(() => '');
    throw new Error(`Fetch failed: ${res.status} ${text}`);
  }
  const data = await res.json().catch(() => null) as unknown;
  if (data && typeof data === 'object') {
    const obj = data as Record<string, unknown>;
    const viewsProp = obj.views;
    if (Array.isArray(viewsProp)) return viewsProp as ViewModel[];
  }
  return [];
}
