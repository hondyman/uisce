import { describe, it, expect, vi, beforeEach } from 'vitest';
import { apiFetch } from '../../lib/apiClient';
import { getOperation } from '../../studio-core/operations/registry';
import '../../features/data-domains/studio';

vi.mock('../../lib/apiClient', () => ({ apiFetch: vi.fn() }));

const ok = (body: unknown) =>
  new Response(JSON.stringify(body), { status: 200, headers: { 'Content-Type': 'application/json' } });
const parentId = '11111111-1111-4111-8111-111111111111';
const childId = '22222222-2222-4222-8222-222222222222';
const stored = [
  { id: parentId, name: 'Risk', slug: 'risk', parent_id: null, level: 2, description: { String: 'Risk data', Valid: true } },
  { id: childId, name: 'Credit', slug: 'credit', parent_id: parentId, level: 3, description: '' },
];

describe('domains.* Page Studio operations', () => {
  beforeEach(() => {
    vi.mocked(apiFetch).mockReset();
  });

  it('lists domains with the parent name and tolerates the NullString description shape', async () => {
    vi.mocked(apiFetch).mockResolvedValueOnce(ok(stored));
    const out = (await getOperation('domains.list')!.run({})) as { rows: { name: string; parent_name: string; description: string }[] };
    expect(out.rows.find((r) => r.name === 'Credit')).toMatchObject({ parent_name: 'Risk' });
    expect(out.rows.find((r) => r.name === 'Risk')).toMatchObject({ description: 'Risk data' });
  });

  it('creates under a parent with level = parent level + 1 and the NullString description', async () => {
    vi.mocked(apiFetch)
      .mockResolvedValueOnce(ok(stored)) // list
      .mockResolvedValueOnce(new Response(JSON.stringify({}), { status: 201 })); // POST
    await getOperation('domains.save')!.run({
      draft: { name: 'Liquidity', slug: '', parent_id: parentId, description: 'Cash and funding' },
    });
    const [url, init] = vi.mocked(apiFetch).mock.calls[1];
    expect(url).toBe('/api/data-domains');
    expect(init).toMatchObject({ method: 'POST' });
    const body = JSON.parse(String((init as RequestInit).body));
    expect(body).toMatchObject({ name: 'Liquidity', parent_id: parentId, level: 3 });
    expect(body.description).toEqual({ String: 'Cash and funding', Valid: true });
    expect(body.slug).toBeTruthy();
  });

  it('updates in place, keeping the stored slug when the draft leaves it blank', async () => {
    vi.mocked(apiFetch)
      .mockResolvedValueOnce(ok(stored))
      .mockResolvedValueOnce(new Response('{}', { status: 200 }));
    await getOperation('domains.save')!.run({
      draft: { id: childId, name: 'Credit risk', slug: '', parent_id: null, description: '' },
    });
    const [url, init] = vi.mocked(apiFetch).mock.calls[1];
    expect(url).toBe(`/api/data-domains/${childId}`);
    expect(init).toMatchObject({ method: 'PUT' });
    expect(JSON.parse(String((init as RequestInit).body))).toMatchObject({ slug: 'credit', level: 1 });
  });

  it('refuses a domain as its own parent, and a move under one of its children', async () => {
    vi.mocked(apiFetch).mockImplementation(async () => ok(stored));
    await expect(
      getOperation('domains.save')!.run({ draft: { id: parentId, name: 'Risk', parent_id: parentId } }),
    ).rejects.toThrow('cannot be its own parent');
    await expect(
      getOperation('domains.save')!.run({ draft: { id: parentId, name: 'Risk', parent_id: childId } }),
    ).rejects.toThrow('own children');
    // Neither refusal reaches the API write.
    const writes = vi.mocked(apiFetch).mock.calls.filter(([, init]) => init && (init as RequestInit).method);
    expect(writes).toHaveLength(0);
  });

  it('surfaces the API refusal (for example a non-core administrator) as the error', async () => {
    vi.mocked(apiFetch)
      .mockResolvedValueOnce(ok(stored))
      .mockResolvedValueOnce(new Response('only gold-copy administrators may modify data domains', { status: 403 }));
    await expect(getOperation('domains.save')!.run({ draft: { name: 'X', parent_id: null } })).rejects.toThrow(
      'only gold-copy administrators',
    );
  });
});
