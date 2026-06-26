import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import PropertyEditor from '../PropertyEditor';
import { vi } from 'vitest';

vi.mock('../../contexts/TenantContext', () => ({ useTenant: () => ({ tenant: { id: 't-1' } }) }));
vi.mock('../../api/lookups', () => ({ useLookupValues: (tenantId?: string, lookupId?: string, parentId?: string | null, parentValue?: string | null) => {
  const items = [
    { id: 'a', name: 'A', parent_id: null },
    { id: 'b', name: 'B', parent_id: 'parent-x' }
  ];
  // Simulate server-side cascade filtering based on parentId
  const filtered = parentId ? items.filter(i => i.parent_id === parentId) : items.filter(i => i.parent_id == null);
  return { data: filtered, isLoading: false };
} }));

describe('PropertyEditor cascade lookups', () => {
  test('filters lookup values based on cascade_from parent property', async () => {
    const lookupProp = {
      name: 'subdomain', label: 'Subdomain', data_type: 'string', nullable: true, input_type: 'lookup', lookup_id: 'lu-domains', cascade_from: 'domain'
    } as any;

    const onChange = vi.fn();

    render(<PropertyEditor property={lookupProp} value={''} onChange={onChange} allProperties={{ domain: 'parent-x' }} />);

    // Should show only the option for id 'b'
    const select = screen.getByLabelText('Subdomain');
    fireEvent.mouseDown(select);
    expect(screen.queryByText('A')).toBeNull();
    expect(screen.getByText('B')).toBeTruthy();
  });
});
