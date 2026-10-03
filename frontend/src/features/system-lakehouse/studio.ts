import { registerOperations, type OperationDef } from '../../studio-core/operations/registry';
import { systemLakehouseApi } from './api';
import { auditChangeText, chainText, retentionText, toRow } from './format';

/**
 * Operations behind System > Tenant lakehouse (ADR-032). Everything they read and
 * write is control-plane metadata in alpha, through the global-admin System API;
 * the page itself carries no URLs, and authorization stays in that API.
 */

const DOMAIN = 'systemLakehouse';
const PAGE_SIZE = 50;

const need = (p: Record<string, unknown>, name: string): string => {
  const v = p[name];
  if (typeof v !== 'string' || v === '') throw new Error(`${name} is required`);
  return v;
};

/** The retention form gives a string or a number; only a whole number of days is valid. */
export function parseDays(raw: unknown): number {
  const n = typeof raw === 'number' ? raw : Number(String(raw ?? '').trim());
  if (String(raw ?? '').trim() === '' || !Number.isInteger(n) || n < 1) {
    throw new Error('Enter a whole number of days, 1 or more.');
  }
  return n;
}

const operations: OperationDef[] = [
  {
    id: 'systemLakehouse.list',
    domain: DOMAIN,
    kind: 'query',
    label: 'List tenant lakehouses',
    description: 'Every tenant with its lakehouse state and audit retention (global admin).',
    params: [
      { name: 'q', label: 'Search', type: 'string' },
      { name: 'offset', label: 'Offset', type: 'number' },
    ],
    rowsPath: 'rows',
    rowFields: [
      { name: 'name', label: 'Tenant', type: 'string' },
      { name: 'code_text', label: 'Code', type: 'string' },
      { name: 'state', label: 'State', type: 'string' },
      { name: 'state_label', label: 'State', type: 'string' },
      { name: 'retention_text', label: 'Audit retention', type: 'string' },
      { name: 'warehouse_name', label: 'Warehouse', type: 'string' },
      { name: 'can_provision', label: 'Can provision', type: 'boolean' },
      { name: 'provision_hint', label: 'Provision hint', type: 'string' },
    ],
    run: async (p) => {
      const q = typeof p.q === 'string' ? p.q : '';
      const offset = Number(p.offset) > 0 ? Number(p.offset) : 0;
      const r = await systemLakehouseApi.list({ q, limit: PAGE_SIZE, offset });
      return { rows: r.items.map(toRow), total: r.total, empty_text: q ? 'No tenant matches that search.' : 'There are no tenants.' };
    },
  },
  {
    id: 'systemLakehouse.editorStart',
    domain: DOMAIN,
    kind: 'query',
    label: 'Start the retention editor',
    description: "One tenant's current lakehouse settings, shaped for the editor dialog.",
    params: [{ name: 'tenant_id', type: 'string', required: true }],
    fields: [
      { name: 'key', type: 'string' },
      { name: 'title', type: 'string' },
      { name: 'draft', type: 'object' },
      { name: 'note', type: 'string' },
    ],
    run: async (p) => {
      const c = await systemLakehouseApi.get(need(p, 'tenant_id'));
      const set = c.audit_retention_days !== null && c.audit_retention_days !== undefined;
      return {
        key: `${c.tenant_id}:${c.version ?? 0}`,
        title: `Lakehouse: ${c.tenant_name || c.tenant_code || c.tenant_id}`,
        draft: { audit_retention_days: c.audit_retention_days ?? '' },
        note: set
          ? `Currently ${retentionText(c.audit_retention_days)}. It can be extended, never shortened.`
          : 'No retention is set. Compliance retention cannot be shortened later, so choose it deliberately.',
      };
    },
  },
  {
    id: 'systemLakehouse.setRetention',
    domain: DOMAIN,
    kind: 'mutation',
    label: 'Set a tenant\'s audit retention',
    description: 'Sets or extends the tenant\'s default audit retention, in days. Audited. It cannot be lowered.',
    params: [
      { name: 'tenant_id', type: 'string', required: true },
      { name: 'audit_retention_days', type: 'number', required: true },
    ],
    run: async (p) => toRow(await systemLakehouseApi.setRetention(need(p, 'tenant_id'), parseDays(p.audit_retention_days))),
  },
  {
    id: 'systemLakehouse.provision',
    domain: DOMAIN,
    kind: 'mutation',
    label: 'Provision a tenant\'s lakehouse',
    description: "Starts creating the tenant's bucket, key, credential and Iceberg warehouse. Needs the retention set first.",
    params: [{ name: 'tenant_id', type: 'string', required: true }],
    run: async (p) => systemLakehouseApi.provision(need(p, 'tenant_id')),
  },
  {
    id: 'systemLakehouse.audit',
    domain: DOMAIN,
    kind: 'query',
    label: 'Lakehouse audit trail',
    description: "A tenant's lakehouse configuration audit trail, newest first, with the chain's integrity.",
    params: [{ name: 'tenant_id', type: 'string', required: true }],
    rowsPath: 'rows',
    rowFields: [
      { name: 'at', label: 'When', type: 'datetime' },
      { name: 'actor', label: 'Who', type: 'string' },
      { name: 'action', label: 'Action', type: 'string' },
      { name: 'change', label: 'Change', type: 'string' },
    ],
    run: async (p) => {
      const a = await systemLakehouseApi.audit(need(p, 'tenant_id'));
      return {
        rows: a.entries.map((e) => ({
          id: String(e.id),
          at: e.at,
          actor: e.actor_id,
          role: e.actor_role,
          action: e.action,
          change: auditChangeText(e),
        })),
        chain_intact: a.chain_intact,
        chain_text: chainText(a.chain_intact, a.first_broken_id),
        empty_text: 'No lakehouse changes have been recorded for this tenant.',
      };
    },
  },
];

registerOperations(operations);
