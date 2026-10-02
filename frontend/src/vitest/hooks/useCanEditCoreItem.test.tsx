import { renderHook, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';

const state = { tenantId: 'tenant-a' as string | undefined, goldCopy: 'gold-1' as string | null };

vi.mock('../../contexts/TenantContext', () => ({
  useTenant: () => ({ tenant: state.tenantId ? { id: state.tenantId } : null }),
}));
vi.mock('../../utils/goldCopy', () => ({
  resolveGoldCopyTenantId: () => Promise.resolve(state.goldCopy),
}));

import { useCanEditCoreItem } from '../../hooks/useCanEditCoreItem';

describe('useCanEditCoreItem', () => {
  beforeEach(() => {
    state.tenantId = 'tenant-a';
    state.goldCopy = 'gold-1';
  });

  it('fails closed until the gold-copy id is resolved', () => {
    const { result } = renderHook(() => useCanEditCoreItem({ tenant_id: 'tenant-a' }));
    expect(result.current.canEdit).toBe(false);
  });

  it('lets a tenant edit its own items once resolved', async () => {
    const { result } = renderHook(() => useCanEditCoreItem({ tenant_id: 'tenant-a' }));
    await waitFor(() => expect(result.current.canEdit).toBe(true));
    expect(result.current.isCore).toBe(false);
  });

  it('blocks a regular tenant from editing gold-copy items', async () => {
    const { result } = renderHook(() => useCanEditCoreItem({ tenant_id: 'gold-1' }));
    await waitFor(() => expect(result.current.isCore).toBe(true));
    expect(result.current.canEdit).toBe(false);
    expect(result.current.disabledReason).toMatch(/gold copy/i);
  });

  it('allows the gold-copy tenant itself to edit core items', async () => {
    state.tenantId = 'gold-1';
    const { result } = renderHook(() => useCanEditCoreItem({ tenant_id: 'gold-1' }));
    await waitFor(() => expect(result.current.canEdit).toBe(true));
    expect(result.current.isGoldCopyTenant).toBe(true);
  });

  it('does not treat "no tenant" as the gold-copy tenant when the id is unresolved', async () => {
    state.tenantId = undefined;
    state.goldCopy = null;
    const { result } = renderHook(() => useCanEditCoreItem({ tenant_id: undefined, is_core: true }));
    await waitFor(() => expect(result.current.isCore).toBe(true));
    expect(result.current.isGoldCopyTenant).toBe(false);
    expect(result.current.canEdit).toBe(false);
  });
});
