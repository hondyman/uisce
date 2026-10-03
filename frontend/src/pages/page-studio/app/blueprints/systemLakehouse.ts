import type { ComponentDefinition, CorePageDefinition, PageLayout } from '../../../../types/pageStudio';
import type { Action, CellSpec, ColumnDef, ConditionNode } from '../appModel';

/**
 * System > Tenant lakehouse (ADR-032): every tenant's one Iceberg warehouse, its
 * per-tenant audit retention (no default; can only be extended), provisioning, and
 * the audit trail of changes. For a new or an existing tenant alike; the list is
 * every tenant, configured or not.
 *
 * Built only from Page Studio blocks. All of its data is control-plane metadata in
 * alpha, served by the global-admin System API through the operations in
 * features/system-lakehouse/studio.ts; the page carries no URLs.
 */

const cond = (field: string, operator: string, value?: unknown): ConditionNode => ({ type: 'condition', field, operator, value });
const set = (name: string, value: unknown = null): Action => ({ kind: 'setVariable', name, value });
const op = (operation: string, params: Record<string, unknown>, onSuccess: Action[] = [], more: Record<string, unknown> = {}): Action =>
  ({ kind: 'runOperation', operation, params, onSuccess, ...more } as Action);
const col = (id: string, header: string, cell: CellSpec | CellSpec[], more: Partial<ColumnDef> = {}): ColumnDef =>
  Array.isArray(cell) ? { id, header, stack: cell, ...more } : { id, header, cell, ...more };

const fit = { flex: '0 0 auto' };

type NodeSpec = { type: 'Row' | 'Column' | 'Dialog' | 'Drawer'; children: string[]; style?: Record<string, string>; props?: Record<string, unknown> };
const layout = (root: string, spec: Record<string, NodeSpec>): PageLayout => ({
  root,
  nodes: Object.fromEntries(Object.entries(spec).map(([id, n]) => [id, { id, ...n }])),
});

const START = 'queries.lhStart.data';
const AUDIT = 'queries.lhAudit.data';
const DRAFT = 'vars.lhDraft';

