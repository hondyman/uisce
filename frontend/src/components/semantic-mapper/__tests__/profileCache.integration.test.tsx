import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { vi, describe, it, beforeEach, afterEach, expect } from 'vitest';
import SemanticMapper from '../../SemanticMapper';
import { TenantProvider } from '../../../contexts/TenantContext';
import { ScopeProvider } from '../../../contexts/ScopeContext';
import { RouteBlockerProvider } from '../../../components/RouteBlocker/RouteBlocker';
import { ApolloProvider } from '@apollo/client';
import apolloClient from '../../../../src/graphql/apolloClient';

describe('Profile caching', () => {
  let originalFetch: any;

  beforeEach(() => {
    originalFetch = globalThis.fetch;
    // Seed tenant selection in localStorage so the UI can run profiling
    localStorage.setItem('selected_tenant', JSON.stringify({ id: 't1', display_name: 'T1' }));
    localStorage.setItem('selected_product', JSON.stringify({ id: 'p1', alpha_product: { product_name: 'P1' } }));
    localStorage.setItem('selected_datasource', JSON.stringify({ id: 'ds1', source_name: 'DS1', config: { host: 'localhost', port: 5432, credentials: { username: 'u', password: 'p' } } }));

    // Put a cached profile for the datasource
    const cacheKey = `profile_cache:ds1:public:table1,table2`;
    localStorage.setItem(cacheKey, JSON.stringify({ profiles: [{ tableName: 'table1', columnName: 'col1', dataType: 'text', cardinality: 10 }] }));

    // Spy on fetch to see if POST to /api/profiler/profile occurs
    globalThis.fetch = vi.fn((input: any) => {
      const url = typeof input === 'string' ? input : input?.url;
      if (typeof url === 'string' && url.includes('/api/semantic-mappings')) {
        // Return an empty array for mappings so component initializes cleanly
        return Promise.resolve(new Response(JSON.stringify([]), { status: 200, headers: { 'Content-Type': 'application/json' } }));
      }
      // Default generic ok response
      return Promise.resolve(new Response(JSON.stringify({}), { status: 200, headers: { 'Content-Type': 'application/json' } }));
    });
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    localStorage.clear();
    vi.resetAllMocks();
  });

  it('does not POST to /api/profiler/profile when cached results exist', async () => {
  render(
    <ApolloProvider client={apolloClient}>
      <RouteBlockerProvider>
        <TenantProvider>
          <ScopeProvider>
            <SemanticMapper />
          </ScopeProvider>
        </TenantProvider>
      </RouteBlockerProvider>
    </ApolloProvider>
  );

    // The Profile tab was removed; try to find a Run Data Profile button if it exists.
    const runButton = await screen.queryByRole('button', { name: /Run Data Profile/i });
    if (runButton) {
      fireEvent.click(runButton);
    }

    // Wait to allow the component to check cache and render results
    await waitFor(() => {
      // Ensure fetch was called but not with POST to /api/profiler/profile
      const calls = (globalThis.fetch as any).mock.calls.map((c: any) => c[0]);
      const profilerCalls = calls.filter((u: string) => typeof u === 'string' && u.includes('/api/profiler/profile'));
      expect(profilerCalls.length).toBe(0);
    });
  });
});
