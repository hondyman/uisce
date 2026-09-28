import React from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { registerOperations, type OperationDef } from '../../studio-core/operations/registry';
import { registerDomainComponents } from '../../studio-core/components/registry';
import { type ConfigChange, type ConfigKind, type ConfigRow, isSeries, masteringApi, masteringConfigApi } from './api';
import ConfigRowEditor, { type EditorMode } from './ConfigRowEditor';

/**
 * Mastering configuration in Page Studio: the Vendor registry, Source
 * hierarchy and Match rules core pages (blueprints/mdmConfig.ts). The domain
 * shapes rows for display (group, scope, keys, origin) and decides who may do
 * what (own / core / overridden, mine / can decide); pages only display.
 * Changes are maker-checker: proposed here, applied when a second
 * administrator approves (backend internal/mastering/configedit.go).
 */

const str = (p: Record<string, unknown>, k: string) => (p[k] === undefined || p[k] === null ? '' : String(p[k]));
const kindOf = (p: Record<string, unknown>) => str(p, 'kind') as ConfigKind;
const blank = (v: unknown) => v === undefined || v === null || v === '' || v === '*' || v === 'ALL';

/** The column a hierarchy row is grouped by: field group (records) or price type (prices). */
const GROUP_COLS = ['field_group', 'price_type_cd'];
const SETTING_LABELS: Record<string, string> = { is_fallback: 'fallback', is_active: 'active', max_staleness_minutes: 'stale after' };

function fuzzyText(v: unknown): string[] {
  return Array.isArray(v)
    ? v.map((k) => (k && typeof k === 'object'
      ? `${(k as Record<string, unknown>).field} (${(k as Record<string, unknown>).method}, ${(k as Record<string, unknown>).weight})`
      : String(k)))
    : [];
}

/** One configuration row, flattened for a grid. */
function displayRow(r: ConfigRow, keyCols: string[]) {
  const v = r.values;
  const group = GROUP_COLS.map((c) => v[c]).find((x) => !blank(x));
  const scope = keyCols.filter((c) => !GROUP_COLS.includes(c) && c !== 'source' && c !== 'rule_cd' && c !== 'code' && c !== 'price_entity_type' && !blank(v[c]))
    .map((c) => String(v[c]));
  const settings = Object.entries(SETTING_LABELS)
    .filter(([c]) => c in v && c !== 'is_active')
    .map(([c, l]) => (typeof v[c] === 'boolean' ? (v[c] ? l : '') : v[c] === null || v[c] === undefined ? '' : `${l} ${v[c]} min`))
    .filter(Boolean);
  const meta = (v.metadata && typeof v.metadata === 'object' ? v.metadata : {}) as Record<string, unknown>;
  return {
    ...v,
    id: r.id,
    row: r,
    origin: r.origin,
    state: r.overridden ? 'overridden' : r.inherited ? 'core' : 'own',
    state_label: r.overridden ? 'Core · overridden' : r.inherited ? 'Core' : r.origin === 'core' ? 'Core (gold copy)' : 'Yours',
    pending: r.pending,
    active: v.is_active !== false,
    group: group === undefined ? '' : String(group),
    scope_label: scope.join(' · ') || 'All',
    settings,
    deterministic: Array.isArray(v.deterministic_keys) ? (v.deterministic_keys as unknown[]).map(String) : [],
    fuzzy: fuzzyText(v.fuzzy_keys),
    description: String(meta.description ?? ''),
  };
}

/** What a proposal changes, one line per column: "priority: 10 → 1". */
function changeLines(c: ConfigChange): string[] {
  if (c.action === 'delete') return [];
  const before = (c.before ?? {}) as Record<string, unknown>;
  return Object.entries(c.values ?? {}).map(([k, v]) => {
    const show = (x: unknown) => (x === undefined || x === null ? '∅' : typeof x === 'object' ? JSON.stringify(x) : String(x));
    return k in before ? `${k}: ${show(before[k])} → ${show(v)}` : `${k}: ${show(v)}`;
  });
}

function changeRow(c: ConfigChange, canDecide: boolean) {
  const v = { ...(c.before ?? {}), ...(c.values ?? {}) } as Record<string, unknown>;
  const subject = [v.rule_cd, v.code, GROUP_COLS.map((g) => v[g]).find((x) => !blank(x)), v.source].filter((x) => !blank(x)).join(' · ');
  return {
    ...c,
    subject: subject || (c.target_id ?? '').slice(0, 8),
    what: c.action === 'delete' ? 'Remove' : c.target_id ? 'Change' : 'Add / override',
    lines: changeLines(c),
    // Only an administrator who did not propose it may decide.
    state: c.status !== 'pending' ? 'closed' : c.mine ? 'mine' : canDecide ? 'can_decide' : 'waiting',
    requested_by_label: c.requested_by_name || c.requested_by,
    reviewed_by_label: c.reviewed_by_name || '—',
  };
}

const KIND = { name: 'kind', type: 'string' as const, required: true, description: 'source_system, source_priority or match_rule' };
const ENTITY = { name: 'entity', type: 'string' as const, description: 'Empty for the vendor registry' };

