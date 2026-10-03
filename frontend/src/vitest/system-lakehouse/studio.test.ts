import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('../../features/system-lakehouse/api', async (orig) => {
  const real = await orig<typeof import('../../features/system-lakehouse/api')>();
  return {
    ...real,
    systemLakehouseApi: { list: vi.fn(), get: vi.fn(), setRetention: vi.fn(), provision: vi.fn(), audit: vi.fn() },
  };
});

import { systemLakehouseApi } from '../../features/system-lakehouse/api';
import { parseDays } from '../../features/system-lakehouse/studio';
import { getOperation } from '../../studio-core/operations/registry';

const api = systemLakehouseApi as unknown as Record<keyof typeof systemLakehouseApi, ReturnType<typeof vi.fn>>;
const run = (id: string, params: Record<string, unknown> = {}) => getOperation(id)!.run(params);
const config = (over = {}) => ({
  tenant_id: 't-1', tenant_name: 'Acme', tenant_code: 'acme', configured: true, audit_retention_days: 365,
  lifecycle_state: 'provisioning', provisioned: false, warehouse_name: 'ivy-t-x', version: 3, ...over,
});

beforeEach(() => Object.values(api).forEach((f) => f.mockReset()));

describe('operations are registered', () => {
  it('registers every id the page uses, with the right kind', () => {
    expect(getOperation('systemLakehouse.list')?.kind).toBe('query');
    expect(getOperation('systemLakehouse.editorStart')?.kind).toBe('query');
    expect(getOperation('systemLakehouse.audit')?.kind).toBe('query');
    expect(getOperation('systemLakehouse.setRetention')?.kind).toBe('mutation');
    expect(getOperation('systemLakehouse.provision')?.kind).toBe('mutation');
  });
});

describe('parseDays', () => {
  it('accepts a whole number, as a number or a string', () => {
    expect(parseDays(2555)).toBe(2555);
    expect(parseDays('2555')).toBe(2555);
    expect(parseDays(' 30 ')).toBe(30);
  });
  it('rejects everything else, so no default can creep in', () => {
    for (const bad of ['', '   ', null, undefined, 'abc', 1.5, '1.5', 0, -3, NaN]) {
      expect(() => parseDays(bad), String(bad)).toThrow(/whole number of days/);
    }
  });
});

describe('systemLakehouse.list', () => {
  it('maps tenants to rows and passes the search through', async () => {
    api.list.mockResolvedValue({ items: [config(), config({ tenant_id: 't-2', audit_retention_days: null, configured: false, lifecycle_state: 'unconfigured' })], total: 2, limit: 50, offset: 0 });
    const r = (await run('systemLakehouse.list', { q: 'ac' })) as { rows: { id: string; can_provision: boolean }[]; total: number; empty_text: string };
    expect(api.list).toHaveBeenCalledWith({ q: 'ac', limit: 50, offset: 0 });
    expect(r.rows.map((x) => [x.id, x.can_provision])).toEqual([['t-1', true], ['t-2', false]]);
    expect(r.total).toBe(2);
  });

  it('words the empty state by whether a search was made', async () => {
    api.list.mockResolvedValue({ items: [], total: 0, limit: 50, offset: 0 });
    expect(((await run('systemLakehouse.list', { q: 'zz' })) as { empty_text: string }).empty_text).toMatch(/matches that search/);
    expect(((await run('systemLakehouse.list', {})) as { empty_text: string }).empty_text).toMatch(/no tenants/);
  });
});

describe('systemLakehouse.editorStart', () => {
  it('seeds the form with the current retention and warns it can only be extended', async () => {
    api.get.mockResolvedValue(config({ audit_retention_days: 365 }));
    const r = (await run('systemLakehouse.editorStart', { tenant_id: 't-1' })) as { key: string; title: string; draft: { audit_retention_days: unknown }; note: string };
    expect(r.title).toBe('Lakehouse: Acme');
    expect(r.draft.audit_retention_days).toBe(365);
    expect(r.note).toMatch(/extended, never shortened/);
    expect(r.key).toBe('t-1:3');
  });

  it('starts empty, with no default, when nothing is set', async () => {
    api.get.mockResolvedValue(config({ audit_retention_days: null, configured: false, version: undefined }));
    const r = (await run('systemLakehouse.editorStart', { tenant_id: 't-1' })) as { draft: { audit_retention_days: unknown }; note: string };
    expect(r.draft.audit_retention_days).toBe('');
    expect(r.note).toMatch(/No retention is set/);
  });

  it('requires a tenant', async () => {
    await expect(run('systemLakehouse.editorStart', {})).rejects.toThrow(/tenant_id is required/);
    expect(api.get).not.toHaveBeenCalled();
  });
});

describe('mutations', () => {
  it('setRetention sends a validated whole number for the named tenant', async () => {
    api.setRetention.mockResolvedValue(config({ audit_retention_days: 2555 }));
    await run('systemLakehouse.setRetention', { tenant_id: 't-1', audit_retention_days: '2555' });
    expect(api.setRetention).toHaveBeenCalledWith('t-1', 2555);
  });

  it('setRetention never reaches the API with an invalid number or no tenant', async () => {
    await expect(run('systemLakehouse.setRetention', { tenant_id: 't-1', audit_retention_days: '' })).rejects.toThrow(/whole number/);
    await expect(run('systemLakehouse.setRetention', { tenant_id: 't-1', audit_retention_days: 0 })).rejects.toThrow(/whole number/);
    await expect(run('systemLakehouse.setRetention', { audit_retention_days: 30 })).rejects.toThrow(/tenant_id is required/);
    expect(api.setRetention).not.toHaveBeenCalled();
  });

  it("setRetention surfaces the API's refusal to lower retention", async () => {
    api.setRetention.mockRejectedValue(new Error('audit retention can only be extended once set'));
    await expect(run('systemLakehouse.setRetention', { tenant_id: 't-1', audit_retention_days: 30 })).rejects.toThrow(/only be extended/);
  });

  it('provision calls the API for exactly the named tenant', async () => {
    api.provision.mockResolvedValue({ workflow_id: 'w', tenant_id: 't-1' });
    await run('systemLakehouse.provision', { tenant_id: 't-1' });
    expect(api.provision).toHaveBeenCalledWith('t-1');
    await expect(run('systemLakehouse.provision', {})).rejects.toThrow(/tenant_id is required/);
  });
});

describe('systemLakehouse.audit', () => {
  it('shapes entries and carries the chain verdict', async () => {
    api.audit.mockResolvedValue({
      chain_intact: false, first_broken_id: 4,
      entries: [{ id: 5, at: '2026-10-02T00:00:00Z', actor_id: 'alice', actor_role: 'global_admin', action: 'retention_extended',
        before: { audit_retention_days: 365 }, after: { audit_retention_days: 730 }, prev_hash: 'a', hash: 'b' }],
    });
    const r = (await run('systemLakehouse.audit', { tenant_id: 't-1' })) as { rows: { id: string; actor: string; change: string }[]; chain_intact: boolean; chain_text: string };
    expect(r.rows[0]).toMatchObject({ id: '5', actor: 'alice' });
    expect(r.rows[0].change).toMatch(/extended from 1 year/);
    expect(r.chain_intact).toBe(false);
    expect(r.chain_text).toMatch(/broken at entry 4/);
  });
});
