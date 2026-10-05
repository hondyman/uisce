import apiClient from '../../utils/apiClient';
import type {
  CubeDefinition,
  CubeImpactPreviewAction,
  CubeImpactPreviewReport,
  CubeImpactReport,
  CubeListResponse,
  CubeMetricOption,
  CubeScope,
  CubeValidateResponse,
} from './types';

export type ListCubesParams = {
  scope?: CubeScope;
  cursor?: string;
  limit?: number;
  boId?: string;
};

export async function listCubeMetrics(boId?: string): Promise<CubeMetricOption[]> {
  const q = new URLSearchParams();
  if (boId) q.set('boId', boId);
  const qs = q.toString();
  const res = await apiClient<{ metrics: CubeMetricOption[] }>(`cubes/metrics${qs ? `?${qs}` : ''}`);
  return res.metrics ?? [];
}

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

export type CubeMaterializeStart = {
  grain: string[];
  grain_hash?: string;
  workflow_id?: string;
  attempt_id?: string;
  noop?: boolean;
  noop_reason?: string;
  already_running?: boolean;
  error?: string;
};

export type CubeMaterializeStartResponse = {
  cube_id: string;
  force: boolean;
  starts: CubeMaterializeStart[];
};

async function startCubeMaterialize(
  path: string,
  body?: { grain?: string[]; force?: boolean },
): Promise<CubeMaterializeStartResponse> {
  try {
    return await apiClient(path, {
      method: 'POST',
      body: body ? JSON.stringify(body) : undefined,
    });
  } catch (e) {
    // 409 already_running still returns the starts receipt in the error body.
    const msg = e instanceof Error ? e.message : String(e);
    const idx = msg.indexOf('{"cube_id"');
    if (idx >= 0) {
      try {
        const parsed = JSON.parse(msg.slice(idx)) as CubeMaterializeStartResponse;
        if (parsed && Array.isArray(parsed.starts)) return parsed;
      } catch {
        /* fall through */
      }
    }
    throw e;
  }
}

export async function deployCube(
  id: string,
  body?: { grain?: string[]; force?: boolean },
): Promise<CubeMaterializeStartResponse> {
  return startCubeMaterialize(`cubes/${encodeURIComponent(id)}/deploy`, body);
}

export async function refreshCube(
  id: string,
  body?: { grain?: string[]; force?: boolean },
): Promise<CubeMaterializeStartResponse> {
  return startCubeMaterialize(`cubes/${encodeURIComponent(id)}/refresh`, body);
}

export async function getCubeImpact(
  id: string,
  opts?: { includePhysical?: boolean },
): Promise<CubeImpactReport> {
  const q = new URLSearchParams();
  if (opts?.includePhysical === false) q.set('includePhysical', '0');
  const qs = q.toString();
  return apiClient<CubeImpactReport>(
    `cubes/${encodeURIComponent(id)}/impact${qs ? `?${qs}` : ''}`,
  );
}

export async function previewCubeImpact(
  id: string,
  body: {
    action: CubeImpactPreviewAction;
    patch?: Record<string, unknown>;
  },
): Promise<CubeImpactPreviewReport> {
  return apiClient<CubeImpactPreviewReport>(`cubes/${encodeURIComponent(id)}/impact/preview`, {
    method: 'POST',
    body: JSON.stringify(body),
  });
}
