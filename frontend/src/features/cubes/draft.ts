import type {
  CubeDefinition,
  CubeDraft,
  CubeFederation,
  CubeMaterialization,
  FederationKeySample,
} from './types';
import { federationIsActive, normalizeFederation } from './types';

export const DEFAULT_CUBE_MATERIALIZATION: CubeMaterialization = {
  strategy: 'starrocks_mv',
  refreshStrategy: 'manual',
  hotEngine: 'starrocks',
  coldEngine: 'iceberg',
  stalePolicy: 'serve_with_flag',
};

/** Empty working draft for `/build/cubes/new` and `cubes.editorStart`. */
export function emptyCubeDraft(overrides?: Partial<CubeDraft>): CubeDraft {
  return {
    name: '',
    description: '',
    boId: 'account',
    dimensions: [],
    metricIds: [],
    grains: [],
    materialization: { ...DEFAULT_CUBE_MATERIALIZATION },
    federation: {},
    federationKeySamples: [],
    ...overrides,
  };
}

export function cubeDraftFromDefinition(c: CubeDefinition): CubeDraft {
  return {
    name: c.name,
    description: c.description || '',
    boId: c.boId,
    dimensions: c.dimensions || [],
    metricIds: c.metricIds || [],
    grains: c.grains || [],
    materialization: { ...DEFAULT_CUBE_MATERIALIZATION, ...(c.materialization || {}) },
    federation: (c.federation || {}) as CubeFederation,
    federationKeySamples: [],
  };
}

/** Body for create/patch/validate from a CubeDraft. */
export function cubeDraftPayload(
  d: CubeDraft,
  opts?: { includeKeySamples?: boolean },
): Record<string, unknown> {
  const federation = normalizeFederation(d.federation);
  const body: Record<string, unknown> = {
    name: d.name.trim(),
    description: d.description,
    boId: d.boId.trim(),
    dimensions: d.dimensions,
    metricIds: d.metricIds,
    grains: d.grains,
    materialization: d.materialization,
    federation,
  };
  if (opts?.includeKeySamples && federationIsActive(federation) && d.federationKeySamples?.length) {
    body.federationKeySamples = d.federationKeySamples.map((s: FederationKeySample) => ({
      leftAlias: s.leftAlias,
      rightAlias: s.rightAlias,
      leftKeys: s.leftKeys,
      rightKeys: s.rightKeys,
      matched: s.matched,
    }));
  }
  return body;
}
