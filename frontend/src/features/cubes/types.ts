export type CubeScope = 'all' | 'core' | 'custom' | 'adopted';

export type CubeDimension = {
  termNodeId: string;
  drillPath?: string[];
};

export type CubeTimeDimension = {
  termNodeId: string;
  defaultGrain?: string;
};

export type CubeMaterialization = {
  strategy?: string;
  refreshStrategy?: string;
  refreshIntervalMinutes?: number;
  partitionGrain?: string;
  retentionDays?: number;
  retentionDaysHot?: number;
  retentionDaysCold?: number;
  hotEngine?: string;
  coldEngine?: string;
  stalePolicy?: string;
};

export type CubeFederationSource = {
  boId: string;
  alias: string;
  bindingHint?: string;
};

export type CubeFederationJoin = {
  leftAlias: string;
  rightAlias: string;
  keyKind: 'common' | 'transform' | string;
  leftTermIds: string[];
  rightTermIds: string[];
  transformTermId?: string;
};

export type CubeFederation = {
  sources?: CubeFederationSource[];
  joins?: CubeFederationJoin[];
  orphanRateMaxPercent?: number;
};

/** Validate/deploy orphan-gate fixture (not persisted on cube_definition). */
export type FederationKeySample = {
  leftAlias: string;
  rightAlias: string;
  leftKeys: number;
  rightKeys: number;
  matched: number;
};

export type CubeDefinition = {
  id: string;
  tenantId: string;
  name: string;
  description?: string;
  boId: string;
  dimensions: CubeDimension[];
  timeDimension?: CubeTimeDimension | null;
  metricIds: string[];
  grains: string[][];
  materialization: CubeMaterialization;
  federation: CubeFederation;
  contractVersion: number;
  contentHash: string;
  isCore: boolean;
  status: string;
  archivedAt?: string | null;
  createdBy?: string | null;
  createdAt: string;
  updatedAt: string;
};

export type CubeListResponse = {
  cubes: CubeDefinition[];
  scope: CubeScope;
  nextCursor?: string;
};

/** A1 GET /api/cubes/{id}/impact */
export type CubeImpactMetric = { id: string; name?: string; status?: string };
export type CubeImpactPhysicalGrain = {
  nodeName?: string;
  grainHash?: string;
  grain?: string[];
  lifecycleStatus?: string;
  icebergTable?: string;
  dualCommitWatermark?: string | null;
  attemptId?: string;
  contractVersion?: number;
};
export type CubeImpactComposition = {
  cubeId: string;
  name: string;
  status: string;
  contractVersion: number;
  contentHash: string;
  isCore: boolean;
  boId: string;
  dimensions: CubeDimension[];
  timeDimension?: CubeTimeDimension | null;
  metrics: CubeImpactMetric[];
  grains: string[][];
  federation: CubeFederation;
  materialization: CubeMaterialization;
  physical?: { grains: CubeImpactPhysicalGrain[] };
};
export type CubeImpactConsumer = {
  kind: string;
  id: string;
  label: string;
  href?: string;
  severity: 'info' | 'warning' | 'blocking' | string;
  blocking: boolean;
  detail?: string;
  enabled?: boolean;
  pin?: { contractVersion?: number | string };
};
export type CubeImpactSummary = {
  consumerCount: number;
  blockingCount: number;
  warningCount: number;
  physicalGrainCount: number;
};
export type CubeImpactReport = {
  cubeId: string;
  composition: CubeImpactComposition;
  consumers: CubeImpactConsumer[];
  summary: CubeImpactSummary;
};

/** A2 POST /api/cubes/{id}/impact/preview */
export type CubeImpactPreviewAction = 'archive' | 'patch' | 'publish_version';
export type CubeImpactPreviewReport = CubeImpactReport & {
  changeClass: 'archive' | 'non_breaking_patch' | 'breaking_contract' | string;
  breakReasons: string[];
  blockingCount: number;
  confirmToken: string;
  allowedModes: string[];
  recommendedMode: string;
  nextVersion?: number;
  patchHash?: string;
};

export type FederationJoinOrphanStats = {
  leftAlias: string;
  rightAlias: string;
  leftKeys: number;
  rightKeys: number;
  matched: number;
  leftOrphanPct: number;
  rightOrphanPct: number;
  ok: boolean;
  detail?: string;
};

export type FederationOrphanReport = {
  joins?: FederationJoinOrphanStats[];
  maxPercent: number;
  observedMaxPercent: number;
  ok: boolean;
  skipped?: boolean;
  detail?: string;
};

export type FederationTransformProjection = {
  joinIndex: number;
  leftAlias: string;
  rightAlias: string;
  transformTermId: string;
  leftTermIds: string[];
  rightTermIds: string[];
  compiledExpr: string;
};

export type CubeValidateResponse = {
  ok: boolean;
  structuralOk: boolean;
  structuralError?: string;
  metricsOk: boolean;
  metricsError?: string;
  federationOk?: boolean;
  federationError?: string;
  federationTransforms?: FederationTransformProjection[];
  orphanReport?: FederationOrphanReport;
  breakReasons?: string[];
  contentHash?: string;
  contractVersion?: number;
  orphanRateMaxPct?: number;
};

export type CubeMetricOption = {
  id: string;
  name: string;
  description?: string;
  boId: string;
  contentHash?: string;
  decomposable: boolean;
  isCore: boolean;
  status: string;
};

export type CubeDraft = {
  name: string;
  description: string;
  boId: string;
  dimensions: CubeDimension[];
  metricIds: string[];
  grains: string[][];
  materialization: CubeMaterialization;
  federation: CubeFederation;
  /** Session-only fixtures sent with validate when federation is declared. */
  federationKeySamples: FederationKeySample[];
};

/** True when the draft declares at least one federation source or join. */
export function federationIsActive(f?: CubeFederation | null): boolean {
  if (!f) return false;
  return (f.sources?.length ?? 0) > 0 || (f.joins?.length ?? 0) > 0;
}

/** Normalize federation for create/patch/validate (empty → {}). */
export function normalizeFederation(f?: CubeFederation | null): CubeFederation {
  if (!f) return {};
  const sources = (f.sources || [])
    .map((s) => ({
      boId: (s.boId || '').trim(),
      alias: (s.alias || '').trim().toLowerCase(),
      bindingHint: (s.bindingHint || '').trim() || undefined,
    }))
    .filter((s) => s.boId && s.alias);
  const joins = (f.joins || [])
    .map((j) => ({
      leftAlias: (j.leftAlias || '').trim().toLowerCase(),
      rightAlias: (j.rightAlias || '').trim().toLowerCase(),
      keyKind: (j.keyKind || 'common').trim().toLowerCase(),
      leftTermIds: (j.leftTermIds || []).map((t) => t.trim()).filter(Boolean),
      rightTermIds: (j.rightTermIds || []).map((t) => t.trim()).filter(Boolean),
      transformTermId:
        (j.keyKind || '').toLowerCase() === 'transform'
          ? (j.transformTermId || '').trim() || undefined
          : undefined,
    }))
    .filter((j) => j.leftAlias && j.rightAlias && j.leftTermIds.length > 0 && j.rightTermIds.length > 0);
  if (sources.length === 0 && joins.length === 0) return {};
  const out: CubeFederation = { sources, joins };
  if (f.orphanRateMaxPercent != null && f.orphanRateMaxPercent > 0) {
    out.orphanRateMaxPercent = f.orphanRateMaxPercent;
  }
  return out;
}
