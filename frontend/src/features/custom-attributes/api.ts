import { apiFetch } from '../../lib/apiClient';
import type {
  AttributeDef,
  CreateAttributeInput,
  EligibleEntity,
  PreviewResponse,
  UpdateAttributeInput,
} from './types';

async function parseJSON<T>(res: Response): Promise<T> {
  if (!res.ok) {
    let message = res.statusText;
    try {
      const body = await res.json();
      message = body.error || body.message || message;
    } catch {
      // ignore
    }
    throw new Error(message);
  }
  if (res.status === 204) {
    return undefined as T;
  }
  return res.json();
}

export async function listEligibleEntities(): Promise<EligibleEntity[]> {
  const res = await apiFetch('/api/v1/attributes/entities');
  const data = await parseJSON<{ entities: EligibleEntity[] }>(res);
  return data.entities || [];
}

export async function listAttributes(entityType: string): Promise<AttributeDef[]> {
  const res = await apiFetch(
    `/api/v1/attributes?entity_type=${encodeURIComponent(entityType)}`,
  );
  const data = await parseJSON<{ attributes: AttributeDef[] }>(res);
  return data.attributes || [];
}

export async function createAttribute(input: CreateAttributeInput): Promise<AttributeDef> {
  const res = await apiFetch('/api/v1/attributes', {
    method: 'POST',
    body: JSON.stringify(input),
  });
  return parseJSON<AttributeDef>(res);
}

export async function updateAttribute(
  id: string,
  input: UpdateAttributeInput,
): Promise<AttributeDef> {
  const res = await apiFetch(`/api/v1/attributes/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(input),
  });
  return parseJSON<AttributeDef>(res);
}

export async function deleteAttribute(id: string): Promise<void> {
  const res = await apiFetch(`/api/v1/attributes/${id}`, { method: 'DELETE' });
  await parseJSON<void>(res);
}

export async function previewAttributes(params: {
  entityType: string;
  tableRef?: string;
  limit?: number;
  offset?: number;
}): Promise<PreviewResponse> {
  const q = new URLSearchParams({
    entity_type: params.entityType,
    limit: String(params.limit ?? 50),
    offset: String(params.offset ?? 0),
  });
  if (params.tableRef) {
    q.set('table_ref', params.tableRef);
  }
  const res = await apiFetch(`/api/v1/attributes/preview?${q.toString()}`);
  return parseJSON<PreviewResponse>(res);
}
