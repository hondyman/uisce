import { describe, expect, it } from 'vitest';
import type { LakehouseConfig } from '../../features/system-lakehouse/api';
import { apiMessage } from '../../features/system-lakehouse/api';
import { auditChangeText, chainText, retentionText, stateLabel, toRow } from '../../features/system-lakehouse/format';

const cfg = (over: Partial<LakehouseConfig> = {}): LakehouseConfig => ({
  tenant_id: 't-1', tenant_name: 'Acme', tenant_code: 'acme', configured: true,
  audit_retention_days: 2555, retention_applied_days: null, retention_pending: false,
  lifecycle_state: 'provisioning', provisioned: false, warehouse_name: 'ivy-t-abc', ...over,
});

describe('retentionText', () => {
  it('says years when the days are whole years, and always the days', () => {
    expect(retentionText(2555)).toBe('7 years (2,555 days)');
    expect(retentionText(3650)).toBe('10 years (3,650 days)');
    expect(retentionText(365)).toBe('1 year (365 days)');
  });
  it('says only days otherwise', () => {
    expect(retentionText(400)).toBe('400 days');
    expect(retentionText(1)).toBe('1 day');
  });
  it('says "Not set" rather than inventing a default', () => {
    expect(retentionText(null)).toBe('Not set');
    expect(retentionText(undefined)).toBe('Not set');
  });
});

describe('toRow', () => {
  it('lets a configured, unprovisioned tenant be provisioned', () => {
    const r = toRow(cfg());
    expect(r.can_provision).toBe(true);
    expect(r.provision_hint).toBe('');
    expect(r.retention_text).toBe('7 years (2,555 days)');
    expect(r.name).toBe('Acme');
  });

  it('never offers provisioning before retention is set, and says why', () => {
    const r = toRow(cfg({ audit_retention_days: null }));
    expect(r.can_provision).toBe(false);
    expect(r.provision_hint).toBe('Set the audit retention first');
  });

  it('never offers provisioning for a tenant with no lakehouse row', () => {
    const r = toRow(cfg({ configured: false, audit_retention_days: null, lifecycle_state: 'unconfigured', warehouse_name: undefined }));
    expect(r.can_provision).toBe(false);
    expect(r.state_label).toBe('Not configured');
    expect(r.warehouse_name).toBe('');
  });

  it('does not offer provisioning twice', () => {
    const r = toRow(cfg({ provisioned: true, lifecycle_state: 'active' }));
    expect(r.can_provision).toBe(false);
    expect(r.provision_hint).toBe('Already provisioned');
  });

  it('does not offer provisioning while offboarding or suspended', () => {
    for (const state of ['suspended', 'offboarding', 'offboarded'] as const) {
      expect(toRow(cfg({ lifecycle_state: state })).can_provision).toBe(false);
    }
  });

  it('falls back to code then id when a tenant has no name', () => {
    expect(toRow(cfg({ tenant_name: '' })).name).toBe('acme');
    expect(toRow(cfg({ tenant_name: '', tenant_code: '' })).name).toBe('t-1');
  });
});

describe('retention sync', () => {
  const provisioned = (over: Partial<LakehouseConfig> = {}) =>
    cfg({ provisioned: true, lifecycle_state: 'active', retention_applied_days: 365, audit_retention_days: 2555, retention_pending: true, ...over });

  it('shows what the bucket enforces and offers a sync while it is behind', () => {
    const r = toRow(provisioned());
    expect(r.retention_text).toBe('7 years (2,555 days)');
    expect(r.applied_text).toBe('1 year (365 days)');
    expect(r.retention_pending).toBe(true);
    expect(r.can_sync).toBe(true);
  });

  it('offers nothing once the bucket has caught up', () => {
    const r = toRow(provisioned({ retention_applied_days: 2555, retention_pending: false }));
    expect(r.applied_text).toBe('7 years (2,555 days)');
    expect(r.retention_pending).toBe(false);
    expect(r.can_sync).toBe(false);
  });

  it('never offers a sync for a tenant with no bucket, whatever the flag says', () => {
    const r = toRow(cfg({ provisioned: false, retention_pending: true, retention_applied_days: null }));
    expect(r.can_sync).toBe(false);
    expect(r.retention_pending).toBe(false);
    expect(r.applied_text).toBe(''); // nothing is enforced before there is a bucket
  });

  it('says plainly when a provisioned bucket reports nothing applied', () => {
    expect(toRow(provisioned({ retention_applied_days: null })).applied_text).toBe('Not set');
  });
});

describe('audit text', () => {
  it('describes each action in words', () => {
    expect(auditChangeText({ action: 'configured', after: { audit_retention_days: 365 } })).toBe('Audit retention set to 1 year (365 days)');
    expect(auditChangeText({ action: 'retention_extended', before: { audit_retention_days: 365 }, after: { audit_retention_days: 2555 } }))
      .toBe('Audit retention extended from 1 year (365 days) to 7 years (2,555 days)');
    expect(auditChangeText({ action: 'provision_requested' })).toBe('Provisioning requested');
    expect(auditChangeText({ action: 'retention_applied', before: { retention_applied_days: 365 }, after: { retention_applied_days: 2555 } }))
      .toBe('Bucket now enforces 7 years (2,555 days) by default (was 1 year (365 days))');
    expect(auditChangeText({ action: 'retention_applied', after: { retention_applied_days: 365 } }))
      .toBe('Bucket now enforces 1 year (365 days) by default (was Not set)');
    expect(auditChangeText({ action: 'retention_sync_failed', after: { step: 'ExtendBucketRetention', error: 'governance mode' } }))
      .toBe("Raising the bucket's retention failed at ExtendBucketRetention: governance mode. The bucket still enforces its previous retention; fix the cause and sync again.");
    expect(auditChangeText({ action: 'provision_failed', after: { step: 'EnsureLakehouseBucket', error: 'bucket exists without Object Lock' } }))
      .toBe('Provisioning failed at EnsureLakehouseBucket: bucket exists without Object Lock. Fix the cause and provision again; steps that already finished are safe to re-run.');
    expect(auditChangeText({ action: 'provision_failed' })).toBe('Provisioning failed. Fix the cause and provision again; steps that already finished are safe to re-run.');
    expect(auditChangeText({ action: 'something_new' })).toBe('something_new');
  });

  it('reports a broken chain loudly and an intact one plainly', () => {
    expect(chainText(true)).toMatch(/verified/);
    expect(chainText(false, 7)).toBe('Audit chain broken at entry 7: an entry was altered or removed.');
  });

  it('passes an unknown state through rather than hiding it', () => {
    expect(stateLabel('mystery')).toBe('mystery');
  });
});

describe('apiMessage', () => {
  it("extracts the API's own message from apiClient's error envelope", () => {
    const e = new Error('API Error: 409 Conflict - {"code":"retention_lowered","error":"audit retention can only be extended once set"}');
    expect(apiMessage(e)).toBe('audit retention can only be extended once set');
  });
  it('keeps the raw text when there is no JSON body', () => {
    expect(apiMessage(new Error('API Error: 502 Bad Gateway'))).toBe('API Error: 502 Bad Gateway');
    expect(apiMessage(new Error('API Error: 500 - {not json'))).toBe('API Error: 500 - {not json');
    expect(apiMessage('plain')).toBe('plain');
  });
});
