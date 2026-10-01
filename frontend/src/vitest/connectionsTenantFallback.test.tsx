/**
 * Checklist line 3: a tenant-required route with no tenant present must render the soft
 * fallback — no crash, and critically NO stale or wrong data.
 *
 * "Does not crash" passes on far too much, so this asserts positively on the fallback
 * surface AND negatively on the data region. The regression being guarded is wrong
 * content being rendered; only the negative assertion catches that.
 *
 * `Connections` is the subject because its guard is the clearest tenant-required gate
 * in the app: `if (!scopedTenant && !isPlatformOperator) return <fallback/>`.
 */
import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import React from 'react';

const NO_TENANT_ACCESS = {
  currentTenant: null,
  setDatasourceScope: () => {},
  isPlatformOperator: false,
};

vi.mock('../contexts/AccessContext', () => ({
  useAccess: () => NO_TENANT_ACCESS,
  useAccessOptional: () => NO_TENANT_ACCESS,
}));

// Keep the data layer inert: this test is about the guard, not about fetching.
vi.mock('../api/apiClient', () => ({ apiFetch: vi.fn().mockResolvedValue([]) }));
vi.mock('../lib/apiClient', () => ({ apiFetch: vi.fn().mockResolvedValue([]) }));

const { default: Connections } = await import('../features/connections/pages/Connections');

const renderConnections = () =>
  render(
    <MemoryRouter>
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}
      >
        <Connections />
      </QueryClientProvider>
    </MemoryRouter>,
  );

describe('Connections with no tenant in scope', () => {
  it('renders the soft fallback instead of the data view', () => {
    renderConnections();

    // Positive: the fallback the guard promises is actually on screen.
    expect(screen.getByText(/select a tenant scope to view connections/i)).toBeInTheDocument();

    // Negative, and this is the load-bearing one: the tenant guard must be what stopped
    // the render, not the loading guard. With a tenant in scope this component renders
    // "Loading..." instead, so asserting its absence is what distinguishes "correctly
    // gated on tenant" from "gated on something else". An earlier draft asserted the
    // "Connection Management" heading was absent, which was vacuous — that heading never
    // renders in this test at all, because the loading guard returns first.
    expect(screen.queryByText(/loading/i)).not.toBeInTheDocument();
    expect(screen.queryByRole('table')).not.toBeInTheDocument();
  });
});
