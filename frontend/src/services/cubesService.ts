export type FetchCubesParams = {
  tenantId?: string;
  datasourceId?: string;
  pageSize?: number;
  q?: string;
  status?: string; // optional, backend may interpret
};
export type CubeModel = Record<string, unknown>;

export async function fetchCubes(
  params: FetchCubesParams = {},
  options: { fetchFn?: typeof fetch; signal?: AbortSignal } = {}
): Promise<CubeModel[]> {
  const { tenantId, datasourceId, pageSize = 100, q, status } = params;
  const fetchFn = options.fetchFn ?? fetch;

  const query = new URLSearchParams();
  if (tenantId) query.set('tenant_id', tenantId);
  if (datasourceId) query.set('datasource_id', datasourceId);
  if (pageSize) query.set('page_size', String(pageSize));
  if (status) query.set('status', status);
  if (q) query.set('q', q);

  const url = `/api/fabric/models?${query.toString()}`;
  const reqInit: RequestInit = { signal: options.signal, cache: 'no-store' };
  const res = await fetchFn(url, reqInit);
  if (!res.ok) {
    const text = await res.text().catch(() => '');
    throw new Error(`Fetch failed: ${res.status} ${text}`);
  }

  const data = (await res.json().catch(() => null)) as unknown;

  // The models endpoint returns { models: [...] }
  if (data && typeof data === 'object') {
    const obj = data as Record<string, unknown>;
    const modelsProp = obj.models;
    if (Array.isArray(modelsProp)) return modelsProp as CubeModel[];
  }
  return [];
}
