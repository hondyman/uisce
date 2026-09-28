import React from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { registerOperations, type OperationDef } from '../../studio-core/operations/registry';
import { registerDomainComponents } from '../../studio-core/components/registry';
import { msgcatApi } from '../message-catalog/api';
import { type Binding, type Change, diffFields, stagingBindingsApi } from './api';
import BindingEditor from './BindingEditor';

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

const operations: OperationDef[] = [
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

// The editor refreshes its own query keys when it proposes; the page's
// queries live under the domain prefix, so refresh those on close too.
function EditorOverlay({ inputs, emit }: { inputs: Record<string, unknown>; emit: (e: string) => void }) {
  const qc = useQueryClient();
  if (!inputs.open) return null;
  const b = inputs.binding as Binding | null | undefined;
  const close = () => {
    qc.invalidateQueries({ queryKey: ['sb'] });
    emit('close');
  };
  return <BindingEditor key={b?.id ?? 'new'} open onClose={close} binding={b?.id ? b : undefined} />;
}

registerDomainComponents([
  {
    id: 'stagingBindings.BindingEditor', domain: 'sb', label: 'Propose a binding', overlay: true,
    description: 'Map a business object\'s fields to a staging table\'s columns (with suggestions) and send the proposal for approval.',
    inputs: [
      { name: 'open', type: 'boolean' },
      { name: 'binding', type: 'object', description: 'The binding to change; empty = a new binding' },
    ],
    events: [{ name: 'close' }],
    render: EditorOverlay,
  },
]);
