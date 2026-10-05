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
};
