import type { BusinessObjectDataSourceConfig, DataSourceDefinition } from '../../types/pageStudio';
import { buildBODataSource } from '../../pages/page-studio/generatePageDraft';
import type { RelatedObjectDragPayload } from './boRelationships';
import { fkColumnFromJoinCondition } from './boRelationships';

const BINDABLE = ['Table', 'Form', 'Slicer', 'LineChart', 'KPIGroup'];

export const isBindableWidget = (type: string) => BINDABLE.includes(type);

/**
 * Idempotent: one data source per related BO (`bo_{id}`), with masterFilter
 * from the relationship join. relatedTable kinds are not data sources.
 *
 * Studio-agnostic: moved out of pages/page-studio so Report Studio can drop
 * a related BO onto a report body and get the same master-detail wiring
 * Page Studio already has, per HANDOFF_REPORT_BUILDER_SPINE_PLAN.md Phase 1.
 * `buildBODataSource` itself stays in generatePageDraft.ts for now — its
 * impact graph is HIGH risk (NewPageWizard, PageStudioListPage, generator
 * flow) and moving it is out of scope for this extraction; only the
 * genuinely decoupled relationship/data-source-binding logic moved.
 */
export async function ensureRelatedDataSource(
  dataSources: DataSourceDefinition[],
  payload: RelatedObjectDragPayload,
): Promise<{ sources: DataSourceDefinition[]; sourceId: string } | null> {
  if (payload.kind === 'relatedTable') return null;
  const existing = dataSources.find((d) => {
    if (d.type !== 'business_object') return false;
    const cfg = d.config as unknown as BusinessObjectDataSourceConfig;
    return cfg.boId === payload.targetObjectId || d.id === `bo_${payload.targetObjectId}`;
  });
  const fkField =
    payload.joinColumns?.[0]?.source ||
    fkColumnFromJoinCondition(payload.joinCondition);
  let source = existing;
  if (!source) {
    source = await buildBODataSource(
      payload.targetObjectId,
      payload.relatedObjectName,
      payload.relatedObjectName,
      payload.relatedObjectName,
      [],
    );
  }
  const nextConfig: BusinessObjectDataSourceConfig = {
    ...(source.config as unknown as BusinessObjectDataSourceConfig),
    ...(fkField ? { masterFilter: { fkField } } : {}),
  };
  const bound: DataSourceDefinition = { ...source, config: nextConfig as unknown as Record<string, unknown> };
  const without = dataSources.filter((d) => d.id !== bound.id);
  const parented = without.map((d) => {
    if (d.id !== payload.parentSourceId) return d;
    const cfg = d.config as unknown as BusinessObjectDataSourceConfig;
    const relatedBoIds = Array.from(new Set([...(cfg.relatedBoIds || []), payload.targetObjectId]));
    return { ...d, config: { ...cfg, relatedBoIds } as unknown as Record<string, unknown> };
  });
  return { sources: [...parented, bound], sourceId: bound.id };
}
