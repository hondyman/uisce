import { registerOperations, type OperationDef } from '../../studio-core/operations/registry';
import i18n from '../../i18n';
import { msgcatApi } from '../message-catalog/api';
import { type Column, pipelinesApi, platformApi } from '../data-pipelines/api';
import { type Binding, type Change, diffFields, stagingBindingsApi } from './api';

/**
 * The staging-bindings domain's Page Studio surface (the Staging bindings
 * core page, app/blueprints/stagingBindings.ts). Row derivations - who may
 * approve, what a proposal changes, whether a binding has a change waiting -
 * live here, next to the API; the page only displays them.
 */

const str = (p: Record<string, unknown>, k: string) => (p[k] === undefined || p[k] === null ? '' : String(p[k]));

async function meId(): Promise<string | undefined> {
  try {
    return ((await msgcatApi.me()) as { user_id?: string } | undefined)?.user_id;
  } catch {
    return undefined;
  }
}

/** One line per changed field: "field: before → after" (or "unbound"). */
function diffLines(c: Change): string[] {
  if (c.action === 'delete') return [];
  return diffFields(c.before, c.fields)
    .filter((d) => d.kind !== 'same')
    .map((d) => `${d.field}: ${d.before ? `${d.before} → ` : ''}${d.after ?? '∅'}`);
}

function changeRow(c: Change, me?: string) {
  const diff = diffLines(c);
  return {
    ...c,
    state: c.status !== 'pending' ? 'closed' : me && c.requested_by === me ? 'mine' : 'can_vote',
    requested_by_label: c.requested_by_name || c.requested_by,
    reviewed_by_label: c.reviewed_by_name || c.reviewed_by || '—',
    diff,
    diff_count: diff.length,
    field_count: Object.keys(c.fields || {}).length,
    is_delete: c.action === 'delete',
  };
}

const ID = { name: 'id', type: 'string' as const, required: true };

/** Keys mastering reads from a source besides BO fields (see stagingbind/keys.go). */
const SOURCE_KEY = '@source_key';
const AS_OF_KEY = '@as_of';
const t = (k: string, o?: Record<string, unknown>) => i18n.t(k, o) as string;

/** A business object's fields as mapping rows, then the mastering keys every source may bind. */
interface MapRow { key: string; label: string; caption: string; group: string; type?: string }

async function editorRows(boKey: string): Promise<{ rows: MapRow[]; offers_values: boolean }> {
  const { fields } = await platformApi.boSchema(boKey);
  return {
    rows: [
      ...fields.map((f) => ({ key: f.name, label: f.displayName || f.name, caption: `${f.name}${f.type ? ` · ${f.type}` : ''}`, group: 'fields', type: f.type })),
      { key: SOURCE_KEY, label: t('stagingBindings.editor.sourceKey'), caption: SOURCE_KEY, group: 'keys' },
      { key: AS_OF_KEY, label: t('stagingBindings.editor.asOf'), caption: AS_OF_KEY, group: 'keys' },
    ],
    // Price columns are for a time-series (price) object.
    offers_values: boKey === 'price',
  };
}

const tableColumns = async (table: string) => ((await pipelinesApi.stagingTables()).find((x) => x.table === table)?.columns ?? []);

