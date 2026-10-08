import type { ComponentDefinition, CorePageDefinition, PageLayout } from '../../../../types/pageStudio';

/**
 * Platform > Lakehouse status (ADR-049 area). Read-only: cluster health, resource groups,
 * per-tenant wiring, and cross-check warnings. NO buttons that write — config stays in env
 * + git per the runbook gap 4 ("the warnings column is the panel").
 *
 * Mirrors the systemLakehouse blueprint's structure: header + filter bar + main layout,
 * all bound to one read-only operation (lakehouseStatus.platform). The tenant page uses
 * lakehouseStatus.tenant instead — same shape minus cluster + resource_groups.
 *
 * Page rule (per docs/page-designer-handoff.md): every widget here is a Page Studio block.
 * Domain decisions live in the operation adapter (features/lakehouse/studio.ts), not here.
 */

const layout = (root: string, spec: Record<string, { type: 'Row' | 'Column' | 'Dialog' | 'Drawer'; children: string[]; style?: Record<string, string>; props?: Record<string, unknown> }>): PageLayout => ({
  root,
  nodes: Object.fromEntries(Object.entries(spec).map(([id, n]) => [id, { id, ...n }])),
});

// Condition helpers — see appModel.ts ConditionNode. Rule engine operators include
// `is_not_empty`, `is_empty`, `greater_than`, `is_true`, etc. (no `is_not_true`).
const cond = (field: string, operator: string, value?: unknown) => ({ type: 'condition' as const, field, operator, value });

// The platform handler returns the full Status payload. Bindings below reach into it
// via {{queries.lhStart.data.X}} — see features/system-lakehouse/studio.ts for the
// precedent. lhStart is the single platform-status query id; lhStartTenant is the
// tenant-scoped variant.
const START = 'queries.lhStart.data';
const START_TENANT = 'queries.lhStartTenant.data';

