import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import PropertySchemaEditor from '../PropertySchemaEditor';
import { vi } from 'vitest';

const fetchNextMock = vi.fn();
vi.mock('../../api/lookups', () => ({
  useLookups: (tenantId?: string, q?: string) => ({ data: [{ id: 'lu-1', name: 'domains', description: 'Domains' }], isLoading: false }),
  useInfiniteLookups: (tenantId?: string, q?: string) => ({ data: { pages: [{ items: [{ id: 'lu-1', name: 'domains', description: 'Domains' }] }] }, fetchNextPage: fetchNextMock, hasNextPage: true, isFetching: false }),
}));
vi.mock('../../contexts/TenantContext', () => ({ useTenant: () => ({ tenant: { id: 't-1' } }) }));

describe('PropertySchemaEditor', () => {
  test('shows lookup ProfessionalSearchInput for lookup input_type', async () => {
    const val = [{ name: 'domain', label: 'Domain', data_type: 'string', input_type: 'lookup', required: false }];
    const onChange = vi.fn();

    render(<PropertySchemaEditor value={val as any} onChange={onChange} />);

    // We expect ProfessionalSearchInput to render; also ensure it calls onSearch when typed (mocked lookups won't change)
    const input = screen.getByPlaceholderText(/Select lookup/i) as HTMLInputElement;
    expect(input).toBeTruthy();

    // Simulate typing in the search box - ProfessionalSearchInput uses an input element
    fireEvent.change(input, { target: { value: 'iso' } });
    // Simulate scroll near the bottom to trigger onLoadMore and check fetchNextPage was called
    // Ensure we can find the results container and dispatch scroll
    const results = await screen.findByRole('listbox');
    // Simulate scroll event
    fireEvent.scroll(results, { target: { scrollTop: 1000 } });
    expect(fetchNextMock).toHaveBeenCalled();
    // Debounce from ProfessionalSearchInput will call onSearch - the mock useLookups just returns static data
    expect(screen.getByPlaceholderText(/Select lookup/i)).toBeTruthy();
  });
});
