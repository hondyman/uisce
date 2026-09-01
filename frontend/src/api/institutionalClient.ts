export interface APIResponse<T> {
  data: T;
  error?: string;
  status: number;
}

export interface CatalogNodeFilter {
  tenantId?: string;
  nodeTypeId?: string;
  type?: string;
  limit?: number;
}

const DEFAULT_GOLD_COPY_TENANT = '00000000-0000-0000-0000-000000000000';

class InstitutionalClient {
  private baseUrl: string;

  constructor(baseUrl: string = '/api') {
    this.baseUrl = baseUrl;
  }

  private sanitizeUUID(id?: string): string | undefined {
    if (!id) return undefined;
    const uuidRegex = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;
    return uuidRegex.test(id) ? id : undefined;
  }

  async getCatalogNodes(filter: CatalogNodeFilter): Promise<any[]> {
    try {
      const params = new URLSearchParams();
      const safeTenant = this.sanitizeUUID(filter.tenantId) || DEFAULT_GOLD_COPY_TENANT;
      params.append('tenant_id', safeTenant);

      if (filter.nodeTypeId) {
        const safeNodeType = this.sanitizeUUID(filter.nodeTypeId);
        if (safeNodeType) params.append('node_type_id', safeNodeType);
      }

      if (filter.type) {
        params.append('type', filter.type);
      }

      const safeLimit = Math.min(Math.max(filter.limit || 500, 1), 2000);
      params.append('limit', safeLimit.toString());

      const res = await fetch(`${this.baseUrl}/catalog/nodes?${params.toString()}`);
      if (!res.ok) {
        console.warn(`[InstitutionalClient] /catalog/nodes returned status ${res.status}. Falling back to empty array.`);
        return [];
      }
      const data = await res.json();
      return Array.isArray(data) ? data : [];
    } catch (err) {
      console.error('[InstitutionalClient] getCatalogNodes failed:', err);
      return [];
    }
  }

  async listBusinessObjects(tenantId?: string): Promise<any[]> {
    try {
      const safeTenant = this.sanitizeUUID(tenantId) || DEFAULT_GOLD_COPY_TENANT;
      const res = await fetch(`${this.baseUrl}/business-objects?tenant_id=${safeTenant}`);
      if (!res.ok) {
        console.warn(`[InstitutionalClient] /business-objects returned status ${res.status}. Returning fallback.`);
        return [];
      }
      const data = await res.json();
      return Array.isArray(data) ? data : [];
    } catch (err) {
      console.error('[InstitutionalClient] listBusinessObjects network exception:', err);
      return [];
    }
  }

  async triggerMasterSaga(payload: Record<string, any>): Promise<any> {
    const res = await fetch(`${this.baseUrl}/v1/saga/institutional/execute`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });
    if (!res.ok) {
      const err = await res.json().catch(() => ({ error: 'Saga execution failed' }));
      throw new Error(err.error || `HTTP error ${res.status}`);
    }
    return res.json();
  }
}

export const institutionalApi = new InstitutionalClient();
