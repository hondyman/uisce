import type {
  CorePageDefinition,
  ComponentDefinition,
  DataSourceDefinition,
  BusinessObjectDataSourceConfig,
} from '../../types/pageStudio';
import type { GeneratedPageSpec, GeneratedRelatedBO } from '../../api/pageStudio';
import { fetchBusinessObjectBindings } from '../../features/query-builder/services/queryBuilderApi';
import { getLayoutTemplate } from './layoutTemplates';
import { fkColumnFromJoinCondition } from '../../studio-core/binding/boRelationships';

export interface BOOption {
  id: string;
  key: string;
  name: string;
  display_name: string;
}

const randomId = (prefix: string) => `${prefix}_${Math.random().toString(36).slice(2, 8)}`;

const slugify = (name: string) =>
  name
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '') || 'page';

export const buildBODataSource = async (
  id: string,
  key: string,
  name: string,
  displayName: string,
  relatedBoIds: string[],
  masterFilter?: { fkField: string },
): Promise<DataSourceDefinition> => {
  const bindings = await fetchBusinessObjectBindings(id).catch(() => []);
  const defaultBinding = bindings.find((b) => b.isDefault) || bindings[0];
  const config: BusinessObjectDataSourceConfig = {
    boId: id,
    boKey: key,
    bindingId: defaultBinding?.bindingId || '',
    displayName,
    relatedBoIds,
    ...(masterFilter ? { masterFilter } : {}),
  };
  return { id: `bo_${id}`, name, type: 'business_object', config: config as unknown as Record<string, unknown> };
};

const masterFilterFor = (rel: GeneratedRelatedBO): { fkField: string } | undefined => {
  const fk = rel.joinCondition ? fkColumnFromJoinCondition(rel.joinCondition) : '';
  return fk ? { fkField: fk } : undefined;
};

const placeSections = (
  sections: GeneratedPageSpec['sections'],
  layout: import('../../types/pageStudio').PageLayout,
  sectionIds: string[],
  components: Record<string, ComponentDefinition>,
  dataSourceIdByBOKey: Map<string, string>,
  fallbackSourceId: string,
) => {
  sections.forEach((section, i) => {
    const id = randomId('comp');
    components[id] = {
      id,
      type: section.type,
      label: section.title,
      props: { dataSourceId: dataSourceIdByBOKey.get(section.boKey) || fallbackSourceId },
    };
    const sectionId = sectionIds[Math.min(i, Math.max(0, sectionIds.length - 1))];
    const layoutSection = layout.nodes[sectionId];
    if (!layoutSection) return;
    layout.nodes[sectionId] = { ...layoutSection, children: [...(layoutSection.children || []), id] };
  });
};

/**
 * Expands a GeneratedPageSpec (a title + layout template + a list of
 * sections, each bound to either the primary BO or one of its real related
 * Business Objects) into an actual, savable CorePageDefinition draft: one
 * `business_object` data source per BO actually used - built the same way
 * DataBindingsPanel.tsx builds one when a user adds a BO by hand - plus one
 * component per generated section, placed into the chosen template's
 * sections in order. Widget data binding itself is NOT set here -
 * PageComponentRenderer resolves dimensions/measures live from each
 * component's data source at render time, so components only need a
 * dataSourceId.
 */
