import type { ComponentDefinition, CorePageDefinition, PageLayout } from '../../../../types/pageStudio';
import '../../../../features/fabric/studio';

type Spec = Record<string, { type: 'Row' | 'Column'; children: string[]; style?: Record<string, string> }>;
const layout = (root: string, spec: Spec): PageLayout => ({
  root,
  nodes: Object.fromEntries(Object.entries(spec).map(([id, n]) => [id, { id, ...n }])),
});

function makeSingleDCBlueprint(opts: {
  name: string;
  slug: string;
  description: string;
  icon: string;
  dcId: string;
  title: string;
  subtitle: string;
}): Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'> {
  const components: Record<string, ComponentDefinition> = {
    header: {
      id: 'header',
      type: 'PageHeader',
      props: {
        icon: opts.icon,
        title: opts.title,
        subtitle: opts.subtitle,
      },
    },
    dc: {
      id: 'dc',
      type: 'DomainComponent',
      props: {
        component: opts.dcId,
        inputs: {},
      },
    },
  };

  const l = layout('root', {
    root: { type: 'Column', children: ['dc'], style: { gap: '16px' } },
  });

  const fb = layout('fb_root', {
    fb_root: { type: 'Row', children: ['header'], style: { alignItems: 'center', width: '100%' } },
  });

  return {
    name: opts.name,
    slug: opts.slug,
    description: opts.description,
    version: 1,
    isCore: true,
    status: 'published',
    layout: l,
    components,
    dataSources: [],
    tabs: [],
    presentationEvents: [],
    filterBar: fb,
    appModel: {
      chrome: 'none',
      surface: { padding: 3, maxWidth: 1600 },
      queries: [],
      variables: [],
    },
  };
}

export const calculationsLibraryBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'Calculations Library',
    slug: 'fabric-calculations',
    description: 'Central registry of metrics, business formulas, and reusable calculations.',
    icon: 'Percent',
    dcId: 'fabric.CalculationsLibrary',
    title: 'Calculations Library',
    subtitle: 'Central registry of metrics, business formulas, and reusable calculations.',
  });

export const preaggregationsBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'Preaggregations',
    slug: 'fabric-preaggregations',
    description: 'Pre-computed rollup cubes, partition refreshes, and acceleration telemetry.',
    icon: 'Cpu',
    dcId: 'fabric.Preaggregations',
    title: 'Preaggregations Management',
    subtitle: 'Pre-computed rollup cubes, partition refreshes, and acceleration telemetry.',
  });

export const fabricDashboardBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'Fabric Dashboard',
    slug: 'fabric-dashboard',
    description: 'Unified semantic fabric overview, node topology, and performance monitors.',
    icon: 'Layout',
    dcId: 'fabric.Dashboard',
    title: 'Fabric Dashboard',
    subtitle: 'Unified semantic fabric overview, node topology, and performance monitors.',
  });

export const fabricSettingsBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'Fabric Settings',
    slug: 'fabric-settings',
    description: 'System configurations, caching policies, and query engine routing rules.',
    icon: 'Sliders',
    dcId: 'fabric.Settings',
    title: 'Fabric Settings',
    subtitle: 'System configurations, caching policies, and query engine routing rules.',
  });

export const reportLibraryBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'Report Library',
    slug: 'reports-library',
    description: 'Enterprise catalog of published executive reports, templates, and analytics.',
    icon: 'FileText',
    dcId: 'fabric.ReportLibrary',
    title: 'Report Library',
    subtitle: 'Enterprise catalog of published executive reports, templates, and analytics.',
  });

export const reportBuilderBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'Report Builder',
    slug: 'reports-builder',
    description: 'Interactive visual designer for building reports and parameterized views.',
    icon: 'PieChart',
    dcId: 'fabric.ReportBuilder',
    title: 'Report Builder',
    subtitle: 'Interactive visual designer for building reports and parameterized views.',
  });

export const queryLibraryBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'Query Builder',
    slug: 'reports-queries',
    description: 'Visual query builder and saved query management repository.',
    icon: 'Search',
    dcId: 'fabric.QueryLibrary',
    title: 'Query Builder & Library',
    subtitle: 'Visual query builder and saved query management repository.',
  });

export const semanticModelsBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'Semantic Models',
    slug: 'reports-models',
    description: 'Manage analytical semantic models, dimensions, and measures.',
    icon: 'Database',
    dcId: 'fabric.SemanticModels',
    title: 'Semantic Models',
    subtitle: 'Manage analytical semantic models, dimensions, and measures.',
  });

export const bundleExplorerBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'Bundle Explorer',
    slug: 'bundle-explorer',
    description: 'Inspect micro-bundles, metadata packages, and cross-tenant exports.',
    icon: 'Package',
    dcId: 'fabric.BundleExplorer',
    title: 'Bundle Explorer',
    subtitle: 'Inspect micro-bundles, metadata packages, and cross-tenant exports.',
  });

export const viewsCatalogBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'Views Catalog',
    slug: 'views',
    description: 'Catalog of materialized, logical, and dynamic semantic views.',
    icon: 'Eye',
    dcId: 'fabric.ViewsCatalog',
    title: 'Views Catalog',
    subtitle: 'Catalog of materialized, logical, and dynamic semantic views.',
  });

export const businessObjectsBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'Business Objects',
    slug: 'business-objects',
    description: 'Enterprise business entities, driving tables, and bitemporal mappings.',
    icon: 'Box',
    dcId: 'fabric.BusinessObjects',
    title: 'Business Objects Catalog',
    subtitle: 'Enterprise business entities, driving tables, and bitemporal mappings.',
  });
