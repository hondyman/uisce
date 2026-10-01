import type { ComponentDefinition, CorePageDefinition, PageLayout, PageTab } from '../../../../types/pageStudio';

type Spec = Record<string, { type: 'Row' | 'Column'; children: string[]; style?: Record<string, string> }>;
const layout = (root: string, spec: Spec): PageLayout => ({
  root,
  nodes: Object.fromEntries(Object.entries(spec).map(([id, n]) => [id, { id, ...n }])),
});

export function lakehouseStreamingBlueprint(): Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'> {
  const components: Record<string, ComponentDefinition> = {};
  const reg = (id: string, type: string, props: Record<string, unknown>, extra: Partial<ComponentDefinition> = {}): string => {
    components[id] = { id, type, props, ...extra };
    return id;
  };

  // Header
  reg('hdr', 'PageHeader', {
    icon: 'stream',
    title: 'Lakehouse & CDC Stream Ingestion',
    subtitle: 'Apache Iceberg Lakekeeper REST catalog, Debezium CDC gatekeeper, Layer 2 Tenant Assertion, and DLQ telemetry',
  }, { style: { flex: '1 1 320px' } });

  // Tab 1 Components
  reg('gatekeeper_widget', 'lakehouse.StreamingGatekeeperMonitor', {});

  // Tab 2 Components
  reg('catalog_widget', 'lakehouse.LakekeeperCatalogManager', {});

  // Tab 3 Components
  reg('dlq_widget', 'lakehouse.DLQInspectorPanel', {});

  // Tabs
  const tabs: PageTab[] = [
    {
      id: 'telemetry',
      label: 'Ingestion & Gatekeeper',
      layout: layout('telemetry_root', {
        telemetry_root: { type: 'Column', children: ['gatekeeper_widget'] },
      }),
    },
    {
      id: 'catalog',
      label: 'Iceberg & Lakekeeper Catalog',
      layout: layout('catalog_root', {
        catalog_root: { type: 'Column', children: ['catalog_widget'] },
      }),
    },
    {
      id: 'dlq',
      label: 'Dead-Letter Queue (DLQ)',
      layout: layout('dlq_root', {
        dlq_root: { type: 'Column', children: ['dlq_widget'] },
      }),
    },
  ];

  return {
    name: 'Lakehouse & CDC Stream Ingestion',
    slug: 'lakehouse-streaming',
    description: 'Apache Iceberg REST catalog management, Debezium CDC ingestion pipelines, Layer 2 tenant assertion, and streaming gatekeeper inspection.',
    version: 1,
    isCore: true,
    status: 'published',
    layout: layout('page_root', {
      page_root: { type: 'Column', children: ['gatekeeper_widget'], style: { gap: '16px' } },
    }),
    tabs,
    components,
    dataSources: [],
    presentationEvents: [],
    filterBar: layout('filter_root', {
      filter_root: { type: 'Column', children: ['header_row'], style: { gap: '12px' } },
      header_row: { type: 'Row', children: ['hdr'], style: { alignItems: 'center', gap: '8px', flexWrap: 'wrap' } },
    }),
    app: {
      chrome: 'none',
      surface: { maxWidth: 1400, padding: 3 },
      tabVariable: 'tab',
      variables: [
        { name: 'tab', default: 'telemetry', url: true },
      ],
      queries: [
        {
          id: 'lakehouse_overview',
          operation: 'lakehouse.overview',
          params: {},
        },
      ],
    },
  };
}
