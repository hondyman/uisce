/**
 * Shared QuerySubject for Query Builder and Report Builder (CUBE-1.6).
 * One shell, one picker: business_object | cube.
 */

export type QuerySubjectKind = 'business_object' | 'cube';

export type ContractVersionPin = number | 'latest';

export type BusinessObjectSubject = {
  kind: 'business_object';
  boId: string;
  bindingId: string;
  relatedBoIds?: string[];
};

export type CubeSubject = {
  kind: 'cube';
  cubeId: string;
  contractVersion: ContractVersionPin;
};

export type QuerySubject = BusinessObjectSubject | CubeSubject;

export type ServedFrom = 'hot' | 'cold' | 'raw';

export type CubeRouteBadge = {
  servedFrom: ServedFrom;
  stale?: boolean;
  cubeId?: string;
  cubeName?: string;
  materialization?: string;
  contractVersion?: number;
  missReason?: string;
};

export function isCubeSubject(s: QuerySubject | null | undefined): s is CubeSubject {
  return !!s && s.kind === 'cube';
}

export function isBusinessObjectSubject(
  s: QuerySubject | null | undefined,
): s is BusinessObjectSubject {
  return !!s && s.kind === 'business_object';
}

/** Synthesize a BO subject from legacy flat context fields. */
export function businessObjectSubject(
  boId: string,
  bindingId: string,
  relatedBoIds?: string[],
): BusinessObjectSubject {
  return {
    kind: 'business_object',
    boId,
    bindingId,
    ...(relatedBoIds && relatedBoIds.length ? { relatedBoIds } : {}),
  };
}

export function cubeSubject(
  cubeId: string,
  contractVersion: ContractVersionPin = 'latest',
): CubeSubject {
  return { kind: 'cube', cubeId, contractVersion };
}
