/**
 * Regression test for the 2b fix: TenantsManagementPage's entitlement guard.
 *
 * The guard is an early return, and the page's load effect calls
 * `api.fetchTenantIPWhitelist` once per tenant. Hoisting that effect above the guard
 * makes the hook count stable, but on its own it would fire one request per tenant for
 * a user who is immediately redirected away and never sees the page.
 *
 * So both halves are asserted here, and they pull in opposite directions on purpose:
 *   isVisible=false -> <Navigate> renders AND zero whitelist requests are made
 *   isVisible=true  -> the page renders AND the fetch DOES happen
 *
 * The second case is the discrimination control. Without it this test could pass for
 * the wrong reason (e.g. if the fetch were wired up to never run at all). If the first
 * case starts failing, or the second stops firing the fetch, something is wrong.
 */
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import React from 'react';

const fetchTenantIPWhitelist = vi.fn().mockResolvedValue([]);

// Stable reference: the load effect depends on `accessibleTenants`. If the mock built a
// fresh array on every call the effect would re-run on every render and the test would
// measure a render loop rather than the guard.
const ACCESSIBLE_TENANTS = [{
  id: 't-1',
  name: 'Acme',
  display_name: 'Acme',
  // isActiveTenant() requires an explicit active flag; without it the tenant is
  // filtered out and the load loop never runs, which would make the "no fetch"
  // assertion below pass for the wrong reason.
  is_active: true,
  is_deleted: false,
}];

const SCOPE = { level: 'all', tenantId: null };

let isVisible = false;

vi.mock('../contexts/useOrganizationEntitlement', () => ({
  useOrganizationEntitlement: () => ({
    level: 'none',
    canRead: true,
    canWrite: false,
    isVisible,
  }),
}));

vi.mock('../features/fabric/hooks/useIPWhitelist', () => ({
  useIPWhitelistAPI: () => ({
    fetchTenantIPWhitelist,
    deleteTenant: vi.fn().mockResolvedValue(true),
  }),
}));

vi.mock('../contexts/AccessContext', () => ({
  useAccess: () => ({
    accessibleTenants: ACCESSIBLE_TENANTS,
    scope: SCOPE,
    currentTenant: null,
  }),
}));

vi.mock('../contexts/TenantContext', () => ({ useTenant: () => ({ tenant: null }) }));

vi.mock('../hooks/useNotification', () => ({
  useNotification: () => ({ success: vi.fn(), error: vi.fn(), info: vi.fn() }),
}));

vi.mock('../hooks/useRegions', () => ({ useRegions: () => ({ regions: [] }) }));

vi.mock('../lib/apiClient', () => ({ apiFetch: vi.fn().mockResolvedValue({ ok: true }) }));

const { default: TenantsManagementPage } = await import(
  '../features/fabric/pages/TenantsManagementPage'
);

const renderPage = () =>
  render(
    <MemoryRouter>
      <TenantsManagementPage />
    </MemoryRouter>,
  );

describe('TenantsManagementPage entitlement guard', () => {
  beforeEach(() => {
    fetchTenantIPWhitelist.mockClear();
    isVisible = false;
  });

  it('redirects and fetches nothing when the user has no organization entitlement', async () => {
    isVisible = false;
    const { container } = renderPage();

    // The redirect: <Navigate> resolves to a location change, so the page body is empty.
    await waitFor(() => {
      expect(container.textContent).toBe('');
    });
    expect(screen.queryByText('Connection Management')).not.toBeInTheDocument();

    // The behaviour that a bare hoist would have broken: no per-tenant whitelist calls.
    await waitFor(() => {
      expect(fetchTenantIPWhitelist).not.toHaveBeenCalled();
    });
  });

  it('CONTROL: fetches per tenant when the user does have entitlement', async () => {
    isVisible = true;
    renderPage();

    // Proves the previous assertion is discriminating rather than vacuous: with
    // entitlement the effect does run and does call the API.
    await waitFor(() => {
      expect(fetchTenantIPWhitelist).toHaveBeenCalledWith('t-1');
    });
  });
});
