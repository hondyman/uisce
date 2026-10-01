import type {
  SavedQueryState,
  QueryDimension,
  QueryFilter,
} from '../types/queryDef';
import type { SemanticTermView } from '../../../studio-core/binding/businessObjectApi';

export type DrillResolution =
  | { status: 'ok'; path: readonly string[]; currentLevel: number }
  | { status: 'disabled'; reason: 'no-hierarchy-configured' };

export interface DrillLevel {
  dimension: string;
  value: unknown;
}

export interface DrillStep {
  dimension: QueryDimension;
  filter: QueryFilter;
  label: string;
  value: string | number | boolean;
}

export interface DrillThroughTarget {
  type: 'page' | 'modal' | 'query';
  target: string;
  label: string;
  contextMapping?: Record<string, string>;
}

/**
 * Standard fallback dimension hierarchies.
 * @deprecated Kept for pre-metadata catalogs; overridable via tenant config.
 */
export const DEFAULT_HIERARCHIES: Record<string, readonly string[]> = {
  region: ['region', 'country', 'state_province', 'city'],
  country: ['country', 'state_province', 'city', 'postal_code'],
  asset_class: ['asset_class', 'sector', 'industry', 'security_id'],
  strategy: ['strategy', 'sub_strategy', 'mandate', 'account_id'],
  date: ['year', 'quarter', 'month', 'day'],
};

/**
 * Priority:
 *  1. Semantic catalog term metadata (drillPath or drill_down_dimensions authored in EditSemanticTermDialog)
 *  2. Standard fallback hierarchies (DEPRECATED — kept for pre-metadata catalogs; overridable via tenant config)
 *  3. Disabled — never invent synthetic levels.
 */
export function resolveDrillPath(
  termNodeId: string,
  currentDimension: string,
  catalog?: SemanticTermView[],
  fallbacks: Record<string, readonly string[]> = DEFAULT_HIERARCHIES,
): DrillResolution {
  const term = catalog?.find((t) => t.termNodeId === termNodeId || t.termKey === currentDimension);
  
  let metaPath: string[] | undefined;
  if (term) {
    const raw = (term as any).metadata?.drillPath || (term as any).metadata?.drill_down_dimensions;
    if (Array.isArray(raw)) {
      metaPath = raw;
    } else if (typeof raw === 'string' && raw.trim()) {
      metaPath = raw.split(',').map((s) => s.trim()).filter(Boolean);
    }
  }

  const path = metaPath?.length ? metaPath : fallbacks[currentDimension.toLowerCase()];

  if (!path || path.length < 2) {
    return { status: 'disabled', reason: 'no-hierarchy-configured' };
  }

  const idx = path.findIndex((p) => p.toLowerCase() === currentDimension.toLowerCase());
  if (idx === -1 || idx === path.length - 1) {
    return { status: 'disabled', reason: 'no-hierarchy-configured' }; // leaf or unrelated dimension
  }

  return { status: 'ok', path, currentLevel: idx };
}

export function drillDown(
  res: DrillResolution,
  memberValue: unknown,
  stack: DrillLevel[]
): { next: DrillLevel[] | null; childDimension: string | null } {
  if (res.status !== 'ok') return { next: null, childDimension: null };
  const child = res.path[res.currentLevel + 1];
  return {
    next: [...stack, { dimension: res.path[res.currentLevel], value: memberValue }],
    childDimension: child,
  };
}

/**
 * Computes a drilled SavedQueryState by swapping primary dimension and pushing filter stack.
 */
export function computeDrillState(
  baseState: SavedQueryState,
  drillSteps: DrillStep[],
  targetDimension?: QueryDimension
): SavedQueryState {
  if (drillSteps.length === 0) {
    return baseState;
  }

  const activeDim = targetDimension || drillSteps[drillSteps.length - 1].dimension;
  const pushedFilters: QueryFilter[] = drillSteps.map((s) => s.filter);

  return {
    ...baseState,
    dimensions: [activeDim, ...baseState.dimensions.slice(1)],
    filters: [...baseState.filters, ...pushedFilters],
  };
}

/**
 * Extracts context mapping for drill-through navigation from the active drill stack and selection.
 */
export function buildDrillThroughContext(
  drillSteps: DrillStep[],
  recordId?: string,
  extraContext?: Record<string, unknown>
): Record<string, unknown> {
  const context: Record<string, unknown> = { ...(extraContext || {}) };
  if (recordId) {
    context.recordId = recordId;
  }

  drillSteps.forEach((step) => {
    context[step.dimension.alias] = step.value;
    context[step.filter.termNodeId] = step.value;
  });

  return context;
}
