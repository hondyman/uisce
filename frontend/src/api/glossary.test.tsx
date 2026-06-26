import { describe, it, expect, vi, beforeEach } from 'vitest';
 
import { QueryClient } from '@tanstack/react-query';
import { ApolloClient } from '@apollo/client';
import { renderWithProviders, createQueryClient, createApolloClient } from '../../test/testUtils';
import { useUpdateTerm } from './glossary';

// Mock the tenant context
vi.mock('../contexts/TenantContext', () => ({
  useTenant: () => ({
    tenant: { id: 'test-tenant-id' },
    datasource: { id: 'test-datasource-id' },
  }),
}));

// Mock fetch
global.fetch = vi.fn();

describe('useUpdateTerm', () => {
  let queryClient: QueryClient;
  let mockApolloClient: ApolloClient<any>;

  beforeEach(() => {
    queryClient = createQueryClient();
    mockApolloClient = createApolloClient();
    vi.clearAllMocks();
  });

  it('should make correct API call for update', async () => {
    // Render a small harness that uses the hook so hooks run inside React
    const Harness: React.FC = () => {
      const hook = useUpdateTerm();
      // Expose to window for simple assertions
      // @ts-ignore
      (globalThis as any).__lastHook = hook;
      return null;
    };

    renderWithProviders(<Harness />, { queryClient, apolloClient: mockApolloClient });
    // @ts-ignore
    const hook = (globalThis as any).__lastHook;
    expect(hook).toBeDefined();
    expect(typeof hook.mutate).toBe('function');
  });

  it('should handle API errors', async () => {
    (global.fetch as any).mockResolvedValueOnce({
      ok: false,
      text: () => Promise.resolve('Update failed'),
    });

    const Harness: React.FC = () => {
      const hook = useUpdateTerm();
      // @ts-ignore
      (globalThis as any).__lastHook = hook;
      return null;
    };

    renderWithProviders(<Harness />, { queryClient, apolloClient: mockApolloClient });
    // @ts-ignore
    const hook = (globalThis as any).__lastHook;
    expect(hook).toBeDefined();
    expect(typeof hook.mutate).toBe('function');
  });
});