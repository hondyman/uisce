import type { ComponentDefinition, CorePageDefinition, PageLayout } from '../../../../types/pageStudio';
import '../../../../features/catalog/studio';

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
    filterBar: fb,
    tabs: [],
    components,
    dataSources: [],
    presentationEvents: [],
    app: {
      chrome: 'none',
      surface: { maxWidth: 1600, padding: 3 },
      variables: [],
      queries: [],
    },
  };
}

export function apiInventoryBlueprint(): Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'> {
  return makeSingleDCBlueprint({
    name: 'API Inventory',
    slug: 'catalog-api-inventory',
    description: 'Catalog and inspect endpoints, payloads, schema contracts, and authentication',
    icon: 'api',
    dcId: 'catalog.ApiInventory',
    title: 'API Inventory',
    subtitle: 'Unified interface directory, OpenAPI specs, and contract governance',
  });
}

export function glossaryBlueprint(): Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'> {
  return makeSingleDCBlueprint({
    name: 'Glossary & Semantic Terms',
    slug: 'catalog-glossary',
    description: 'Explore business glossary, semantic terms, graph relationships, and classifications',
    icon: 'menuBook',
    dcId: 'catalog.GlossaryExplorer',
    title: 'Glossary & Semantic Terms',
    subtitle: 'Enterprise taxonomy, semantic terms, and graph associations',
  });
}

export function businessTermsBlueprint(): Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'> {
  return makeSingleDCBlueprint({
    name: 'Business Terms',
    slug: 'catalog-business-terms',
    description: 'Governed business taxonomy and terminology explorer',
    icon: 'label',
    dcId: 'catalog.BusinessTerms',
    title: 'Business Terms',
    subtitle: 'Governed business taxonomy and terminology glossary',
  });
}

export function customFieldsBlueprint(): Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'> {
  return makeSingleDCBlueprint({
    name: 'Custom Fields',
    slug: 'catalog-custom-fields',
    description: 'Manage dynamic custom attributes and entity extensions',
    icon: 'tune',
    dcId: 'catalog.CustomFields',
    title: 'Custom Fields',
    subtitle: 'Dynamic attributes, entity extensions, and custom property sets',
  });
}

export function domainsBlueprint(): Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'> {
  return makeSingleDCBlueprint({
    name: 'Data Domains',
    slug: 'core-domains',
    description: 'Business data domains, ownership boundaries, and semantic scope',
    icon: 'category',
    dcId: 'catalog.DomainsManagement',
    title: 'Data Domains',
    subtitle: 'Domain boundaries, ownership, data stewardship, and policy assignment',
  });
}

export function schemaExplorerBlueprint(): Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'> {
  return makeSingleDCBlueprint({
    name: 'Schema Explorer',
    slug: 'catalog-schema-explorer',
    description: 'Deep introspection of underlying database tables, views, and columns',
    icon: 'database',
    dcId: 'catalog.SchemaExplorer',
    title: 'Schema Explorer',
    subtitle: 'Physical database schema introspection, column statistics, and keys',
  });
}

export function semanticMapperBlueprint(): Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'> {
  return makeSingleDCBlueprint({
    name: 'Semantic Mapper',
    slug: 'core-semantic-mapper',
    description: 'Map physical columns and upstream assets to canonical business concepts',
    icon: 'altRoute',
    dcId: 'catalog.SemanticMapper',
    title: 'Semantic Mapper',
    subtitle: 'Intelligent physical-to-semantic concept mapping and automated bindings',
  });
}

export function aiSuggestionsBlueprint(): Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'> {
  return makeSingleDCBlueprint({
    name: 'AI Term Suggestions',
    slug: 'catalog-ai-suggestions',
    description: 'AI-assisted terminology recommendations, match confidence, and review workflow',
    icon: 'autoAwesome',
    dcId: 'catalog.AISuggestions',
    title: 'AI Term Suggestions',
    subtitle: 'Machine-learning discovered semantic terms and review approval pipeline',
  });
}