function systemLakehousePage(): Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'> {
  const components: Record<string, ComponentDefinition> = {};
  const w = (id: string, type: string, props: Record<string, unknown>, extra: Partial<ComponentDefinition> = {}) => {
    components[id] = { id, type, props, ...extra };
  };

  w('hdr', 'PageHeader', {
    icon: 'storage',
    title: 'Tenant lakehouse',
    subtitle: 'One Iceberg warehouse per tenant. Audit retention is set per tenant, has no default, and can only be extended.',
  }, { style: { flex: '1 1 320px' } });
  w('q', 'SearchInput', { variable: 'q', placeholder: 'Search tenants', maxWidth: 420 }, { style: { flex: '1 1 260px' } });
  w('info', 'AlertBanner', {
    severity: 'info',
    text: 'Metadata for every tenant stays in alpha. This page configures where a tenant\'s own data and its immutable audit copy are kept.',
  });

  // --- the list of tenants ----------------------------------------------------------
  w('grid', 'DataGrid', {
    query: 'lakehouses', rowsPath: 'rows', rowKey: 'id', emptyText: '{{queries.lakehouses.data.empty_text}}',
    columns: [
      col('tenant', 'Tenant', [
        { kind: 'text', value: '{{row.name}}', bold: true },
        { kind: 'text', value: '{{row.code_text}}', caption: true, visibleWhen: cond('row.code_text', 'is_not_empty') },
      ]),
      col('state', 'State', {
        kind: 'chip', value: '{{row.state}}', label: '{{row.state_label}}', variant: 'outlined',
        colorMap: { active: 'success', provisioning: 'warning', suspended: 'warning', offboarding: 'warning', unconfigured: 'default', offboarded: 'default', '*': 'default' },
      }),
      col('retention', 'Audit retention', { kind: 'text', value: '{{row.retention_text}}' }),
      col('warehouse', 'Warehouse', { kind: 'text', value: '{{row.warehouse_name}}', caption: true }),
      col('act', '', {
        kind: 'actions', buttons: [
          { label: 'Set retention', icon: 'edit', onClick: [set('lhTenant', '{{row.tenant_id}}'), set('lhOpen', true)] },
          {
            label: 'Provision', icon: 'play', visibleWhen: cond('row.can_provision', 'is_true'),
            onClick: [op('systemLakehouse.provision', { tenant_id: '{{row.tenant_id}}' }, [], {
              confirm: {
                title: 'Provision lakehouse',
                text: 'Create the bucket, key, credential and Iceberg warehouse for {{row.name}} with audit retention of {{row.retention_text}}? Compliance retention cannot be shortened once objects are locked.',
                confirmLabel: 'Provision',
              },
              successMessage: 'Provisioning started',
            })],
          },
          { label: 'Audit trail', icon: 'review', onClick: [set('auditTenant', '{{row.tenant_id}}'), set('auditOpen', true)] },
        ],
      }, { align: 'right', nowrap: true }),
    ] satisfies ColumnDef[],
  });

  // --- set / extend retention (dialog) ----------------------------------------------
  const closeEditor = [set('lhOpen', false), set('lhTenant'), set('lhDraft')];
  w('lh_note', 'AlertBanner', { severity: 'warning', text: `{{${START}.note}}` }, { visibleWhen: cond(`${START}.note`, 'is_not_empty') });
  w('lh_form', 'Form', {
    variable: 'lhDraft', initFrom: `{{${START}.draft}}`, seedKey: `{{${START}.key}}`,
    fields: [
      {
        name: 'audit_retention_days', kind: 'number', label: 'Audit retention (days)', required: true, step: 1, wide: true,
        helperText: 'A whole number of days. 2555 is seven years. Once set it can only be extended.',
      },
    ],
  });

  // --- audit trail (drawer) ---------------------------------------------------------
  w('au_ok', 'AlertBanner', { severity: 'success', text: `{{${AUDIT}.chain_text}}` }, { visibleWhen: cond(`${AUDIT}.chain_intact`, 'is_true') });
  w('au_broken', 'AlertBanner', { severity: 'error', text: `{{${AUDIT}.chain_text}}` }, { visibleWhen: cond(`${AUDIT}.chain_intact`, 'is_false') });
  w('au_grid', 'DataGrid', {
    query: 'lhAudit', rowsPath: 'rows', rowKey: 'id', emptyText: `{{${AUDIT}.empty_text}}`,
    columns: [
      col('at', 'When', { kind: 'datetime', value: '{{row.at}}' }, { nowrap: true }),
      col('who', 'Who', [
        { kind: 'text', value: '{{row.actor}}' },
        { kind: 'text', value: '{{row.role}}', caption: true, visibleWhen: cond('row.role', 'is_not_empty') },
      ]),
      col('change', 'Change', { kind: 'text', value: '{{row.change}}' }),
    ] satisfies ColumnDef[],
  });

  const closeAudit = [set('auditOpen', false), set('auditTenant')];
  const nodes: Record<string, NodeSpec> = {
    top_root: { type: 'Column', children: ['top_header', 'info', 'lh_dialog', 'au_drawer'], style: { gap: '8px' } },
    top_header: { type: 'Row', children: ['hdr', 'q'], style: { alignItems: 'center', gap: '12px', flexWrap: 'wrap' } },
    lh_dialog: {
      type: 'Dialog', children: ['lh_body'], props: {
        title: `{{${START}.title}}`, maxWidth: 'sm', openWhen: cond('vars.lhOpen', 'is_true'), onClose: closeEditor,
        buttons: [
          { label: 'Cancel', onClick: closeEditor },
          {
            label: 'Save', variant: 'contained',
            disabledWhen: cond(`${DRAFT}.audit_retention_days`, 'is_empty'),
            onClick: [op('systemLakehouse.setRetention',
              { tenant_id: '{{vars.lhTenant}}', audit_retention_days: `{{${DRAFT}.audit_retention_days}}` },
              closeEditor, { successMessage: 'Audit retention saved' })],
          },
        ],
      },
    },
    lh_body: { type: 'Column', children: ['lh_note', 'lh_form'], style: { gap: '12px' } },
    au_drawer: {
      type: 'Drawer', children: ['au_body'], props: {
        title: 'Lakehouse audit trail', subtitle: 'Changes to this tenant\'s lakehouse configuration, newest first', width: 640,
        openWhen: cond('vars.auditOpen', 'is_true'), onClose: closeAudit,
      },
    },
    au_body: { type: 'Column', children: ['au_ok', 'au_broken', 'au_grid'], style: { gap: '12px' } },
  };

  const main = layout('page_root', { page_root: { type: 'Column', children: ['grid'], style: { gap: '12px' } } });
  return {
    name: 'Tenant lakehouse',
    slug: 'system-lakehouse',
    description: "System: each tenant's one Iceberg warehouse and its per-tenant audit retention, for new and existing tenants. Built in Page Studio.",
    version: 1,
    isCore: true,
    status: 'published',
    layout: main,
    tabs: [],
    filterBar: layout('top_root', nodes),
    components,
    dataSources: [],
    presentationEvents: [],
    app: {
      chrome: 'none' as const,
      surface: { maxWidth: 1400, padding: 3 },
      variables: [
        { name: 'q', default: '' },
        { name: 'lhOpen', default: false },
        { name: 'lhTenant', description: 'The tenant whose retention is being edited' },
        { name: 'lhDraft', description: 'The retention form' },
        { name: 'auditOpen', default: false },
        { name: 'auditTenant', description: "The tenant whose audit trail is open" },
      ],
      queries: [
        { id: 'lakehouses', operation: 'systemLakehouse.list', params: { q: '{{vars.q}}' }, keepPrevious: true, debounceMs: 300 },
        { id: 'lhStart', operation: 'systemLakehouse.editorStart', params: { tenant_id: '{{vars.lhTenant}}' }, enabledWhen: cond('vars.lhOpen', 'is_true') },
        { id: 'lhAudit', operation: 'systemLakehouse.audit', params: { tenant_id: '{{vars.auditTenant}}' }, enabledWhen: cond('vars.auditOpen', 'is_true') },
      ],
    },
  };
}

export const systemLakehouseBlueprint = () => structuredClone(systemLakehousePage());
