import apiClient from '../../utils/apiClient';
import type {
  CubeDefinition,
  CubeListResponse,
  CubeScope,
  CubeValidateResponse,
} from './types';

export type ListCubesParams = {
  scope?: CubeScope;
  cursor?: string;
  limit?: number;
  boId?: string;
};

export async function listCubes(params: ListCubesParams = {}): Promise<CubeListResponse> {
  const q = new URLSearchParams();
  if (params.scope) q.set('scope', params.scope);
  if (params.cursor) q.set('cursor', params.cursor);
  if (params.limit) q.set('limit', String(params.limit));
  if (params.boId) q.set('boId', params.boId);
  const qs = q.toString();
  return apiClient<CubeListResponse>(`cubes${qs ? `?${qs}` : ''}`);
}

export async function getCube(id: string): Promise<CubeDefinition> {
  const res = await apiClient<{ cube: CubeDefinition }>(`cubes/${encodeURIComponent(id)}`);
  return res.cube;
}

export async function createCube(body: Record<string, unknown>): Promise<CubeDefinition> {
  const res = await apiClient<{ cube: CubeDefinition }>('cubes', {
    method: 'POST',
    body: JSON.stringify(body),
  });
  return res.cube;
}

export async function patchCube(id: string, body: Record<string, unknown>): Promise<{
  cube: CubeDefinition;
  noop?: boolean;
  reason?: string;
}> {
  return apiClient(`cubes/${encodeURIComponent(id)}`, {
    method: 'PATCH',
    body: JSON.stringify(body),
  });
}

export async function validateCube(
  id: string,
  body?: Record<string, unknown>,
): Promise<CubeValidateResponse> {
  return apiClient(`cubes/${encodeURIComponent(id)}/validate`, {
    method: 'POST',
    body: body ? JSON.stringify(body) : undefined,
  });
}

export async function publishCubeVersion(
  id: string,
  body: Record<string, unknown>,
): Promise<{ cube: CubeDefinition; breakReasons?: string[]; previousVersion?: number }> {
  return apiClient(`cubes/${encodeURIComponent(id)}/versions`, {
    method: 'POST',
    body: JSON.stringify(body),
  });
}