function lakehouseStatusBlueprintCore(): Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'> {
  const components: Record<string, ComponentDefinition> = {};
  const w = (id: string, type: string, props: Record<string, unknown>, extra: Partial<ComponentDefinition> = {}) => {
    components[id] = { id, type, props, ...extra };
  };

  // --- header ---
  w('hdr', 'PageHeader', {
    icon: 'monitor_heart',
    title: 'Lakehouse status',
    subtitle: 'Cluster health, per-tenant wiring, and cross-check warnings. Read-only.',
  }, { style: { flex: '1 1 320px' } });

  // --- top notes banner — only visible when the platform call returned caveats ---
  w('note_info', 'AlertBanner', {
    severity: 'info',
    text: `{{${START}.notes_text}}`,
  }, { visibleWhen: cond(`${START}.notes_text`, 'is_not_empty') });

  // --- cluster summary: a KeyValue block reading the shaped rows from the operation ---
  // (The operation adapter exposes `cluster_rows` as one entry per FE/BE so this widget
  // doesn't need a separate query.)
  w('cluster_caption', 'TextBlock', {
    text: 'Cluster',
    caption: 'Frontends and backends from SHOW FRONTENDS / SHOW BACKENDS.',
  });
  w('cluster_kv', 'KeyValue', {
    label: '',
    rowsPath: `${START}.cluster_rows`,
    emptyText: 'Cluster not configured (LAKEHOUSE_STARROCKS_DSN unset).',
    pairs: [
      { label: '{{row.label}}', value: '{{row.value}}', sub: '{{row.sub}}' },
    ],
  });

  // --- resource groups table ---
  w('rg_caption', 'TextBlock', {
    text: 'Resource groups',
    caption: 'CPU weight, memory limit, concurrency, and the classifier routing that pins each tenant user to a group.',
  });
  w('rg_grid', 'DataGrid', {
    query: 'lhStart',
    rowsPath: 'resource_groups',
    rowKey: 'name',
    emptyText: `{{${START}.rg_empty_text}}`,
    columns: [
      { id: 'name', header: 'Group', cell: { kind: 'text', value: '{{row.name}}', bold: true } },
      { id: 'cpu', header: 'CPU', cell: { kind: 'number', value: '{{row.cpu_weight}}', suffix: ' bw' } },
      { id: 'mem', header: 'Memory', cell: { kind: 'text', value: '{{row.mem_limit}}' } },
      { id: 'conc', header: 'Concurrency', cell: { kind: 'number', value: '{{row.concurrency_limit}}' } },
      { id: 'cls', header: 'Classifiers', cell: { kind: 'text', value: '{{row.classifiers_text}}', caption: true } },
    ],
  });

  // --- tenants table (warnings column is the panel) ---
  w('tenants_caption', 'TextBlock', {
    text: 'Tenants',
    caption: 'Per-tenant wiring (DSN, init resource group), warehouse DB size, and audit-copy posture. Warnings show the silent-failure modes the runbook gap 4 calls out.',
  });
  w('tenants_grid', 'DataGrid', {
    query: 'lhStart',
    rowsPath: 'tenants',
    rowKey: 'tenant_id',
    emptyText: `{{${START}.tenants_empty_text}}`,
    columns: [
      {
        id: 'name', header: 'Tenant',
        cell: { kind: 'text', value: '{{row.name}}', bold: true },
      },
      {
        id: 'slug', header: 'Slug',
        cell: { kind: 'text', value: '{{row.slug}}', monospace: true },
      },
      {
        id: 'env', header: 'Wiring',
        stack: [
          { kind: 'chip', label: '{{row.dsn_label}}', color: '{{row.dsn_color}}', variant: 'outlined' },
          {
            kind: 'text', value: 'init rg: {{row.init_rg_text}}', caption: true,
            visibleWhen: cond('row.init_rg_text', 'is_not_empty'),
          },
        ],
      },
      {
        id: 'db', header: 'Database',
        stack: [
          { kind: 'text', value: '{{row.db_size_text}}' },
          {
            kind: 'text', value: '{{row.db_table_text}}', caption: true,
            visibleWhen: cond('row.db_table_text', 'is_not_empty'),
          },
        ],
      },
      {
        id: 'audit', header: 'Audit copy',
        cell: { kind: 'text', value: '{{row.audit_text}}', caption: true },
      },
      {
        id: 'warn', header: 'Warnings',
        stack: [
          {
            kind: 'chip', label: '{{row.warnings_label}}', color: 'warning', variant: 'outlined',
            visibleWhen: cond('row.warnings_count', 'greater_than', 0),
          },
          {
            kind: 'chip', label: 'OK', color: 'success', variant: 'outlined',
            visibleWhen: cond('row.warnings_count', 'less_than', 1),
          },
        ],
      },
    ],
  });

  // --- main layout ---
  const main = layout('page_root', {
    page_root: {
      type: 'Column',
      children: ['rg_caption', 'rg_grid', 'tenants_caption', 'tenants_grid'],
      style: { gap: '16px' },
    },
  });

  return {
    name: 'Lakehouse status',
    slug: 'lakehouse-status',
    description:
      'Read-only: cluster health, resource groups, per-tenant wiring, and cross-check warnings. Built in Page Studio.',
    version: 1,
    isCore: true,
    status: 'published',
    layout: main,
    tabs: [],
    components,
    dataSources: [],
    presentationEvents: [],
    filterBar: layout('top_root', {
      top_root: { type: 'Column', children: ['top_header', 'note_info', 'cluster_caption', 'cluster_kv'], style: { gap: '12px' } },
      top_header: { type: 'Row', children: ['hdr'], style: { alignItems: 'center', gap: '12px', flexWrap: 'wrap' } },
    }),
    app: {
      chrome: 'none' as const,
      surface: { maxWidth: 1400, padding: 3 },
      variables: [],
      queries: [
        // Single platform query — every widget on the page binds to {{queries.lhStart.data.*}}.
        // Multiple queries with the same operation create duplicate-query warnings at runtime;
        // bindings reach into the shared data path instead.
        { id: 'lhStart', operation: 'lakehouseStatus.platform' },
        // Tenant handler: different endpoint, different shape. Always-on: the tenant handler
        // strips the cluster + RGs sections; if it's reached, only this query runs.
        { id: 'lhStartTenant', operation: 'lakehouseStatus.tenant' },
      ],
    },
  };
}

/**
 * lakehouseStatusBlueprint is the entry the blueprints index calls. structuredClone lets
 * a "From blueprint" draft start from this page without leaking shared references into
 * the saved JSON (the page-designer-handoff docs warn about this exact failure mode).
 */
export const lakehouseStatusBlueprint = (): Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'> =>
  structuredClone(lakehouseStatusBlueprintCore());