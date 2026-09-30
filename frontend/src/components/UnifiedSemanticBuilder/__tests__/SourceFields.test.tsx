// React import removed (unused)
import { render, screen } from '@testing-library/react';
import { vi } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { TenantProvider } from '../../../contexts/TenantContext';
import SourceFields from '../SourceFields';

// EXPLICIT STUB: useTenant is stubbed, so this file is evidence about SourceFields'
// rendering only — not about the tenant context contract. See fixtures/tenantContextStub.
// The mock also supplies a pass-through TenantProvider, so the existing wrapper below
// keeps working untouched.
vi.mock('../../../contexts/TenantContext', async () => {
  const { tenantContextMockFactory } = await import('../../../vitest/fixtures/tenantContextStub');
  return tenantContextMockFactory();
});

// The ApolloProvider wrapper this test used to carry was removed: @apollo/client is not a
// dependency, `src/graphql/apolloClient` no longer exists, and SourceFields never used it.
// It could only have thrown on import, so this file has never been runnable.
//
// A QueryClientProvider is now supplied because SimpleTableAutocomplete calls useQuery
// and previously blew up on the missing client. No test asserts query behaviour here.
describe('SourceFields', () => {
  it('renders and updates inputs', () => {
    const setFormData = vi.fn();
    const formData = { sourceTable: 's.t', sourceColumn: 'c' };
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    render(
      <QueryClientProvider client={queryClient}>
        <TenantProvider>
          <SourceFields formData={formData} setFormData={setFormData} />
        </TenantProvider>
      </QueryClientProvider>
    );
    expect(screen.getByDisplayValue('s.t')).toBeInTheDocument();
    expect(screen.getByDisplayValue('c')).toBeInTheDocument();
  });
});
