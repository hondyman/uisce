import type {
  CorePageDefinition,
  ComponentDefinition,
  DataSourceDefinition,
  BusinessObjectDataSourceConfig,
} from '../../types/pageStudio';
import type { GeneratedPageSpec } from '../../api/pageStudio';
import { fetchBusinessObjectBindings } from '../../features/query-builder/services/queryBuilderApi';
import { getLayoutTemplate } from './layoutTemplates';

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

export const buildBODataSource = async (id: string, key: string, name: string, displayName: string, relatedBoIds: string[]): Promise<DataSourceDefinition> => {
  const bindings = await fetchBusinessObjectBindings(id).catch(() => []);
  const defaultBinding = bindings.find((b) => b.isDefault) || bindings[0];
  const config: BusinessObjectDataSourceConfig = {
    boId: id,
    boKey: key,
    bindingId: defaultBinding?.bindingId || '',
    displayName,
    relatedBoIds,
  };
  return { id: `bo_${id}`, name, type: 'business_object', config: config as unknown as Record<string, unknown> };
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
    spec.relatedBusinessObjects.map((r) => buildBODataSource(r.boId, r.boKey, r.displayName, r.displayName, []))
  );
  const dataSources = [primarySource, ...relatedSources];
  const dataSourceIdByBOKey = new Map<string, string>([
    ['', primarySource.id],
    ...spec.relatedBusinessObjects.map((r): [string, string] => [r.boKey, `bo_${r.boId}`]),
  ]);

  const template = getLayoutTemplate(spec.layoutTemplate);
  const { layout, sectionIds } = template.build();

  const components: Record<string, ComponentDefinition> = {};
  spec.sections.forEach((section, i) => {
    const id = randomId('comp');
    components[id] = {
      id,
      type: section.type,
      label: section.title,
      props: { dataSourceId: dataSourceIdByBOKey.get(section.boKey) || primarySource.id },
    };
    // More sections than the template has slots for (e.g. dashboard-grid's
    // 4 sections vs a 6-section spec): pile any overflow into the last
    // section rather than dropping them, since every generated widget
    // should end up somewhere visible.
    const sectionId = sectionIds[Math.min(i, sectionIds.length - 1)];
    const layoutSection = layout.nodes[sectionId];
    layout.nodes[sectionId] = { ...layoutSection, children: [...(layoutSection.children || []), id] };
  });

  const now = new Date().toISOString();
  return {
    id: '',
    name: spec.title,
    slug: `${slugify(spec.title)}-${Math.random().toString(36).slice(2, 6)}`,
    description,
    tenantId,
    layout,
    components,
    dataSources,
    isCore: false,
    status: 'draft',
    createdAt: now,
    updatedAt: now,
    version: 1,
  };
}