export async function generatePageDraft(
  bo: BOOption,
  spec: GeneratedPageSpec,
  description: string,
  tenantId: string
): Promise<CorePageDefinition> {
  const relatedBoIds = spec.relatedBusinessObjects.map((r) => r.boId);
  const primarySource = await buildBODataSource(bo.id, bo.key, bo.name, bo.display_name || bo.name, relatedBoIds);
  const relatedSources = await Promise.all(
    spec.relatedBusinessObjects.map((r) =>
      buildBODataSource(r.boId, r.boKey, r.displayName, r.displayName, [], masterFilterFor(r)),
    ),
  );
  const dataSources = [primarySource, ...relatedSources];
  const dataSourceIdByBOKey = new Map<string, string>([
    ['', primarySource.id],
    ...spec.relatedBusinessObjects.map((r): [string, string] => [r.boKey, `bo_${r.boId}`]),
  ]);

  const template = getLayoutTemplate(spec.layoutTemplate);
  const { layout, sectionIds } = template.build();

  const components: Record<string, ComponentDefinition> = {};
  placeSections(spec.sections, layout, sectionIds, components, dataSourceIdByBOKey, primarySource.id);

  let filterBar: CorePageDefinition['filterBar'];
  if (spec.filterBar && spec.filterBar.length > 0) {
    const barRoot = 'filter_root';
    filterBar = { root: barRoot, nodes: { [barRoot]: { id: barRoot, type: 'Row', children: [] } } };
    placeSections(spec.filterBar, filterBar, [barRoot], components, dataSourceIdByBOKey, primarySource.id);
  }

  const now = new Date().toISOString();
  return {
    id: '',
    name: spec.title,
    slug: `${slugify(spec.title)}-${Math.random().toString(36).slice(2, 6)}`,
    description,
    tenantId,
    layout,
    filterBar,
    components,
    dataSources,
    pageKind: spec.pageKind,
    isCore: false,
    status: 'draft',
    createdAt: now,
    updatedAt: now,
    version: 1,
  };
}

/**
 * Copilot follow-up: add any spec sections that are not already on the page.
 * Does not replace layout, events, or existing widgets.
 */
export async function mergeGeneratedSpecIntoDraft(
  draft: CorePageDefinition,
  spec: GeneratedPageSpec,
): Promise<CorePageDefinition> {
  const existingTypesBySource = new Set(
    Object.values(draft.components || {}).map((c) => `${c.type}:${(c.props?.dataSourceId as string) || ''}`),
  );
  const sources = [...(draft.dataSources || [])];
  const byBoId = new Set(
    sources.filter((d) => d.type === 'business_object').map((d) => (d.config as unknown as BusinessObjectDataSourceConfig).boId),
  );

  for (const rel of spec.relatedBusinessObjects || []) {
    if (byBoId.has(rel.boId)) continue;
    sources.push(await buildBODataSource(rel.boId, rel.boKey, rel.displayName, rel.displayName, [], masterFilterFor(rel)));
    byBoId.add(rel.boId);
    const primary = sources.find((d) => d.type === 'business_object');
    if (primary) {
      const cfg = primary.config as unknown as BusinessObjectDataSourceConfig;
      primary.config = { ...cfg, relatedBoIds: Array.from(new Set([...(cfg.relatedBoIds || []), rel.boId])) } as unknown as Record<string, unknown>;
    }
  }

  const dataSourceIdByBOKey = new Map<string, string>();
  sources.filter((d) => d.type === 'business_object').forEach((d, i) => {
    const cfg = d.config as unknown as BusinessObjectDataSourceConfig;
    if (i === 0) dataSourceIdByBOKey.set('', d.id);
    dataSourceIdByBOKey.set(cfg.boKey, d.id);
  });
  const fallback = sources.find((d) => d.type === 'business_object')?.id || '';

  const components = { ...draft.components };
  const layout = {
    ...draft.layout,
    nodes: { ...draft.layout.nodes },
  };
  const rootId = layout.root;
  const root = layout.nodes[rootId];
  const overflowId = (root?.children || []).slice(-1)[0] || rootId;

  const toPlace = spec.sections.filter((section) => {
    const dsId = dataSourceIdByBOKey.get(section.boKey) || fallback;
    return !existingTypesBySource.has(`${section.type}:${dsId}`);
  });
  toPlace.forEach((section) => {
    const id = randomId('comp');
    const dsId = dataSourceIdByBOKey.get(section.boKey) || fallback;
    components[id] = { id, type: section.type, label: section.title, props: { dataSourceId: dsId } };
    const slot = layout.nodes[overflowId] ? overflowId : rootId;
    const node = layout.nodes[slot];
    if (node) layout.nodes[slot] = { ...node, children: [...(node.children || []), id] };
    existingTypesBySource.add(`${section.type}:${dsId}`);
  });

  return { ...draft, components, layout, dataSources: sources, updatedAt: new Date().toISOString() };
}