const operations: OperationDef[] = [
  {
    id: 'stagingBindings.businessObjects', domain: 'sb', kind: 'query', label: 'Business objects to bind', params: [],
    fields: [{ name: 'name' }, { name: 'label' }],
    run: async () => (await platformApi.businessObjects()).map((b) => ({ ...b, label: `${b.display_name} (${b.name})` })),
  },
  {
    id: 'stagingBindings.stagingTables', domain: 'sb', kind: 'query', label: 'Staging tables', params: [],
    fields: [{ name: 'table' }],
    run: async () => (await pipelinesApi.stagingTables()).map((x) => ({ table: x.table })),
  },
  {
    id: 'stagingBindings.editorRows', domain: 'sb', kind: 'query', label: 'What a binding maps',
    description: 'Mapping rows {key, label, caption, group}: the business object\'s fields (group fields), then the mastering keys (group keys). offers_values: a price object.',
    params: [{ name: 'bo_key', type: 'string', required: true }],
    fields: [{ name: 'rows', type: 'array' }, { name: 'offers_values', type: 'boolean' }],
    run: (p) => editorRows(str(p, 'bo_key')),
  },
  {
    id: 'stagingBindings.tableColumns', domain: 'sb', kind: 'query', label: 'A staging table\'s columns',
    params: [{ name: 'table', type: 'string', required: true }],
    fields: [{ name: 'name' }, { name: 'type' }],
    run: (p) => tableColumns(str(p, 'table')),
  },
  {
    id: 'stagingBindings.suggest', domain: 'sb', kind: 'mutation', label: 'Suggest a field mapping',
    description: 'Fills fields not already bound with the best-matching column. result.draft: the draft with fields filled; result.hints: per field {value, label, color, tooltip}.',
    params: [{ name: 'draft', type: 'object', required: true, description: '{bo_key, staging_table, fields}' }],
    invalidates: [],
    run: async (p) => {
      const draft = (p.draft ?? {}) as { bo_key?: string; staging_table?: string; fields?: Record<string, string> };
      const [columns, { rows }] = await Promise.all([tableColumns(draft.staging_table ?? ''), editorRows(draft.bo_key ?? '')]);
      const fields = rows.filter((r) => r.group === 'fields');
      const list = await pipelinesApi.suggestMapping(
        columns.map((c) => ({ name: c.name, type: (c.type || 'string') as Column['type'] })),
        fields.map((f) => ({ name: f.key, label: f.label, type: f.type })),
      );
      const best: Record<string, { from: string; confidence: number; reason: string }> = {};
      for (const s of list) if (!best[s.to] || best[s.to].confidence < s.confidence) best[s.to] = s;
      // Fill only fields not already bound; the user reviews every one.
      const mapping = { ...(draft.fields ?? {}) };
      for (const [field, s] of Object.entries(best)) if (!mapping[field]) mapping[field] = s.from;
      return {
        draft: { ...draft, fields: mapping },
        hints: Object.fromEntries(Object.entries(best).map(([field, s]) => [field, {
          value: s.from, tooltip: s.reason, color: s.confidence >= 0.8 ? 'success' : 'warning',
          label: t('stagingBindings.editor.suggested', { pct: Math.round(s.confidence * 100) }),
        }])),
      };
    },
  },
  {
    id: 'stagingBindings.propose', domain: 'sb', kind: 'mutation', label: 'Propose a binding',
    description: 'The bound fields (unbound rows left out) of {bo_key, staging_table, fields, reason}. Nothing changes until another administrator approves.',
    params: [{ name: 'draft', type: 'object', required: true }],
    run: (p) => {
      const d = (p.draft ?? {}) as { bo_key?: string; staging_table?: string; fields?: Record<string, unknown>; reason?: string };
      return stagingBindingsApi.propose({
        bo_key: d.bo_key ?? '', staging_table: d.staging_table ?? '', action: 'upsert', reason: (d.reason ?? '').trim() || undefined,
        fields: Object.fromEntries(Object.entries(d.fields ?? {}).filter(([, c]) => !!c).map(([k, c]) => [k, String(c)])),
      });
    },
  },
  {
    id: 'stagingBindings.list', domain: 'sb', kind: 'query', label: 'Staging bindings',
    description: 'Every binding visible to the tenant (core ones inherited read-only). pending: a change is waiting for approval.',
    params: [],
    fields: [
      { name: 'id' }, { name: 'staging_table' }, { name: 'bo_key' }, { name: 'bo_name' }, { name: 'field_count', type: 'number' },
      { name: 'fields_tooltip', description: 'field ← column, one per line' }, { name: 'origin' }, { name: 'inherited', type: 'boolean' },
      { name: 'pending', type: 'boolean' }, { name: 'version', type: 'number' }, { name: 'updated_at', type: 'datetime' },
    ],
    run: async () => {
      const [{ bindings }, { changes }] = await Promise.all([stagingBindingsApi.list(), stagingBindingsApi.changes('pending')]);
      const waiting = new Set(changes.map((c) => `${c.bo_key}|${c.staging_table}`));
      return bindings.map((b: Binding) => ({
        ...b,
        field_count: Object.keys(b.fields).length,
        fields_tooltip: Object.entries(b.fields).map(([f, c]) => `${f} ← ${c}`).join('\n'),
        pending: waiting.has(`${b.bo_key}|${b.staging_table}`),
      }));
    },
  },
  {
    id: 'stagingBindings.changes', domain: 'sb', kind: 'query', label: 'Proposed binding changes',
    description: 'status pending, or empty for the full history. state: mine (withdraw), can_vote (approve/reject), closed. diff: one line per changed field.',
    params: [{ name: 'status', type: 'string' }],
    fields: [
      { name: 'id' }, { name: 'staging_table' }, { name: 'bo_key' }, { name: 'action' }, { name: 'is_delete', type: 'boolean' },
      { name: 'diff', type: 'array' }, { name: 'diff_count', type: 'number' }, { name: 'field_count', type: 'number' }, { name: 'reason' },
      { name: 'status' }, { name: 'state' }, { name: 'requested_by_label' }, { name: 'requested_at', type: 'datetime' },
      { name: 'reviewed_by_label' }, { name: 'review_comment' },
    ],
    run: async (p) => {
      const [{ changes }, me] = await Promise.all([
        stagingBindingsApi.changes((str(p, 'status') || undefined) as 'pending' | undefined),
        meId(),
      ]);
      return changes.map((c) => changeRow(c, me));
    },
  },
  {
    id: 'stagingBindings.approve', domain: 'sb', kind: 'mutation', label: 'Approve a binding change',
    params: [ID, { name: 'comment', type: 'string' }],
    run: (p) => stagingBindingsApi.approve(str(p, 'id'), str(p, 'comment') || undefined),
  },
  {
    id: 'stagingBindings.reject', domain: 'sb', kind: 'mutation', label: 'Reject a binding change',
    params: [ID, { name: 'comment', type: 'string' }],
    run: (p) => stagingBindingsApi.reject(str(p, 'id'), str(p, 'comment') || undefined),
  },
  {
    id: 'stagingBindings.withdraw', domain: 'sb', kind: 'mutation', label: 'Withdraw my binding change',
    params: [ID],
    run: (p) => stagingBindingsApi.withdraw(str(p, 'id')),
  },
  {
    id: 'stagingBindings.proposeDelete', domain: 'sb', kind: 'mutation', label: 'Propose deleting a binding',
    description: 'Nothing changes until another administrator approves.',
    params: [{ name: 'bo_key', type: 'string', required: true }, { name: 'staging_table', type: 'string', required: true }],
    run: (p) => stagingBindingsApi.propose({ bo_key: str(p, 'bo_key'), staging_table: str(p, 'staging_table'), action: 'delete' }),
  },
];

registerOperations(operations);
