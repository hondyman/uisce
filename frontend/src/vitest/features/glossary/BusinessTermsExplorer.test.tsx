import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';

// Mock all the data hooks so the component receives stable empty data
// (no real network calls). If the component had an infinite render loop,
// react-testing-library would surface the warning or the test would time out.
vi.mock('@/features/glossary/BusinessTermsExplorer', async () => {
  const actual = await vi.importActual<any>('@/features/glossary/BusinessTermsExplorer');
  return actual;
});

// Mock data hooks used by the component
vi.mock('@/api/glossary', () => ({
  useDeleteTerm: () => ({ mutateAsync: vi.fn() }),
  useUpdateTerm: () => ({ mutateAsync: vi.fn() }),
  useCreateTerm: () => ({ mutateAsync: vi.fn() }),
}));

vi.mock('@/api/nodeTypes', () => ({
  useNodeTypes: () => ({ data: [] }),
}));

vi.mock('@/hooks/usePropertyLookupMaps', () => ({
  usePropertyLookupMaps: () => ({ data: {} }),
}));

vi.mock('@/contexts/AccessContext', () => ({
  useAccess: () => ({
    currentTenant: { id: 't1', display_name: 'Test Tenant' },
    isPlatformOperator: false,
    accessLevel: 'tenant_user',
  }),
}));

vi.mock('@/utils/tenantScope', () => ({
  readCachedSelection: () => ({ tenant: { id: 't1' } }),
}));

vi.mock('@/utils/apiClient', () => ({
  default: vi.fn().mockImplementation((url: string) => {
    if (url.includes('node_type_id=21645d21')) {
      return Promise.resolve([
        { id: 'bt-1', node_name: 'Test Term', description: 'Test description', node_type_id: '21645d21-de5f-4feb-af99-99273ea75626' },
      ]);
    }
    return Promise.resolve([]);
  }),
}));

import BusinessTermsExplorer from '@/features/glossary/BusinessTermsExplorer';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';

describe('BusinessTermsExplorer', () => {
  it('mounts without infinite rendering', () => {
    const qc = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });

    // Spy on console.error to catch React's "Maximum update depth" warning
    const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

    render(
      <QueryClientProvider client={qc}>
        <MemoryRouter initialEntries={['/core/business-terms']}>
          <BusinessTermsExplorer />
        </MemoryRouter>
      </QueryClientProvider>
    );

    // Verify the component's outer chrome rendered
    expect(screen.getByText('Business Terms Glossary')).toBeInTheDocument();

    // Verify no "Maximum update depth" warning was logged
    const maxUpdateWarnings = consoleErrorSpy.mock.calls.filter((call) =>
      String(call[0] || '').includes('Maximum update depth')
    );
    expect(maxUpdateWarnings).toHaveLength(0);

    consoleErrorSpy.mockRestore();
  });

  it('declares the three-tab structure (details, relationships, lineage)', () => {
    // Static check that the tab set is consistent — pill onClicks now route
    // through handleTabChange() rather than calling setActiveTab inline.
    const fs = require('fs');
    const path = require('path');
    const src = fs.readFileSync(
      path.resolve(__dirname, '../../../features/glossary/BusinessTermsExplorer.tsx'),
      'utf8'
    );
    expect(src).toContain("handleTabChange('details')");
    expect(src).toContain("handleTabChange('relationships')");
    expect(src).toContain("handleTabChange('lineage')");
    // Properties tab should be gone (rolled into Details)
    expect(src).not.toContain("activeTab === 'properties'");
  });
});