const operations: OperationDef[] = [
  {
    id: 'mdmConfig.entities', domain: 'mdmcfg', kind: 'query', label: 'Entities with this configuration',
    description: 'Mastered entities for a configuration kind (match rules: record entities only).', params: [KIND],
    fields: [{ name: 'entity' }, { name: 'display_name' }],
    run: async (p) => (await masteringApi.profiles()).profiles
      .filter((x) => kindOf(p) !== 'match_rule' || !isSeries(x))
      .map((x) => ({ entity: x.entity_cd.toLowerCase(), display_name: x.display_name })),
  },
  {
    id: 'mdmConfig.table', domain: 'mdmcfg', kind: 'query', label: 'Configuration rows',
    description: 'Returns {can_edit, rows}. state: own (edit/remove), core (read-only; override), overridden (core, replaced by an own row).',
    params: [KIND, ENTITY],
    fields: [
      { name: 'id' }, { name: 'state' }, { name: 'origin' }, { name: 'pending', type: 'boolean' }, { name: 'active', type: 'boolean' },
      { name: 'group' }, { name: 'scope_label' }, { name: 'source' }, { name: 'priority', type: 'number' }, { name: 'settings', type: 'array' },
      { name: 'rule_cd' }, { name: 'rule_name' }, { name: 'deterministic', type: 'array' }, { name: 'fuzzy', type: 'array' },
      { name: 'threshold_auto_match', type: 'number' }, { name: 'threshold_review', type: 'number' },
      { name: 'code' }, { name: 'display_name' }, { name: 'feed_type' }, { name: 'description' },
    ],
    run: async (p) => {
      const { table, can_edit } = await masteringConfigApi.table(kindOf(p), str(p, 'entity') || undefined);
      const keyCols = table.columns.filter((c) => c.key).map((c) => c.name);
      return { can_edit, rows: table.rows.map((r) => displayRow(r, keyCols)) };
    },
  },
  {
    id: 'mdmConfig.changes', domain: 'mdmcfg', kind: 'query', label: 'Configuration changes',
    description: 'status pending, or empty for the history. state: mine (withdraw), can_decide (approve/reject), waiting (not an administrator), closed.',
    params: [KIND, ENTITY, { name: 'status', type: 'string' }],
    fields: [
      { name: 'id' }, { name: 'subject' }, { name: 'what' }, { name: 'lines', type: 'array' }, { name: 'reason' }, { name: 'status' },
      { name: 'state' }, { name: 'requested_by_label' }, { name: 'requested_at', type: 'datetime' }, { name: 'reviewed_by_label' },
      { name: 'review_comment' },
    ],
    run: async (p) => {
      const r = await masteringConfigApi.changes({ kind: kindOf(p), entity: str(p, 'entity') || undefined, status: str(p, 'status') || undefined });
      return r.changes.map((c) => changeRow(c, r.can_decide));
    },
  },
  {
    id: 'mdmConfig.proposeRemove', domain: 'mdmcfg', kind: 'mutation', label: 'Propose removing a row',
    description: 'Only the tenant\'s own rows; applied when another administrator approves.',
    params: [KIND, ENTITY, { name: 'id', type: 'string', required: true }, { name: 'reason', type: 'string' }],
    run: (p) => masteringConfigApi.propose({
      kind: kindOf(p), entity: str(p, 'entity') || undefined, action: 'delete', target_id: str(p, 'id'), reason: str(p, 'reason'),
    }),
  },
  ...(['approve', 'reject', 'withdraw'] as const).map((d): OperationDef => ({
    id: `mdmConfig.${d}`, domain: 'mdmcfg', kind: 'mutation', label: `${d[0].toUpperCase()}${d.slice(1)} a configuration change`,
    description: d === 'approve' ? 'Applies the change. Never the proposer.' : undefined,
    params: [{ name: 'id', type: 'string', required: true }, { name: 'comment', type: 'string' }],
    run: (p) => masteringConfigApi.decide(str(p, 'id'), d, str(p, 'comment') || undefined),
  })),
];

registerOperations(operations);

// The editor proposes through the API itself; refresh the page's queries
// (domain prefix) when it is done.
function EditorOverlay({ inputs, emit }: { inputs: Record<string, unknown>; emit: (e: string) => void }) {
  const qc = useQueryClient();
  if (!inputs.open) return null;
  const row = (inputs.row as { row?: ConfigRow } | null)?.row ?? null;
  const done = (event: string) => {
    qc.invalidateQueries({ queryKey: ['mdmcfg'] });
    emit(event);
  };
  return (
    <ConfigRowEditor kind={String(inputs.kind) as ConfigKind} entity={(inputs.entity as string) || undefined}
      mode={(String(inputs.mode || 'new')) as EditorMode} row={row}
      onClose={() => emit('close')} onProposed={() => done('proposed')} />
  );
}

registerDomainComponents([
  {
    id: 'mdmConfig.RowEditor', domain: 'mdmcfg', label: 'Propose a configuration row', overlay: true,
    description: 'New row, change to an own row, or override of a gold-copy row - sent for approval. Form from the table\'s columns.',
    inputs: [
      { name: 'open', type: 'boolean' },
      { name: 'kind', type: 'string', required: true },
      { name: 'entity', type: 'string' },
      { name: 'mode', type: 'string', description: 'new, edit or override' },
      { name: 'row', type: 'object', description: 'The grid row being edited or overridden' },
    ],
    events: [{ name: 'close' }, { name: 'proposed' }],
    render: EditorOverlay,
  },
]);
