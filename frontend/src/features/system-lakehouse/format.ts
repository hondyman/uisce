import type { LakehouseAuditEntry, LakehouseConfig, LifecycleState } from './api';

// "Domains decide, pages display": the derived text, state and action flags a row
// needs live here, not in the page.

/** 2555 -> "7 years (2,555 days)"; 400 -> "400 days"; null -> "Not set". */
export function retentionText(days: number | null | undefined): string {
  if (days === null || days === undefined) return 'Not set';
  const n = days.toLocaleString('en-US');
  if (days >= 365 && days % 365 === 0) {
    const years = days / 365;
    return `${years} ${years === 1 ? 'year' : 'years'} (${n} days)`;
  }
  return `${n} ${days === 1 ? 'day' : 'days'}`;
}

const STATE_LABELS: Record<LifecycleState, string> = {
  unconfigured: 'Not configured',
  provisioning: 'Configured, not provisioned',
  active: 'Active',
  suspended: 'Suspended',
  offboarding: 'Offboarding',
  offboarded: 'Offboarded',
};

export function stateLabel(state: string): string {
  return STATE_LABELS[state as LifecycleState] ?? state;
}

export interface LakehouseRow {
  id: string;
  tenant_id: string;
  name: string;
  code_text: string;
  state: string;
  state_label: string;
  retention_days: number | null;
  retention_text: string;
  warehouse_name: string;
  provisioned: boolean;
  /** Retention is set and there is nothing yet provisioned. */
  can_provision: boolean;
  /** Why provisioning is unavailable, shown under the button; empty when it is available. */
  provision_hint: string;
}

export function toRow(c: LakehouseConfig): LakehouseRow {
  const retentionSet = c.audit_retention_days !== null && c.audit_retention_days !== undefined;
  const can = c.configured && retentionSet && !c.provisioned && c.lifecycle_state === 'provisioning';
  let hint = '';
  if (!can) {
    if (c.provisioned) hint = 'Already provisioned';
    else if (!retentionSet) hint = 'Set the audit retention first';
    else hint = `Not available while ${stateLabel(c.lifecycle_state).toLowerCase()}`;
  }
  return {
    id: c.tenant_id,
    tenant_id: c.tenant_id,
    name: c.tenant_name || c.tenant_code || c.tenant_id,
    code_text: c.tenant_code ?? '',
    state: c.lifecycle_state,
    state_label: stateLabel(c.lifecycle_state),
    retention_days: c.audit_retention_days ?? null,
    retention_text: retentionText(c.audit_retention_days),
    warehouse_name: c.warehouse_name ?? '',
    provisioned: c.provisioned,
    can_provision: can,
    provision_hint: hint,
  };
}

/** What an audit entry changed, in words. */
export function auditChangeText(e: Pick<LakehouseAuditEntry, 'action' | 'before' | 'after'>): string {
  const before = e.before?.audit_retention_days;
  const after = e.after?.audit_retention_days;
  switch (e.action) {
    case 'configured':
      return after !== undefined ? `Audit retention set to ${retentionText(after)}` : 'Lakehouse configured';
    case 'retention_extended':
      return `Audit retention extended from ${retentionText(before)} to ${retentionText(after)}`;
    case 'provision_requested':
      return 'Provisioning requested';
    case 'provisioned':
      return 'Provisioned';
    case 'state_changed':
      return 'Lifecycle state changed';
    default:
      return e.action;
  }
}

/** What to tell an admin about the integrity of the audit chain. */
export function chainText(intact: boolean, firstBrokenId?: number): string {
  return intact
    ? 'Audit chain verified: every entry links to the one before it.'
    : `Audit chain broken at entry ${firstBrokenId ?? 'unknown'}: an entry was altered or removed.`;
}
