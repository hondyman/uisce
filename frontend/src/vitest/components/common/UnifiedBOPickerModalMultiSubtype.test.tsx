import { describe, it, expect, vi } from 'vitest';
import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { UnifiedBOPickerModal } from '../../../components/common/UnifiedBOPickerModal';

vi.mock('../../../features/data-explorer/services/dataExplorerApi', () => ({
  fetchBusinessObjects: vi.fn().mockResolvedValue([
    { id: 'bo-account', name: 'account', displayName: 'Account', description: 'Account BO' },
  ]),
  fetchJSON: vi.fn().mockResolvedValue({
    bindings: [{ id: 'default-postgres-binding', name: 'Primary', isDefault: true }],
    businessObject: {
      id: 'bo-account',
      name: 'account',
      displayName: 'Account',
      description: 'Account BO',
      subtypes: {
        institutional: { displayName: 'Institutional', subtypeFields: [{ name: 'sponsor_id' }] },
        retail_wealth: { displayName: 'Retail Wealth', subtypeFields: [{ name: 'advisor_id' }] },
      },
    },
    related_bos: [],
  }),
}));

describe('UnifiedBOPickerModal multi-subtype support', () => {
  it('allows selecting multiple subtypes for context="page" and reports them all', async () => {
    const onSelect = vi.fn();
    render(<UnifiedBOPickerModal open context="page" onClose={() => {}} onSelect={onSelect} />);

    await waitFor(() => expect(screen.getByText('Institutional')).toBeInTheDocument());

    fireEvent.click(screen.getByText('Institutional'));
    fireEvent.click(screen.getByText('Retail Wealth'));
    fireEvent.click(screen.getByText(/Confirm & Create Page/i));

    expect(onSelect).toHaveBeenCalledTimes(1);
    const [, , , , , selectedSubtypeKeys] = onSelect.mock.calls[0];
    expect(selectedSubtypeKeys).toEqual(expect.arrayContaining(['institutional', 'retail_wealth']));
    expect(selectedSubtypeKeys).toHaveLength(2);
  });

  it('keeps single-select radio behavior for context="report"', async () => {
    const onSelect = vi.fn();
    render(<UnifiedBOPickerModal open context="report" onClose={() => {}} onSelect={onSelect} />);

    await waitFor(() => expect(screen.getByText('Institutional')).toBeInTheDocument());

    fireEvent.click(screen.getByText('Institutional'));
    fireEvent.click(screen.getByText(/Confirm & Create Report/i));

    expect(onSelect).toHaveBeenCalledTimes(1);
    const [, , , , selectedSubtypeKey, selectedSubtypeKeys] = onSelect.mock.calls[0];
    expect(selectedSubtypeKey).toBe('institutional');
    expect(selectedSubtypeKeys).toEqual(['institutional']);
  });
});
