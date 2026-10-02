import { render, screen, fireEvent, within } from '@testing-library/react';
import { describe, it, expect, beforeEach, vi } from 'vitest';

vi.mock('../../contexts/AccessContext', () => ({
  useAccess: () => ({ currentTenant: { id: 't1', name: 'Acme' } }),
}));
vi.mock('../../hooks/useRegions', () => ({
  useRegions: () => ({
    loading: false,
    regions: [
      { value: 'us-east-1', label: 'US East', fromLookup: true },
      { value: 'eu-central-1', label: 'EUR Central', fromLookup: true },
    ],
  }),
}));

import RegionPicker from '../../components/RegionPicker';
import { getSelectedRegion, setSelectedRegion } from '../../lib/region';

describe('RegionPicker', () => {
  beforeEach(() => localStorage.clear());

  it('shows an explicit "Select region" state, never a pre-selected region', () => {
    render(<RegionPicker />);
    expect(screen.getByText('Select region')).toBeTruthy();
    expect(screen.getByText('Region required')).toBeTruthy();
    expect(getSelectedRegion()).toBe('');
  });

  it('shows the region that is actually sent', () => {
    setSelectedRegion('eu-central-1');
    render(<RegionPicker />);
    expect(screen.getByText('EUR Central')).toBeTruthy();
    expect(screen.queryByText('Region required')).toBeNull();
  });

  it('stores the region the user picks', () => {
    render(<RegionPicker />);
    fireEvent.mouseDown(screen.getByRole('combobox'));
    fireEvent.click(within(screen.getByRole('listbox')).getByText('US East'));
    expect(getSelectedRegion()).toBe('us-east-1');
    expect(screen.queryByText('Region required')).toBeNull();
  });

  it('keeps a region that is in use selectable even if the lookup does not list it', () => {
    setSelectedRegion('ap-unknown-9');
    render(<RegionPicker />);
    expect(screen.getByText('ap-unknown-9')).toBeTruthy();
  });
});
