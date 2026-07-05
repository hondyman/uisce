import { describe, it, expect, vi, beforeEach } from 'vitest';
import { abbreviationApiClient } from '../../utils/abbreviationApi';

vi.mock('../../utils/apiClient', () => ({
  default: vi.fn(() => Promise.reject(new Error('API Error: 500 relation "sml.abbreviation_lookup" does not exist'))),
}));

describe('abbreviationApi dev fallback', () => {
  const tenantId = 'test-tenant-123';

  beforeEach(() => {
    localStorage.clear();
  });

  it('returns gold-copy core abbreviations when the backend fails', async () => {
    const result = await abbreviationApiClient.getAbbreviations(50, 0, '', tenantId);

    expect(result.total_count).toBeGreaterThan(0);
    expect(result.items.length).toBeGreaterThan(0);
    expect(result.items.some((a) => a.is_core)).toBe(true);
    expect(result.items.find((a) => a.abbreviation === 'AMT')?.full_word).toBe('AMOUNT');
  });

  it('persists custom tenant abbreviations locally when the backend fails', async () => {
    await abbreviationApiClient.addAbbreviation('FOO', 'FOOBAR', 'demo note', tenantId);

    const result = await abbreviationApiClient.getAbbreviations(50, 0, '', tenantId);
    const custom = result.items.filter((a) => !a.is_core);

    expect(custom.length).toBe(1);
    expect(custom[0].abbreviation).toBe('FOO');
    expect(custom[0].full_word).toBe('FOOBAR');
    expect(custom[0].tenant_id).toBe(tenantId);
  });

  it('updates locally persisted abbreviations', async () => {
    await abbreviationApiClient.addAbbreviation('FOO', 'FOOBAR', '', tenantId);
    let list = await abbreviationApiClient.getAbbreviations(50, 0, '', tenantId);
    const custom = list.items.find((a) => !a.is_core)!;

    await abbreviationApiClient.updateAbbreviation(custom.id, 'FOO', 'FOOBAR_UPDATED', 'updated', tenantId);
    list = await abbreviationApiClient.getAbbreviations(50, 0, '', tenantId);
    const updated = list.items.find((a) => a.id === custom.id);

    expect(updated?.full_word).toBe('FOOBAR_UPDATED');
    expect(updated?.notes).toBe('updated');
  });

  it('deletes locally persisted abbreviations', async () => {
    await abbreviationApiClient.addAbbreviation('FOO', 'FOOBAR', '', tenantId);
    let list = await abbreviationApiClient.getAbbreviations(50, 0, '', tenantId);
    const custom = list.items.find((a) => !a.is_core)!;

    await abbreviationApiClient.deleteAbbreviation(custom.id, tenantId);
    list = await abbreviationApiClient.getAbbreviations(50, 0, '', tenantId);

    expect(list.items.some((a) => a.id === custom.id)).toBe(false);
  });
});
